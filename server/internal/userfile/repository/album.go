package repository

import (
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type AlbumRepository struct {
	db *gorm.DB
}

func NewAlbumRepository(db *gorm.DB) *AlbumRepository {
	return &AlbumRepository{db: db}
}

func (r *AlbumRepository) Create(album *model.Album) error {
	return r.db.Create(album).Error
}

func (r *AlbumRepository) FindByID(id uint64) (*model.Album, error) {
	var a model.Album
	err := r.db.Where("id = ?", id).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AlbumRepository) FindByOwner(ownerID uint64, page, pageSize int) ([]model.Album, int64, error) {
	var albums []model.Album
	var total int64
	query := r.db.Model(&model.Album{}).Where("owner_id = ?", ownerID)
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&albums).Error
	// fill file_count
	for i := range albums {
		var cnt int64
		r.db.Model(&model.AlbumFile{}).Where("album_id = ?", albums[i].ID).Count(&cnt)
		albums[i].FileCount = int(cnt)
	}
	return albums, total, err
}

func (r *AlbumRepository) Update(album *model.Album) error {
	return r.db.Save(album).Error
}

func (r *AlbumRepository) Delete(id uint64) error {
	return r.db.Delete(&model.Album{}, id).Error
}

// AlbumFile operations

func (r *AlbumRepository) AddFiles(albumID uint64, fileIDs []uint64) error {
	records := make([]model.AlbumFile, 0, len(fileIDs))
	for _, fid := range fileIDs {
		records = append(records, model.AlbumFile{AlbumID: albumID, FileID: fid})
	}
	return r.db.Create(&records).Error
}

func (r *AlbumRepository) RemoveFile(albumID, fileID uint64) error {
	return r.db.Delete(&model.AlbumFile{}, "album_id = ? AND file_id = ?", albumID, fileID).Error
}

func (r *AlbumRepository) FindFilesByAlbum(albumID uint64, page, pageSize int) ([]model.AlbumFile, int64, error) {
	var files []model.AlbumFile
	var total int64
	query := r.db.Model(&model.AlbumFile{}).Where("album_id = ?", albumID)
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("added_at DESC").Offset(offset).Limit(pageSize).Find(&files).Error
	return files, total, err
}

func (r *AlbumRepository) FindFileIDsByAlbum(albumID uint64) ([]uint64, error) {
	var files []model.AlbumFile
	err := r.db.Where("album_id = ?", albumID).Find(&files).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(files))
	for _, f := range files {
		ids = append(ids, f.FileID)
	}
	return ids, nil
}

func (r *AlbumRepository) IsFileInAlbum(albumID, fileID uint64) bool {
	var count int64
	r.db.Model(&model.AlbumFile{}).Where("album_id = ? AND file_id = ?", albumID, fileID).Count(&count)
	return count > 0
}
