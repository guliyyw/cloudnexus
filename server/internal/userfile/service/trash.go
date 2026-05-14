package service

import (
	"context"

	"github.com/cloudnexus/server/internal/userfile/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"

	"github.com/minio/minio-go/v7"
)

type TrashService struct {
	fileRepo  *repository.FileRepository
	quotaRepo *repository.QuotaRepository
	quotaSvc  *QuotaService
	minio     *minio.Client
	bucket    string
}

func NewTrashService(
	fileRepo *repository.FileRepository,
	quotaRepo *repository.QuotaRepository,
	quotaSvc *QuotaService,
	minioClient *minio.Client,
	bucket string,
) *TrashService {
	return &TrashService{
		fileRepo:  fileRepo,
		quotaRepo: quotaRepo,
		quotaSvc:  quotaSvc,
		minio:     minioClient,
		bucket:    bucket,
	}
}

func (s *TrashService) ListTrash(userID uint64, page, pageSize int) ([]model.File, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	return s.fileRepo.FindDeletedByUser(userID, page, pageSize)
}

func (s *TrashService) Restore(userID, fileID uint64) error {
	file, err := s.fileRepo.FindByID(fileID)
	if err != nil {
		return apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	if file.DeletedAt == nil {
		return apperrors.NewAppError(400, "文件不在回收站中", apperrors.ErrBadRequest)
	}

	if err := s.fileRepo.RestoreFromTrash(fileID, userID); err != nil {
		return apperrors.NewAppError(500, "恢复文件失败", err)
	}

	// Add back to quota
	_ = s.quotaRepo.AddStorageUsed(userID, file.Size)
	return nil
}

func (s *TrashService) PermanentDelete(userID, fileID uint64) error {
	file, err := s.fileRepo.FindByIDIncludingDeleted(fileID)
	if err != nil {
		return apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}

	if !file.IsDir {
		s.minio.RemoveObject(context.Background(), s.bucket, file.StorageKey, minio.RemoveObjectOptions{})
	}

	// Also clean up versions
	s.fileRepo.CleanupFileVersionsByFileID(fileID)

	return s.fileRepo.ForceDelete(fileID)
}

func (s *TrashService) EmptyTrash(userID uint64) (int64, error) {
	// Get all deleted files for user
	files, _, err := s.fileRepo.FindDeletedByUser(userID, 1, 10000)
	if err != nil {
		return 0, apperrors.NewAppError(500, "读取回收站失败", err)
	}

	var count int64
	ids := make([]uint64, 0)
	for _, f := range files {
		if !f.IsDir {
			s.minio.RemoveObject(context.Background(), s.bucket, f.StorageKey, minio.RemoveObjectOptions{})
		}
		s.fileRepo.CleanupFileVersionsByFileID(f.ID)
		ids = append(ids, f.ID)
		count++
	}

	if len(ids) > 0 {
		if err := s.fileRepo.BatchForceDelete(ids); err != nil {
			return count, apperrors.NewAppError(500, "清空回收站失败", err)
		}
	}
	return count, nil
}

// CleanupExpiredTrash removes files deleted more than 30 days ago.
func (s *TrashService) CleanupExpiredTrash(threshold interface{}) (int64, error) {
	files, err := s.fileRepo.FindDeletedExpired(threshold)
	if err != nil {
		return 0, err
	}

	var deleted int64
	ids := make([]uint64, 0)
	for _, f := range files {
		if !f.IsDir {
			s.minio.RemoveObject(context.Background(), s.bucket, f.StorageKey, minio.RemoveObjectOptions{})
		}
		s.fileRepo.CleanupFileVersionsByFileID(f.ID)
		ids = append(ids, f.ID)
	}
	if len(ids) > 0 {
		if err := s.fileRepo.BatchForceDelete(ids); err != nil {
			return deleted, err
		}
		deleted = int64(len(ids))
	}
	return deleted, nil
}
