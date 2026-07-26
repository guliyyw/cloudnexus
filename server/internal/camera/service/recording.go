package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudnexus/server/internal/camera/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/snowflake"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RecordingOptions struct {
	SegmentSeconds int `json:"segment_seconds"`
	RetentionDays  int `json:"retention_days"`
	MaxStorageMB   int `json:"max_storage_mb"`
}

type RecordingStatus struct {
	Recording      bool             `json:"recording"`
	CameraID       uint64           `json:"camera_id,string"`
	StartedAt      *time.Time       `json:"started_at"`
	SegmentSeconds int              `json:"segment_seconds"`
	RetentionDays  int              `json:"retention_days"`
	MaxStorageMB   int              `json:"max_storage_mb"`
	LastError      string           `json:"last_error"`
	ActiveJobs     []RecordingJobUI `json:"active_jobs,omitempty"`
}

type RecordingJobUI struct {
	CameraID  uint64    `json:"camera_id,string"`
	StartedAt time.Time `json:"started_at"`
	LastError string    `json:"last_error"`
}

type recordingJob struct {
	cameraID uint64
	ownerID  uint64
	dir      string
	pattern  string
	opts     RecordingOptions
	started  time.Time
	cancel   context.CancelFunc
	done     chan struct{}
	lastErr  string
}

type RecordingService struct {
	repo    *repository.CameraRepository
	baseDir string
	mu      sync.Mutex
	jobs    map[uint64]*recordingJob
}

func NewRecordingService(repo *repository.CameraRepository, baseDir string) *RecordingService {
	if baseDir == "" {
		baseDir = "/app/recordings"
	}
	return &RecordingService{
		repo:    repo,
		baseDir: baseDir,
		jobs:    make(map[uint64]*recordingJob),
	}
}

func (s *RecordingService) Start(cameraID, ownerID uint64, opts RecordingOptions) (*RecordingStatus, error) {
	c, err := s.repo.FindCameraByID(cameraID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "camera not found", apperrors.ErrNotFound)
	}
	if c.OwnerID != ownerID {
		return nil, apperrors.NewAppError(403, "forbidden", apperrors.ErrForbidden)
	}
	opts = normalizeRecordingOptions(opts)

	s.mu.Lock()
	if job := s.jobs[cameraID]; job != nil {
		status := job.status()
		s.mu.Unlock()
		return status, nil
	}
	s.mu.Unlock()

	cameraDir := filepath.Join(s.baseDir, fmt.Sprintf("cam_%d", cameraID))
	if err := os.MkdirAll(cameraDir, 0755); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	started := time.Now()
	job := &recordingJob{
		cameraID: cameraID,
		ownerID:  ownerID,
		dir:      cameraDir,
		pattern:  filepath.Join(cameraDir, fmt.Sprintf("cam_%d_%%Y%%m%%d_%%H%%M%%S.mp4", cameraID)),
		opts:     opts,
		started:  started,
		cancel:   cancel,
		done:     make(chan struct{}),
	}

	s.mu.Lock()
	s.jobs[cameraID] = job
	s.mu.Unlock()

	go s.runFFmpeg(ctx, c.StreamURL, job)
	return job.status(), nil
}

func (s *RecordingService) Stop(cameraID, ownerID uint64) error {
	c, err := s.repo.FindCameraByID(cameraID)
	if err != nil {
		return apperrors.NewAppError(404, "camera not found", apperrors.ErrNotFound)
	}
	if c.OwnerID != ownerID {
		return apperrors.NewAppError(403, "forbidden", apperrors.ErrForbidden)
	}

	s.mu.Lock()
	job := s.jobs[cameraID]
	if job != nil {
		delete(s.jobs, cameraID)
	}
	s.mu.Unlock()
	if job == nil {
		return nil
	}

	job.cancel()
	select {
	case <-job.done:
	case <-time.After(10 * time.Second):
	}
	return nil
}

func (s *RecordingService) Status(cameraID, ownerID uint64) (*RecordingStatus, error) {
	c, err := s.repo.FindCameraByID(cameraID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "camera not found", apperrors.ErrNotFound)
	}
	if c.OwnerID != ownerID {
		return nil, apperrors.NewAppError(403, "forbidden", apperrors.ErrForbidden)
	}
	s.mu.Lock()
	job := s.jobs[cameraID]
	s.mu.Unlock()
	if job == nil {
		return &RecordingStatus{Recording: false, CameraID: cameraID}, nil
	}
	return job.status(), nil
}

func (s *RecordingService) List(cameraID, ownerID uint64, offset, limit int) ([]model.CameraRecording, int64, error) {
	c, err := s.repo.FindCameraByID(cameraID)
	if err != nil {
		return nil, 0, apperrors.NewAppError(404, "camera not found", apperrors.ErrNotFound)
	}
	if c.OwnerID != ownerID {
		return nil, 0, apperrors.NewAppError(403, "forbidden", apperrors.ErrForbidden)
	}
	return s.repo.ListRecordings(cameraID, ownerID, offset, limit)
}

func (s *RecordingService) Get(recordingID, ownerID uint64) (*model.CameraRecording, error) {
	rec, err := s.repo.FindRecordingByID(recordingID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "recording not found", apperrors.ErrNotFound)
	}
	if rec.OwnerID != ownerID {
		return nil, apperrors.NewAppError(403, "forbidden", apperrors.ErrForbidden)
	}
	return rec, nil
}

