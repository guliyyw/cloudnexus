package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudnexus/server/internal/drama/repository"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/minio/minio-go/v7"
)

type DramaService struct {
	repo        *repository.DramaRepository
	minioClient *minio.Client
	bucket      string
	taskRunner  *TaskRunner
}

func NewDramaService(repo *repository.DramaRepository, minioClient *minio.Client, bucket string) *DramaService {
	return &DramaService{repo: repo, minioClient: minioClient, bucket: bucket}
}

func (s *DramaService) SetTaskRunner(runner *TaskRunner) {
	s.taskRunner = runner
}

type ProjectDetail struct {
	Project     *model.DramaProject            `json:"project"`
	Storyboards []model.DramaStoryboard        `json:"storyboards"`
	Media       []model.DramaStoryboardMedia   `json:"media"`
	Segments    []model.DramaStoryboardSegment `json:"segments"`
	Assets      []model.DramaAsset             `json:"assets"`
	Tasks       []model.DramaTask              `json:"tasks"`
	Summary     map[string]interface{}         `json:"summary"`
}

type ExportPayload struct {
	Version     int                            `json:"version"`
	ExportedAt  time.Time                      `json:"exported_at"`
	Project     model.DramaProject             `json:"project"`
	Storyboards []model.DramaStoryboard        `json:"storyboards"`
	Segments    []model.DramaStoryboardSegment `json:"segments"`
	Media       []model.DramaStoryboardMedia   `json:"media"`
	Assets      []model.DramaAsset             `json:"assets"`
}

type CreateTaskInput struct {
	Type          string   `json:"type"`
	StoryboardIDs []string `json:"storyboard_ids"`
	Payload       string   `json:"payload"`
}

type AudioImportResult struct {
	FileName   string                 `json:"file_name"`
	Storyboard *model.DramaStoryboard `json:"storyboard,omitempty"`
	Matched    bool                   `json:"matched"`
	Reason     string                 `json:"reason,omitempty"`
}

type AIAssetImport struct {
	Characters []map[string]interface{} `json:"characters"`
	Scenes     []map[string]interface{} `json:"scenes"`
}

type AISegmentImport struct {
	Segments []AISegment `json:"segments"`
}

type AISegment struct {
	Seq               int      `json:"seq"`
	Title             string   `json:"title"`
	DurationSec       int      `json:"duration_sec"`
	Purpose           string   `json:"purpose"`
	Characters        []string `json:"characters"`
	Scene             string   `json:"scene"`
	Dialogue          string   `json:"dialogue"`
	Action            string   `json:"action"`
	Shot              string   `json:"shot"`
	CompositionPrompt string   `json:"composition_prompt"`
	ReferencePrompt   string   `json:"reference_prompt"`
	VideoPrompt       string   `json:"video_prompt"`
	NegativePrompt    string   `json:"negative_prompt"`
}

func (s *DramaService) ListProjects(ownerID uint64, keyword, sort string, page, pageSize int) ([]model.DramaProject, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListProjects(ownerID, keyword, sort, page, pageSize)
}

func (s *DramaService) CreateProject(ownerID uint64, title, description string) (*model.DramaProject, error) {
	project := &model.DramaProject{
		OwnerID:     ownerID,
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(description),
		Settings:    "{}",
	}
	if project.Title == "" {
		project.Title = "未命名短剧"
	}
	if err := s.repo.CreateProject(project); err != nil {
		return nil, err
	}
	if _, err := s.ensureProjectFolders(ownerID, project.Title); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *DramaService) GetProject(ownerID, id uint64) (*ProjectDetail, error) {
	project, err := s.repo.GetProject(ownerID, id)
	if err != nil {
		return nil, err
	}
	storyboards, err := s.repo.ListStoryboards(ownerID, id)
	if err != nil {
		return nil, err
	}
	assets, err := s.repo.ListAssets(ownerID, id)
	if err != nil {
		return nil, err
	}
	media, err := s.repo.ListStoryboardMedia(ownerID, id)
	if err != nil {
		return nil, err
	}
	segments, err := s.repo.ListStoryboardSegments(ownerID, id)
	if err != nil {
		return nil, err
	}
	hydrateSegmentReferenceFiles(segments, assets)
	tasks, err := s.repo.ListTasks(ownerID, id)
	if err != nil {
		return nil, err
	}
	modified := 0
	for _, storyboard := range storyboards {
		if storyboard.Modified {
			modified++
		}
	}
	return &ProjectDetail{
		Project:     project,
		Storyboards: storyboards,
		Media:       media,
		Segments:    segments,
		Assets:      assets,
		Tasks:       tasks,
		Summary: map[string]interface{}{
			"storyboard_count": len(storyboards),
			"asset_count":      len(assets),
			"modified_count":   modified,
		},
	}, nil
}

