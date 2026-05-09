package repository

import (
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
