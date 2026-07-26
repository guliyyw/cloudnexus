package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cloudnexus/server/internal/camera/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MediaMTX path config request.
type pathConfig struct {
	Name           string `json:"name"`
	Source         string `json:"source"`
	SourceOnDemand bool   `json:"sourceOnDemand"`
}

// CameraService manages cameras and the MediaMTX proxy.
type CameraService struct {
	repo             *repository.CameraRepository
	mediamtxURL      string // e.g. http://mediamtx:8889
	mediamtxUser     string
	mediamtxPassword string
	liveMu           sync.Mutex
	liveStreams      map[uint64]*liveStream
}

type liveStream struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func NewCameraService(repo *repository.CameraRepository, mediamtxURL, mediamtxUser, mediamtxPassword string) *CameraService {
	return &CameraService{
		repo:             repo,
		mediamtxURL:      mediamtxURL,
		mediamtxUser:     mediamtxUser,
		mediamtxPassword: mediamtxPassword,
		liveStreams:      make(map[uint64]*liveStream),
	}
}

func (s *CameraService) StartStatusMonitor(interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	go func() {
		s.refreshCameraStatuses()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.refreshCameraStatuses()
		}
	}()
}

func (s *CameraService) refreshCameraStatuses() {
	cameras, err := s.repo.ListAllCameras()
	if err != nil {
		zap.L().Warn("list cameras for status monitor failed", zap.Error(err))
		return
	}

	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i := range cameras {
		camera := cameras[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			online := probeCameraAddress(ctx, &camera)
			if err := s.repo.UpdateCameraHealth(camera.ID, online, time.Now()); err != nil {
				zap.L().Warn("update camera health failed",
					zap.Uint64("camera_id", camera.ID),
					zap.Error(err),
				)
			}
		}()
	}
	wg.Wait()
}