func (s *DramaService) SelectStoryboardMedia(ownerID, projectID, storyboardID, mediaID uint64) (*model.DramaStoryboard, error) {
	storyboard, err := s.repo.GetStoryboard(ownerID, projectID, storyboardID)
	if err != nil {
		return nil, err
	}
	media, err := s.repo.GetStoryboardMedia(ownerID, projectID, storyboardID, mediaID)
	if err != nil {
		return nil, err
	}
	if media.SegmentID != 0 {
		return nil, fmt.Errorf("片段图片不能设为分镜当前图")
	}
	if err := s.repo.SelectStoryboardMedia(ownerID, projectID, storyboardID, mediaID, media.Kind); err != nil {
		return nil, err
	}
	switch media.Kind {
	case "image":
		storyboard.ImageFileID = media.FileID
	case "video":
		storyboard.VideoFileID = media.FileID
	}
	if err := s.repo.UpdateStoryboard(storyboard); err != nil {
		return nil, err
	}
	return storyboard, nil
}

func (s *DramaService) DeleteStoryboardMedia(ownerID, projectID, storyboardID, mediaID uint64) (*model.DramaStoryboard, []model.DramaStoryboardMedia, error) {
	storyboard, err := s.repo.GetStoryboard(ownerID, projectID, storyboardID)
	if err != nil {
		return nil, nil, err
	}
	media, err := s.repo.GetStoryboardMedia(ownerID, projectID, storyboardID, mediaID)
	if err != nil {
		return nil, nil, err
	}
	if media.FileID != 0 && media.Source != "asset_reference" {
		if err := s.repo.SoftDeleteFiles(ownerID, []uint64{media.FileID}); err != nil {
			return nil, nil, err
		}
	}
	if err := s.repo.DeleteStoryboardMedia(ownerID, projectID, storyboardID, mediaID); err != nil {
		return nil, nil, err
	}

	remaining, err := s.repo.ListStoryboardMediaByStoryboard(ownerID, projectID, storyboardID, media.Kind)
	if err != nil {
		return nil, nil, err
	}
	storyboardLevelRemaining := filterStoryboardLevelMedia(remaining)
	currentDeleted := media.SegmentID == 0 && media.Selected
	switch media.Kind {
	case "image":
		currentDeleted = currentDeleted || (media.SegmentID == 0 && storyboard.ImageFileID == media.FileID)
	case "video":
		currentDeleted = currentDeleted || (media.SegmentID == 0 && storyboard.VideoFileID == media.FileID)
	}
	if currentDeleted {
		if len(storyboardLevelRemaining) > 0 {
			replacement := storyboardLevelRemaining[0]
			if err := s.repo.SelectStoryboardMedia(ownerID, projectID, storyboardID, replacement.ID, replacement.Kind); err != nil {
				return nil, nil, err
			}
			switch replacement.Kind {
			case "image":
				storyboard.ImageFileID = replacement.FileID
			case "video":
				storyboard.VideoFileID = replacement.FileID
			}
		} else {
			switch media.Kind {
			case "image":
				storyboard.ImageFileID = 0
			case "video":
				storyboard.VideoFileID = 0
			}
		}
		if err := s.repo.UpdateStoryboard(storyboard); err != nil {
			return nil, nil, err
		}
		remaining, err = s.repo.ListStoryboardMediaByStoryboard(ownerID, projectID, storyboardID, media.Kind)
		if err != nil {
			return nil, nil, err
		}
	}
	allMedia, err := s.repo.ListStoryboardMediaByStoryboard(ownerID, projectID, storyboardID, "")
	if err != nil {
		return nil, nil, err
	}
	return storyboard, allMedia, nil
}

func filterStoryboardLevelMedia(media []model.DramaStoryboardMedia) []model.DramaStoryboardMedia {
	result := make([]model.DramaStoryboardMedia, 0, len(media))
	for _, item := range media {
		if item.SegmentID == 0 {
			result = append(result, item)
		}
	}
	return result
}

func (s *DramaService) ImportStoryboardSegments(ownerID, projectID, storyboardID uint64, text string) ([]model.DramaStoryboardSegment, error) {
	if _, err := s.repo.GetProject(ownerID, projectID); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetStoryboard(ownerID, projectID, storyboardID); err != nil {
		return nil, err
	}
	payload, err := parseAISegments(text)
	if err != nil {
		return nil, err
	}
	segments := make([]model.DramaStoryboardSegment, 0, len(payload.Segments))
	for index, item := range payload.Segments {
		seq := item.Seq
		if seq <= 0 {
			seq = index + 1
		}
		duration := item.DurationSec
		if duration <= 0 {
			duration = 3
		}
		segments = append(segments, model.DramaStoryboardSegment{
			ProjectID:         projectID,
			StoryboardID:      storyboardID,
			OwnerID:           ownerID,
			Seq:               seq,
			Title:             strings.TrimSpace(item.Title),
			DurationSec:       duration,
			Purpose:           strings.TrimSpace(item.Purpose),
			Characters:        strings.Join(item.Characters, ", "),
			Scene:             strings.TrimSpace(item.Scene),
			Dialogue:          strings.TrimSpace(item.Dialogue),
			Action:            strings.TrimSpace(item.Action),
			Shot:              strings.TrimSpace(item.Shot),
			CompositionPrompt: strings.TrimSpace(item.CompositionPrompt),
			ReferencePrompt:   strings.TrimSpace(item.ReferencePrompt),
			VideoPrompt:       strings.TrimSpace(item.VideoPrompt),
			NegativePrompt:    strings.TrimSpace(item.NegativePrompt),
		})
	}
	assets, err := s.repo.ListAssets(ownerID, projectID)
	if err != nil {
		return nil, err
	}
	hydrateSegmentReferenceFiles(segments, assets)
	if err := s.repo.ReplaceStoryboardSegments(ownerID, projectID, storyboardID, segments); err != nil {
		return nil, err
	}
	all, err := s.repo.ListStoryboardSegments(ownerID, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]model.DramaStoryboardSegment, 0, len(segments))
	for _, segment := range all {
		if segment.StoryboardID == storyboardID {
			result = append(result, segment)
		}
	}
	return result, nil
}

