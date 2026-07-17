package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cloudnexus/server/pkg/model"
	"github.com/minio/minio-go/v7"
)

type generationTaskPayload struct {
	AssetID       string                 `json:"asset_id"`
	AssetType     string                 `json:"asset_type"`
	Name          string                 `json:"name"`
	Prompt        string                 `json:"prompt"`
	ImageCount    int                    `json:"image_count"`
	StoryboardIDs []string               `json:"storyboard_ids"`
	Results       []generationTaskResult `json:"results,omitempty"`
	PromptLog     []generationPromptLog  `json:"prompt_log,omitempty"`
}

type generationTaskResult struct {
	Kind         string `json:"kind"`
	FileID       string `json:"file_id"`
	StoryboardID string `json:"storyboard_id,omitempty"`
	AssetID      string `json:"asset_id,omitempty"`
	Title        string `json:"title,omitempty"`
	Prompt       string `json:"prompt,omitempty"`
}

type generationPromptLog struct {
	Target string `json:"target"`
	Prompt string `json:"prompt"`
}

func (s *DramaService) GetComfyStatus(ctx context.Context, ownerID uint64, overrideURL string) (ComfyStatus, error) {
	setting, err := s.repo.GetSetting(ownerID)
	if err != nil {
		return ComfyStatus{}, err
	}
	comfyURL := strings.TrimSpace(overrideURL)
	if comfyURL == "" {
		comfyURL = setting.ComfyUIURL
	}
	return NewComfyClient(comfyURL).Status(ctx), nil
}

func (s *DramaService) CancelTask(ownerID, projectID, taskID uint64) (*model.DramaTask, error) {
	if s.taskRunner == nil {
		return nil, fmt.Errorf("task runner is not available")
	}
	return s.taskRunner.Cancel(ownerID, projectID, taskID)
}

func (s *DramaService) RetryTask(ownerID, projectID, taskID uint64) (*model.DramaTask, error) {
	if s.taskRunner == nil {
		return nil, fmt.Errorf("task runner is not available")
	}
	return s.taskRunner.Retry(ownerID, projectID, taskID)
}

func (s *DramaService) SubscribeTaskEvents(ownerID uint64) (<-chan TaskEvent, func(), error) {
	if s.taskRunner == nil {
		return nil, nil, fmt.Errorf("task runner is not available")
	}
	events, unsubscribe := s.taskRunner.Subscribe(ownerID)
	return events, unsubscribe, nil
}

func (s *DramaService) executeGenerationTask(ctx context.Context, task *model.DramaTask, update func(int, string)) error {
	switch task.Type {
	case "image", "asset_reference":
		return s.executeImageTask(ctx, task, update)
	case "tts":
		return fmt.Errorf("TTS engine will be connected in the next phase")
	case "video":
		return fmt.Errorf("video rendering will be connected after image and audio generation")
	default:
		return fmt.Errorf("unsupported task type: %s", task.Type)
	}
}

func (s *DramaService) executeImageTask(ctx context.Context, task *model.DramaTask, update func(int, string)) error {
	project, err := s.repo.GetProject(task.OwnerID, task.ProjectID)
	if err != nil {
		return err
	}
	setting, err := s.repo.GetSetting(task.OwnerID)
	if err != nil {
		return err
	}
	client := NewComfyClient(setting.ComfyUIURL)
	status := client.Status(ctx)
	if !status.Connected {
		return fmt.Errorf("ComfyUI is not reachable: %s", status.Error)
	}
	imageSettings := defaultImageGenerationSettings(setting.ImageSettings)
	if imageSettings.Checkpoint == "" && len(status.Checkpoints) > 0 {
		imageSettings.Checkpoint = status.Checkpoints[0]
	}
	if imageSettings.Checkpoint == "" {
		return fmt.Errorf("ComfyUI is connected, but no checkpoint model was detected")
	}

	var payload generationTaskPayload
	_ = json.Unmarshal([]byte(task.Payload), &payload)
	if task.Type == "asset_reference" {
		return s.generateAssetReference(ctx, task, client, imageSettings, project, payload, update)
	}
	return s.generateStoryboardImages(ctx, task, client, imageSettings, project, payload.StoryboardIDs, payload.ImageCount, update)
}

