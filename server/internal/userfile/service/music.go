package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudnexus/server/internal/userfile/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/dhowden/tag"
	"github.com/minio/minio-go/v7"
)

type MusicService struct {
	musicRepo *repository.MusicRepository
	fileRepo  *repository.FileRepository
	minio     *minio.Client
	bucket    string
}

// extinfRe 用于解析 M3U 文件的 #EXTINF 行
var extinfRe = regexp.MustCompile(`#EXTINF:(\d+),(.+)`)

func NewMusicService(musicRepo *repository.MusicRepository, fileRepo *repository.FileRepository, minioClient *minio.Client, bucket string) *MusicService {
	return &MusicService{musicRepo: musicRepo, fileRepo: fileRepo, minio: minioClient, bucket: bucket}
}

type TrackInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration int    `json:"duration"`
	MimeType string `json:"mime_type"`
	FileSize int64  `json:"file_size"`
	Source   string `json:"source"`
}

type LibraryResponse struct {
	Tracks []TrackInfo `json:"tracks"`
	Total  int64       `json:"total"`
}

func (s *MusicService) GetLibrary(userID uint64, source string, page, pageSize int) (*LibraryResponse, error) {
	tracks := make([]TrackInfo, 0)
	var total int64

	if source == "" || source == "public" || source == "all" {
		pubTracks, pubTotal, err := s.musicRepo.FindAllTracks(page, pageSize)
		if err != nil {
			return nil, err
		}
		total += pubTotal
		for _, t := range pubTracks {
			tracks = append(tracks, TrackInfo{
				ID:       strconv.FormatUint(t.ID, 10),
				Title:    t.Title,
				Artist:   t.Artist,
				Album:    t.Album,
				Duration: t.Duration,
				MimeType: t.MimeType,
				FileSize: t.FileSize,
				Source:   "public",
			})
		}
	}

	if source == "" || source == "cloud" || source == "private" || source == "all" {
		cloudFiles, cloudTotal, err := s.musicRepo.FindAudioFilesByUser(userID, page, pageSize)
		if err != nil {
			return nil, err
		}
		total += cloudTotal
		for _, f := range cloudFiles {
			tracks = append(tracks, TrackInfo{
				ID:       strconv.FormatUint(f.ID, 10),
				Title:    f.Name,
				Artist:   "",
				Album:    "",
				Duration: 0,
				MimeType: f.MimeType,
				FileSize: f.Size,
				Source:   "cloud",
			})
		}
	}

	return &LibraryResponse{Tracks: tracks, Total: total}, nil
}

func (s *MusicService) UploadPublicTrack(uploadedBy uint64, title, artist, album string, duration int, storageKey, mimeType string, fileSize int64) (*model.PublicTrack, error) {
	track := &model.PublicTrack{
		Title:      title,
		Artist:     artist,
		Album:      album,
		Duration:   duration,
		StorageKey: storageKey,
		MimeType:   mimeType,
		FileSize:   fileSize,
		UploadedBy: uploadedBy,
	}
	if err := s.musicRepo.CreateTrack(track); err != nil {
		return nil, apperrors.NewAppError(500, "上传音乐失败", apperrors.ErrInternalServer)
	}
	return track, nil
}

func (s *MusicService) DeletePublicTrack(id, userID uint64) error {
	track, err := s.musicRepo.FindTrackByID(id)
	if err != nil {
		return apperrors.NewAppError(404, "曲目不存在", apperrors.ErrNotFound)
	}
	if track.UploadedBy != userID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	return s.musicRepo.DeleteTrack(id)
}

func (s *MusicService) getStorageKey(trackID uint64, source string, userID uint64) (string, string, int64, error) {
	if source == "cloud" {
		f, err := s.fileRepo.FindByID(trackID)
		if err != nil {
			return "", "", 0, apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
		}
		if f.UserID != userID {
			return "", "", 0, apperrors.NewAppError(403, "无权访问", apperrors.ErrForbidden)
		}
		return f.StorageKey, f.MimeType, f.Size, nil
	}
	track, err := s.musicRepo.FindTrackByID(trackID)
	if err != nil {
		return "", "", 0, apperrors.NewAppError(404, "曲目不存在", apperrors.ErrNotFound)
	}
	return track.StorageKey, track.MimeType, track.FileSize, nil
}

