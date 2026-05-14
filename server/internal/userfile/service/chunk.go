package service

import (
	"context"
	"fmt"
	"math"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"time"

	"github.com/cloudnexus/server/internal/userfile/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type ChunkService struct {
	chunkRepo *repository.ChunkRepository
	fileRepo  *repository.FileRepository
	quotaRepo *repository.QuotaRepository
	quotaSvc  *QuotaService
	minio     *minio.Client
	bucket    string
}

func NewChunkService(
	chunkRepo *repository.ChunkRepository,
	fileRepo *repository.FileRepository,
	quotaRepo *repository.QuotaRepository,
	quotaSvc *QuotaService,
	minioClient *minio.Client,
	bucket string,
) *ChunkService {
	return &ChunkService{
		chunkRepo: chunkRepo,
		fileRepo:  fileRepo,
		quotaRepo: quotaRepo,
		quotaSvc:  quotaSvc,
		minio:     minioClient,
		bucket:    bucket,
	}
}

// InitUpload starts a chunked upload by validating quota and creating a tracking record.
func (s *ChunkService) InitUpload(userID uint64, parentID uint64, fileName string, fileSize int64, mimeType string) (*model.ChunkUpload, error) {
	if fileSize <= 0 {
		return nil, apperrors.NewAppError(400, "文件大小无效", apperrors.ErrBadRequest)
	}
	if fileName == "" {
		return nil, apperrors.NewAppError(400, "文件名不能为空", apperrors.ErrBadRequest)
	}

	// Check quota
	if err := s.quotaSvc.CheckQuota(userID, fileSize); err != nil {
		return nil, err
	}

	const chunkSize = 10 * 1024 * 1024 // 10MB
	totalChunks := int(math.Ceil(float64(fileSize) / float64(chunkSize)))

	uploadID := uuid.New().String()
	chunk := &model.ChunkUpload{
		UserID:      userID,
		UploadID:    uploadID,
		FileName:    fileName,
		FileSize:    fileSize,
		ChunkSize:   chunkSize,
		MimeType:    mimeType,
		ParentID:    parentID,
		TotalChunks: totalChunks,
		Status:      "uploading",
	}

	if err := s.chunkRepo.Create(chunk); err != nil {
		return nil, apperrors.NewAppError(500, "创建上传记录失败", err)
	}

	return chunk, nil
}

