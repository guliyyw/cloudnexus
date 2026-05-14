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
	const batchSize = 200
	var totalDeleted int64

	for {
		files, total, err := s.fileRepo.FindDeletedByUser(userID, 1, batchSize)
		if err != nil {
			return totalDeleted, apperrors.NewAppError(500, "读取回收站失败", err)
		}
		if len(files) == 0 {
			break
		}

		ids := make([]uint64, 0, len(files))
		for _, f := range files {
			if !f.IsDir {
				s.minio.RemoveObject(context.Background(), s.bucket, f.StorageKey, minio.RemoveObjectOptions{})
			}
			s.fileRepo.CleanupFileVersionsByFileID(f.ID)
			ids = append(ids, f.ID)
		}
		if len(ids) > 0 {
			if err := s.fileRepo.BatchForceDelete(ids); err != nil {
				return totalDeleted, apperrors.NewAppError(500, "清空回收站失败", err)
			}
			totalDeleted += int64(len(ids))
		}

		// If we got fewer than requested, we're done
		if int64(len(files)) < total && total <= int64(batchSize) {
			break
		}
		if len(files) < batchSize {
			break
		}
	}
	return totalDeleted, nil
}

// CleanupExpiredTrash removes files deleted more than 30 days ago.
// Processes in batches of 500 to avoid loading too many rows at once.
func (s *TrashService) CleanupExpiredTrash(threshold interface{}) (int64, error) {
	const batchSize = 500
	var totalDeleted int64

	for {
		files, err := s.fileRepo.FindDeletedExpired(threshold, batchSize)
		if err != nil {
			return totalDeleted, err
		}
		if len(files) == 0 {
			break
		}

		ids := make([]uint64, 0, len(files))
		for _, f := range files {
			if !f.IsDir {
				s.minio.RemoveObject(context.Background(), s.bucket, f.StorageKey, minio.RemoveObjectOptions{})
			}
			s.fileRepo.CleanupFileVersionsByFileID(f.ID)
			ids = append(ids, f.ID)
		}
		if len(ids) > 0 {
			if err := s.fileRepo.BatchForceDelete(ids); err != nil {
				return totalDeleted, err
			}
			totalDeleted += int64(len(ids))
		}
	}
	return totalDeleted, nil
}
