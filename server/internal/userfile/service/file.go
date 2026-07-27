package service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudnexus/server/internal/userfile/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/minio/minio-go/v7"
)

type officeTemplate struct {
	ext      string
	mimeType string
	content  []byte
}

type FileService struct {
	repo      *repository.FileRepository
	quotaRepo *repository.QuotaRepository
	quotaSvc  *QuotaService
	minio     *minio.Client
	bucket    string
}

func NewFileService(repo *repository.FileRepository, quotaRepo *repository.QuotaRepository, quotaSvc *QuotaService, minioClient *minio.Client, bucket string) *FileService {
	return &FileService{repo: repo, quotaRepo: quotaRepo, quotaSvc: quotaSvc, minio: minioClient, bucket: bucket}
}

func (s *FileService) Upload(userID uint64, parentID uint64, header *multipart.FileHeader, versionMessage string) (*model.File, error) {
	// Quota check
	if err := s.quotaSvc.CheckQuota(userID, header.Size); err != nil {
		return nil, err
	}
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

	// 检查是否存在同名文件，保存旧版本
	existing, _ := s.repo.FindByNameAndParent(userID, parentID, header.Filename)
	if existing != nil && !existing.IsDir {
		maxNum, _ := s.repo.GetMaxVersionNum(existing.ID)
		v := &model.FileVersion{
			FileID:     existing.ID,
			VersionNum: maxNum + 1,
			StorageKey: existing.StorageKey,
			Size:       existing.Size,
			SHA256:     existing.StorageSHA256,
			Message:    versionMessage,
		}
		if err := s.repo.CreateVersion(v); err != nil {
			s.minio.RemoveObject(context.Background(), s.bucket, storageKey, minio.RemoveObjectOptions{})
			return nil, apperrors.NewAppError(500, "保存版本失败", err)
		}

		existing.Size = header.Size
		existing.MimeType = contentType
		existing.StorageKey = storageKey
		existing.StorageSHA256 = ""
		if err := s.repo.Update(existing); err != nil {
			s.minio.RemoveObject(context.Background(), s.bucket, storageKey, minio.RemoveObjectOptions{})
			_ = s.repo.DeleteVersion(v.ID)
			return nil, apperrors.NewAppError(500, "更新文件信息失败", err)
		}
		return existing, nil
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
	// Update quota
	_ = s.quotaRepo.AddStorageUsed(userID, header.Size)
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

type tempFileReadCloser struct {
	*os.File
	dir string
}

func (r *tempFileReadCloser) Close() error {
	err := r.File.Close()
	_ = os.RemoveAll(r.dir)
	return err
}

func (s *FileService) SaveTextContent(userID, fileID uint64, content string, versionMessage string) (*model.File, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "file not found", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return nil, apperrors.NewAppError(403, "forbidden", apperrors.ErrForbidden)
	}
	if file.IsDir {
		return nil, apperrors.NewAppError(400, "directories cannot be edited", apperrors.ErrBadRequest)
	}

	maxNum, _ := s.repo.GetMaxVersionNum(file.ID)
	if file.StorageKey != "" {
		v := &model.FileVersion{
			FileID:     file.ID,
			VersionNum: maxNum + 1,
			StorageKey: file.StorageKey,
			Size:       file.Size,
			SHA256:     file.StorageSHA256,
			Message:    versionMessage,
		}
		if err := s.repo.CreateVersion(v); err != nil {
			return nil, apperrors.NewAppError(500, "save version failed", err)
		}
	}

	ext := filepath.Ext(file.Name)
	storageKey := fmt.Sprintf("%d/%d/%d%s", userID, file.ParentID, time.Now().UnixNano(), ext)
	reader := strings.NewReader(content)
	contentType := file.MimeType
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	_, err = s.minio.PutObject(context.Background(), s.bucket, storageKey, reader, int64(reader.Len()), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, apperrors.NewAppError(500, "save content failed", err)
	}

	oldSize := file.Size
	file.StorageKey = storageKey
	file.Size = int64(len(content))
	if err := s.repo.Update(file); err != nil {
		_ = s.minio.RemoveObject(context.Background(), s.bucket, storageKey, minio.RemoveObjectOptions{})
		return nil, apperrors.NewAppError(500, "update file failed", err)
	}
	_ = s.quotaRepo.AddStorageUsed(userID, file.Size-oldSize)
	return file, nil
}

func (s *FileService) SaveBinaryContent(userID, fileID uint64, content []byte, contentType, versionMessage string) (*model.File, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "file not found", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return nil, apperrors.NewAppError(403, "forbidden", apperrors.ErrForbidden)
	}
	if file.IsDir {
		return nil, apperrors.NewAppError(400, "directories cannot be edited", apperrors.ErrBadRequest)
	}

	maxNum, _ := s.repo.GetMaxVersionNum(file.ID)
	if file.StorageKey != "" {
		v := &model.FileVersion{
			FileID:     file.ID,
			VersionNum: maxNum + 1,
			StorageKey: file.StorageKey,
			Size:       file.Size,
			SHA256:     file.StorageSHA256,
			Message:    versionMessage,
		}
		if err := s.repo.CreateVersion(v); err != nil {
			return nil, apperrors.NewAppError(500, "save version failed", err)
		}
	}

	ext := filepath.Ext(file.Name)
	storageKey := fmt.Sprintf("%d/%d/%d%s", userID, file.ParentID, time.Now().UnixNano(), ext)
	if contentType == "" {
		contentType = file.MimeType
	}
	reader := bytes.NewReader(content)
	_, err = s.minio.PutObject(context.Background(), s.bucket, storageKey, reader, int64(reader.Len()), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, apperrors.NewAppError(500, "save content failed", err)
	}

	oldSize := file.Size
	file.StorageKey = storageKey
	file.Size = int64(len(content))
	if contentType != "" {
		file.MimeType = contentType
	}
	if err := s.repo.Update(file); err != nil {
		_ = s.minio.RemoveObject(context.Background(), s.bucket, storageKey, minio.RemoveObjectOptions{})
		return nil, apperrors.NewAppError(500, "update file failed", err)
	}
	_ = s.quotaRepo.AddStorageUsed(userID, file.Size-oldSize)
	return file, nil
}

