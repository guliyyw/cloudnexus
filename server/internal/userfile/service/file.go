package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/cloudnexus/server/internal/userfile/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/minio/minio-go/v7"
)

type FileService struct {
	repo   *repository.FileRepository
	minio  *minio.Client
	bucket string
}

func NewFileService(repo *repository.FileRepository, minioClient *minio.Client, bucket string) *FileService {
	return &FileService{repo: repo, minio: minioClient, bucket: bucket}
}

func (s *FileService) Upload(userID uint64, parentID uint64, header *multipart.FileHeader) (*model.File, error) {
	src, err := header.Open()
	if err != nil {
		return nil, apperrors.NewAppError(500, "打开文件失败", err)
	}
	defer src.Close()

	ext := filepath.Ext(header.Filename)
	storageKey := fmt.Sprintf("%d/%d/%d%s", userID, parentID, time.Now().UnixNano(), ext)
	contentType := header.Header.Get("Content-Type")

	_, err = s.minio.PutObject(context.Background(), s.bucket, storageKey, src, header.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, apperrors.NewAppError(500, "存储文件失败", err)
	}

	file := &model.File{
		UserID:     userID,
		Name:       header.Filename,
		ParentID:   parentID,
		Size:       header.Size,
		MimeType:   contentType,
		StorageKey: storageKey,
	}
	if err := s.repo.Create(file); err != nil {
		s.minio.RemoveObject(context.Background(), s.bucket, storageKey, minio.RemoveObjectOptions{})
		return nil, apperrors.NewAppError(500, "保存文件信息失败", err)
	}
	return file, nil
}

func (s *FileService) Download(userID, fileID uint64) (io.ReadCloser, *model.File, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return nil, nil, apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return nil, nil, apperrors.NewAppError(403, "无权访问", apperrors.ErrForbidden)
	}
	if file.IsDir {
		return nil, nil, apperrors.NewAppError(400, "不能下载目录", apperrors.ErrBadRequest)
	}

	obj, err := s.minio.GetObject(context.Background(), s.bucket, file.StorageKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, apperrors.NewAppError(500, "获取文件失败", err)
	}
	return obj, file, nil
}

func (s *FileService) ListFiles(userID, parentID uint64, page, pageSize int) ([]model.File, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	return s.repo.FindByUserAndParent(userID, parentID, page, pageSize)
}

func (s *FileService) DeleteFile(userID, fileID uint64) error {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return apperrors.NewAppError(403, "无权删除", apperrors.ErrForbidden)
	}
	if !file.IsDir {
		s.minio.RemoveObject(context.Background(), s.bucket, file.StorageKey, minio.RemoveObjectOptions{})
	}
	return s.repo.SoftDelete(fileID, userID)
}

func (s *FileService) Mkdir(userID uint64, parentID uint64, name string) (*model.File, error) {
	dir := &model.File{
		UserID:   userID,
		Name:     name,
		IsDir:    true,
		ParentID: parentID,
	}
	if err := s.repo.Create(dir); err != nil {
		return nil, apperrors.NewAppError(500, "创建目录失败", err)
	}
	return dir, nil
}

func (s *FileService) BatchDelete(userID uint64, ids []uint64) (int64, []string) {
	var errs []string
	validIDs := make([]uint64, 0, len(ids))

	for _, id := range ids {
		file, err := s.repo.FindByID(id)
		if err != nil {
			errs = append(errs, fmt.Sprintf("文件 %d: 不存在", id))
			continue
		}
		if file.UserID != userID {
			errs = append(errs, fmt.Sprintf("%s: 无权删除", file.Name))
			continue
		}
		if !file.IsDir {
			s.minio.RemoveObject(context.Background(), s.bucket, file.StorageKey, minio.RemoveObjectOptions{})
		}
		validIDs = append(validIDs, id)
	}

	if len(validIDs) == 0 {
		return 0, errs
	}

	deleted, err := s.repo.BatchSoftDelete(validIDs, userID)
	if err != nil {
		errs = append(errs, fmt.Sprintf("数据库错误: %s", err.Error()))
	}
	return deleted, errs
}

func (s *FileService) Search(userID uint64, keyword string, page, pageSize int) ([]model.File, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	return s.repo.SearchFiles(userID, keyword, page, pageSize)
}
