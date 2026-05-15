package service

import (
	"context"
	"io"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/minio/minio-go/v7"
)

type MusicService struct {
	musicRepo *repository.MusicRepository
	fileRepo  *repository.FileRepository
	minio     *minio.Client
	bucket    string
}

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