func (s *DramaService) generateAssetReference(ctx context.Context, task *model.DramaTask, client *ComfyClient, settings ImageGenerationSettings, project *model.DramaProject, payload generationTaskPayload, update func(int, string)) error {
	assetID, err := strconv.ParseUint(payload.AssetID, 10, 64)
	if err != nil || assetID == 0 {
		return fmt.Errorf("asset task is missing a valid asset id")
	}
	asset, err := s.repo.GetAsset(project.OwnerID, project.ID, assetID)
	if err != nil {
		return err
	}
	prompt := strings.TrimSpace(payload.Prompt)
	if prompt == "" {
		prompt = asset.Description
	}
	prompt = strings.TrimSpace(prompt + ", reference sheet, realistic cinematic lighting, clear subject, complete composition")
	s.appendTaskPromptLog(task, asset.Name, prompt)
	update(15, fmt.Sprintf("Generating reference image: %s", asset.Name))
	data, filename, err := client.Generate(ctx, prompt, settings, update)
	if err != nil {
		return err
	}
	fileID, err := s.saveGeneratedImage(project.OwnerID, project.ID, project.Title, "assets", asset.Name, filename, data)
	if err != nil {
		return err
	}
	asset.ReferenceFileID = fileID
	if err := s.repo.UpdateAsset(asset); err != nil {
		return err
	}
	s.appendTaskResult(task, generationTaskResult{
		Kind: "asset_reference", FileID: strconv.FormatUint(fileID, 10), AssetID: strconv.FormatUint(asset.ID, 10), Title: asset.Name, Prompt: prompt,
	})
	return nil
}