func probeCameraAddress(ctx context.Context, camera *model.Camera) bool {
	parsed, err := url.Parse(camera.StreamURL)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	if host == "" {
		return false
	}
	port := parsed.Port()
	if port == "" {
		switch strings.ToLower(parsed.Scheme) {
		case "rtsp":
			port = "554"
		case "rtmp":
			port = "1935"
		case "https":
			port = "443"
		default:
			port = "80"
		}
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
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
	s.stopLiveStream(cameraID)
	_ = s.deleteMediaMTXPath(pathName)

	cfg := pathConfig{
		Name:           pathName,
		Source:         "publisher",
		SourceOnDemand: false,
	}

	body, _ := json.Marshal(cfg)
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/v3/config/paths/add/%s", s.mediamtxURL, pathName),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.setMediaMTXAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("mediamtx API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("mediamtx 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	if err := s.startLiveStream(cameraID, pathName, c.StreamURL); err != nil {
		_ = s.deleteMediaMTXPath(pathName)
		return "", "", err
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
	s.stopLiveStream(c.ID)
	return s.deleteMediaMTXPath(pathName)
}

func (s *CameraService) deleteMediaMTXPath(pathName string) error {
	req, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/v3/config/paths/delete/%s", s.mediamtxURL, pathName), nil)
	s.setMediaMTXAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (s *CameraService) startLiveStream(cameraID uint64, pathName, streamURL string) error {
	publishURL, err := s.mediaMTXPublishURL(pathName)
	if err != nil {
		return fmt.Errorf("build mediamtx publish URL: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "ffmpeg", buildLiveFFmpegArgs(streamURL, publishURL)...)
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start live audio transcoder: %w", err)
	}

	stream := &liveStream{cancel: cancel, done: make(chan struct{})}
	s.liveMu.Lock()
	s.liveStreams[cameraID] = stream
	s.liveMu.Unlock()

	go func() {
		err := cmd.Wait()
		close(stream.done)

		s.liveMu.Lock()
		if s.liveStreams[cameraID] == stream {
			delete(s.liveStreams, cameraID)
		}
		s.liveMu.Unlock()

		if err != nil && ctx.Err() == nil {
			zap.L().Warn("live camera transcoder stopped",
				zap.Uint64("camera_id", cameraID),
				zap.Error(err))
		}
	}()

	if err := s.waitForPublishedPath(ctx, pathName, 12*time.Second); err != nil {
		s.stopLiveStream(cameraID)
		return err
	}
	if err := s.waitForHLSManifest(ctx, pathName, 12*time.Second); err != nil {
		s.stopLiveStream(cameraID)
		return err
	}
	return nil
}

func (s *CameraService) stopLiveStream(cameraID uint64) {
	s.liveMu.Lock()
	stream := s.liveStreams[cameraID]
	if stream != nil {
		delete(s.liveStreams, cameraID)
	}
	s.liveMu.Unlock()
	if stream == nil {
		return
	}

	stream.cancel()
	select {
	case <-stream.done:
	case <-time.After(3 * time.Second):
		zap.L().Warn("timed out stopping live camera transcoder", zap.Uint64("camera_id", cameraID))
	}
}

func (s *CameraService) waitForPublishedPath(ctx context.Context, pathName string, timeout time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
			fmt.Sprintf("%s/v3/paths/get/%s", s.mediamtxURL, pathName), nil)
		s.setMediaMTXAuth(req)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("live camera transcoder stopped before publishing")
		case <-deadline.C:
			return fmt.Errorf("timed out waiting for live camera stream")
		case <-ticker.C:
		}
	}
}

func (s *CameraService) waitForHLSManifest(ctx context.Context, pathName string, timeout time.Duration) error {
	apiURL, err := url.Parse(s.mediamtxURL)
	if err != nil {
		return err
	}
	host := apiURL.Hostname()
	if host == "" {
		return fmt.Errorf("mediamtx API URL has no hostname")
	}

	manifestURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, "8888"),
		Path:   "/" + pathName + "/index.m3u8",
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		req, _ := http.NewRequestWithContext(waitCtx, http.MethodGet, manifestURL.String(), nil)
		s.setMediaMTXAuth(req)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-waitCtx.Done():
			return fmt.Errorf("timed out waiting for HLS manifest")
		case <-ticker.C:
		}
	}
}

func (s *CameraService) mediaMTXPublishURL(pathName string) (string, error) {
	apiURL, err := url.Parse(s.mediamtxURL)
	if err != nil {
		return "", err
	}
	host := apiURL.Hostname()
	if host == "" {
		return "", fmt.Errorf("mediamtx API URL has no hostname")
	}

	publishURL := &url.URL{
		Scheme: "rtsp",
		Host:   net.JoinHostPort(host, "8554"),
		Path:   "/" + pathName,
	}
	if s.mediamtxUser != "" || s.mediamtxPassword != "" {
		publishURL.User = url.UserPassword(s.mediamtxUser, s.mediamtxPassword)
	}
	return publishURL.String(), nil
}

func buildLiveFFmpegArgs(streamURL, publishURL string) []string {
	return []string{
		"-hide_banner",
		"-loglevel", "warning",
		"-rtsp_transport", "tcp",
		"-i", streamURL,
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-c:v", "copy",
		"-c:a", "aac",
		"-ar", "48000",
		"-ac", "1",
		"-b:a", "64k",
		"-f", "rtsp",
		"-rtsp_transport", "tcp",
		publishURL,
	}
}

func (s *CameraService) setMediaMTXAuth(req *http.Request) {
	if s.mediamtxUser != "" || s.mediamtxPassword != "" {
		req.SetBasicAuth(s.mediamtxUser, s.mediamtxPassword)
	}
}

// ListEvents returns recognition events for a camera.
func (s *CameraService) ListEvents(cameraID uint64, offset, limit int) ([]model.RecognitionEvent, int64, error) {
	return s.repo.ListEvents(cameraID, offset, limit)
}
