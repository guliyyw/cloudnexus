package repository

import (
	"errors"
	"strings"

	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type MusicRepository struct {
	db *gorm.DB
}

func NewMusicRepository(db *gorm.DB) *MusicRepository {
	return &MusicRepository{db: db}
}

// PublicTrack CRUD

func (r *MusicRepository) CreateTrack(track *model.PublicTrack) error {
	return r.db.Create(track).Error
}

func (r *MusicRepository) FindTrackByID(id uint64) (*model.PublicTrack, error) {
	var t model.PublicTrack
	err := r.db.First(&t, id).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *MusicRepository) FindAllTracks(page, pageSize int) ([]model.PublicTrack, int64, error) {
	var tracks []model.PublicTrack
	var total int64
	query := r.db.Model(&model.PublicTrack{})
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&tracks).Error
	return tracks, total, err
}

func (r *MusicRepository) FindPublicTrackByMetadata(title, artist, album string) (*model.PublicTrack, error) {
	var track model.PublicTrack
	err := r.db.Where(
		"LOWER(TRIM(title)) = ? AND LOWER(TRIM(COALESCE(artist, ''))) = ? AND LOWER(TRIM(COALESCE(album, ''))) = ?",
		strings.ToLower(strings.TrimSpace(title)),
		strings.ToLower(strings.TrimSpace(artist)),
		strings.ToLower(strings.TrimSpace(album)),
	).First(&track).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func (r *MusicRepository) DeleteTrack(id uint64) error {
	return r.db.Delete(&model.PublicTrack{}, id).Error
}

func (r *MusicRepository) FindAudioFilesByUser(userID uint64, page, pageSize int) ([]model.File, int64, error) {
	var files []model.File
	var total int64
	query := r.db.Model(&model.File{}).Where("user_id = ? AND mime_type LIKE ? AND deleted_at IS NULL", userID, "audio/%")
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&files).Error
	return files, total, err
}

// Playlist CRUD

func (r *MusicRepository) CreatePlaylist(pl *model.Playlist) error {
	return r.db.Create(pl).Error
}

func (r *MusicRepository) FindPlaylistByID(id uint64) (*model.Playlist, error) {
	var pl model.Playlist
	err := r.db.First(&pl, id).Error
	if err != nil {
		return nil, err
	}
	return &pl, nil
}

func (r *MusicRepository) FindPlaylistsByOwner(ownerID uint64) ([]model.Playlist, error) {
	var pls []model.Playlist
	err := r.db.Where("owner_id = ?", ownerID).Order("created_at DESC").Find(&pls).Error
	for i := range pls {
		var cnt int64
		r.db.Model(&model.PlaylistTrack{}).Where("playlist_id = ?", pls[i].ID).Count(&cnt)
		pls[i].TrackCount = int(cnt)
	}
	return pls, err
}

func (r *MusicRepository) UpdatePlaylist(pl *model.Playlist) error {
	return r.db.Save(pl).Error
}

func (r *MusicRepository) DeletePlaylist(id uint64) error {
	r.db.Delete(&model.PlaylistTrack{}, "playlist_id = ?", id)
	return r.db.Delete(&model.Playlist{}, id).Error
}

// PlaylistTrack operations

func (r *MusicRepository) AddTrackToPlaylist(pt *model.PlaylistTrack) error {
	return r.db.Create(pt).Error
}

func (r *MusicRepository) RemoveTrackFromPlaylist(playlistID, trackID uint64) error {
	return r.db.Delete(&model.PlaylistTrack{}, "playlist_id = ? AND track_id = ?", playlistID, trackID).Error
}

func (r *MusicRepository) FindTracksByPlaylist(playlistID uint64) ([]model.PlaylistTrack, error) {
	var tracks []model.PlaylistTrack
	err := r.db.Where("playlist_id = ?", playlistID).Order("sort_order ASC").Find(&tracks).Error
	return tracks, err
}
