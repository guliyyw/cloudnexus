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

func (s *FileService) isAncestorOf(userID, sourceID, targetID uint64) (bool, error) {
	visited := make(map[uint64]bool)
	current := targetID
	for current != 0 {
		if current == sourceID {
			return true, nil
		}
		if visited[current] {
			return false, nil
		}
		visited[current] = true
		f, err := s.repo.FindByID(current)
		if err != nil || f == nil {
			return false, nil
		}
		current = f.ParentID
	}
	return false, nil
}

func (s *FileService) Move(userID, fileID, targetParentID uint64) (*model.File, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil || file == nil {
		return nil, apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return nil, apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}

	if targetParentID != 0 {
		target, err := s.repo.FindByID(targetParentID)
		if err != nil || target == nil {
			return nil, apperrors.NewAppError(404, "目标目录不存在", apperrors.ErrNotFound)
		}
		if target.UserID != userID {
			return nil, apperrors.NewAppError(403, "无权访问目标目录", apperrors.ErrForbidden)
		}
		if !target.IsDir {
			return nil, apperrors.NewAppError(400, "目标必须是目录", apperrors.ErrBadRequest)
		}
		if file.IsDir {
			if fileID == targetParentID {
				return nil, apperrors.NewAppError(400, "不能将目录移动到自身", apperrors.ErrBadRequest)
			}
			if isAncestor, _ := s.isAncestorOf(userID, fileID, targetParentID); isAncestor {
				return nil, apperrors.NewAppError(400, "不能将目录移动到其子目录中", apperrors.ErrBadRequest)
			}
		}
	}

	if file.ParentID != targetParentID {
		existing, err := s.repo.FindByNameAndParent(userID, targetParentID, file.Name)
		if err != nil {
			return nil, apperrors.NewAppError(500, "检查重名失败", err)
		}
		if existing != nil && existing.ID != file.ID {
			return nil, apperrors.NewAppError(409, "目标目录已存在同名文件或目录", apperrors.ErrConflict)
		}
	}

	if file.ParentID == targetParentID {
		return file, nil
	}

	file.ParentID = targetParentID
	if err := s.repo.Update(file); err != nil {
		return nil, apperrors.NewAppError(500, "移动失败", err)
	}
	return file, nil
}

func (s *FileService) Copy(userID, fileID, targetParentID uint64) (*model.File, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil || file == nil {
		return nil, apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return nil, apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}

	if targetParentID != 0 {
		target, err := s.repo.FindByID(targetParentID)
		if err != nil || target == nil {
			return nil, apperrors.NewAppError(404, "目标目录不存在", apperrors.ErrNotFound)
		}
		if target.UserID != userID {
			return nil, apperrors.NewAppError(403, "无权访问目标目录", apperrors.ErrForbidden)
		}
		if !target.IsDir {
			return nil, apperrors.NewAppError(400, "目标必须是目录", apperrors.ErrBadRequest)
		}
		if file.IsDir {
			if isAncestor, _ := s.isAncestorOf(userID, fileID, targetParentID); isAncestor {
				return nil, apperrors.NewAppError(400, "不能将目录复制到其子目录中", apperrors.ErrBadRequest)
			}
		}
	}

	existing, err := s.repo.FindByNameAndParent(userID, targetParentID, file.Name)
	if err != nil {
		return nil, apperrors.NewAppError(500, "检查重名失败", err)
	}
	if existing != nil {
		return nil, apperrors.NewAppError(409, "目标目录已存在同名文件或目录", apperrors.ErrConflict)
	}

	return s.copyRecursive(userID, file, targetParentID)
}

func (s *FileService) copyRecursive(userID uint64, src *model.File, targetParentID uint64) (*model.File, error) {
	if !src.IsDir {
		ext := filepath.Ext(src.Name)
		newKey := fmt.Sprintf("%d/%d/%d%s", userID, targetParentID, time.Now().UnixNano(), ext)
		_, err := s.minio.CopyObject(context.Background(),
			minio.CopyDestOptions{Bucket: s.bucket, Object: newKey},
			minio.CopySrcOptions{Bucket: s.bucket, Object: src.StorageKey},
		)
		if err != nil {
			return nil, apperrors.NewAppError(500, "复制文件内容失败", err)
		}

		newFile := &model.File{
			UserID:     userID,
			Name:       src.Name,
			ParentID:   targetParentID,
			Size:       src.Size,
			MimeType:   src.MimeType,
			StorageKey: newKey,
		}
		if err := s.repo.Create(newFile); err != nil {
			s.minio.RemoveObject(context.Background(), s.bucket, newKey, minio.RemoveObjectOptions{})
			return nil, apperrors.NewAppError(500, "保存副本信息失败", err)
		}
		return newFile, nil
	}

	newDir := &model.File{
		UserID:   userID,
		Name:     src.Name,
		IsDir:    true,
		ParentID: targetParentID,
	}
	if err := s.repo.Create(newDir); err != nil {
		return nil, apperrors.NewAppError(500, "创建目录副本失败", err)
	}

	children, err := s.repo.FindAllByParent(userID, src.ID)
	if err != nil {
		return nil, apperrors.NewAppError(500, "读取子目录失败", err)
	}
	for _, child := range children {
		if _, err := s.copyRecursive(userID, &child, newDir.ID); err != nil {
			return nil, err
		}
	}
	return newDir, nil
}
