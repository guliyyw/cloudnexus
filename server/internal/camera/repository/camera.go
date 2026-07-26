package repository

import (
	"time"

	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type CameraRepository struct {
	db *gorm.DB
}

func NewCameraRepository(db *gorm.DB) *CameraRepository {
	return &CameraRepository{db: db}
}

// --- Camera ---

func (r *CameraRepository) ListCameras(ownerID uint64, offset, limit int) ([]model.Camera, int64, error) {
	var total int64
	var cameras []model.Camera
	q := r.db.Model(&model.Camera{}).Where("owner_id = ?", ownerID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&cameras).Error; err != nil {
		return nil, 0, err
	}
	return cameras, total, nil
}

func (r *CameraRepository) FindCameraByID(id uint64) (*model.Camera, error) {
	var c model.Camera
	if err := r.db.First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CameraRepository) ListAllCameras() ([]model.Camera, error) {
	var cameras []model.Camera
	err := r.db.Order("created_at ASC").Find(&cameras).Error
	return cameras, err
}

func (r *CameraRepository) CreateCamera(c *model.Camera) error {
	return r.db.Create(c).Error
}

func (r *CameraRepository) UpdateCamera(c *model.Camera) error {
	return r.db.Save(c).Error
}

func (r *CameraRepository) DeleteCamera(id uint64) error {
	return r.db.Delete(&model.Camera{}, "id = ?", id).Error
}

func (r *CameraRepository) UpdateCameraStatus(id uint64, status string) error {
	return r.db.Model(&model.Camera{}).Where("id = ?", id).Update("status", status).Error
}

func (r *CameraRepository) UpdateCameraHealth(id uint64, online bool, checkedAt time.Time) error {
	updates := map[string]interface{}{
		"status": "offline",
	}
	if online {
		updates["status"] = "online"
		updates["last_seen_at"] = checkedAt
	}
	return r.db.Model(&model.Camera{}).Where("id = ?", id).Updates(updates).Error
}

// --- RecognitionEvent ---

func (r *CameraRepository) ListEvents(cameraID uint64, offset, limit int) ([]model.RecognitionEvent, int64, error) {
	var total int64
	var events []model.RecognitionEvent
	q := r.db.Model(&model.RecognitionEvent{}).Where("camera_id = ?", cameraID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&events).Error; err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (r *CameraRepository) CreateEvent(e *model.RecognitionEvent) error {
	return r.db.Create(e).Error
}

// --- CameraRecording ---

func (r *CameraRepository) CreateRecording(rec *model.CameraRecording) error {
	return r.db.Create(rec).Error
}

func (r *CameraRepository) FindRecordingByID(id uint64) (*model.CameraRecording, error) {
	var rec model.CameraRecording
	if err := r.db.First(&rec, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *CameraRepository) FindRecordingByPath(path string) (*model.CameraRecording, error) {
	var rec model.CameraRecording
	if err := r.db.First(&rec, "file_path = ?", path).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *CameraRepository) ListRecordings(cameraID, ownerID uint64, offset, limit int) ([]model.CameraRecording, int64, error) {
	return r.ListRecordingsInRange(cameraID, ownerID, nil, nil, offset, limit)
}

func (r *CameraRepository) ListRecordingsInRange(
	cameraID, ownerID uint64,
	from, to *time.Time,
	offset, limit int,
) ([]model.CameraRecording, int64, error) {
	var total int64
	var recordings []model.CameraRecording
	q := r.db.Model(&model.CameraRecording{}).
		Where("camera_id = ? AND owner_id = ?", cameraID, ownerID).
		Where(`file_id = 0 OR EXISTS (
			SELECT 1 FROM files
			WHERE files.id = camera_recordings.file_id
			  AND files.deleted_at IS NULL
		)`)
	if from != nil {
		q = q.Where("started_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("started_at < ?", *to)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("started_at DESC").Offset(offset).Limit(limit).Find(&recordings).Error; err != nil {
		return nil, 0, err
	}
	return recordings, total, nil
}

func (r *CameraRepository) ListRecordingsBefore(cameraID uint64, before time.Time) ([]model.CameraRecording, error) {
	var recordings []model.CameraRecording
	err := r.db.Where("camera_id = ? AND started_at < ?", cameraID, before).
		Order("started_at ASC").
		Find(&recordings).Error
	return recordings, err
}

func (r *CameraRepository) ListRecordingsByCamera(cameraID uint64) ([]model.CameraRecording, error) {
	var recordings []model.CameraRecording
	err := r.db.Where("camera_id = ?", cameraID).
		Order("started_at DESC").
		Find(&recordings).Error
	return recordings, err
}

func (r *CameraRepository) DeleteRecording(id uint64) error {
	return r.db.Delete(&model.CameraRecording{}, "id = ?", id).Error
}

func (r *CameraRepository) ListLegacyRecordings(limit int) ([]model.CameraRecording, error) {
	var recordings []model.CameraRecording
	err := r.db.Where("file_id = 0").
		Order("started_at ASC").
		Limit(limit).
		Find(&recordings).Error
	return recordings, err
}

func (r *CameraRepository) SetRecordingCloudFile(id, fileID uint64) error {
	return r.db.Model(&model.CameraRecording{}).
		Where("id = ? AND file_id = 0", id).
		Update("file_id", fileID).Error
}

func (r *CameraRepository) FindCloudFileByID(id uint64) (*model.File, error) {
	var file model.File
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&file).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *CameraRepository) FindCloudFileByName(ownerID, parentID uint64, name string) (*model.File, error) {
	var file model.File
	err := r.db.Where(
		"user_id = ? AND parent_id = ? AND name = ? AND deleted_at IS NULL",
		ownerID, parentID, name,
	).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *CameraRepository) CreateCloudFile(file *model.File) error {
	return r.db.Create(file).Error
}

func (r *CameraRepository) SoftDeleteCloudFile(id, ownerID uint64) error {
	return r.db.Model(&model.File{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, ownerID).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *CameraRepository) AddCloudStorageUsed(ownerID uint64, delta int64) error {
	if err := r.db.Exec(
		`INSERT INTO user_quota (user_id, storage_used, created_at, updated_at)
		 VALUES (?, 0, NOW(), NOW())
		 ON CONFLICT (user_id) DO NOTHING`,
		ownerID,
	).Error; err != nil {
		return err
	}
	return r.db.Exec(
		`UPDATE user_quota
		 SET storage_used = GREATEST(storage_used + ?, 0), updated_at = NOW()
		 WHERE user_id = ?`,
		delta, ownerID,
	).Error
}

// --- FaceProfile ---

func (r *CameraRepository) ListFaceProfiles(ownerID uint64) ([]model.FaceProfile, error) {
	var profiles []model.FaceProfile
	err := r.db.Where("owner_id = ?", ownerID).Order("created_at DESC").Find(&profiles).Error
	return profiles, err
}

func (r *CameraRepository) FindFaceProfileByID(id uint64) (*model.FaceProfile, error) {
	var p model.FaceProfile
	if err := r.db.First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *CameraRepository) CreateFaceProfile(p *model.FaceProfile) error {
	return r.db.Create(p).Error
}

func (r *CameraRepository) UpdateFaceProfile(p *model.FaceProfile) error {
	return r.db.Save(p).Error
}

func (r *CameraRepository) DeleteFaceProfile(id uint64) error {
	return r.db.Delete(&model.FaceProfile{}, "id = ?", id).Error
}

func (r *CameraRepository) AllFaceProfiles(ownerID uint64) ([]model.FaceProfile, error) {
	var profiles []model.FaceProfile
	err := r.db.Where("owner_id = ?", ownerID).Find(&profiles).Error
	return profiles, err
}

// --- FaceRecognitionEvent ---

func (r *CameraRepository) CreateFaceEvent(e *model.FaceRecognitionEvent) error {
	return r.db.Create(e).Error
}

func (r *CameraRepository) ListFaceEvents(cameraID uint64, offset, limit int) ([]model.FaceRecognitionEvent, int64, error) {
	var total int64
	var events []model.FaceRecognitionEvent
	q := r.db.Model(&model.FaceRecognitionEvent{}).Where("camera_id = ?", cameraID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&events).Error; err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (r *CameraRepository) DeleteFaceEventsByCamera(cameraID uint64) (int64, error) {
	result := r.db.Where("camera_id = ?", cameraID).Delete(&model.FaceRecognitionEvent{})
	return result.RowsAffected, result.Error
}

// --- FaceAttendanceSession ---

func (r *CameraRepository) FindActiveAttendanceSession(faceID, cameraID uint64, date string) (*model.FaceAttendanceSession, error) {
	var s model.FaceAttendanceSession
	err := r.db.Where("face_id = ? AND camera_id = ? AND date = ?", faceID, cameraID, date).
		Order("end_time DESC").First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *CameraRepository) UpsertAttendanceSession(s *model.FaceAttendanceSession) error {
	// Try insert; if conflict on unique constraint, update end_time
	return r.db.Where("face_id = ? AND camera_id = ? AND date = ? AND start_time = ?",
		s.FaceID, s.CameraID, s.Date, s.StartTime).
		Assign(map[string]interface{}{"end_time": s.EndTime}).
		FirstOrCreate(s).Error
}

func (r *CameraRepository) ListAttendanceByFace(faceID uint64, dateFrom, dateTo string) ([]model.FaceAttendanceSession, error) {
	var sessions []model.FaceAttendanceSession
	err := r.db.Where("face_id = ? AND date >= ? AND date <= ?", faceID, dateFrom, dateTo).
		Order("start_time ASC").Find(&sessions).Error
	return sessions, err
}

func (r *CameraRepository) ListAttendanceByDate(date string) ([]model.FaceAttendanceSession, error) {
	var sessions []model.FaceAttendanceSession
	err := r.db.Where("date = ?", date).Order("start_time ASC").Find(&sessions).Error
	return sessions, err
}

func (r *CameraRepository) ListAttendanceByCamera(cameraID uint64, date string) ([]model.FaceAttendanceSession, error) {
	var sessions []model.FaceAttendanceSession
	err := r.db.Where("camera_id = ? AND date = ?", cameraID, date).
		Order("start_time ASC").Find(&sessions).Error
	return sessions, err
}

func (r *CameraRepository) FindAttendanceSessionByID(id uint64) (*model.FaceAttendanceSession, error) {
	var s model.FaceAttendanceSession
	if err := r.db.First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *CameraRepository) DeleteAttendanceSession(id uint64) error {
	return r.db.Delete(&model.FaceAttendanceSession{}, "id = ?", id).Error
}

func (r *CameraRepository) DeleteAttendanceByFaceDate(faceID uint64, date string) (int64, error) {
	result := r.db.Where("face_id = ? AND date = ?", faceID, date).Delete(&model.FaceAttendanceSession{})
	return result.RowsAffected, result.Error
}
