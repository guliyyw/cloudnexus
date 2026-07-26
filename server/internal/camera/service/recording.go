package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cloudnexus/server/internal/camera/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/snowflake"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RecordingOptions struct {
	SegmentSeconds       int `json:"segment_seconds"`
	DurationSeconds      int `json:"duration_seconds"`
	RetentionDays        int `json:"retention_days"`
	MaxStorageMB         int `json:"max_storage_mb"`
	TimezoneOffsetMinute int `json:"timezone_offset_minutes"`
}

type RecordingStatus struct {
	Recording       bool             `json:"recording"`
	CameraID        uint64           `json:"camera_id,string"`
	StartedAt       *time.Time       `json:"started_at"`
	EndsAt          *time.Time       `json:"ends_at"`
	SegmentSeconds  int              `json:"segment_seconds"`
	DurationSeconds int              `json:"duration_seconds"`
	RetentionDays   int              `json:"retention_days"`
	MaxStorageMB    int              `json:"max_storage_mb"`
	LastError       string           `json:"last_error"`
	ActiveJobs      []RecordingJobUI `json:"active_jobs,omitempty"`
}

type RecordingJobUI struct {
	CameraID  uint64    `json:"camera_id,string"`
	StartedAt time.Time `json:"started_at"`
	LastError string    `json:"last_error"`
}

type recordingJob struct {
	cameraID   uint64
	cameraName string
	ownerID    uint64
	dir        string
	pattern    string
	opts       RecordingOptions
	started    time.Time
	cancel     context.CancelFunc
	done       chan struct{}
	lastErr    string
}

type RecordingService struct {
	repo       *repository.CameraRepository
	baseDir    string
	minio      *minio.Client
	bucket     string
	mu         sync.Mutex
	cloudDirMu sync.Mutex
	jobs       map[uint64]*recordingJob
}

func NewRecordingService(
	repo *repository.CameraRepository,
	baseDir string,
	minioClient *minio.Client,
	bucket string,
) *RecordingService {
	if baseDir == "" {
		baseDir = "/app/recordings"
	}
	return &RecordingService{
		repo:    repo,
		baseDir: baseDir,
		minio:   minioClient,
		bucket:  bucket,
		jobs:    make(map[uint64]*recordingJob),
	}
}