func hydrateSegmentReferenceFiles(segments []model.DramaStoryboardSegment, assets []model.DramaAsset) {
	if len(segments) == 0 || len(assets) == 0 {
		return
	}
	for index := range segments {
		if segments[index].ReferenceFileID != 0 {
			continue
		}
		if fileID := matchSegmentAssetReference(segments[index], assets); fileID != 0 {
			segments[index].ReferenceFileID = fileID
		}
	}
}

func matchSegmentAssetReference(segment model.DramaStoryboardSegment, assets []model.DramaAsset) uint64 {
	scene := strings.TrimSpace(segment.Scene)
	characters := strings.TrimSpace(segment.Characters)
	for _, asset := range assets {
		if asset.ReferenceFileID == 0 {
			continue
		}
		name := strings.TrimSpace(asset.Name)
		if name == "" {
			continue
		}
		if asset.Type == "scene" && scene != "" && strings.Contains(scene, name) {
			return asset.ReferenceFileID
		}
	}
	for _, asset := range assets {
		if asset.ReferenceFileID == 0 || asset.Type != "character" {
			continue
		}
		name := strings.TrimSpace(asset.Name)
		if name != "" && strings.Contains(characters, name) {
			return asset.ReferenceFileID
		}
	}
	return 0
}

func (s *DramaService) UpdateProject(ownerID, id uint64, title, description, preface, settings string) (*model.DramaProject, error) {
	project, err := s.repo.GetProject(ownerID, id)
	if err != nil {
		return nil, err
	}
	if title != "" {
		project.Title = strings.TrimSpace(title)
	}
	project.Description = description
	project.Preface = preface
	if settings != "" {
		project.Settings = settings
	}
	if err := s.repo.UpdateProject(project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *DramaService) DeleteProject(ownerID, id uint64) error {
	project, err := s.repo.GetProject(ownerID, id)
	if err != nil {
		return err
	}
	if err := s.moveProjectFilesToTrash(ownerID, project.ID, project.Title); err != nil {
		return err
	}
	return s.repo.DeleteProject(ownerID, id)
}

func (s *DramaService) ParseAndSave(ownerID, projectID uint64, script string) (*ProjectDetail, error) {
	project, err := s.repo.GetProject(ownerID, projectID)
	if err != nil {
		return nil, err
	}
	parsed := ParseScript(script)
	project.RawScript = script
	project.Preface = parsed.Preface
	if err := s.repo.UpdateProject(project); err != nil {
		return nil, err
	}

	storyboards := make([]model.DramaStoryboard, 0, len(parsed.Storyboards))
	for _, p := range parsed.Storyboards {
		storyboards = append(storyboards, model.DramaStoryboard{
			ProjectID:   projectID,
			OwnerID:     ownerID,
			Seq:         p.Seq,
			Title:       p.Title,
			Content:     p.Content,
			Original:    p.Content,
			Prompt:      p.Prompt,
			Dialogue:    p.Dialogue,
			SceneAnchor: p.SceneAnchor,
			Plot:        p.Plot,
		})
	}
	if err := s.repo.ReplaceStoryboards(ownerID, projectID, storyboards); err != nil {
		return nil, err
	}
	if err := s.extractAndSaveAssets(ownerID, projectID, parsed.Preface); err != nil {
		return nil, err
	}
	return s.GetProject(ownerID, projectID)
}

func (s *DramaService) UpdateStoryboard(ownerID, projectID, storyboardID uint64, content, prompt string) (*model.DramaStoryboard, error) {
	storyboard, err := s.repo.GetStoryboard(ownerID, projectID, storyboardID)
	if err != nil {
		return nil, err
	}
	storyboard.Content = content
	if prompt != "" {
		storyboard.Prompt = prompt
	} else {
		storyboard.Prompt = synthesizePrompt(content)
	}
	storyboard.Dialogue = extractDialogue(content)
	storyboard.SceneAnchor = extractSection(content, "场景锚定")
	storyboard.Plot = extractSection(content, "整镜剧情")
	storyboard.Modified = storyboard.Content != storyboard.Original
	if err := s.repo.UpdateStoryboard(storyboard); err != nil {
		return nil, err
	}
	return storyboard, nil
}

func (s *DramaService) AppendToStoryboards(ownerID, projectID uint64, suffix string) ([]model.DramaStoryboard, error) {
	storyboards, err := s.repo.ListStoryboards(ownerID, projectID)
	if err != nil {
		return nil, err
	}
	for i := range storyboards {
		storyboards[i].Content = strings.TrimRight(storyboards[i].Content, "\r\n ") + "\n" + suffix
		storyboards[i].Prompt = strings.TrimRight(storyboards[i].Prompt, "，, ") + "，" + suffix
		storyboards[i].Modified = storyboards[i].Content != storyboards[i].Original
		if err := s.repo.UpdateStoryboard(&storyboards[i]); err != nil {
			return nil, err
		}
	}
	return storyboards, nil
}

func (s *DramaService) UpdateAsset(ownerID, projectID, assetID uint64, name, description, referencePrompt, voiceName string) (*model.DramaAsset, error) {
	asset, err := s.repo.GetAsset(ownerID, projectID, assetID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) != "" {
		asset.Name = strings.TrimSpace(name)
	}
	asset.Description = description
	asset.ReferencePrompt = strings.TrimSpace(referencePrompt)
	asset.VoiceName = strings.TrimSpace(voiceName)
	if err := s.repo.UpdateAsset(asset); err != nil {
		return nil, err
	}
	return asset, nil
}

func (s *DramaService) ImportAssetsFromAI(ownerID, projectID uint64, text string) ([]model.DramaAsset, error) {
	if _, err := s.repo.GetProject(ownerID, projectID); err != nil {
		return nil, err
	}
	assets := parseAIAssets(ownerID, projectID, text)
	if len(assets) == 0 {
		characters, scenes := ExtractAssets(text)
		for _, character := range characters {
			assets = append(assets, model.DramaAsset{OwnerID: ownerID, ProjectID: projectID, Type: "character", Name: character.Name, Description: character.Description})
		}
		for _, scene := range scenes {
			assets = append(assets, model.DramaAsset{OwnerID: ownerID, ProjectID: projectID, Type: "scene", Name: scene.Name, Description: scene.Description})
		}
	}
	existing, err := s.repo.ListAssets(ownerID, projectID)
	if err != nil {
		return nil, err
	}
	preserveAssetReferences(assets, existing)
	if err := s.repo.UpsertAssets(assets); err != nil {
		return nil, err
	}
	return s.repo.ListAssets(ownerID, projectID)
}

func preserveAssetReferences(next []model.DramaAsset, existing []model.DramaAsset) {
	byKey := make(map[string]model.DramaAsset, len(existing))
	for _, asset := range existing {
		byKey[assetIdentityKey(asset.Type, asset.Name)] = asset
	}
	for index := range next {
		old, ok := byKey[assetIdentityKey(next[index].Type, next[index].Name)]
		if !ok {
			continue
		}
		if next[index].ReferenceFileID == 0 {
			next[index].ReferenceFileID = old.ReferenceFileID
		}
		if strings.TrimSpace(next[index].VoiceName) == "" {
			next[index].VoiceName = old.VoiceName
		}
	}
}

func assetIdentityKey(assetType, name string) string {
	return strings.ToLower(strings.TrimSpace(assetType)) + "\x00" + strings.TrimSpace(name)
}

func (s *DramaService) CreateTask(ownerID, projectID uint64, input CreateTaskInput) (*model.DramaTask, error) {
	if _, err := s.repo.GetProject(ownerID, projectID); err != nil {
		return nil, err
	}
	taskType := strings.TrimSpace(input.Type)
	if taskType == "" {
		taskType = "custom"
	}
	var payloadData map[string]interface{}
	if err := json.Unmarshal([]byte(input.Payload), &payloadData); err != nil || payloadData == nil {
		payloadData = make(map[string]interface{})
	}
	if len(input.StoryboardIDs) > 0 {
		payloadData["storyboard_ids"] = input.StoryboardIDs
	}
	payloadBytes, _ := json.Marshal(payloadData)
	task := &model.DramaTask{
		ProjectID: projectID,
		OwnerID:   ownerID,
		Type:      taskType,
		Status:    "pending",
		Progress:  0,
		Message:   "任务已创建，等待执行",
		Payload:   string(payloadBytes),
	}
	if err := s.repo.CreateTask(task); err != nil {
		return nil, err
	}
	if s.taskRunner != nil {
		if err := s.taskRunner.Enqueue(task.ID); err != nil {
			task.Status = "failed"
			task.Message = "任务入队失败：" + err.Error()
			now := time.Now()
			task.FinishedAt = &now
			_ = s.repo.UpdateTask(task)
			return task, nil
		}
		s.taskRunner.Publish(*task)
	}
	return task, nil
}

func (s *DramaService) UploadAssetReference(ownerID, projectID, assetID uint64, header *multipart.FileHeader) (*model.DramaAsset, error) {
	project, err := s.repo.GetProject(ownerID, projectID)
	if err != nil {
		return nil, err
	}
	asset, err := s.repo.GetAsset(ownerID, projectID, assetID)
	if err != nil {
		return nil, err
	}
	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	folders, err := s.ensureProjectFolders(ownerID, project.Title)
	if err != nil {
		return nil, err
	}
	parentID := folders["assets"]
	storageKey := fmt.Sprintf("drama/%d/%d/assets/%d-%s", ownerID, projectID, time.Now().UnixNano(), path.Base(header.Filename))
	if _, err := s.minioClient.PutObject(context.Background(), s.bucket, storageKey, file, header.Size, minio.PutObjectOptions{ContentType: header.Header.Get("Content-Type")}); err != nil {
		return nil, err
	}
	cloudFile := &model.File{
		UserID:     ownerID,
		Name:       header.Filename,
		ParentID:   parentID,
		Size:       header.Size,
		MimeType:   header.Header.Get("Content-Type"),
		StorageKey: storageKey,
	}
	if err := s.repo.CreateFile(cloudFile); err != nil {
		return nil, err
	}
	asset.ReferenceFileID = cloudFile.ID
	if err := s.repo.UpdateAsset(asset); err != nil {
		return nil, err
	}
	return asset, nil
}

func (s *DramaService) UploadStoryboardAudio(ownerID, projectID, storyboardID uint64, header *multipart.FileHeader, durationMS int) (*model.DramaStoryboard, error) {
	project, err := s.repo.GetProject(ownerID, projectID)
	if err != nil {
		return nil, err
	}
	storyboard, err := s.repo.GetStoryboard(ownerID, projectID, storyboardID)
	if err != nil {
		return nil, err
	}
	fileID, err := s.saveAudioToDrive(ownerID, projectID, project.Title, header)
	if err != nil {
		return nil, err
	}
	if durationMS <= 0 {
		durationMS = estimateDurationMS(storyboard.Dialogue)
	}
	subtitleSource := storyboard.Dialogue
	if strings.TrimSpace(subtitleSource) == "" {
		subtitleSource = storyboard.Content
	}
	storyboard.AudioFileID = fileID
	storyboard.AudioDurationMS = durationMS
	storyboard.SubtitleASS = BuildSubtitleASS(subtitleSource, durationMS)
	if err := s.repo.UpdateStoryboard(storyboard); err != nil {
		return nil, err
	}
	return storyboard, nil
}

func (s *DramaService) BatchImportAudio(ownerID, projectID uint64, headers []*multipart.FileHeader) ([]AudioImportResult, error) {
	project, err := s.repo.GetProject(ownerID, projectID)
	if err != nil {
		return nil, err
	}
	storyboards, err := s.repo.ListStoryboards(ownerID, projectID)
	if err != nil {
		return nil, err
	}
	bySeq := make(map[int]*model.DramaStoryboard)
	for i := range storyboards {
		bySeq[storyboards[i].Seq] = &storyboards[i]
	}
	sort.Slice(headers, func(i, j int) bool {
		return headers[i].Filename < headers[j].Filename
	})
	results := make([]AudioImportResult, 0, len(headers))
	nextIndex := 0
	for _, header := range headers {
		storyboard := matchStoryboardByAudioName(header.Filename, bySeq)
		if storyboard == nil {
			for nextIndex < len(storyboards) && storyboards[nextIndex].AudioFileID != 0 {
				nextIndex++
			}
			if nextIndex < len(storyboards) {
				storyboard = &storyboards[nextIndex]
				nextIndex++
			}
		}
		if storyboard == nil {
			results = append(results, AudioImportResult{FileName: header.Filename, Matched: false, Reason: "no storyboard matched"})
			continue
		}
		fileID, err := s.saveAudioToDrive(ownerID, projectID, project.Title, header)
		if err != nil {
			results = append(results, AudioImportResult{FileName: header.Filename, Matched: false, Reason: err.Error()})
			continue
		}
		subtitleSource := storyboard.Dialogue
		if strings.TrimSpace(subtitleSource) == "" {
			subtitleSource = storyboard.Content
		}
		durationMS := estimateDurationMS(subtitleSource)
		storyboard.AudioFileID = fileID
		storyboard.AudioDurationMS = durationMS
		storyboard.SubtitleASS = BuildSubtitleASS(subtitleSource, durationMS)
		if err := s.repo.UpdateStoryboard(storyboard); err != nil {
			results = append(results, AudioImportResult{FileName: header.Filename, Matched: false, Reason: err.Error()})
			continue
		}
		copy := *storyboard
		results = append(results, AudioImportResult{FileName: header.Filename, Storyboard: &copy, Matched: true})
	}
	return results, nil
}

func (s *DramaService) ExportProject(ownerID, projectID uint64, saveToDrive bool) ([]byte, string, error) {
	detail, err := s.GetProject(ownerID, projectID)
	if err != nil {
		return nil, "", err
	}
	payload := ExportPayload{
		Version:     1,
		ExportedAt:  time.Now(),
		Project:     *detail.Project,
		Storyboards: detail.Storyboards,
		Segments:    detail.Segments,
		Media:       detail.Media,
		Assets:      detail.Assets,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, "", err
	}
	filename := sanitizeName(detail.Project.Title) + "-短剧项目.json"
	if saveToDrive {
		if err := s.saveExportToDrive(ownerID, detail.Project.ID, detail.Project.Title, filename, data); err != nil {
			return nil, "", err
		}
	}
	return data, filename, nil
}

func (s *DramaService) ImportProject(ownerID uint64, data []byte) (*model.DramaProject, error) {
	var payload ExportPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	project := &model.DramaProject{
		OwnerID:     ownerID,
		Title:       payload.Project.Title + "（导入）",
		Description: payload.Project.Description,
		Preface:     payload.Project.Preface,
		RawScript:   payload.Project.RawScript,
		Settings:    payload.Project.Settings,
	}
	if project.Settings == "" {
		project.Settings = "{}"
	}
	if err := s.repo.CreateProject(project); err != nil {
		return nil, err
	}
	storyboards := make([]model.DramaStoryboard, 0, len(payload.Storyboards))
	for _, old := range payload.Storyboards {
		old.ID = 0
		old.ProjectID = project.ID
		old.OwnerID = ownerID
		old.ImageFileID = 0
		old.AudioFileID = 0
		old.VideoFileID = 0
		storyboards = append(storyboards, old)
	}
	if err := s.repo.ReplaceStoryboards(ownerID, project.ID, storyboards); err != nil {
		return nil, err
	}
	newStoryboards, err := s.repo.ListStoryboards(ownerID, project.ID)
	if err != nil {
		return nil, err
	}
	oldStoryboardSeq := make(map[uint64]int, len(payload.Storyboards))
	for _, old := range payload.Storyboards {
		oldStoryboardSeq[old.ID] = old.Seq
	}
	newStoryboardBySeq := make(map[int]model.DramaStoryboard, len(newStoryboards))
	for _, storyboard := range newStoryboards {
		newStoryboardBySeq[storyboard.Seq] = storyboard
	}
	segmentsByStoryboard := make(map[uint64][]model.DramaStoryboardSegment)
	for _, old := range payload.Segments {
		seq, ok := oldStoryboardSeq[old.StoryboardID]
		if !ok {
			continue
		}
		storyboard, ok := newStoryboardBySeq[seq]
		if !ok {
			continue
		}
		old.ID = 0
		old.ProjectID = project.ID
		old.OwnerID = ownerID
		old.StoryboardID = storyboard.ID
		old.ReferenceFileID = 0
		old.VideoFileID = 0
		segmentsByStoryboard[storyboard.ID] = append(segmentsByStoryboard[storyboard.ID], old)
	}
	for storyboardID, segments := range segmentsByStoryboard {
		if err := s.repo.ReplaceStoryboardSegments(ownerID, project.ID, storyboardID, segments); err != nil {
			return nil, err
		}
	}
	assets := make([]model.DramaAsset, 0, len(payload.Assets))
	for _, old := range payload.Assets {
		old.ID = 0
		old.ProjectID = project.ID
		old.OwnerID = ownerID
		old.ReferenceFileID = 0
		assets = append(assets, old)
	}
	if err := s.repo.UpsertAssets(assets); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *DramaService) GetSetting(ownerID uint64) (*model.DramaSetting, error) {
	return s.repo.GetSetting(ownerID)
}

func (s *DramaService) SaveSetting(ownerID uint64, input model.DramaSetting) (*model.DramaSetting, error) {
	setting, err := s.repo.GetSetting(ownerID)
	if err != nil {
		return nil, err
	}
	setting.ComfyUIURL = input.ComfyUIURL
	if input.ImageSettings != "" {
		setting.ImageSettings = input.ImageSettings
	}
	if input.TTSEngine != "" {
		setting.TTSEngine = input.TTSEngine
	}
	if input.TTSConfig != "" {
		setting.TTSConfig = input.TTSConfig
	}
	if input.VideoSettings != "" {
		setting.VideoSettings = input.VideoSettings
	}
	if input.StorageRoot != "" {
		setting.StorageRoot = input.StorageRoot
	}
	if err := s.repo.SaveSetting(setting); err != nil {
		return nil, err
	}
	return setting, nil
}

func (s *DramaService) extractAndSaveAssets(ownerID, projectID uint64, preface string) error {
	characters, scenes := ExtractAssets(preface)
	assets := make([]model.DramaAsset, 0, len(characters)+len(scenes))
	for _, character := range characters {
		assets = append(assets, model.DramaAsset{OwnerID: ownerID, ProjectID: projectID, Type: "character", Name: character.Name, Description: character.Description})
	}
	for _, scene := range scenes {
		assets = append(assets, model.DramaAsset{OwnerID: ownerID, ProjectID: projectID, Type: "scene", Name: scene.Name, Description: scene.Description})
	}
	return s.repo.UpsertAssets(assets)
}

func (s *DramaService) ensureProjectFolders(ownerID uint64, title string) (map[string]uint64, error) {
	setting, err := s.repo.GetSetting(ownerID)
	if err != nil {
		return nil, err
	}
	root, err := s.ensureFolder(ownerID, 0, setting.StorageRoot)
	if err != nil {
		return nil, err
	}
	project, err := s.ensureFolder(ownerID, root.ID, sanitizeName(title))
	if err != nil {
		return nil, err
	}
	result := map[string]uint64{"root": root.ID, "project": project.ID}
	for _, name := range []string{"images", "audio", "videos", "exports", "assets"} {
		folder, err := s.ensureFolder(ownerID, project.ID, name)
		if err != nil {
			return nil, err
		}
		result[name] = folder.ID
	}
	return result, nil
}

func (s *DramaService) moveProjectFilesToTrash(ownerID, projectID uint64, title string) error {
	idSet := make(map[uint64]bool)
	setting, err := s.repo.GetSetting(ownerID)
	if err != nil {
		return err
	}
	root, err := s.repo.FindFileByNameAndParent(ownerID, 0, setting.StorageRoot)
	if err != nil || root == nil {
		if err != nil {
			return err
		}
	} else if projectFolder, err := s.repo.FindFileByNameAndParent(ownerID, root.ID, sanitizeName(title)); err != nil {
		return err
	} else if projectFolder != nil {
		ids, err := s.collectFileTree(ownerID, projectFolder.ID)
		if err != nil {
			return err
		}
		ids = append(ids, projectFolder.ID)
		for _, id := range ids {
			idSet[id] = true
		}
	}
	prefixFiles, err := s.repo.ListFilesByStoragePrefix(ownerID, fmt.Sprintf("drama/%d/%d/", ownerID, projectID))
	if err != nil {
		return err
	}
	for _, file := range prefixFiles {
		idSet[file.ID] = true
	}
	ids := make([]uint64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	return s.repo.SoftDeleteFiles(ownerID, ids)
}

func (s *DramaService) collectFileTree(ownerID, parentID uint64) ([]uint64, error) {
	children, err := s.repo.ListFilesByParent(ownerID, parentID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(children))
	for _, child := range children {
		if child.IsDir {
			nested, err := s.collectFileTree(ownerID, child.ID)
			if err != nil {
				return nil, err
			}
			ids = append(ids, nested...)
		}
		ids = append(ids, child.ID)
	}
	return ids, nil
}

func (s *DramaService) ensureFolder(ownerID, parentID uint64, name string) (*model.File, error) {
	existing, err := s.repo.FindFileByNameAndParent(ownerID, parentID, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	folder := &model.File{UserID: ownerID, Name: name, IsDir: true, ParentID: parentID}
	if err := s.repo.CreateFile(folder); err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *DramaService) saveExportToDrive(ownerID, projectID uint64, title, filename string, data []byte) error {
	folders, err := s.ensureProjectFolders(ownerID, title)
	if err != nil {
		return err
	}
	storageKey := fmt.Sprintf("drama/%d/%d/exports/%d-%s", ownerID, projectID, time.Now().UnixNano(), filename)
	if _, err := s.minioClient.PutObject(context.Background(), s.bucket, storageKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: "application/json"}); err != nil {
		return err
	}
	return s.repo.CreateFile(&model.File{
		UserID:     ownerID,
		Name:       filename,
		ParentID:   folders["exports"],
		Size:       int64(len(data)),
		MimeType:   "application/json",
		StorageKey: storageKey,
	})
}

func (s *DramaService) saveAudioToDrive(ownerID, projectID uint64, title string, header *multipart.FileHeader) (uint64, error) {
	folders, err := s.ensureProjectFolders(ownerID, title)
	if err != nil {
		return 0, err
	}
	file, err := header.Open()
	if err != nil {
		return 0, err
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/mpeg"
	}
	storageKey := fmt.Sprintf("drama/%d/%d/audio/%d-%s", ownerID, projectID, time.Now().UnixNano(), path.Base(header.Filename))
	if _, err := s.minioClient.PutObject(context.Background(), s.bucket, storageKey, file, header.Size, minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return 0, err
	}
	cloudFile := &model.File{
		UserID:     ownerID,
		Name:       header.Filename,
		ParentID:   folders["audio"],
		Size:       header.Size,
		MimeType:   contentType,
		StorageKey: storageKey,
	}
	if err := s.repo.CreateFile(cloudFile); err != nil {
		return 0, err
	}
	return cloudFile.ID, nil
}

var audioSeqPattern = regexp.MustCompile(`\d+`)

func matchStoryboardByAudioName(filename string, bySeq map[int]*model.DramaStoryboard) *model.DramaStoryboard {
	matches := audioSeqPattern.FindAllString(filename, -1)
	for _, match := range matches {
		seq, err := strconv.Atoi(match)
		if err != nil {
			continue
		}
		if storyboard := bySeq[seq]; storyboard != nil {
			return storyboard
		}
	}
	return nil
}

func estimateDurationMS(dialogue string) int {
	text := stripDialogueNames(dialogue)
	runes := len([]rune(strings.TrimSpace(text)))
	if runes == 0 {
		return 3000
	}
	seconds := float64(runes) / 4.5
	if seconds < 2 {
		seconds = 2
	}
	return int(seconds * 1000)
}

func BuildSubtitleASS(dialogue string, durationMS int) string {
	lines := dialogueLines(dialogue)
	if len(lines) == 0 {
		lines = []string{strings.TrimSpace(dialogue)}
	}
	if len(lines) == 0 || lines[0] == "" {
		return ""
	}
	if durationMS <= 0 {
		durationMS = estimateDurationMS(dialogue)
	}
	header := `[Script Info]
ScriptType: v4.00+
PlayResX: 1920
PlayResY: 1080

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Microsoft YaHei,42,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,0,0,0,0,100,100,0,0,1,2,0,2,80,80,72,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`
	var builder strings.Builder
	builder.WriteString(header)
	totalChars := 0
	for _, line := range lines {
		totalChars += maxInt(1, len([]rune(stripDialogueNames(line))))
	}
	elapsed := 0
	for i, line := range lines {
		weight := maxInt(1, len([]rune(stripDialogueNames(line))))
		part := durationMS * weight / totalChars
		if i == len(lines)-1 {
			part = durationMS - elapsed
		}
		start := elapsed
		end := elapsed + part
		builder.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", assTime(start), assTime(end), escapeASS(stripDialogueNames(line))))
		elapsed = end
	}
	return builder.String()
}

func dialogueLines(dialogue string) []string {
	rawLines := strings.Split(dialogue, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func stripDialogueNames(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if idx := strings.IndexAny(line, ":："); idx >= 0 && idx < 12 {
			lines[i] = strings.TrimSpace(line[idx+1:])
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func assTime(ms int) string {
	if ms < 0 {
		ms = 0
	}
	cs := ms / 10
	h := cs / 360000
	cs %= 360000
	m := cs / 6000
	cs %= 6000
	s := cs / 100
	cs %= 100
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

func escapeASS(text string) string {
	text = strings.ReplaceAll(text, "{", "\\{")
	text = strings.ReplaceAll(text, "}", "\\}")
	text = strings.ReplaceAll(text, "\n", "\\N")
	return text
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "未命名短剧"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(name)
}

func parseAIAssets(ownerID, projectID uint64, text string) []model.DramaAsset {
	text = cleanJSONText(text)
	var payload AIAssetImport
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil
	}
	assets := make([]model.DramaAsset, 0, len(payload.Characters)+len(payload.Scenes))
	for _, item := range payload.Characters {
		name := stringField(item, "name")
		if name == "" {
			continue
		}
		assets = append(assets, model.DramaAsset{
			OwnerID:         ownerID,
			ProjectID:       projectID,
			Type:            "character",
			Name:            name,
			Description:     assetDescription(item, []string{"age", "appearance", "clothing", "personality", "voice_suggestion", "notes"}),
			ReferencePrompt: stringField(item, "reference_prompt"),
			VoiceName:       stringField(item, "voice_name"),
		})
	}
	for _, item := range payload.Scenes {
		name := stringField(item, "name")
		if name == "" {
			continue
		}
		assets = append(assets, model.DramaAsset{
			OwnerID:         ownerID,
			ProjectID:       projectID,
			Type:            "scene",
			Name:            name,
			Description:     assetDescription(item, []string{"environment", "lighting", "style", "notes"}),
			ReferencePrompt: stringField(item, "reference_prompt"),
		})
	}
	return assets
}

func parseAISegments(text string) (AISegmentImport, error) {
	var payload AISegmentImport
	err := json.Unmarshal([]byte(cleanJSONText(text)), &payload)
	if err != nil {
		return payload, err
	}
	if len(payload.Segments) == 0 {
		return payload, fmt.Errorf("segments is empty")
	}
	return payload, nil
}

func cleanJSONText(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```JSON")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}
	return text
}

func stringField(item map[string]interface{}, key string) string {
	if v, ok := item[key]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}

func assetDescription(item map[string]interface{}, keys []string) string {
	labels := map[string]string{
		"age":              "年龄",
		"appearance":       "外貌",
		"clothing":         "服装",
		"personality":      "性格",
		"voice_suggestion": "音色建议",
		"reference_prompt": "参考图提示词",
		"environment":      "环境",
		"lighting":         "光线",
		"style":            "风格",
		"notes":            "备注",
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := stringField(item, key)
		if value == "" {
			continue
		}
		label := labels[key]
		if label == "" {
			label = key
		}
		parts = append(parts, label+"："+value)
	}
	return strings.Join(parts, "\n")
}