func (s *RecordingService) Delete(recordingID, ownerID uint64) error {
	rec, err := s.Get(recordingID, ownerID)
	if err != nil {
		return err
	}
	if err := os.Remove(rec.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return s.repo.DeleteRecording(recordingID)
}

func (s *RecordingService) runFFmpeg(ctx context.Context, streamURL string, job *recordingJob) {
	defer close(job.done)
	defer func() {
		s.scanSegments(job, true)
		s.cleanup(job)
		s.mu.Lock()
		if s.jobs[job.cameraID] == job {
			delete(s.jobs, job.cameraID)
		}
		s.mu.Unlock()
	}()

	args := buildFFmpegArgs(streamURL, job.pattern, job.opts.SegmentSeconds)
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	errCh := make(chan error, 1)
	go func() { errCh <- cmd.Run() }()

	for {
		select {
		case <-ticker.C:
			s.scanSegments(job, false)
			s.cleanup(job)
		case err := <-errCh:
			if err != nil && ctx.Err() == nil {
				job.lastErr = err.Error()
				zap.L().Warn("camera recording stopped", zap.Uint64("camera_id", job.cameraID), zap.Error(err))
			}
			return
		case <-ctx.Done():
			<-errCh
			return
		}
	}
}

func buildFFmpegArgs(streamURL, pattern string, segmentSeconds int) []string {
	args := []string{"-hide_banner", "-loglevel", "error"}
	if strings.HasPrefix(strings.ToLower(streamURL), "rtsp://") {
		args = append(args, "-rtsp_transport", "tcp")
	}
	args = append(args,
		"-i", streamURL,
		"-an",
		"-c:v", "copy",
		"-f", "segment",
		"-segment_time", strconv.Itoa(segmentSeconds),
		"-reset_timestamps", "1",
		"-strftime", "1",
		pattern,
	)
	return args
}

func (s *RecordingService) scanSegments(job *recordingJob, includeRecent bool) {
	files, err := filepath.Glob(filepath.Join(job.dir, fmt.Sprintf("cam_%d_*.mp4", job.cameraID)))
	if err != nil {
		job.lastErr = err.Error()
		return
	}
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			continue
		}
		if !includeRecent && time.Since(info.ModTime()) < 3*time.Second {
			continue
		}
		if _, err := s.repo.FindRecordingByPath(path); err == nil {
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			job.lastErr = err.Error()
			continue
		}

		started := parseSegmentStart(job.cameraID, filepath.Base(path), info.ModTime())
		ended := started.Add(time.Duration(job.opts.SegmentSeconds) * time.Second)
		rec := &model.CameraRecording{
			BaseModel:       model.BaseModel{ID: snowflake.Uint64()},
			CameraID:        job.cameraID,
			OwnerID:         job.ownerID,
			FileName:        filepath.Base(path),
			FilePath:        path,
			Status:          "ready",
			StartedAt:       started,
			EndedAt:         &ended,
			DurationSeconds: job.opts.SegmentSeconds,
			SizeBytes:       info.Size(),
		}
		if err := s.repo.CreateRecording(rec); err != nil {
			job.lastErr = err.Error()
		}
	}
}

func (s *RecordingService) cleanup(job *recordingJob) {
	if job.opts.RetentionDays > 0 {
		before := time.Now().AddDate(0, 0, -job.opts.RetentionDays)
		old, err := s.repo.ListRecordingsBefore(job.cameraID, before)
		if err == nil {
			for _, rec := range old {
				_ = os.Remove(rec.FilePath)
				_ = s.repo.DeleteRecording(rec.ID)
			}
		}
	}
	if job.opts.MaxStorageMB <= 0 {
		return
	}
	recordings, err := s.repo.ListRecordingsByCamera(job.cameraID)
	if err != nil {
		job.lastErr = err.Error()
		return
	}
	var total int64
	for _, rec := range recordings {
		total += rec.SizeBytes
	}
	limit := int64(job.opts.MaxStorageMB) * 1024 * 1024
	if total <= limit {
		return
	}
	sort.Slice(recordings, func(i, j int) bool {
		return recordings[i].StartedAt.Before(recordings[j].StartedAt)
	})
	for _, rec := range recordings {
		if total <= limit {
			break
		}
		_ = os.Remove(rec.FilePath)
		_ = s.repo.DeleteRecording(rec.ID)
		total -= rec.SizeBytes
	}
}

func parseSegmentStart(cameraID uint64, name string, fallback time.Time) time.Time {
	prefix := fmt.Sprintf("cam_%d_", cameraID)
	stamp := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".mp4")
	if t, err := time.ParseInLocation("20060102_150405", stamp, time.Local); err == nil {
		return t
	}
	return fallback
}

func normalizeRecordingOptions(opts RecordingOptions) RecordingOptions {
	if opts.SegmentSeconds < 10 {
		opts.SegmentSeconds = 60
	}
	if opts.SegmentSeconds > 3600 {
		opts.SegmentSeconds = 3600
	}
	if opts.RetentionDays < 1 {
		opts.RetentionDays = 7
	}
	if opts.RetentionDays > 365 {
		opts.RetentionDays = 365
	}
	if opts.MaxStorageMB < 128 {
		opts.MaxStorageMB = 1024
	}
	return opts
}

func (j *recordingJob) status() *RecordingStatus {
	started := j.started
	return &RecordingStatus{
		Recording:      true,
		CameraID:       j.cameraID,
		StartedAt:      &started,
		SegmentSeconds: j.opts.SegmentSeconds,
		RetentionDays:  j.opts.RetentionDays,
		MaxStorageMB:   j.opts.MaxStorageMB,
		LastError:      j.lastErr,
	}
}