func (s *RecordingService) StartLegacyMigration() {
	go func() {
		for {
			recordings, err := s.repo.ListLegacyRecordings(20)
			if err != nil {
				zap.L().Warn("legacy camera recording migration failed", zap.Error(err))
				return
			}
			if len(recordings) == 0 {
				return
			}

			migrated := 0
			for i := range recordings {
				rec := &recordings[i]
				info, err := os.Stat(rec.FilePath)
				if err != nil || info.Size() == 0 {
					continue
				}
				camera, err := s.repo.FindCameraByID(rec.CameraID)
				if err != nil {
					continue
				}
				job := &recordingJob{
					cameraID:   rec.CameraID,
					cameraName: camera.Name,
					ownerID:    rec.OwnerID,
				}
				cloudFile, err := s.uploadSegmentToCloudDrive(job, rec.FilePath, info, rec.StartedAt)
				if err != nil {
					zap.L().Warn("legacy camera recording cloud upload failed",
						zap.Uint64("recording_id", rec.ID),
						zap.Error(err),
					)
					continue
				}
				if err := s.repo.SetRecordingCloudFile(rec.ID, cloudFile.ID); err != nil {
					s.rollbackCloudFile(cloudFile, true)
					continue
				}
				if err := os.Remove(rec.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
					zap.L().Warn("legacy camera recording local cleanup failed",
						zap.Uint64("recording_id", rec.ID),
						zap.Error(err),
					)
				}
				migrated++
			}
			if migrated == 0 {
				return
			}
		}
	}()
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

	started := time.Now()
	var ctx context.Context
	var cancel context.CancelFunc
	if opts.DurationSeconds > 0 {
		ctx, cancel = context.WithTimeout(
			context.Background(),
			time.Duration(opts.DurationSeconds)*time.Second,
		)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	job := &recordingJob{
		cameraID:   cameraID,
		cameraName: c.Name,
		ownerID:    ownerID,
		dir:        cameraDir,
		pattern:    filepath.Join(cameraDir, fmt.Sprintf("cam_%d_%%Y%%m%%d_%%H%%M%%S.mp4", cameraID)),
		opts:       opts,
		started:    started,
		cancel:     cancel,
		done:       make(chan struct{}),
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

func (s *RecordingService) List(
	cameraID, ownerID uint64,
	from, to *time.Time,
	offset, limit int,
) ([]model.CameraRecording, int64, error) {
	c, err := s.repo.FindCameraByID(cameraID)
	if err != nil {
		return nil, 0, apperrors.NewAppError(404, "camera not found", apperrors.ErrNotFound)
	}
	if c.OwnerID != ownerID {
		return nil, 0, apperrors.NewAppError(403, "forbidden", apperrors.ErrForbidden)
	}
	return s.repo.ListRecordingsInRange(cameraID, ownerID, from, to, offset, limit)
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
	return s.deleteRecordingAssets(rec)
}

type readSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

func (s *RecordingService) OpenPlayback(recordingID, ownerID uint64) (readSeekCloser, *model.CameraRecording, error) {
	rec, err := s.Get(recordingID, ownerID)
	if err != nil {
		return nil, nil, err
	}
	if rec.FileID == 0 {
		file, err := os.Open(rec.FilePath)
		if err != nil {
			return nil, nil, err
		}
		return file, rec, nil
	}

	cloudFile, err := s.repo.FindCloudFileByID(rec.FileID)
	if err != nil {
		return nil, nil, apperrors.NewAppError(404, "recording file not found in cloud drive", err)
	}
	if cloudFile.UserID != ownerID {
		return nil, nil, apperrors.NewAppError(403, "forbidden", apperrors.ErrForbidden)
	}
	object, err := s.minio.GetObject(context.Background(), s.bucket, cloudFile.StorageKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, err
	}
	if _, err := object.Stat(); err != nil {
		object.Close()
		return nil, nil, err
	}
	return object, rec, nil
}

func (s *RecordingService) deleteRecordingAssets(rec *model.CameraRecording) error {
	if err := os.Remove(rec.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if rec.FileID != 0 {
		cloudFile, err := s.repo.FindCloudFileByID(rec.FileID)
		if err == nil {
			if err := s.minio.RemoveObject(
				context.Background(),
				s.bucket,
				cloudFile.StorageKey,
				minio.RemoveObjectOptions{},
			); err != nil {
				return err
			}
			if err := s.repo.SoftDeleteCloudFile(cloudFile.ID, rec.OwnerID); err != nil {
				return err
			}
			if err := s.repo.AddCloudStorageUsed(rec.OwnerID, -cloudFile.Size); err != nil {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return s.repo.DeleteRecording(rec.ID)
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
	cmd := exec.Command("ffmpeg", args...)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	if err := cmd.Start(); err != nil {
		job.lastErr = err.Error()
		zap.L().Warn("camera recording failed to start", zap.Uint64("camera_id", job.cameraID), zap.Error(err))
		return
	}

	errCh := make(chan error, 1)
	go func() { errCh <- cmd.Wait() }()

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
			// Let FFmpeg finalize the current MP4 segment before forcing exit.
			if cmd.Process != nil {
				_ = cmd.Process.Signal(syscall.SIGINT)
			}
			select {
			case <-errCh:
			case <-time.After(10 * time.Second):
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				<-errCh
			}
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
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "128k",
		"-f", "segment",
		"-segment_time", strconv.Itoa(segmentSeconds),
		"-segment_format_options", "movflags=+faststart",
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
	for _, path := range completedSegmentPaths(files, includeRecent) {
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
		durationSeconds, err := probeDurationSeconds(path)
		if err != nil {
			if includeRecent {
				job.lastErr = err.Error()
				zap.L().Warn("camera recording segment is not finalized",
					zap.Uint64("camera_id", job.cameraID),
					zap.String("file", filepath.Base(path)),
					zap.Error(err),
				)
			}
			continue
		}
		ended := started.Add(time.Duration(durationSeconds) * time.Second)
		cloudFile, err := s.uploadSegmentToCloudDrive(job, path, info, started)
		if err != nil {
			job.lastErr = err.Error()
			zap.L().Warn("camera recording cloud upload failed",
				zap.Uint64("camera_id", job.cameraID),
				zap.String("file", filepath.Base(path)),
				zap.Error(err),
			)
			continue
		}
		rec := &model.CameraRecording{
			BaseModel:       model.BaseModel{ID: snowflake.Uint64()},
			CameraID:        job.cameraID,
			OwnerID:         job.ownerID,
			FileID:          cloudFile.ID,
			FileName:        filepath.Base(path),
			FilePath:        path,
			Status:          "ready",
			StartedAt:       started,
			EndedAt:         &ended,
			DurationSeconds: durationSeconds,
			SizeBytes:       info.Size(),
		}
		if err := s.repo.CreateRecording(rec); err != nil {
			s.rollbackCloudFile(cloudFile, true)
			job.lastErr = err.Error()
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			job.lastErr = err.Error()
		}
	}
}

func completedSegmentPaths(files []string, includeActive bool) []string {
	paths := append([]string(nil), files...)
	sort.Strings(paths)
	if !includeActive && len(paths) > 0 {
		paths = paths[:len(paths)-1]
	}
	return paths
}

func (s *RecordingService) uploadSegmentToCloudDrive(
	job *recordingJob,
	path string,
	info os.FileInfo,
	started time.Time,
) (*model.File, error) {
	localStarted := started.In(recordingLocation(job.opts.TimezoneOffsetMinute))
	s.cloudDirMu.Lock()
	parentID, err := s.ensureRecordingCloudDirectory(job, localStarted)
	s.cloudDirMu.Unlock()
	if err != nil {
		return nil, err
	}

	fileID := snowflake.Uint64()
	displayName := localStarted.Format("15-04-05") + ".mp4"
	storageKey := fmt.Sprintf(
		"%d/%d/camera-recordings/%d.mp4",
		job.ownerID,
		parentID,
		fileID,
	)
	input, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer input.Close()

	if _, err := s.minio.PutObject(
		context.Background(),
		s.bucket,
		storageKey,
		input,
		info.Size(),
		minio.PutObjectOptions{ContentType: "video/mp4"},
	); err != nil {
		return nil, err
	}

	cloudFile := &model.File{
		BaseModel:  model.BaseModel{ID: fileID},
		UserID:     job.ownerID,
		Name:       displayName,
		ParentID:   parentID,
		Size:       info.Size(),
		MimeType:   "video/mp4",
		StorageKey: storageKey,
	}
	if err := s.repo.CreateCloudFile(cloudFile); err != nil {
		_ = s.minio.RemoveObject(context.Background(), s.bucket, storageKey, minio.RemoveObjectOptions{})
		return nil, err
	}
	if err := s.repo.AddCloudStorageUsed(job.ownerID, info.Size()); err != nil {
		s.rollbackCloudFile(cloudFile, false)
		return nil, err
	}
	return cloudFile, nil
}

func (s *RecordingService) ensureRecordingCloudDirectory(job *recordingJob, started time.Time) (uint64, error) {
	root, err := s.ensureCloudDirectory(job.ownerID, 0, "监控录像")
	if err != nil {
		return 0, err
	}
	cameraDir, err := s.ensureCloudDirectory(
		job.ownerID,
		root.ID,
		recordingCameraFolderName(job.cameraName, job.cameraID),
	)
	if err != nil {
		return 0, err
	}
	dateDir, err := s.ensureCloudDirectory(job.ownerID, cameraDir.ID, started.Format("2006-01-02"))
	if err != nil {
		return 0, err
	}
	return dateDir.ID, nil
}

func (s *RecordingService) ensureCloudDirectory(ownerID, parentID uint64, name string) (*model.File, error) {
	existing, err := s.repo.FindCloudFileByName(ownerID, parentID, name)
	if err == nil {
		if !existing.IsDir {
			return nil, fmt.Errorf("cloud drive path %q is not a directory", name)
		}
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	dir := &model.File{
		BaseModel: model.BaseModel{ID: snowflake.Uint64()},
		UserID:    ownerID,
		Name:      name,
		IsDir:     true,
		ParentID:  parentID,
	}
	if err := s.repo.CreateCloudFile(dir); err != nil {
		return nil, err
	}
	return dir, nil
}

func recordingCameraFolderName(name string, cameraID uint64) string {
	name = strings.TrimSpace(name)
	name = strings.NewReplacer("/", "_", "\\", "_").Replace(name)
	if name == "" {
		name = "摄像头"
	}
	id := strconv.FormatUint(cameraID, 10)
	if len(id) > 6 {
		id = id[len(id)-6:]
	}
	return fmt.Sprintf("%s-%s", name, id)
}

func (s *RecordingService) rollbackCloudFile(file *model.File, adjustQuota bool) {
	_ = s.minio.RemoveObject(context.Background(), s.bucket, file.StorageKey, minio.RemoveObjectOptions{})
	_ = s.repo.SoftDeleteCloudFile(file.ID, file.UserID)
	if adjustQuota {
		_ = s.repo.AddCloudStorageUsed(file.UserID, -file.Size)
	}
}

func probeDurationSeconds(path string) (int, error) {
	output, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffprobe %s: %w: %s", filepath.Base(path), err, strings.TrimSpace(string(output)))
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("parse duration for %s: %w", filepath.Base(path), err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("invalid duration for %s", filepath.Base(path))
	}
	seconds := int(duration + 0.5)
	if seconds < 1 {
		seconds = 1
	}
	return seconds, nil
}

func (s *RecordingService) cleanup(job *recordingJob) {
	if job.opts.RetentionDays > 0 {
		before := time.Now().AddDate(0, 0, -job.opts.RetentionDays)
		old, err := s.repo.ListRecordingsBefore(job.cameraID, before)
		if err == nil {
			for _, rec := range old {
				_ = s.deleteRecordingAssets(&rec)
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
		_ = s.deleteRecordingAssets(&rec)
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
		opts.SegmentSeconds = 300
	}
	if opts.SegmentSeconds > 3600 {
		opts.SegmentSeconds = 3600
	}
	if opts.DurationSeconds < 0 {
		opts.DurationSeconds = 0
	}
	if opts.DurationSeconds > 0 && opts.DurationSeconds < 10 {
		opts.DurationSeconds = 10
	}
	if opts.DurationSeconds > 7*24*60*60 {
		opts.DurationSeconds = 7 * 24 * 60 * 60
	}
	if opts.RetentionDays < 0 {
		opts.RetentionDays = 0
	}
	if opts.RetentionDays > 365 {
		opts.RetentionDays = 365
	}
	if opts.MaxStorageMB < 0 {
		opts.MaxStorageMB = 0
	}
	if opts.TimezoneOffsetMinute < -840 || opts.TimezoneOffsetMinute > 840 {
		opts.TimezoneOffsetMinute = 0
	}
	return opts
}

func recordingLocation(offsetMinute int) *time.Location {
	return time.FixedZone("recording-client", -offsetMinute*60)
}

func (j *recordingJob) status() *RecordingStatus {
	started := j.started
	var endsAt *time.Time
	if j.opts.DurationSeconds > 0 {
		end := started.Add(time.Duration(j.opts.DurationSeconds) * time.Second)
		endsAt = &end
	}
	return &RecordingStatus{
		Recording:       true,
		CameraID:        j.cameraID,
		StartedAt:       &started,
		EndsAt:          endsAt,
		SegmentSeconds:  j.opts.SegmentSeconds,
		DurationSeconds: j.opts.DurationSeconds,
		RetentionDays:   j.opts.RetentionDays,
		MaxStorageMB:    j.opts.MaxStorageMB,
		LastError:       j.lastErr,
	}
}