func (s *FileService) ConvertHTMLToWord(userID, fileID uint64, html string) ([]byte, *model.File, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return nil, nil, apperrors.NewAppError(404, "file not found", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return nil, nil, apperrors.NewAppError(403, "forbidden", apperrors.ErrForbidden)
	}
	ext := strings.ToLower(filepath.Ext(file.Name))
	if ext != ".docx" && ext != ".doc" {
		return nil, nil, apperrors.NewAppError(400, "only Word documents can be edited", apperrors.ErrBadRequest)
	}
	if strings.TrimSpace(html) == "" {
		html = "<p></p>"
	}

	content, err := convertHTMLToDOCX(html)
	if err != nil {
		return nil, nil, err
	}
	return content, file, nil
}

func (s *FileService) SaveWordHTML(userID, fileID uint64, html, versionMessage string) (*model.File, error) {
	content, _, err := s.ConvertHTMLToWord(userID, fileID, html)
	if err != nil {
		return nil, err
	}
	return s.SaveBinaryContent(
		userID,
		fileID,
		content,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		versionMessage,
	)
}

func convertHTMLToDOCX(html string) ([]byte, error) {
	bin := "libreoffice"
	if _, err := exec.LookPath(bin); err != nil {
		bin = "soffice"
		if _, sofficeErr := exec.LookPath(bin); sofficeErr != nil {
			return nil, apperrors.NewAppError(501, "LibreOffice is required for Word editing", err)
		}
	}

	tmpDir, err := os.MkdirTemp("", "cloudnexus-html-docx-*")
	if err != nil {
		return nil, apperrors.NewAppError(500, "create temp dir failed", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "document.html")
	if err := os.WriteFile(inputPath, []byte(html), 0600); err != nil {
		return nil, apperrors.NewAppError(500, "write Word content failed", err)
	}

	profileDir := filepath.Join(tmpDir, "lo-profile")
	cmd := exec.Command(
		bin,
		"-env:UserInstallation=file://"+filepath.ToSlash(profileDir),
		"--headless",
		"--convert-to",
		"docx:Office Open XML Text",
		"--outdir",
		tmpDir,
		inputPath,
	)
	cmd.Env = append(os.Environ(), "HOME="+tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, apperrors.NewAppError(500, "HTML to Word conversion failed: "+string(output), err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "document.docx"))
	if err != nil {
		return nil, apperrors.NewAppError(500, "converted Word file not found", err)
	}
	return content, nil
}

func (s *FileService) ConvertWordToPDF(userID, fileID uint64) (io.ReadCloser, string, int64, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return nil, "", 0, apperrors.NewAppError(404, "file not found", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return nil, "", 0, apperrors.NewAppError(403, "forbidden", apperrors.ErrForbidden)
	}
	ext := strings.ToLower(filepath.Ext(file.Name))
	if ext != ".docx" && ext != ".doc" {
		return nil, "", 0, apperrors.NewAppError(400, "only Word documents can be converted", apperrors.ErrBadRequest)
	}
	if _, err := exec.LookPath("libreoffice"); err != nil {
		if _, sofficeErr := exec.LookPath("soffice"); sofficeErr != nil {
			return nil, "", 0, apperrors.NewAppError(501, "LibreOffice is required for Word to PDF conversion", err)
		}
	}

	obj, _, err := s.Download(userID, fileID)
	if err != nil {
		return nil, "", 0, err
	}
	defer obj.Close()

	tmpDir, err := os.MkdirTemp("", "cloudnexus-word-pdf-*")
	if err != nil {
		return nil, "", 0, apperrors.NewAppError(500, "create temp dir failed", err)
	}
	inputPath := filepath.Join(tmpDir, file.Name)
	input, err := os.Create(inputPath)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, "", 0, apperrors.NewAppError(500, "create temp file failed", err)
	}
	if _, err := io.Copy(input, obj); err != nil {
		input.Close()
		_ = os.RemoveAll(tmpDir)
		return nil, "", 0, apperrors.NewAppError(500, "write temp file failed", err)
	}
	input.Close()

	bin := "libreoffice"
	if _, err := exec.LookPath(bin); err != nil {
		bin = "soffice"
	}
	profileDir := filepath.Join(tmpDir, "lo-profile")
	cmd := exec.Command(
		bin,
		"-env:UserInstallation=file://"+filepath.ToSlash(profileDir),
		"--headless",
		"--convert-to",
		"pdf",
		"--outdir",
		tmpDir,
		inputPath,
	)
	cmd.Env = append(os.Environ(), "HOME="+tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, "", 0, apperrors.NewAppError(500, "Word to PDF conversion failed: "+string(output), err)
	}

	pdfPath := filepath.Join(tmpDir, strings.TrimSuffix(file.Name, ext)+".pdf")
	pdf, err := os.Open(pdfPath)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, "", 0, apperrors.NewAppError(500, "converted PDF not found", err)
	}
	info, err := pdf.Stat()
	if err != nil {
		pdf.Close()
		_ = os.RemoveAll(tmpDir)
		return nil, "", 0, apperrors.NewAppError(500, "read converted PDF failed", err)
	}
	return &tempFileReadCloser{File: pdf, dir: tmpDir}, strings.TrimSuffix(file.Name, ext) + ".pdf", info.Size(), nil
}

