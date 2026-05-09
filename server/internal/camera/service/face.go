package service

import (
	"encoding/json"
	"math"
	"time"

	"github.com/cloudnexus/server/internal/camera/repository"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/snowflake"
	apperrors "github.com/cloudnexus/server/pkg/errors"
)

// FaceService handles face profile CRUD and similarity matching.
type FaceService struct {
	repo *repository.CameraRepository
}

func NewFaceService(repo *repository.CameraRepository) *FaceService {
	return &FaceService{repo: repo}
}

// --- Profile CRUD ---

func (s *FaceService) ListProfiles(ownerID uint64) ([]model.FaceProfile, error) {
	return s.repo.ListFaceProfiles(ownerID)
}

func (s *FaceService) CreateProfile(ownerID uint64, name string, embedding []float64, thumbnailURL string) (*model.FaceProfile, error) {
	embJSON, err := json.Marshal(embedding)
	if err != nil {
		return nil, apperrors.NewAppError(400, "embedding 格式错误", apperrors.ErrBadRequest)
	}
	p := &model.FaceProfile{
		OwnerID:      ownerID,
		Name:         name,
		Embedding:    string(embJSON),
		ThumbnailURL: thumbnailURL,
	}
	p.ID = snowflake.Uint64()
	if err := s.repo.CreateFaceProfile(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *FaceService) UpdateProfile(id, ownerID uint64, name string) error {
	existing, err := s.repo.FindFaceProfileByID(id)
	if err != nil {
		return apperrors.NewAppError(404, "人脸不存在", apperrors.ErrNotFound)
	}
	if existing.OwnerID != ownerID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	existing.Name = name
	return s.repo.UpdateFaceProfile(existing)
}

func (s *FaceService) DeleteProfile(id, ownerID uint64) error {
	existing, err := s.repo.FindFaceProfileByID(id)
	if err != nil {
		return apperrors.NewAppError(404, "人脸不存在", apperrors.ErrNotFound)
	}
	if existing.OwnerID != ownerID {
		return apperrors.NewAppError(403, "无权操作", apperrors.ErrForbidden)
	}
	return s.repo.DeleteFaceProfile(id)
}

// --- Matching ---

// MatchResult is the result of matching a face embedding against the library.
type MatchResult struct {
	Matched    bool    `json:"matched"`
	FaceID     *uint64 `json:"face_id,string"`
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
}

// MatchEmbedding compares a query embedding against all stored profiles.
func (s *FaceService) MatchEmbedding(ownerID uint64, query []float64) (*MatchResult, error) {
	profiles, err := s.repo.AllFaceProfiles(ownerID)
	if err != nil {
		return nil, err
	}

	var bestMatch *model.FaceProfile
	var bestScore float64

	for i := range profiles {
		var stored []float64
		if err := json.Unmarshal([]byte(profiles[i].Embedding), &stored); err != nil {
			continue
		}
		score := cosineSimilarity(query, stored)
		if score > bestScore {
			bestScore = score
			p := profiles[i]
			bestMatch = &p
		}
	}

	if bestMatch != nil && bestScore >= 0.6 {
		id := bestMatch.ID
		return &MatchResult{
			Matched:    true,
			FaceID:     &id,
			Name:       bestMatch.Name,
			Confidence: bestScore,
		}, nil
	}
	return &MatchResult{Matched: false, Confidence: bestScore}, nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (normA * normB)
}

// --- Events ---

func (s *FaceService) RecordEvent(cameraID uint64, result *MatchResult, bbox map[string]float64) (*model.FaceRecognitionEvent, error) {
	bboxJSON, _ := json.Marshal(bbox)
	event := &model.FaceRecognitionEvent{
		ID:        snowflake.Uint64(),
		CameraID:  cameraID,
		FaceID:    result.FaceID,
		FaceName:  result.Name,
		Confidence: result.Confidence,
		BboxJSON: string(bboxJSON),
	}
	if err := s.repo.CreateFaceEvent(event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *FaceService) ListEvents(cameraID uint64, offset, limit int) ([]model.FaceRecognitionEvent, int64, error) {
	return s.repo.ListFaceEvents(cameraID, offset, limit)
}

// --- Attendance ---

const attendanceGracePeriod = 5 * time.Minute

// RecordAttendance upserts an attendance session for a matched face.
// If the same face was seen within the grace period, extend the existing session.
// Otherwise, create a new session.
func (s *FaceService) RecordAttendance(cameraID uint64, match *MatchResult) (*model.FaceAttendanceSession, error) {
	if !match.Matched || match.FaceID == nil {
		return nil, nil
	}
	now := time.Now()
	date := now.Format("2006-01-02")

	existing, err := s.repo.FindActiveAttendanceSession(*match.FaceID, cameraID, date)
	if err == nil && existing != nil {
		// Extend if within grace period
		if now.Sub(existing.EndTime) <= attendanceGracePeriod {
			existing.EndTime = now
			if err := s.repo.UpsertAttendanceSession(existing); err != nil {
				return nil, err
			}
			return existing, nil
		}
	}

	// New session
	session := &model.FaceAttendanceSession{
		ID:        snowflake.Uint64(),
		FaceID:    *match.FaceID,
		FaceName:  match.Name,
		CameraID:  cameraID,
		StartTime: now,
		EndTime:   now,
		Date:      date,
	}
	if err := s.repo.UpsertAttendanceSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

// DailyAttendance summarizes check-in/check-out for all faces on a date.
type DailyAttendance struct {
	FaceID     uint64    `json:"face_id,string"`
	FaceName   string    `json:"face_name"`
	Date       string    `json:"date"`
	CheckIn    time.Time `json:"check_in"`  // earliest session start
	CheckOut   time.Time `json:"check_out"` // latest session end
	SessionCount int     `json:"session_count"`
}

func (s *FaceService) GetDailyAttendance(date string) ([]DailyAttendance, error) {
	sessions, err := s.repo.ListAttendanceByDate(date)
	if err != nil {
		return nil, err
	}

	// Group by face_id: min start_time = check-in, max end_time = check-out
	type agg struct {
		faceName string
		checkIn  time.Time
		checkOut time.Time
		count    int
	}
	grouped := make(map[uint64]*agg)
	for i := range sessions {
		s := &sessions[i]
		a, ok := grouped[s.FaceID]
		if !ok {
			a = &agg{faceName: s.FaceName, checkIn: s.StartTime, checkOut: s.EndTime, count: 0}
			grouped[s.FaceID] = a
		}
		if s.StartTime.Before(a.checkIn) {
			a.checkIn = s.StartTime
		}
		if s.EndTime.After(a.checkOut) {
			a.checkOut = s.EndTime
		}
		a.count++
	}

	result := make([]DailyAttendance, 0, len(grouped))
	for faceID, a := range grouped {
		result = append(result, DailyAttendance{
			FaceID:       faceID,
			FaceName:     a.faceName,
			Date:         date,
			CheckIn:      a.checkIn,
			CheckOut:     a.checkOut,
			SessionCount: a.count,
		})
	}
	return result, nil
}

func (s *FaceService) GetAttendanceByFace(faceID uint64, dateFrom, dateTo string) ([]model.FaceAttendanceSession, error) {
	if dateFrom == "" {
		dateFrom = time.Now().Format("2006-01-02")
	}
	if dateTo == "" {
		dateTo = dateFrom
	}
	return s.repo.ListAttendanceByFace(faceID, dateFrom, dateTo)
}

func (s *FaceService) GetAttendanceByCamera(cameraID uint64, date string) ([]model.FaceAttendanceSession, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	return s.repo.ListAttendanceByCamera(cameraID, date)
}