func (s *MusicService) StreamAudio(trackID uint64, source string, userID uint64) (io.ReadCloser, string, int64, error) {
	key, mime, size, err := s.getStorageKey(trackID, source, userID)
	if err != nil {
		return nil, "", 0, err
	}
	obj, err := s.minio.GetObject(context.Background(), s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", 0, apperrors.NewAppError(500, "获取音频失败", apperrors.ErrInternalServer)
	}
	return obj, mime, size, nil
}

func (s *MusicService) StreamRange(trackID uint64, source string, userID uint64, start, end int64) (io.ReadCloser, string, int64, error) {
	key, mime, size, err := s.getStorageKey(trackID, source, userID)
	if err != nil {
		return nil, "", 0, err
	}
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(start, end); err != nil {
		return nil, "", 0, err
	}
	obj, err := s.minio.GetObject(context.Background(), s.bucket, key, opts)
	if err != nil {
		return nil, "", 0, apperrors.NewAppError(500, "获取音频失败", apperrors.ErrInternalServer)
	}
	return obj, mime, size, nil
}

// Playlist

type CreatePlaylistReq struct {
	Name     string `json:"name"`
	IsPublic bool   `json:"is_public"`
}

type UpdatePlaylistReq struct {
	Name     *string `json:"name"`
	IsPublic *bool   `json:"is_public"`
}

type AddTrackReq struct {
	TrackID   uint64 `json:"track_id,string"`
	Source    string `json:"source"`
	SortOrder *int   `json:"sort_order"`
}

func (s *MusicService) CreatePlaylist(ownerID uint64, req CreatePlaylistReq) (*model.Playlist, error) {
	if req.Name == "" {
		return nil, apperrors.NewAppError(400, "播放列表名称不能为空", apperrors.ErrBadRequest)
	}
	pl := &model.Playlist{OwnerID: ownerID, Name: req.Name, IsPublic: req.IsPublic}
	if err := s.musicRepo.CreatePlaylist(pl); err != nil {
		return nil, apperrors.NewAppError(500, "创建播放列表失败", apperrors.ErrInternalServer)
	}
	return pl, nil
}

func (s *MusicService) ListPlaylists(ownerID uint64) ([]model.Playlist, error) {
	return s.musicRepo.FindPlaylistsByOwner(ownerID)
}

func (s *MusicService) GetPlaylist(id, userID uint64) (*model.Playlist, []model.PlaylistTrack, error) {
	pl, err := s.musicRepo.FindPlaylistByID(id)
	if err != nil {
		return nil, nil, apperrors.NewAppError(404, "播放列表不存在", apperrors.ErrNotFound)
	}
	if !pl.IsPublic && pl.OwnerID != userID {
		return nil, nil, apperrors.NewAppError(403, "无权访问", apperrors.ErrForbidden)
	}
	tracks, err := s.musicRepo.FindTracksByPlaylist(id)
	return pl, tracks, err
}