// DownloadForShare 用于分享下载，跳过所有权检查（由调用方验证分享权限）
func (s *FileService) DownloadForShare(fileID uint64) (io.ReadCloser, *model.File, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return nil, nil, apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
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

func (s *FileService) ListCollabDocs(userID uint64, page, pageSize int, keyword string) ([]model.File, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.FindCollabDocs(userID, page, pageSize, strings.TrimSpace(keyword))
}

func (s *FileService) DeleteFile(userID, fileID uint64) error {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return apperrors.NewAppError(403, "无权删除", apperrors.ErrForbidden)
	}

	files, err := s.repo.FindActiveTree(userID, fileID)
	if err != nil {
		return apperrors.NewAppError(500, "读取目录内容失败", err)
	}
	_, err = s.softDeleteTree(userID, files)
	return err
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
	validRoots := make(map[uint64]struct{}, len(ids))
	filesByID := make(map[uint64]model.File)

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
		tree, err := s.repo.FindActiveTree(userID, id)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: 读取目录内容失败", file.Name))
			continue
		}
		validRoots[id] = struct{}{}
		for _, item := range tree {
			filesByID[item.ID] = item
		}
	}

	if len(validRoots) == 0 {
		return 0, errs
	}

	files := make([]model.File, 0, len(filesByID))
	for _, file := range filesByID {
		files = append(files, file)
	}
	_, err := s.softDeleteTree(userID, files)
	if err != nil {
		errs = append(errs, err.Error())
		return 0, errs
	}
	return int64(len(validRoots)), errs
}