func (s *DramaService) generateStoryboardImages(ctx context.Context, task *model.DramaTask, client *ComfyClient, settings ImageGenerationSettings, project *model.DramaProject, selected []string, imageCount int, update func(int, string)) error {
	storyboards, err := s.repo.ListStoryboards(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	assets, err := s.repo.ListAssets(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	selectedSet := make(map[uint64]bool)
	for _, rawID := range selected {
		if id, err := strconv.ParseUint(rawID, 10, 64); err == nil {
			selectedSet[id] = true
		}
	}
	targets := make([]*model.DramaStoryboard, 0, len(storyboards))
	for i := range storyboards {
		if len(selectedSet) == 0 || selectedSet[storyboards[i].ID] {
			targets = append(targets, &storyboards[i])
		}
	}
	if len(targets) == 0 {
		return fmt.Errorf("no storyboard found for image generation")
	}
	if imageCount <= 0 {
		imageCount = 1
	}
	if imageCount > 8 {
		imageCount = 8
	}
	for index, storyboard := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		prompt := s.buildStoryboardImagePrompt(project, storyboard, assets)
		s.appendTaskPromptLog(task, fmt.Sprintf("Storyboard %d: %s", storyboard.Seq, storyboard.Title), prompt)
		for candidate := 1; candidate <= imageCount; candidate++ {
			total := len(targets) * imageCount
			done := index*imageCount + candidate - 1
			baseProgress := done * 90 / total
			span := 90 / total
			update(5+baseProgress, fmt.Sprintf("Generating storyboard %d/%d image %d/%d: %s", index+1, len(targets), candidate, imageCount, storyboard.Title))
			data, filename, err := client.Generate(ctx, prompt, settings, func(localProgress int, message string) {
				mapped := 5 + baseProgress + localProgress*span/100
				update(mapped, fmt.Sprintf("Storyboard %d/%d image %d/%d: %s", index+1, len(targets), candidate, imageCount, message))
			})
			if err != nil {
				return fmt.Errorf("storyboard %d image %d failed: %w", storyboard.Seq, candidate, err)
			}
			label := fmt.Sprintf("storyboard-%03d-%02d", storyboard.Seq, candidate)
			fileID, err := s.saveGeneratedImage(project.OwnerID, project.ID, project.Title, "images", label, filename, data)
			if err != nil {
				return err
			}
			sortOrder, err := s.repo.NextStoryboardMediaSort(project.OwnerID, project.ID, storyboard.ID)
			if err != nil {
				return err
			}
			media := &model.DramaStoryboardMedia{
				ProjectID:    project.ID,
				StoryboardID: storyboard.ID,
				OwnerID:      project.OwnerID,
				Kind:         "image",
				FileID:       fileID,
				Source:       "generated",
				Prompt:       prompt,
				SortOrder:    sortOrder,
				Selected:     candidate == 1,
			}
			if err := s.repo.CreateStoryboardMedia(media); err != nil {
				return err
			}
			if candidate == 1 {
				if err := s.repo.SelectStoryboardMedia(project.OwnerID, project.ID, storyboard.ID, media.ID, "image"); err != nil {
					return err
				}
				storyboard.ImageFileID = fileID
				if err := s.repo.UpdateStoryboard(storyboard); err != nil {
					return err
				}
			}
			s.appendTaskResult(task, generationTaskResult{
				Kind: "storyboard_image", FileID: strconv.FormatUint(fileID, 10), StoryboardID: strconv.FormatUint(storyboard.ID, 10), Title: fmt.Sprintf("%s #%d", storyboard.Title, candidate), Prompt: prompt,
			})
		}
	}
	return nil
}

func (s *DramaService) buildStoryboardImagePrompt(project *model.DramaProject, storyboard *model.DramaStoryboard, assets []model.DramaAsset) string {
	basePrompt := strings.TrimSpace(storyboard.Prompt)
	if basePrompt == "" {
		basePrompt = synthesizePrompt(storyboard.Content)
	}
	scene := strings.TrimSpace(storyboard.SceneAnchor)
	plot := strings.TrimSpace(storyboard.Plot)
	if plot == "" {
		plot = strings.TrimSpace(storyboard.Content)
	}
	characters := make([]string, 0)
	scenes := make([]string, 0)
	for _, asset := range assets {
		name := strings.TrimSpace(asset.Name)
		description := compactText(asset.Description, 220)
		if name == "" && description == "" {
			continue
		}
		line := strings.TrimSpace(name + ": " + description)
		if asset.Type == "character" {
			characters = append(characters, line)
		} else if asset.Type == "scene" {
			scenes = append(scenes, line)
		}
	}
	parts := []string{
		"cinematic storyboard frame, high quality still image, coherent scene, realistic lighting, detailed composition",
		"Project context: " + compactText(project.Preface, 500),
		"Current shot title: " + strings.TrimSpace(storyboard.Title),
		"Scene anchor: " + scene,
		"Shot action: " + compactText(plot, 500),
		"Visual prompt: " + compactText(basePrompt, 500),
	}
	if len(characters) > 0 {
		parts = append(parts, "Character continuity: "+compactText(strings.Join(characters, "; "), 700))
	}
	if len(scenes) > 0 {
		parts = append(parts, "Scene continuity: "+compactText(strings.Join(scenes, "; "), 500))
	}
	parts = append(parts, "single frame, no text, no subtitles, no watermark, no UI, no comic panels")
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && !strings.HasSuffix(part, ":") {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "\n")
}

func compactText(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if limit > 0 && len([]rune(value)) > limit {
		runes := []rune(value)
		return string(runes[:limit])
	}
	return value
}

func (s *DramaService) appendTaskPromptLog(task *model.DramaTask, target, prompt string) {
	var payload generationTaskPayload
	_ = json.Unmarshal([]byte(task.Payload), &payload)
	payload.PromptLog = append(payload.PromptLog, generationPromptLog{Target: target, Prompt: prompt})
	if data, err := json.Marshal(payload); err == nil {
		task.Payload = string(data)
		_ = s.repo.UpdateTask(task)
	}
}

func (s *DramaService) appendTaskResult(task *model.DramaTask, result generationTaskResult) {
	var payload generationTaskPayload
	_ = json.Unmarshal([]byte(task.Payload), &payload)
	payload.Results = append(payload.Results, result)
	if data, err := json.Marshal(payload); err == nil {
		task.Payload = string(data)
		_ = s.repo.UpdateTask(task)
	}
}

func (s *DramaService) saveGeneratedImage(ownerID, projectID uint64, title, folderKey, label, originalFilename string, data []byte) (uint64, error) {
	folders, err := s.ensureProjectFolders(ownerID, title)
	if err != nil {
		return 0, err
	}
	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext == "" || len(ext) > 8 {
		ext = ".png"
	}
	filename := sanitizeName(label) + "-" + time.Now().Format("20060102-150405") + ext
	storageKey := fmt.Sprintf("drama/%d/%d/%s/%d-%s", ownerID, projectID, folderKey, time.Now().UnixNano(), filename)
	contentType := http.DetectContentType(data)
	if _, err := s.minioClient.PutObject(context.Background(), s.bucket, storageKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return 0, err
	}
	cloudFile := &model.File{
		UserID:     ownerID,
		Name:       filename,
		ParentID:   folders[folderKey],
		Size:       int64(len(data)),
		MimeType:   contentType,
		StorageKey: storageKey,
	}
	if err := s.repo.CreateFile(cloudFile); err != nil {
		return 0, err
	}
	return cloudFile.ID, nil
}

func assetTypeLabel(assetType string) string {
	if assetType == "character" {
		return "character reference"
	}
	return "scene reference"
}