// UploadChunk stores a single chunk to MinIO and updates progress.
func (s *ChunkService) UploadChunk(userID uint64, uploadID string, chunkIndex int32, header *multipart.FileHeader) (int, error) {
	chunk, err := s.chunkRepo.FindByUploadID(uploadID)
	if err != nil {
		return 0, apperrors.NewAppError(404, "上传会话不存在", apperrors.ErrNotFound)
	}
	if chunk.UserID != userID {
		return 0, apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	if chunk.Status != "uploading" {
		return 0, apperrors.NewAppError(400, "上传会话状态异常", apperrors.ErrBadRequest)
	}
	if chunkIndex < 0 || chunkIndex >= int32(chunk.TotalChunks) {
		return 0, apperrors.NewAppError(400, "分片索引无效", apperrors.ErrBadRequest)
	}

	src, err := header.Open()
	if err != nil {
		return 0, apperrors.NewAppError(500, "打开分片失败", err)
	}
	defer src.Close()

	chunkKey := fmt.Sprintf("chunks/%s/%d", uploadID, chunkIndex)
	_, err = s.minio.PutObject(context.Background(), s.bucket, chunkKey, src, header.Size, minio.PutObjectOptions{})
	if err != nil {
		return 0, apperrors.NewAppError(500, "存储分片失败", err)
	}

	if err := s.chunkRepo.AddCompletedChunk(uploadID, chunkIndex); err != nil {
		return 0, apperrors.NewAppError(500, "更新进度失败", err)
	}

	// Re-read to get updated count
	updated, _ := s.chunkRepo.FindByUploadID(uploadID)
	completed := 0
	if updated != nil {
		completed = len(updated.Completed)
	}
	return completed, nil
}

// GetStatus returns the upload status for resume support.
func (s *ChunkService) GetStatus(userID uint64, uploadID string) (*model.ChunkUpload, error) {
	chunk, err := s.chunkRepo.FindByUploadID(uploadID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "上传会话不存在", apperrors.ErrNotFound)
	}
	if chunk.UserID != userID {
		return nil, apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	return chunk, nil
}

// CompleteUpload merges all chunks via MinIO ComposeObject and creates the file record.
func (s *ChunkService) CompleteUpload(userID uint64, uploadID string, versionMessage string) (*model.File, error) {
	chunk, err := s.chunkRepo.FindByUploadID(uploadID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "上传会话不存在", apperrors.ErrNotFound)
	}
	if chunk.UserID != userID {
		return nil, apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	if chunk.Status != "uploading" {
		return nil, apperrors.NewAppError(400, "上传会话状态异常", apperrors.ErrBadRequest)
	}
	if len(chunk.Completed) < chunk.TotalChunks {
		return nil, apperrors.NewAppError(400, fmt.Sprintf("分片未完成 (%d/%d)", len(chunk.Completed), chunk.TotalChunks), apperrors.ErrBadRequest)
	}

	// Set status to merging
	_ = s.chunkRepo.UpdateStatus(uploadID, "merging")

	ext := filepath.Ext(chunk.FileName)
	storageKey := fmt.Sprintf("%d/%d/%d%s", userID, chunk.ParentID, time.Now().UnixNano(), ext)

	// ComposeObject: merge all chunks server-side
	sources := make([]minio.CopySrcOptions, chunk.TotalChunks)
	for i := 0; i < chunk.TotalChunks; i++ {
		sources[i] = minio.CopySrcOptions{
			Bucket: s.bucket,
			Object: fmt.Sprintf("chunks/%s/%d", uploadID, i),
		}
	}

	_, err = s.minio.ComposeObject(context.Background(),
		minio.CopyDestOptions{Bucket: s.bucket, Object: storageKey},
		sources...,
	)
	if err != nil {
		_ = s.chunkRepo.UpdateStatus(uploadID, "uploading")
		return nil, apperrors.NewAppError(500, "合并分片失败", err)
	}

	// Create file record (reuse existing pattern)
	file := &model.File{
		UserID:     userID,
		Name:       chunk.FileName,
		ParentID:   chunk.ParentID,
		Size:       chunk.FileSize,
		MimeType:   chunk.MimeType,
		StorageKey: storageKey,
	}

	// Check versioning
	existing, _ := s.fileRepo.FindByNameAndParent(userID, chunk.ParentID, chunk.FileName)
	if existing != nil && !existing.IsDir {
		maxNum, _ := s.fileRepo.GetMaxVersionNum(existing.ID)
		v := &model.FileVersion{
			FileID:     existing.ID,
			VersionNum: maxNum + 1,
			StorageKey: existing.StorageKey,
			Size:       existing.Size,
			SHA256:     existing.StorageSHA256,
			Message:    versionMessage,
		}
		if err := s.fileRepo.CreateVersion(v); err != nil {
			s.minio.RemoveObject(context.Background(), s.bucket, storageKey, minio.RemoveObjectOptions{})
			return nil, apperrors.NewAppError(500, "保存版本失败", err)
		}
		existing.Size = chunk.FileSize
		existing.MimeType = chunk.MimeType
		existing.StorageKey = storageKey
		existing.StorageSHA256 = ""
		if err := s.fileRepo.Update(existing); err != nil {
			s.minio.RemoveObject(context.Background(), s.bucket, storageKey, minio.RemoveObjectOptions{})
			return nil, apperrors.NewAppError(500, "更新文件信息失败", err)
		}
		file = existing
	} else {
		if err := s.fileRepo.Create(file); err != nil {
			s.minio.RemoveObject(context.Background(), s.bucket, storageKey, minio.RemoveObjectOptions{})
			return nil, apperrors.NewAppError(500, "保存文件信息失败", err)
		}
	}

	// Clean up chunks from MinIO
	go func() {
		for i := 0; i < chunk.TotalChunks; i++ {
			s.minio.RemoveObject(context.Background(), s.bucket,
				fmt.Sprintf("chunks/%s/%d", uploadID, i), minio.RemoveObjectOptions{})
		}
	}()

	_ = s.chunkRepo.UpdateStatus(uploadID, "completed")

	// Update quota
	_ = s.quotaRepo.AddStorageUsed(userID, chunk.FileSize)

	return file, nil
}

// CancelUpload cancels an ongoing upload and cleans up temporary chunks.
func (s *ChunkService) CancelUpload(userID uint64, uploadID string) error {
	chunk, err := s.chunkRepo.FindByUploadID(uploadID)
	if err != nil {
		return apperrors.NewAppError(404, "上传会话不存在", apperrors.ErrNotFound)
	}
	if chunk.UserID != userID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}

	_ = s.chunkRepo.UpdateStatus(uploadID, "cancelled")

	// Clean up chunks from MinIO asynchronously
	go func() {
		for i := 0; i < chunk.TotalChunks; i++ {
			s.minio.RemoveObject(context.Background(), s.bucket,
				fmt.Sprintf("chunks/%s/%d", uploadID, i), minio.RemoveObjectOptions{})
		}
	}()

	return nil
}

// ListIncomplete returns all incomplete uploads for the user (used by resume dialog).
func (s *ChunkService) ListIncomplete(userID uint64) ([]model.ChunkUpload, error) {
	return s.chunkRepo.ListIncompleteByUser(userID)
}

// ChunkDownloadURL generates a presigned URL for downloading a chunk.
func (s *ChunkService) ChunkDownloadURL(uploadID string, chunkIndex int) (string, error) {
	u, err := s.minio.PresignedGetObject(context.Background(), s.bucket,
		fmt.Sprintf("chunks/%s/%d", uploadID, chunkIndex), time.Hour, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