func (s *FileService) softDeleteTree(userID uint64, files []model.File) (int64, error) {
	if len(files) == 0 {
		return 0, nil
	}

	var totalSize int64
	ids := make([]uint64, 0, len(files))
	for _, file := range files {
		ids = append(ids, file.ID)
		if !file.IsDir {
			totalSize += file.Size
		}
	}
	if totalSize > 0 {
		if err := s.quotaSvc.CheckTrashSpace(userID, totalSize, s.repo); err != nil {
			return 0, err
		}
	}

	for _, file := range files {
		if file.IsDir || file.StorageKey == "" {
			continue
		}
		if err := s.minio.RemoveObject(context.Background(), s.bucket, file.StorageKey, minio.RemoveObjectOptions{}); err != nil {
			return 0, apperrors.NewAppError(500, "删除实际文件失败", err)
		}
	}

	deleted, err := s.repo.BatchSoftDelete(ids, userID)
	if err != nil {
		return 0, apperrors.NewAppError(500, "删除文件记录失败", err)
	}
	if totalSize > 0 {
		_ = s.quotaRepo.AddStorageUsed(userID, -totalSize)
	}
	return deleted, nil
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

// ── 协作文档 ──

func (s *FileService) GetFileMeta(userID, fileID uint64) (*model.File, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return nil, apperrors.NewAppError(403, "无权访问", apperrors.ErrForbidden)
	}
	return file, nil
}

func (s *FileService) CreateCollab(userID, parentID uint64, name, collabType string) (*model.File, error) {
	// Check for duplicate name
	existing, err := s.repo.FindByNameAndParent(userID, parentID, name+".clouddoc")
	if err != nil {
		return nil, apperrors.NewAppError(500, "检查重名失败", err)
	}
	if existing != nil {
		return nil, apperrors.NewAppError(409, "已存在同名协作文档", apperrors.ErrConflict)
	}

	mimeType := "application/x-collab-" + collabType
	file := &model.File{
		UserID:     userID,
		Name:       name + ".clouddoc",
		ParentID:   parentID,
		Size:       1,
		MimeType:   mimeType,
		CollabType: collabType,
		StorageKey: "", // set after ID is generated
	}
	if err := s.repo.Create(file); err != nil {
		return nil, apperrors.NewAppError(500, "创建文件记录失败", err)
	}

	// Set deterministic storage key (MinIO object written on first persist)
	storageKey := fmt.Sprintf("collab/%d/ydoc.bin", file.ID)
	file.StorageKey = storageKey
	file.Size = 0

	if err := s.repo.Update(file); err != nil {
		return nil, apperrors.NewAppError(500, "更新存储路径失败", err)
	}
	return file, nil
}