func (s *MusicService) UpdatePlaylist(id, userID uint64, req UpdatePlaylistReq) (*model.Playlist, error) {
	pl, err := s.musicRepo.FindPlaylistByID(id)
	if err != nil {
		return nil, apperrors.NewAppError(404, "播放列表不存在", apperrors.ErrNotFound)
	}
	if pl.OwnerID != userID {
		return nil, apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	if req.Name != nil {
		pl.Name = *req.Name
	}
	if req.IsPublic != nil {
		pl.IsPublic = *req.IsPublic
	}
	if err := s.musicRepo.UpdatePlaylist(pl); err != nil {
		return nil, apperrors.NewAppError(500, "更新失败", apperrors.ErrInternalServer)
	}
	return pl, nil
}

func (s *MusicService) DeletePlaylist(id, userID uint64) error {
	pl, err := s.musicRepo.FindPlaylistByID(id)
	if err != nil {
		return apperrors.NewAppError(404, "播放列表不存在", apperrors.ErrNotFound)
	}
	if pl.OwnerID != userID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	return s.musicRepo.DeletePlaylist(id)
}

func (s *MusicService) AddTrackToPlaylist(playlistID, userID uint64, req AddTrackReq) error {
	pl, err := s.musicRepo.FindPlaylistByID(playlistID)
	if err != nil {
		return apperrors.NewAppError(404, "播放列表不存在", apperrors.ErrNotFound)
	}
	if pl.OwnerID != userID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	pt := &model.PlaylistTrack{
		PlaylistID: playlistID,
		TrackID:    req.TrackID,
		Source:     req.Source,
		SortOrder:  sortOrder,
	}
	return s.musicRepo.AddTrackToPlaylist(pt)
}

func (s *MusicService) RemoveTrackFromPlaylist(playlistID, trackID, userID uint64) error {
	pl, err := s.musicRepo.FindPlaylistByID(playlistID)
	if err != nil {
		return apperrors.NewAppError(404, "播放列表不存在", apperrors.ErrNotFound)
	}
	if pl.OwnerID != userID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	return s.musicRepo.RemoveTrackFromPlaylist(playlistID, trackID)
}

func (s *MusicService) GetLyrics(trackID uint64, source string, userID uint64) (string, error) {
	key, _, _, err := s.getStorageKey(trackID, source, userID)
	if err != nil {
		return "", err
	}

	obj, err := s.minio.GetObject(context.Background(), s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return "", apperrors.NewAppError(500, "获取音频失败", apperrors.ErrInternalServer)
	}
	defer obj.Close()

	tmpFile, err := os.CreateTemp("", "lyrics-*.mp3")
	if err != nil {
		return "", apperrors.NewAppError(500, "创建临时文件失败", apperrors.ErrInternalServer)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, obj); err != nil {
		tmpFile.Close()
		return "", apperrors.NewAppError(500, "下载音频失败", apperrors.ErrInternalServer)
	}
	tmpFile.Close()

	f, err := os.Open(tmpPath)
	if err != nil {
		return "", apperrors.NewAppError(500, "读取临时文件失败", apperrors.ErrInternalServer)
	}
	defer f.Close()

	metadata, err := tag.ReadFrom(f)
	if err != nil {
		return "", nil
	}

	lyrics := metadata.Lyrics()
	if lyrics == "" {
		return "", nil
	}
	return lyrics, nil
}

func (s *MusicService) ExportPlaylist(id, userID uint64, format string) (string, error) {
	pl, _, err := s.GetPlaylist(id, userID)
	if err != nil {
		return "", err
	}

	tracks, err := s.musicRepo.FindTracksByPlaylist(id)
	if err != nil {
		return "", apperrors.NewAppError(500, "获取曲目列表失败", apperrors.ErrInternalServer)
	}

	if format == "m3u" {
		var sb strings.Builder
		sb.WriteString("#EXTM3U\n")
		for _, pt := range tracks {
			var title, artist string
			var duration int
			if pt.Source == "cloud" {
				f, fErr := s.fileRepo.FindByID(pt.TrackID)
				if fErr != nil {
					continue
				}
				title = f.Name
				duration = 0
			} else {
				track, tErr := s.musicRepo.FindTrackByID(pt.TrackID)
				if tErr != nil {
					continue
				}
				title = track.Title
				artist = track.Artist
				duration = track.Duration
				if artist != "" {
					title = artist + " - " + title
				}
			}
			sb.WriteString(fmt.Sprintf("#EXTINF:%d,%s\n", duration, title))
			sb.WriteString(fmt.Sprintf("/api/v1/music/tracks/%d/stream?source=%s\n", pt.TrackID, pt.Source))
		}
		return sb.String(), nil
	}

	// JSON format
	type exportTrack struct {
		Title   string `json:"title"`
		Artist  string `json:"artist"`
		Album   string `json:"album"`
		Duration int   `json:"duration"`
		Source  string `json:"source"`
		TrackID uint64 `json:"track_id,string"`
	}
	type exportPlaylist struct {
		Name   string        `json:"name"`
		Tracks []exportTrack `json:"tracks"`
	}

	exportTracks := make([]exportTrack, 0, len(tracks))
	for _, pt := range tracks {
		if pt.Source == "cloud" {
			f, fErr := s.fileRepo.FindByID(pt.TrackID)
			if fErr != nil {
				continue
			}
			exportTracks = append(exportTracks, exportTrack{
				Title:   f.Name,
				Source:  pt.Source,
				TrackID: pt.TrackID,
			})
			continue
		}
		track, tErr := s.musicRepo.FindTrackByID(pt.TrackID)
		if tErr != nil {
			continue
		}
		exportTracks = append(exportTracks, exportTrack{
			Title:    track.Title,
			Artist:   track.Artist,
			Album:    track.Album,
			Duration: track.Duration,
			Source:   pt.Source,
			TrackID:  pt.TrackID,
		})
	}

	ep := exportPlaylist{
		Name:   pl.Name,
		Tracks: exportTracks,
	}

	data, err := json.MarshalIndent(ep, "", "  ")
	if err != nil {
		return "", apperrors.NewAppError(500, "生成 JSON 失败", apperrors.ErrInternalServer)
	}
	return string(data), nil
}

func (s *MusicService) ImportPlaylist(id, userID uint64, data []byte, format string) error {
	pl, err := s.musicRepo.FindPlaylistByID(id)
	if err != nil {
		return apperrors.NewAppError(404, "播放列表不存在", apperrors.ErrNotFound)
	}
	if pl.OwnerID != userID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}

	if format == "m3u" {
		return s.importM3U(id, string(data))
	}

	// JSON format
	type importTrack struct {
		Title  string `json:"title"`
		Artist string `json:"artist"`
		Source string `json:"source"`
	}
	var playlist struct {
		Tracks []importTrack `json:"tracks"`
	}
	if err := json.Unmarshal(data, &playlist); err != nil {
		return apperrors.NewAppError(400, "无效的 JSON 格式", apperrors.ErrBadRequest)
	}

	added := 0
	for _, t := range playlist.Tracks {
		if t.Title == "" {
			continue
		}
		trackID, found, source := s.findTrackByTitleArtist(t.Title, t.Artist, t.Source)
		if !found {
			continue
		}
		pt := &model.PlaylistTrack{
			PlaylistID: id,
			TrackID:    trackID,
			Source:     source,
			SortOrder:  added,
		}
		if err := s.musicRepo.AddTrackToPlaylist(pt); err != nil {
			continue
		}
		added++
	}

	return nil
}

