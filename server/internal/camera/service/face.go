package service

import (
	"encoding/json"
	"math"

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