func zipParts(parts map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range parts {
		w, err := zw.Create(name)
		if err != nil {
			zw.Close()
			return nil, err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func blankDocx() ([]byte, error) {
	return zipParts(map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml":   `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t></w:t></w:r></w:p><w:sectPr/></w:body></w:document>`,
	})
}

func blankXlsx() ([]byte, error) {
	return zipParts(map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/worksheets/sheet1.xml":   `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/></worksheet>`,
	})
}

func newOfficeTemplate(kind string) (*officeTemplate, error) {
	switch strings.ToLower(kind) {
	case "word", "docx":
		content, err := blankDocx()
		if err != nil {
			return nil, err
		}
		return &officeTemplate{ext: ".docx", mimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", content: content}, nil
	case "excel", "xlsx":
		content, err := blankXlsx()
		if err != nil {
			return nil, err
		}
		return &officeTemplate{ext: ".xlsx", mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", content: content}, nil
	default:
		return nil, apperrors.ErrBadRequest
	}
}

func (s *FileService) CreateOfficeDoc(userID, parentID uint64, name, kind string) (*model.File, error) {
	tpl, err := newOfficeTemplate(kind)
	if err != nil {
		return nil, apperrors.NewAppError(400, "unsupported document type", apperrors.ErrBadRequest)
	}
	fileName := strings.TrimSpace(name)
	if fileName == "" {
		return nil, apperrors.NewAppError(400, "document name is required", apperrors.ErrBadRequest)
	}
	if !strings.HasSuffix(strings.ToLower(fileName), tpl.ext) {
		fileName += tpl.ext
	}
	existing, err := s.repo.FindByNameAndParent(userID, parentID, fileName)
	if err != nil {
		return nil, apperrors.NewAppError(500, "check duplicate file failed", err)
	}
	if existing != nil {
		return nil, apperrors.NewAppError(409, "file already exists", apperrors.ErrConflict)
	}
	if err := s.quotaSvc.CheckQuota(userID, int64(len(tpl.content))); err != nil {
		return nil, err
	}
	storageKey := fmt.Sprintf("%d/%d/%d%s", userID, parentID, time.Now().UnixNano(), tpl.ext)
	reader := bytes.NewReader(tpl.content)
	_, err = s.minio.PutObject(context.Background(), s.bucket, storageKey, reader, int64(reader.Len()), minio.PutObjectOptions{ContentType: tpl.mimeType})
	if err != nil {
		return nil, apperrors.NewAppError(500, "save document failed", err)
	}
	file := &model.File{UserID: userID, Name: fileName, ParentID: parentID, Size: int64(len(tpl.content)), MimeType: tpl.mimeType, StorageKey: storageKey}
	if err := s.repo.Create(file); err != nil {
		_ = s.minio.RemoveObject(context.Background(), s.bucket, storageKey, minio.RemoveObjectOptions{})
		return nil, apperrors.NewAppError(500, "create document record failed", err)
	}
	_ = s.quotaRepo.AddStorageUsed(userID, file.Size)
	return file, nil
}

// ── 文件版本管理 ──

func (s *FileService) ListVersions(userID, fileID uint64, page, pageSize int) ([]model.FileVersion, int64, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return nil, 0, apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return nil, 0, apperrors.NewAppError(403, "无权访问", apperrors.ErrForbidden)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListVersions(fileID, page, pageSize)
}

func (s *FileService) RestoreVersion(userID, fileID, versionID uint64) (*model.File, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return nil, apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	if file.IsDir {
		return nil, apperrors.NewAppError(400, "目录不支持版本恢复", apperrors.ErrBadRequest)
	}

	ver, err := s.repo.FindVersionByID(versionID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "版本不存在", apperrors.ErrNotFound)
	}
	if ver.FileID != fileID {
		return nil, apperrors.NewAppError(400, "版本与文件不匹配", apperrors.ErrBadRequest)
	}

	// 当前文件作为新版本保存
	maxNum, _ := s.repo.GetMaxVersionNum(fileID)
	saveVer := &model.FileVersion{
		FileID:     fileID,
		VersionNum: maxNum + 1,
		StorageKey: file.StorageKey,
		Size:       file.Size,
		SHA256:     file.StorageSHA256,
		Message:    "恢复前自动保存",
	}
	if err := s.repo.CreateVersion(saveVer); err != nil {
		return nil, apperrors.NewAppError(500, "保存当前版本失败", err)
	}

	// 复制旧版本对象到新 key（MinIO copy）
	ext := filepath.Ext(file.Name)
	newKey := fmt.Sprintf("%d/%d/%d%s", userID, file.ParentID, time.Now().UnixNano(), ext)
	_, err = s.minio.CopyObject(context.Background(),
		minio.CopyDestOptions{Bucket: s.bucket, Object: newKey},
		minio.CopySrcOptions{Bucket: s.bucket, Object: ver.StorageKey},
	)
	if err != nil {
		return nil, apperrors.NewAppError(500, "恢复版本内容失败", err)
	}

	file.StorageKey = newKey
	file.Size = ver.Size
	file.StorageSHA256 = ver.SHA256
	if err := s.repo.Update(file); err != nil {
		s.minio.RemoveObject(context.Background(), s.bucket, newKey, minio.RemoveObjectOptions{})
		return nil, apperrors.NewAppError(500, "更新文件失败", err)
	}
	return file, nil
}

func (s *FileService) DownloadVersion(userID, fileID, versionID uint64) (io.ReadCloser, *model.FileVersion, *model.File, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return nil, nil, nil, apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return nil, nil, nil, apperrors.NewAppError(403, "无权访问", apperrors.ErrForbidden)
	}

	ver, err := s.repo.FindVersionByID(versionID)
	if err != nil {
		return nil, nil, nil, apperrors.NewAppError(404, "版本不存在", apperrors.ErrNotFound)
	}
	if ver.FileID != fileID {
		return nil, nil, nil, apperrors.NewAppError(400, "版本与文件不匹配", apperrors.ErrBadRequest)
	}

	obj, err := s.minio.GetObject(context.Background(), s.bucket, ver.StorageKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, nil, apperrors.NewAppError(500, "获取版本文件失败", err)
	}
	return obj, ver, file, nil
}

func (s *FileService) VersionDownloadURL(userID, fileID, versionID uint64) (string, error) {
	file, err := s.repo.FindByID(fileID)
	if err != nil {
		return "", apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if file.UserID != userID {
		return "", apperrors.NewAppError(403, "无权访问", apperrors.ErrForbidden)
	}

	ver, err := s.repo.FindVersionByID(versionID)
	if err != nil {
		return "", apperrors.NewAppError(404, "版本不存在", apperrors.ErrNotFound)
	}
	if ver.FileID != fileID {
		return "", apperrors.NewAppError(400, "版本与文件不匹配", apperrors.ErrBadRequest)
	}

	u, err := s.minio.PresignedGetObject(context.Background(), s.bucket, ver.StorageKey, time.Hour, url.Values{})
	if err != nil {
		return "", apperrors.NewAppError(500, "生成下载链接失败", err)
	}
	return u.String(), nil
}
