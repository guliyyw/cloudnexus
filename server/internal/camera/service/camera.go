package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloudnexus/server/internal/camera/repository"
	"github.com/cloudnexus/server/pkg/model"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MediaMTX path config request.
type pathConfig struct {
	Name         string `json:"name"`
	Source       string `json:"source"`
	SourceOnDemand bool `json:"sourceOnDemand"`
}

// CameraService manages cameras and the MediaMTX proxy.
type CameraService struct {
	repo        *repository.CameraRepository
	mediamtxURL string // e.g. http://mediamtx:8889
}

func NewCameraService(repo *repository.CameraRepository, mediamtxURL string) *CameraService {
	return &CameraService{repo: repo, mediamtxURL: mediamtxURL}
}

func (s *CameraService) ListCameras(ownerID uint64, offset, limit int) ([]model.Camera, int64, error) {
	return s.repo.ListCameras(ownerID, offset, limit)
}

func (s *CameraService) CreateCamera(c *model.Camera) error {
	if c.StreamURL == "" {
		return apperrors.NewAppError(400, "视频流地址不能为空", apperrors.ErrBadRequest)
	}
	if c.Name == "" {
		return apperrors.NewAppError(400, "摄像头名称不能为空", apperrors.ErrBadRequest)
	}
	return s.repo.CreateCamera(c)
}

func (s *CameraService) UpdateCamera(c *model.Camera) error {
	existing, err := s.repo.FindCameraByID(c.ID)
	if err != nil {
		return apperrors.NewAppError(404, "摄像头不存在", apperrors.ErrNotFound)
	}
	existing.Name = c.Name
	existing.StreamURL = c.StreamURL
	existing.Protocol = c.Protocol
	return s.repo.UpdateCamera(existing)
}

func (s *CameraService) DeleteCamera(id uint64, ownerID uint64) error {
	c, err := s.repo.FindCameraByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.NewAppError(404, "摄像头不存在", apperrors.ErrNotFound)
		}
		return err
	}
	if c.OwnerID != ownerID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	// Remove from MediaMTX
	if err := s.stopStream(c); err != nil {
		zap.L().Warn("mediamtx stop stream failed", zap.Error(err))
	}
	return s.repo.DeleteCamera(id)
}

// StartStream registers the camera in MediaMTX and returns the HLS/WebRTC URLs.
func (s *CameraService) StartStream(cameraID uint64, ownerID uint64) (hlsURL, webrtcURL string, err error) {
	c, err := s.repo.FindCameraByID(cameraID)
	if err != nil {
		return "", "", apperrors.NewAppError(404, "摄像头不存在", apperrors.ErrNotFound)
	}
	if c.OwnerID != ownerID {
		return "", "", apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}

	pathName := fmt.Sprintf("cam_%d", cameraID)
	cfg := pathConfig{
		Name:         pathName,
		Source:       c.StreamURL,
		SourceOnDemand: true,
	}

	body, _ := json.Marshal(cfg)
	resp, err := http.Post(
		fmt.Sprintf("%s/v3/config/paths/add/%s", s.mediamtxURL, pathName),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", "", fmt.Errorf("mediamtx API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		// Path already exists — stream is already configured, just return URLs.
		if resp.StatusCode == 400 && bytes.Contains(respBody, []byte("path already exists")) {
			zap.L().Info("mediamtx path already exists, reusing", zap.String("path", pathName))
		} else {
			return "", "", fmt.Errorf("mediamtx 返回 %d: %s", resp.StatusCode, string(respBody))
		}
	}

	// Update camera status
	now := time.Now()
	c.Status = "online"
	c.LastSeenAt = &now
	s.repo.UpdateCamera(c)

	// URLs are relative to nginx proxy (port 80), which proxies to MediaMTX
	hlsURL = fmt.Sprintf("/%s/index.m3u8", pathName)
	webrtcURL = fmt.Sprintf("/%s/whep", pathName)

	return hlsURL, webrtcURL, nil
}

// StopStream removes the camera path from MediaMTX.
func (s *CameraService) StopStream(cameraID uint64, ownerID uint64) error {
	c, err := s.repo.FindCameraByID(cameraID)
	if err != nil {
		return apperrors.NewAppError(404, "摄像头不存在", apperrors.ErrNotFound)
	}
	if c.OwnerID != ownerID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	return s.stopStream(c)
}

func (s *CameraService) stopStream(c *model.Camera) error {
	pathName := fmt.Sprintf("cam_%d", c.ID)
	req, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/v3/config/paths/delete/%s", s.mediamtxURL, pathName), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.Status = "offline"
	s.repo.UpdateCamera(c)
	return nil
}

// ListEvents returns recognition events for a camera.
func (s *CameraService) ListEvents(cameraID uint64, offset, limit int) ([]model.RecognitionEvent, int64, error) {
	return s.repo.ListEvents(cameraID, offset, limit)
}