func (s *MusicService) importM3U(playlistID uint64, content string) error {
	lines := strings.Split(content, "\n")

	added := 0
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "#EXTINF") {
			continue
		}
		matches := extinfRe.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}
		title := strings.TrimSpace(matches[2])

		var artist, trackTitle string
		if idx := strings.Index(title, " - "); idx != -1 {
			artist = title[:idx]
			trackTitle = title[idx+3:]
		} else {
			trackTitle = title
		}

		trackID, found, source := s.findTrackByTitleArtist(trackTitle, artist, "public")
		if !found {
			continue
		}
		pt := &model.PlaylistTrack{
			PlaylistID: playlistID,
			TrackID:    trackID,
			Source:     source,
			SortOrder:  added,
		}
		if err := s.musicRepo.AddTrackToPlaylist(pt); err != nil {
			continue
		}
		added++
	}

	return nil
}

func (s *MusicService) findTrackByTitleArtist(title, artist, preferredSource string) (uint64, bool, string) {
	// 当前仅支持公共曲库匹配，cloud source 暂不匹配
	tracks, _, err := s.musicRepo.FindAllTracks(1, 10000)
	if err != nil {
		return 0, false, "public"
	}
	for _, t := range tracks {
		if strings.EqualFold(t.Title, title) && (artist == "" || strings.EqualFold(t.Artist, artist)) {
			return t.ID, true, "public"
		}
	}
	return 0, false, "public"
}
