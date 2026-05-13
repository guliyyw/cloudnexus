package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/cloudnexus/server/internal/camera/repository"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/snowflake"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"go.uber.org/zap"
)

// DetectResponse wraps the YOLO inference result.
type DetectResponse struct {
	Objects []DetectedObject `json:"objects"`
	Error   string           `json:"error,omitempty"`
}

// DetectedObject represents a single detection.
type DetectedObject struct {
	Class      string  `json:"class"`
	Confidence float64 `json:"confidence"`
	X1         float64 `json:"x1"`
	Y1         float64 `json:"y1"`
	X2         float64 `json:"x2"`
	Y2         float64 `json:"y2"`
}

// RecognitionService handles AI recognition via the ai-inference container.
type RecognitionService struct {
	repo           *repository.CameraRepository
	inferenceURL   string // e.g. http://ai-inference:8000
	inferenceToken string // shared secret for AI inference auth
	stopChans      map[uint64]chan struct{}
}

func NewRecognitionService(repo *repository.CameraRepository, inferenceURL, inferenceToken string) *RecognitionService {
	return &RecognitionService{
		repo:           repo,
		inferenceURL:   inferenceURL,
		inferenceToken: inferenceToken,
		stopChans:      make(map[uint64]chan struct{}),
	}
}

// StartRecognition begins periodic frame capture and AI inference for a camera.
func (s *RecognitionService) StartRecognition(cameraID uint64) error {
	c, err := s.repo.FindCameraByID(cameraID)
	if err != nil {
		return apperrors.NewAppError(404, "摄像头不存在", apperrors.ErrNotFound)
	}
	if c.Status != "online" {
		return apperrors.NewAppError(400, "摄像头未在线，请先开启视频流", apperrors.ErrBadRequest)
	}

	// Prevent duplicate start
	if _, exists := s.stopChans[cameraID]; exists {
		return apperrors.NewAppError(400, "AI 识别已在运行", apperrors.ErrBadRequest)
	}

	stopCh := make(chan struct{})
	s.stopChans[cameraID] = stopCh

	go s.recognitionLoop(c, stopCh)
	zap.L().Info("AI recognition started", zap.Uint64("camera_id", cameraID))
	return nil
}

// StopRecognition stops the recognition loop for a camera.
func (s *RecognitionService) StopRecognition(cameraID uint64) error {
	ch, exists := s.stopChans[cameraID]
	if !exists {
		return apperrors.NewAppError(400, "AI 识别未运行", apperrors.ErrBadRequest)
	}
	close(ch)
	delete(s.stopChans, cameraID)
	zap.L().Info("AI recognition stopped", zap.Uint64("camera_id", cameraID))
	return nil
}

func (s *RecognitionService) recognitionLoop(c *model.Camera, stopCh chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			s.detectAndSave(c)
		}
	}
}

func (s *RecognitionService) detectAndSave(c *model.Camera) {
	// Capture a frame from the MediaMTX stream via JPEG snapshot.
	// MediaMTX exposes snapshot at GET /{path_name}
	frame, err := s.captureFrame(c.ID)
	if err != nil {
		zap.L().Warn("frame capture failed", zap.Uint64("camera_id", c.ID), zap.Error(err))
		return
	}
	if len(frame) == 0 {
		return
	}

	objects, err := s.sendToInference(frame)
	if err != nil {
		zap.L().Warn("inference failed", zap.Uint64("camera_id", c.ID), zap.Error(err))
		return
	}

	for _, obj := range objects {
		meta, _ := json.Marshal(map[string]interface{}{
			"bbox": map[string]float64{"x1": obj.X1, "y1": obj.Y1, "x2": obj.X2, "y2": obj.Y2},
		})
		event := &model.RecognitionEvent{
			ID:        snowflake.Uint64(),
			CameraID:  c.ID,
			EventType: obj.Class,
			Confidence: obj.Confidence,
			Metadata: string(meta),
			CreatedAt: time.Now(),
		}
		if err := s.repo.CreateEvent(event); err != nil {
			zap.L().Error("save event failed", zap.Error(err))
		}
	}

	if len(objects) > 0 {
		zap.L().Info("objects detected",
			zap.Uint64("camera_id", c.ID),
			zap.Int("count", len(objects)),
		)
	}
}

func (s *RecognitionService) captureFrame(cameraID uint64) ([]byte, error) {
	// Use ffmpeg to grab a single JPEG frame from the RTSP stream.
	// Alternative: HTTP GET from MediaMTX snapshot endpoint.
	url := fmt.Sprintf("http://mediamtx:8889/cam_%d", cameraID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (s *RecognitionService) sendToInference(imageBytes []byte) ([]DetectedObject, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("image", "frame.jpg")
	if err != nil {
		return nil, err
	}
	part.Write(imageBytes)
	w.Close()

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/detect", s.inferenceURL), &buf)
	if err != nil {
		return nil, fmt.Errorf("创建推理请求失败: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if s.inferenceToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.inferenceToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("推理服务请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result DetectResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("推理结果解析失败: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("推理服务错误: %s", result.Error)
	}

	// Filter low-confidence detections
	var filtered []DetectedObject
	for _, obj := range result.Objects {
		if obj.Confidence >= 0.5 {
			filtered = append(filtered, obj)
		}
	}
	return filtered, nil
}

// DetectImage performs one-shot object detection on an uploaded image.
func (s *RecognitionService) DetectImage(imageBytes []byte) ([]DetectedObject, error) {
	return s.sendToInference(imageBytes)
}

// VideoDetection groups objects detected at a specific timestamp.
type VideoDetection struct {
	Time    float64         `json:"time"`
	Objects []DetectedObject `json:"objects"`
}

// VideoDetectResponse wraps the /detect-video inference result.
type VideoDetectResponse struct {
	Error          string           `json:"error,omitempty"`
	VideoDuration  float64          `json:"video_duration"`
	FPS            float64          `json:"fps"`
	FramesAnalyzed int              `json:"frames_analyzed"`
	Detections     []VideoDetection `json:"detections"`
}

// DetectVideo sends a video file to the AI inference service for frame-by-frame analysis.
func (s *RecognitionService) DetectVideo(videoBytes []byte, filename string, interval float64) (*VideoDetectResponse, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("video", filename)
	if err != nil {
		return nil, err
	}
	part.Write(videoBytes)
	w.Close()

	url := fmt.Sprintf("%s/detect-video?interval=%.1f", s.inferenceURL, interval)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("创建视频分析请求失败: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if s.inferenceToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.inferenceToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("视频分析请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result VideoDetectResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("视频分析结果解析失败: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("视频分析错误: %s", result.Error)
	}
	return &result, nil
}

