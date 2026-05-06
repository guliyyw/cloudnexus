package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/cloudnexus/server/internal/userfile/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ShareService struct {
	shareRepo *repository.ShareRepository
	fileRepo  *repository.FileRepository
}

func NewShareService(shareRepo *repository.ShareRepository, fileRepo *repository.FileRepository) *ShareService {
	return &ShareService{shareRepo: shareRepo, fileRepo: fileRepo}
}

func generateShareCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type CreateShareReq struct {
	Password  string `json:"password"`
	ExpiresIn int    `json:"expires_in"` // hours, 0 = never
}

type ShareInfo struct {
	model.FileShare
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	MimeType    string `json:"mime_type"`
	HasPassword bool   `json:"has_password"`
}

func (s *ShareService) CreateShare(userID uint64, fileID uint64, req CreateShareReq) (*ShareInfo, error) {
	f, err := s.fileRepo.FindByID(fileID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if f.UserID != userID {
		return nil, apperrors.NewAppError(403, "无权操作此文件", apperrors.ErrForbidden)
	}
	if f.IsDir {
		return nil, apperrors.NewAppError(400, "暂不支持分享目录", apperrors.ErrBadRequest)
	}

	code, err := generateShareCode()
	if err != nil {
		return nil, apperrors.NewAppError(500, "生成分享码失败", apperrors.ErrInternalServer)
	}

	share := &model.FileShare{
		FileID:    fileID,
		OwnerID:   userID,
		ShareCode: code,
	}

	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, apperrors.NewAppError(500, "密码加密失败", apperrors.ErrInternalServer)
		}
		share.Password = string(hash)
	}

	if req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
		share.ExpiresAt = &t
	}

	if err := s.shareRepo.Create(share); err != nil {
		return nil, apperrors.NewAppError(500, "创建分享失败", apperrors.ErrInternalServer)
	}

	return &ShareInfo{
		FileShare:   *share,
		FileName:    f.Name,
		FileSize:    f.Size,
		MimeType:    f.MimeType,
		HasPassword: share.Password != "",
	}, nil
}

func (s *ShareService) GetShareByCode(code string) (*ShareInfo, error) {
	share, err := s.shareRepo.FindByCode(code)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.NewAppError(404, "分享不存在或已失效", apperrors.ErrNotFound)
		}
		return nil, apperrors.NewAppError(500, "查询分享失败", apperrors.ErrInternalServer)
	}

	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return nil, apperrors.NewAppError(410, "分享已过期", apperrors.ErrNotFound)
	}

	f, err := s.fileRepo.FindByID(share.FileID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "原文件不存在", apperrors.ErrNotFound)
	}

	return &ShareInfo{
		FileShare:   *share,
		FileName:    f.Name,
		FileSize:    f.Size,
		MimeType:    f.MimeType,
		HasPassword: share.Password != "",
	}, nil
}

func (s *ShareService) VerifyPassword(code string, password string) (string, error) {
	share, err := s.shareRepo.FindByCode(code)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", apperrors.NewAppError(404, "分享不存在", apperrors.ErrNotFound)
		}
		return "", apperrors.NewAppError(500, "查询分享失败", apperrors.ErrInternalServer)
	}

	if share.Password == "" {
		return "", apperrors.NewAppError(400, "此分享无需密码", apperrors.ErrBadRequest)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(share.Password), []byte(password)); err != nil {
		return "", apperrors.NewAppError(403, "密码错误", apperrors.ErrForbidden)
	}

	return share.ShareCode, nil
}

func (s *ShareService) RecordDownload(shareID uint64) {
	_ = s.shareRepo.IncrementDownloadCount(shareID)
}

func (s *ShareService) ListSharesByFile(userID uint64, fileID uint64) ([]ShareInfo, error) {
	f, err := s.fileRepo.FindByID(fileID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "文件不存在", apperrors.ErrNotFound)
	}
	if f.UserID != userID {
		return nil, apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}

	shares, err := s.shareRepo.FindByFileID(fileID)
	if err != nil {
		return nil, apperrors.NewAppError(500, "查询分享列表失败", apperrors.ErrInternalServer)
	}

	result := make([]ShareInfo, len(shares))
	for i, sh := range shares {
		result[i] = ShareInfo{
			FileShare:   sh,
			FileName:    f.Name,
			FileSize:    f.Size,
			MimeType:    f.MimeType,
			HasPassword: sh.Password != "",
		}
	}
	return result, nil
}

func (s *ShareService) ListMyShares(userID uint64) ([]ShareInfo, error) {
	shares, err := s.shareRepo.FindByOwnerID(userID)
	if err != nil {
		return nil, apperrors.NewAppError(500, "查询我的分享失败", apperrors.ErrInternalServer)
	}

	result := make([]ShareInfo, 0, len(shares))
	for _, sh := range shares {
		f, err := s.fileRepo.FindByID(sh.FileID)
		name := "(已删除)"
		size := int64(0)
		mimeType := ""
		if err == nil {
			name = f.Name
			size = f.Size
			mimeType = f.MimeType
		}
		result = append(result, ShareInfo{
			FileShare:   sh,
			FileName:    name,
			FileSize:    size,
			MimeType:    mimeType,
			HasPassword: sh.Password != "",
		})
	}
	return result, nil
}

func (s *ShareService) DeleteShare(userID uint64, shareID uint64) error {
	share, err := s.shareRepo.FindByID(shareID)
	if err != nil {
		return apperrors.NewAppError(404, "分享不存在", apperrors.ErrNotFound)
	}
	if share.OwnerID != userID {
		return apperrors.NewAppError(403, "无权删除此分享", apperrors.ErrForbidden)
	}
	return s.shareRepo.Delete(shareID)
}
