package service

import (
	"github.com/cloudnexus/server/internal/userfile/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
)

type AlbumService struct {
	albumRepo *repository.AlbumRepository
	fileRepo  *repository.FileRepository
}

func NewAlbumService(albumRepo *repository.AlbumRepository, fileRepo *repository.FileRepository) *AlbumService {
	return &AlbumService{albumRepo: albumRepo, fileRepo: fileRepo}
}

type CreateAlbumReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateAlbumReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	CoverFileID *uint64 `json:"cover_file_id,string"`
}

type AddFilesReq struct {
	FileIDs []uint64 `json:"file_ids"`
}

type AlbumWithFiles struct {
	model.Album
	Files []model.File `json:"files"`
}

func (s *AlbumService) Create(ownerID uint64, req CreateAlbumReq) (*model.Album, error) {
	if req.Name == "" {
		return nil, apperrors.NewAppError(400, "相册名称不能为空", apperrors.ErrBadRequest)
	}
	album := &model.Album{
		OwnerID:     ownerID,
		Name:        req.Name,
		Description: req.Description,
	}
	if err := s.albumRepo.Create(album); err != nil {
		return nil, apperrors.NewAppError(500, "创建相册失败", apperrors.ErrInternalServer)
	}
	return album, nil
}

func (s *AlbumService) GetByID(id uint64) (*model.Album, error) {
	album, err := s.albumRepo.FindByID(id)
	if err != nil {
		return nil, apperrors.NewAppError(404, "相册不存在", apperrors.ErrNotFound)
	}
	return album, nil
}

func (s *AlbumService) ListByOwner(ownerID uint64, page, pageSize int) ([]model.Album, int64, error) {
	return s.albumRepo.FindByOwner(ownerID, page, pageSize)
}

func (s *AlbumService) Update(id, userID uint64, req UpdateAlbumReq) (*model.Album, error) {
	album, err := s.albumRepo.FindByID(id)
	if err != nil {
		return nil, apperrors.NewAppError(404, "相册不存在", apperrors.ErrNotFound)
	}
	if album.OwnerID != userID {
		return nil, apperrors.NewAppError(403, "无权操作此相册", apperrors.ErrForbidden)
	}
	if req.Name != nil {
		album.Name = *req.Name
	}
	if req.Description != nil {
		album.Description = *req.Description
	}
	if req.CoverFileID != nil {
		album.CoverFileID = *req.CoverFileID
	}
	if err := s.albumRepo.Update(album); err != nil {
		return nil, apperrors.NewAppError(500, "更新相册失败", apperrors.ErrInternalServer)
	}
	return album, nil
}

func (s *AlbumService) Delete(id, userID uint64) error {
	album, err := s.albumRepo.FindByID(id)
	if err != nil {
		return apperrors.NewAppError(404, "相册不存在", apperrors.ErrNotFound)
	}
	if album.OwnerID != userID {
		return apperrors.NewAppError(403, "无权操作此相册", apperrors.ErrForbidden)
	}
	return s.albumRepo.Delete(id)
}

func (s *AlbumService) AddFiles(albumID, userID uint64, req AddFilesReq) error {
	album, err := s.albumRepo.FindByID(albumID)
	if err != nil {
		return apperrors.NewAppError(404, "相册不存在", apperrors.ErrNotFound)
	}
	if album.OwnerID != userID {
		return apperrors.NewAppError(403, "无权操作此相册", apperrors.ErrForbidden)
	}
	if len(req.FileIDs) == 0 {
		return apperrors.NewAppError(400, "文件列表不能为空", apperrors.ErrBadRequest)
	}
	return s.albumRepo.AddFiles(albumID, req.FileIDs)
}

func (s *AlbumService) RemoveFile(albumID, fileID, userID uint64) error {
	album, err := s.albumRepo.FindByID(albumID)
	if err != nil {
		return apperrors.NewAppError(404, "相册不存在", apperrors.ErrNotFound)
	}
	if album.OwnerID != userID {
		return apperrors.NewAppError(403, "无权操作此相册", apperrors.ErrForbidden)
	}
	return s.albumRepo.RemoveFile(albumID, fileID)
}

func (s *AlbumService) GetFiles(albumID uint64, page, pageSize int) ([]model.File, int64, error) {
	ids, err := s.albumRepo.FindFileIDsByAlbum(albumID)
	if err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return []model.File{}, 0, nil
	}
	// fetch file info from file repo
	files := make([]model.File, 0, len(ids))
	for _, fid := range ids {
		f, err := s.fileRepo.FindByID(fid)
		if err == nil && f != nil {
			files = append(files, *f)
		}
	}
	// simple pagination
	total := int64(len(files))
	start := (page - 1) * pageSize
	if start >= len(files) {
		return []model.File{}, total, nil
	}
	end := start + pageSize
	if end > len(files) {
		end = len(files)
	}
	return files[start:end], total, nil
}
