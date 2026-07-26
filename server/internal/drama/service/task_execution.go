package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudnexus/server/pkg/model"
	"github.com/minio/minio-go/v7"
)

type generationTaskPayload struct {
	AssetID          string                 `json:"asset_id"`
	AssetType        string                 `json:"asset_type"`
	Name             string                 `json:"name"`
	Prompt           string                 `json:"prompt"`
	ImageCount       int                    `json:"image_count"`
	StoryboardIDs    []string               `json:"storyboard_ids"`
	SegmentIDs       []string               `json:"segment_ids"`
	ForceGenerate    bool                   `json:"force_generate"`
	ReferenceFileIDs []string               `json:"reference_file_ids,omitempty"`
	NegativePrompt   string                 `json:"negative_prompt,omitempty"`
	Width            int                    `json:"width,omitempty"`
	Height           int                    `json:"height,omitempty"`
	Steps            int                    `json:"steps,omitempty"`
	CFG              float64                `json:"cfg,omitempty"`
	ReferenceWeight  float64                `json:"reference_weight,omitempty"`
	Results          []generationTaskResult `json:"results,omitempty"`
	PromptLog        []generationPromptLog  `json:"prompt_log,omitempty"`
}

type generationTaskResult struct {
	Kind         string `json:"kind"`
	FileID       string `json:"file_id"`
	StoryboardID string `json:"storyboard_id,omitempty"`
	SegmentID    string `json:"segment_id,omitempty"`
	AssetID      string `json:"asset_id,omitempty"`
	Title        string `json:"title,omitempty"`
	Prompt       string `json:"prompt,omitempty"`
}

type generationPromptLog struct {
	Target string `json:"target"`
	Prompt string `json:"prompt"`
}

type storyboardImageTarget struct {
	Storyboard *model.DramaStoryboard
	Segment    *model.DramaStoryboardSegment
}

type storyboardVideoStartFrame struct {
	FileID uint64
	Label  string
}

type storyboardVideoTarget struct {
	Storyboard *model.DramaStoryboard
	Segment    *model.DramaStoryboardSegment
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
	case "image_generation":
		return s.executeStandaloneImageTask(ctx, task, update)
	case "image", "asset_reference":
		return s.executeImageTask(ctx, task, update)
	case "tts":
		return fmt.Errorf("TTS engine will be connected in the next phase")
	case "video":
		return s.executeVideoTask(ctx, task, update)
	default:
		return fmt.Errorf("unsupported task type: %s", task.Type)
	}
}

func (s *DramaService) executeVideoTask(ctx context.Context, task *model.DramaTask, update func(int, string)) error {
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
	var payload generationTaskPayload
	_ = json.Unmarshal([]byte(task.Payload), &payload)
	return s.generateStoryboardSegmentVideos(ctx, task, client, project, payload, update)
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
	return s.generateStoryboardImages(ctx, task, client, imageSettings, project, payload, update)
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
		prompt = strings.TrimSpace(asset.ReferencePrompt)
	}
	if prompt == "" {
		prompt = asset.Description
	}
	prompt = buildAssetReferencePrompt(asset, prompt)
	s.appendTaskPromptLog(task, asset.Name, prompt)
	update(15, fmt.Sprintf("Generating reference image: %s", asset.Name))
	data, filename, err := client.Generate(ctx, prompt, assetReferenceSettings(settings), update)
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

func (s *DramaService) generateStoryboardImages(ctx context.Context, task *model.DramaTask, client *ComfyClient, settings ImageGenerationSettings, project *model.DramaProject, payload generationTaskPayload, update func(int, string)) error {
	storyboards, err := s.repo.ListStoryboards(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	assets, err := s.repo.ListAssets(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	segments, err := s.repo.ListStoryboardSegments(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	hydrateSegmentReferenceFiles(segments, assets)
	segmentsByStoryboard := make(map[uint64][]model.DramaStoryboardSegment)
	for _, segment := range segments {
		segmentsByStoryboard[segment.StoryboardID] = append(segmentsByStoryboard[segment.StoryboardID], segment)
	}
	selectedSet := make(map[uint64]bool)
	for _, rawID := range payload.StoryboardIDs {
		if id, err := strconv.ParseUint(rawID, 10, 64); err == nil {
			selectedSet[id] = true
		}
	}
	selectedSegmentSet := make(map[uint64]bool)
	for _, rawID := range payload.SegmentIDs {
		if id, err := strconv.ParseUint(rawID, 10, 64); err == nil {
			selectedSegmentSet[id] = true
		}
	}
	targets := make([]storyboardImageTarget, 0, len(storyboards))
	for i := range storyboards {
		storyboard := &storyboards[i]
		if len(selectedSet) > 0 && !selectedSet[storyboard.ID] {
			continue
		}
		if storyboardSegments := segmentsByStoryboard[storyboard.ID]; len(storyboardSegments) > 0 {
			for index := range storyboardSegments {
				if len(selectedSegmentSet) > 0 && !selectedSegmentSet[storyboardSegments[index].ID] {
					continue
				}
				targets = append(targets, storyboardImageTarget{Storyboard: storyboard, Segment: &storyboardSegments[index]})
			}
			if len(selectedSegmentSet) > 0 {
				continue
			}
		} else {
			if len(selectedSegmentSet) > 0 {
				continue
			}
			targets = append(targets, storyboardImageTarget{Storyboard: storyboard})
		}
	}
	if len(targets) == 0 {
		return fmt.Errorf("no storyboard found for image generation")
	}
	imageCount := payload.ImageCount
	if imageCount <= 0 {
		imageCount = 1
	}
	if imageCount > 8 {
		imageCount = 8
	}
	selectedStoryboards := make(map[uint64]bool)
	for index, target := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		storyboard := target.Storyboard
		prompt := s.buildStoryboardImagePrompt(project, storyboard, assets, target.Segment)
		targetSettings := storyboardImageSettings(settings, target.Segment)
		targetTitle := storyboardImageTargetTitle(storyboard, target.Segment)
		references, err := s.buildComfyReferenceImages(ctx, project.OwnerID, filterAssetsForSegment(assets, target.Segment))
		if err != nil {
			return err
		}
		s.appendTaskPromptLog(task, targetTitle, prompt)
		for candidate := 1; candidate <= imageCount; candidate++ {
			total := len(targets) * imageCount
			done := index*imageCount + candidate - 1
			baseProgress := done * 90 / total
			span := 90 / total
			update(5+baseProgress, fmt.Sprintf("Generating %s image %d/%d", targetTitle, candidate, imageCount))
			data, filename, err := client.GenerateWithReferences(ctx, prompt, targetSettings, references, func(localProgress int, message string) {
				mapped := 5 + baseProgress + localProgress*span/100
				update(mapped, fmt.Sprintf("%s image %d/%d: %s", targetTitle, candidate, imageCount, message))
			})
			if err != nil {
				return fmt.Errorf("%s image %d failed: %w", targetTitle, candidate, err)
			}
			label := storyboardImageFileLabel(storyboard, target.Segment, candidate)
			fileID, err := s.saveGeneratedImage(project.OwnerID, project.ID, project.Title, "images", label, filename, data)
			if err != nil {
				return err
			}
			sortOrder, err := s.repo.NextStoryboardMediaSort(project.OwnerID, project.ID, storyboard.ID)
			if err != nil {
				return err
			}
			segmentID := uint64(0)
			if target.Segment != nil {
				segmentID = target.Segment.ID
			}
			shouldSelect := segmentID == 0 && candidate == 1 && !selectedStoryboards[storyboard.ID]
			media := &model.DramaStoryboardMedia{
				ProjectID:    project.ID,
				StoryboardID: storyboard.ID,
				SegmentID:    segmentID,
				OwnerID:      project.OwnerID,
				Kind:         "image",
				FileID:       fileID,
				Source:       "generated",
				Prompt:       prompt,
				SortOrder:    sortOrder,
				Selected:     shouldSelect,
			}
			if err := s.repo.CreateStoryboardMedia(media); err != nil {
				return err
			}
			if shouldSelect {
				if err := s.repo.SelectStoryboardMedia(project.OwnerID, project.ID, storyboard.ID, media.ID, "image"); err != nil {
					return err
				}
				selectedStoryboards[storyboard.ID] = true
				storyboard.ImageFileID = fileID
				if err := s.repo.UpdateStoryboard(storyboard); err != nil {
					return err
				}
			}
			result := generationTaskResult{
				Kind: "storyboard_image", FileID: strconv.FormatUint(fileID, 10), StoryboardID: strconv.FormatUint(storyboard.ID, 10), Title: fmt.Sprintf("%s #%d", targetTitle, candidate), Prompt: prompt,
			}
			if segmentID != 0 {
				result.Kind = "storyboard_segment_image"
				result.SegmentID = strconv.FormatUint(segmentID, 10)
			}
			s.appendTaskResult(task, result)
		}
	}
	return nil
}

func (s *DramaService) generateStoryboardVideos(ctx context.Context, task *model.DramaTask, client *ComfyClient, project *model.DramaProject, payload generationTaskPayload, update func(int, string)) error {
	storyboards, err := s.repo.ListStoryboards(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	assets, err := s.repo.ListAssets(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	segments, err := s.repo.ListStoryboardSegments(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	media, err := s.repo.ListStoryboardMedia(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	hydrateSegmentReferenceFiles(segments, assets)
	segmentsByStoryboard := make(map[uint64][]model.DramaStoryboardSegment)
	for _, segment := range segments {
		segmentsByStoryboard[segment.StoryboardID] = append(segmentsByStoryboard[segment.StoryboardID], segment)
	}
	mediaByStoryboard := make(map[uint64][]model.DramaStoryboardMedia)
	for _, item := range media {
		mediaByStoryboard[item.StoryboardID] = append(mediaByStoryboard[item.StoryboardID], item)
	}
	for storyboardID := range segmentsByStoryboard {
		sort.SliceStable(segmentsByStoryboard[storyboardID], func(i, j int) bool {
			return segmentsByStoryboard[storyboardID][i].Seq < segmentsByStoryboard[storyboardID][j].Seq
		})
	}
	selectedSet := make(map[uint64]bool)
	for _, rawID := range payload.StoryboardIDs {
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
		return fmt.Errorf("no storyboard found for video generation")
	}
	for index, storyboard := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		originalImageFileID := storyboard.ImageFileID
		{
			storyboardSegments := segmentsByStoryboard[storyboard.ID]
			relevantAssets := filterAssetsForSegments(assets, storyboardSegments)
			if len(relevantAssets) == 0 {
				relevantAssets = assets
			}
			startFrame := selectStoryboardVideoStartFrame(storyboard, storyboardSegments, mediaByStoryboard[storyboard.ID], relevantAssets)
			if startFrame.FileID == 0 {
				return fmt.Errorf("分镜 %d 还没有可用的视频首帧，请先生成至少一张片段图片，或给角色/场景资产添加参考图", storyboard.Seq)
			}
			storyboard.ImageFileID = startFrame.FileID
			s.appendTaskPromptLog(task, fmt.Sprintf("Storyboard %d video start frame", storyboard.Seq), fmt.Sprintf("%s (file_id=%d)", startFrame.Label, startFrame.FileID))
		}
		if storyboard.ImageFileID == 0 {
			return fmt.Errorf("分镜 %d 还没有图片，请先生成分镜图片", storyboard.Seq)
		}
		imageData, imageName, err := s.readCloudFile(ctx, project.OwnerID, storyboard.ImageFileID)
		if err != nil {
			return err
		}
		storyboardSegments := segmentsByStoryboard[storyboard.ID]
		relevantAssets := filterAssetsForSegments(assets, storyboardSegments)
		if len(relevantAssets) == 0 {
			relevantAssets = assets
		}
		prompt := buildStoryboardVideoPrompt(project, storyboard, relevantAssets, storyboardSegments)
		negative := buildStoryboardVideoNegativePrompt(storyboardSegments)
		s.appendTaskPromptLog(task, fmt.Sprintf("Storyboard %d video: %s", storyboard.Seq, storyboard.Title), prompt)
		baseProgress := index * 90 / len(targets)
		span := 90 / len(targets)
		update(5+baseProgress, fmt.Sprintf("Generating storyboard %d/%d video: %s", index+1, len(targets), storyboard.Title))
		data, filename, err := client.GenerateVideoFromImage(ctx, imageData, imageName, prompt, negative, 10, func(localProgress int, message string) {
			update(5+baseProgress+localProgress*span/100, fmt.Sprintf("Storyboard %d/%d video: %s", index+1, len(targets), message))
		})
		if err != nil {
			return fmt.Errorf("分镜 %d 视频生成失败：%w", storyboard.Seq, err)
		}
		fileID, err := s.saveGeneratedFile(project.OwnerID, project.ID, project.Title, "videos", fmt.Sprintf("storyboard-%03d-video", storyboard.Seq), filename, data, "video/mp4")
		if err != nil {
			return err
		}
		storyboard.ImageFileID = originalImageFileID
		storyboard.VideoFileID = fileID
		if err := s.repo.UpdateStoryboard(storyboard); err != nil {
			return err
		}
		sortOrder, err := s.repo.NextStoryboardMediaSort(project.OwnerID, project.ID, storyboard.ID)
		if err != nil {
			return err
		}
		if err := s.repo.CreateStoryboardMedia(&model.DramaStoryboardMedia{
			ProjectID: project.ID, StoryboardID: storyboard.ID, OwnerID: project.OwnerID,
			Kind: "video", FileID: fileID, Source: "generated", Prompt: prompt, SortOrder: sortOrder, Selected: true,
		}); err != nil {
			return err
		}
		s.appendTaskResult(task, generationTaskResult{
			Kind: "storyboard_video", FileID: strconv.FormatUint(fileID, 10), StoryboardID: strconv.FormatUint(storyboard.ID, 10), Title: fmt.Sprintf("%s 10s", storyboard.Title), Prompt: prompt,
		})
	}
	return nil
}

func (s *DramaService) generateStoryboardSegmentVideos(ctx context.Context, task *model.DramaTask, client *ComfyClient, project *model.DramaProject, payload generationTaskPayload, update func(int, string)) error {
	storyboards, err := s.repo.ListStoryboards(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	assets, err := s.repo.ListAssets(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	segments, err := s.repo.ListStoryboardSegments(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	media, err := s.repo.ListStoryboardMedia(project.OwnerID, project.ID)
	if err != nil {
		return err
	}
	hydrateSegmentReferenceFiles(segments, assets)

	segmentsByStoryboard := make(map[uint64][]model.DramaStoryboardSegment)
	for _, segment := range segments {
		segmentsByStoryboard[segment.StoryboardID] = append(segmentsByStoryboard[segment.StoryboardID], segment)
	}
	mediaByStoryboard := make(map[uint64][]model.DramaStoryboardMedia)
	for _, item := range media {
		mediaByStoryboard[item.StoryboardID] = append(mediaByStoryboard[item.StoryboardID], item)
	}
	selectedStoryboards := parseGenerationIDSet(payload.StoryboardIDs)
	selectedSegments := parseGenerationIDSet(payload.SegmentIDs)
	targets := make([]storyboardVideoTarget, 0, len(segments))
	for i := range storyboards {
		storyboard := &storyboards[i]
		if len(selectedStoryboards) > 0 && !selectedStoryboards[storyboard.ID] {
			continue
		}
		storyboardSegments := segmentsByStoryboard[storyboard.ID]
		if len(storyboardSegments) == 0 {
			if len(selectedSegments) == 0 {
				targets = append(targets, storyboardVideoTarget{Storyboard: storyboard})
			}
			continue
		}
		for segmentIndex := range storyboardSegments {
			segment := &storyboardSegments[segmentIndex]
			if len(selectedSegments) > 0 && !selectedSegments[segment.ID] {
				continue
			}
			targets = append(targets, storyboardVideoTarget{Storyboard: storyboard, Segment: segment})
		}
	}
	if len(targets) == 0 {
		return fmt.Errorf("no storyboard segment found for video generation")
	}

	for index, target := range targets {
		if err := ctx.Err(); err != nil {
			return err
		}
		storyboard := target.Storyboard
		storyboardSegments := segmentsByStoryboard[storyboard.ID]
		relevantAssets := filterAssetsForSegments(assets, storyboardSegments)
		startFrame := selectStoryboardVideoStartFrame(storyboard, storyboardSegments, mediaByStoryboard[storyboard.ID], relevantAssets)
		durationSec := 10
		prompt := buildStoryboardVideoPrompt(project, storyboard, relevantAssets, storyboardSegments)
		negative := buildStoryboardVideoNegativePrompt(storyboardSegments)
		if target.Segment != nil {
			relevantAssets = filterAssetsForSegment(assets, target.Segment)
			startFrame = selectSegmentVideoStartFrame(storyboard, target.Segment, mediaByStoryboard[storyboard.ID], relevantAssets)
			durationSec = target.Segment.DurationSec
			if durationSec <= 0 {
				durationSec = 3
			}
			prompt = buildSegmentVideoPrompt(storyboard, target.Segment, relevantAssets)
			negative = buildSegmentVideoNegativePrompt(target.Segment)
		}
		if startFrame.FileID == 0 {
			return fmt.Errorf("%s has no usable start frame; generate its segment image first", storyboardVideoTargetTitle(storyboard, target.Segment))
		}
		imageData, imageName, err := s.readCloudFile(ctx, project.OwnerID, startFrame.FileID)
		if err != nil {
			return err
		}
		width, height := videoDimensionsFromImage(imageData)
		targetTitle := storyboardVideoTargetTitle(storyboard, target.Segment)
		s.appendTaskPromptLog(task, targetTitle+" start frame", fmt.Sprintf("%s (file_id=%d)", startFrame.Label, startFrame.FileID))
		s.appendTaskPromptLog(task, targetTitle, prompt)

		baseProgress := index * 90 / len(targets)
		span := 90 / len(targets)
		update(5+baseProgress, fmt.Sprintf("Generating video %d/%d: %s", index+1, len(targets), targetTitle))
		data, filename, err := client.GenerateVideoFromImageSized(ctx, imageData, imageName, prompt, negative, durationSec, width, height, func(localProgress int, message string) {
			update(5+baseProgress+localProgress*span/100, fmt.Sprintf("Video %d/%d: %s", index+1, len(targets), message))
		})
		if err != nil {
			return fmt.Errorf("%s video generation failed: %w", targetTitle, err)
		}

		fileLabel := fmt.Sprintf("storyboard-%03d-video", storyboard.Seq)
		if target.Segment != nil {
			fileLabel = fmt.Sprintf("storyboard-%03d-segment-%02d-video", storyboard.Seq, target.Segment.Seq)
		}
		fileID, err := s.saveGeneratedFile(project.OwnerID, project.ID, project.Title, "videos", fileLabel, filename, data, "video/mp4")
		if err != nil {
			return err
		}
		segmentID := uint64(0)
		selected := true
		resultKind := "storyboard_video"
		if target.Segment != nil {
			segmentID = target.Segment.ID
			selected = false
			resultKind = "storyboard_segment_video"
			target.Segment.VideoFileID = fileID
			if err := s.repo.UpdateStoryboardSegment(target.Segment); err != nil {
				return err
			}
		} else {
			storyboard.VideoFileID = fileID
			if err := s.repo.UpdateStoryboard(storyboard); err != nil {
				return err
			}
		}
		sortOrder, err := s.repo.NextStoryboardMediaSort(project.OwnerID, project.ID, storyboard.ID)
		if err != nil {
			return err
		}
		if err := s.repo.CreateStoryboardMedia(&model.DramaStoryboardMedia{
			ProjectID: project.ID, StoryboardID: storyboard.ID, SegmentID: segmentID, OwnerID: project.OwnerID,
			Kind: "video", FileID: fileID, Source: "generated", Prompt: prompt, SortOrder: sortOrder, Selected: selected,
		}); err != nil {
			return err
		}
		result := generationTaskResult{
			Kind: resultKind, FileID: strconv.FormatUint(fileID, 10), StoryboardID: strconv.FormatUint(storyboard.ID, 10),
			Title: fmt.Sprintf("%s %ds %dx%d", targetTitle, durationSec, width, height), Prompt: prompt,
		}
		if segmentID != 0 {
			result.SegmentID = strconv.FormatUint(segmentID, 10)
		}
		s.appendTaskResult(task, result)
	}
	return nil
}

func (s *DramaService) buildStoryboardImagePrompt(project *model.DramaProject, storyboard *model.DramaStoryboard, assets []model.DramaAsset, segment *model.DramaStoryboardSegment) string {
	basePrompt := strings.TrimSpace(storyboard.Prompt)
	if basePrompt == "" {
		basePrompt = synthesizePrompt(storyboard.Content)
	}
	scene := strings.TrimSpace(storyboard.SceneAnchor)
	plot := strings.TrimSpace(storyboard.Plot)
	if plot == "" {
		plot = strings.TrimSpace(storyboard.Content)
	}
	segmentTitle := ""
	compositionPrompt := ""
	if segment != nil {
		segmentTitle = strings.TrimSpace(segment.Title)
		compositionPrompt = strings.TrimSpace(segment.CompositionPrompt)
		if strings.TrimSpace(segment.ReferencePrompt) != "" {
			basePrompt = strings.TrimSpace(segment.ReferencePrompt)
		}
		if strings.TrimSpace(segment.Scene) != "" {
			scene = strings.TrimSpace(segment.Scene)
		}
		segmentPlot := strings.Join([]string{
			strings.TrimSpace(segment.Purpose),
			strings.TrimSpace(segment.Action),
			strings.TrimSpace(segment.Dialogue),
			strings.TrimSpace(segment.Shot),
		}, " ")
		if strings.TrimSpace(segmentPlot) != "" {
			plot = segmentPlot
		}
	}
	relevantAssets := assets
	if segment != nil {
		relevantAssets = filterAssetsForSegment(assets, segment)
	}
	characters := make([]string, 0)
	scenes := make([]string, 0)
	for _, asset := range relevantAssets {
		name := strings.TrimSpace(asset.Name)
		description := compactText(strings.TrimSpace(asset.Description+" "+asset.ReferencePrompt), 140)
		if name == "" && description == "" {
			continue
		}
		line := strings.TrimSpace(name + ": " + description)
		if asset.ReferenceFileID != 0 {
			line += fmt.Sprintf(" (approved reference image file_id=%d, keep identity and styling consistent)", asset.ReferenceFileID)
		}
		if asset.Type == "character" {
			characters = append(characters, line)
		} else if asset.Type == "scene" {
			scenes = append(scenes, line)
		}
	}
	parts := []string{
		"Composition priority: " + compactText(compositionPrompt, 420),
		"Realistic cinematic still frame, wide horizontal 16:9, coherent natural lighting, clear faces and hands.",
		"Visible action and dialogue state: " + compactText(plot, 360),
		"Scene: " + compactText(scene, 180),
		"Visual details: " + compactText(basePrompt, 380),
		"Continuity rule: preserve identities, age, hairstyle, clothing, body shape, props, and room layout from approved references. References control identity and environment only; the composition priority controls pose and framing.",
		"Shot: " + strings.TrimSpace(storyboard.Title) + " / " + segmentTitle,
	}
	if len(characters) > 0 {
		parts = append(parts, "Characters: "+compactText(strings.Join(characters, "; "), 420))
	}
	if len(scenes) > 0 {
		parts = append(parts, "Environment reference: "+compactText(strings.Join(scenes, "; "), 280))
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

func filterAssetsForSegment(assets []model.DramaAsset, segment *model.DramaStoryboardSegment) []model.DramaAsset {
	if segment == nil {
		return assets
	}
	characters := strings.TrimSpace(segment.Characters)
	scene := strings.TrimSpace(segment.Scene)
	matched := make([]model.DramaAsset, 0)
	for _, asset := range assets {
		name := strings.TrimSpace(asset.Name)
		if name == "" {
			continue
		}
		switch asset.Type {
		case "character":
			if characters != "" && strings.Contains(characters, name) {
				matched = append(matched, asset)
			}
		case "scene":
			if scene != "" && strings.Contains(scene, name) {
				matched = append(matched, asset)
			}
		}
	}
	if len(matched) > 0 {
		return matched
	}
	return assets
}

func filterAssetsForSegments(assets []model.DramaAsset, segments []model.DramaStoryboardSegment) []model.DramaAsset {
	if len(segments) == 0 {
		return assets
	}
	seen := make(map[uint64]bool)
	matched := make([]model.DramaAsset, 0)
	for i := range segments {
		filtered := filterAssetsForSegment(assets, &segments[i])
		for _, asset := range filtered {
			if seen[asset.ID] {
				continue
			}
			if len(filtered) == len(assets) {
				continue
			}
			seen[asset.ID] = true
			matched = append(matched, asset)
		}
	}
	if len(matched) > 0 {
		return matched
	}
	return assets
}

func (s *DramaService) buildComfyReferenceImages(ctx context.Context, ownerID uint64, assets []model.DramaAsset) ([]ComfyReferenceImage, error) {
	sort.SliceStable(assets, func(i, j int) bool {
		if assets[i].Type == assets[j].Type {
			return assets[i].ID < assets[j].ID
		}
		return assets[i].Type != "scene" && assets[j].Type == "scene"
	})
	fileIDs := make([]uint64, 0, len(assets))
	assetByFileID := make(map[uint64]model.DramaAsset)
	for _, asset := range assets {
		if asset.ReferenceFileID == 0 {
			continue
		}
		if _, exists := assetByFileID[asset.ReferenceFileID]; exists {
			continue
		}
		fileIDs = append(fileIDs, asset.ReferenceFileID)
		assetByFileID[asset.ReferenceFileID] = asset
		if len(fileIDs) >= 5 {
			break
		}
	}
	if len(fileIDs) == 0 {
		return nil, nil
	}
	files, err := s.repo.ListFilesByIDs(ownerID, fileIDs)
	if err != nil {
		return nil, err
	}
	fileByID := make(map[uint64]model.File, len(files))
	for _, file := range files {
		fileByID[file.ID] = file
	}
	refs := make([]ComfyReferenceImage, 0, len(files))
	for _, fileID := range fileIDs {
		file, ok := fileByID[fileID]
		if !ok {
			continue
		}
		asset := assetByFileID[file.ID]
		object, err := s.minioClient.GetObject(ctx, s.bucket, file.StorageKey, minio.GetObjectOptions{})
		if err != nil {
			return nil, fmt.Errorf("读取资产参考图失败：%w", err)
		}
		data, readErr := io.ReadAll(object)
		closeErr := object.Close()
		if readErr != nil {
			return nil, fmt.Errorf("读取资产参考图失败：%w", readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("关闭资产参考图失败：%w", closeErr)
		}
		refs = append(refs, ComfyReferenceImage{
			Name:   safeReferenceFilename(asset.Name, file.Name),
			Data:   data,
			Kind:   asset.Type,
			Weight: referenceAssetWeight(asset),
		})
	}
	return refs, nil
}

func safeReferenceFilename(assetName, fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" || len(ext) > 8 {
		ext = ".png"
	}
	name := sanitizeName(assetName)
	if name == "" {
		name = "asset-reference"
	}
	return name + ext
}

func referenceAssetWeight(asset model.DramaAsset) float64 {
	if asset.Type == "scene" {
		return 0.55
	}
	return 0.62
}

func storyboardImageTargetTitle(storyboard *model.DramaStoryboard, segment *model.DramaStoryboardSegment) string {
	if segment == nil {
		return fmt.Sprintf("Storyboard %d: %s", storyboard.Seq, storyboard.Title)
	}
	title := strings.TrimSpace(segment.Title)
	if title == "" {
		title = fmt.Sprintf("Segment %d", segment.Seq)
	}
	return fmt.Sprintf("Storyboard %d Segment %d: %s", storyboard.Seq, segment.Seq, title)
}

func storyboardImageFileLabel(storyboard *model.DramaStoryboard, segment *model.DramaStoryboardSegment, candidate int) string {
	if segment == nil {
		return fmt.Sprintf("storyboard-%03d-%02d", storyboard.Seq, candidate)
	}
	return fmt.Sprintf("storyboard-%03d-segment-%02d-%02d", storyboard.Seq, segment.Seq, candidate)
}

func selectStoryboardVideoStartFrame(storyboard *model.DramaStoryboard, segments []model.DramaStoryboardSegment, media []model.DramaStoryboardMedia, assets []model.DramaAsset) storyboardVideoStartFrame {
	for _, segment := range segments {
		for _, item := range media {
			if item.Kind == "image" && item.FileID != 0 && item.SegmentID == segment.ID {
				return storyboardVideoStartFrame{FileID: item.FileID, Label: fmt.Sprintf("segment %d generated image", segment.Seq)}
			}
		}
	}
	for _, segment := range segments {
		if segment.ReferenceFileID != 0 {
			return storyboardVideoStartFrame{FileID: segment.ReferenceFileID, Label: fmt.Sprintf("segment %d reference image", segment.Seq)}
		}
	}
	if storyboard.ImageFileID != 0 {
		return storyboardVideoStartFrame{FileID: storyboard.ImageFileID, Label: "storyboard selected image"}
	}
	for _, item := range media {
		if item.Kind == "image" && item.FileID != 0 && item.SegmentID == 0 && item.Selected {
			return storyboardVideoStartFrame{FileID: item.FileID, Label: "selected storyboard image"}
		}
	}
	for _, item := range media {
		if item.Kind == "image" && item.FileID != 0 && item.SegmentID == 0 {
			return storyboardVideoStartFrame{FileID: item.FileID, Label: "storyboard image"}
		}
	}
	for _, asset := range assets {
		if asset.Type == "scene" && asset.ReferenceFileID != 0 {
			return storyboardVideoStartFrame{FileID: asset.ReferenceFileID, Label: "global scene reference image"}
		}
	}
	for _, asset := range assets {
		if asset.Type == "character" && asset.ReferenceFileID != 0 {
			return storyboardVideoStartFrame{FileID: asset.ReferenceFileID, Label: "global character reference image"}
		}
	}
	return storyboardVideoStartFrame{}
}

func selectSegmentVideoStartFrame(storyboard *model.DramaStoryboard, segment *model.DramaStoryboardSegment, media []model.DramaStoryboardMedia, assets []model.DramaAsset) storyboardVideoStartFrame {
	for index := len(media) - 1; index >= 0; index-- {
		item := media[index]
		if item.Kind == "image" && item.FileID != 0 && item.SegmentID == segment.ID {
			return storyboardVideoStartFrame{FileID: item.FileID, Label: fmt.Sprintf("segment %d latest generated image", segment.Seq)}
		}
	}
	if storyboard.ImageFileID != 0 {
		return storyboardVideoStartFrame{FileID: storyboard.ImageFileID, Label: "storyboard selected image"}
	}
	for index := len(media) - 1; index >= 0; index-- {
		item := media[index]
		if item.Kind == "image" && item.FileID != 0 && item.SegmentID == 0 && item.Selected {
			return storyboardVideoStartFrame{FileID: item.FileID, Label: "selected storyboard image"}
		}
	}
	if segment.ReferenceFileID != 0 {
		return storyboardVideoStartFrame{FileID: segment.ReferenceFileID, Label: fmt.Sprintf("segment %d asset reference image", segment.Seq)}
	}
	for _, asset := range assets {
		if asset.Type == "scene" && asset.ReferenceFileID != 0 {
			return storyboardVideoStartFrame{FileID: asset.ReferenceFileID, Label: "scene reference image"}
		}
	}
	for _, asset := range assets {
		if asset.Type == "character" && asset.ReferenceFileID != 0 {
			return storyboardVideoStartFrame{FileID: asset.ReferenceFileID, Label: "character reference image"}
		}
	}
	return storyboardVideoStartFrame{}
}

func buildSegmentVideoPrompt(storyboard *model.DramaStoryboard, segment *model.DramaStoryboardSegment, assets []model.DramaAsset) string {
	motion := strings.TrimSpace(segment.VideoPrompt)
	if motion == "" {
		motion = strings.TrimSpace(segment.Action)
	}
	assetNames := make([]string, 0, len(assets))
	for _, asset := range assets {
		if name := strings.TrimSpace(asset.Name); name != "" {
			assetNames = append(assetNames, name)
		}
	}
	parts := []string{
		"Single continuous cinematic shot. Use the input image as the exact first frame.",
		"Motion: " + compactText(motion, 320),
		"Visible action: " + compactText(segment.Action, 260),
		"Camera and framing: " + compactText(segment.Shot, 180),
		"Characters visible: " + compactText(segment.Characters, 160),
		"Scene: " + compactText(segment.Scene, 160),
		"Continuity assets: " + strings.Join(assetNames, ", "),
		"Preserve every face, hairstyle, costume, body shape, prop, room layout, lighting direction, and framing from the first frame. Use subtle natural body motion. No new person, object, costume, or location.",
		"Storyboard context: " + compactText(storyboard.Title, 120),
	}
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && !strings.HasSuffix(part, ":") {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "\n")
}

func buildSegmentVideoNegativePrompt(segment *model.DramaStoryboardSegment) string {
	characterCount := segmentCharacterCount(segment.Characters)
	characters := strings.ToLower(segment.Characters)
	closeUpRequested := strings.Contains(strings.ToLower(segment.Shot), "close-up") ||
		strings.Contains(segment.Shot, "特写") || strings.Contains(segment.Shot, "近景")
	parts := []string{
		"text", "subtitles", "watermark", "logo", "UI", "flicker", "jitter",
		"distorted face", "deformed hands", "identity change", "clothing change",
		"hairstyle change", "scene change", "room redesign", "extra people",
		"wrong gaze direction", "wrong hand position", "foreground obstruction",
	}
	if characterCount >= 2 {
		parts = append(parts, "solo portrait", "cropped second person", "missing character")
	}
	for _, rawPart := range splitPromptTerms(segment.NegativePrompt) {
		part := strings.TrimSpace(rawPart)
		lower := strings.ToLower(part)
		if part == "" {
			continue
		}
		if characterCount <= 1 && (lower == "single person" || lower == "solo portrait" || lower == "cropped second person" || lower == "missing character") {
			continue
		}
		if !strings.Contains(characters, "丈夫") && lower == "missing husband" {
			continue
		}
		if !strings.Contains(characters, "妻子") && lower == "missing wife" {
			continue
		}
		if closeUpRequested && (lower == "close-up" || lower == "close-up portrait" || lower == "extreme close-up") {
			continue
		}
		parts = append(parts, part)
	}
	return strings.Join(uniquePromptTerms(parts), ", ")
}

func storyboardVideoTargetTitle(storyboard *model.DramaStoryboard, segment *model.DramaStoryboardSegment) string {
	if segment == nil {
		return fmt.Sprintf("Storyboard %d: %s", storyboard.Seq, storyboard.Title)
	}
	title := strings.TrimSpace(segment.Title)
	if title == "" {
		title = fmt.Sprintf("Segment %d", segment.Seq)
	}
	return fmt.Sprintf("Storyboard %d Segment %d: %s", storyboard.Seq, segment.Seq, title)
}

func parseGenerationIDSet(values []string) map[uint64]bool {
	result := make(map[uint64]bool)
	for _, value := range values {
		if id, err := strconv.ParseUint(value, 10, 64); err == nil && id != 0 {
			result[id] = true
		}
	}
	return result
}

func videoDimensionsFromImage(data []byte) (int, int) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return 832, 480
	}
	if config.Width >= config.Height {
		height := roundVideoDimension(832 * config.Height / config.Width)
		return 832, height
	}
	width := roundVideoDimension(832 * config.Width / config.Height)
	return width, 832
}

func roundVideoDimension(value int) int {
	if value < 256 {
		value = 256
	}
	return ((value + 8) / 16) * 16
}

func segmentCharacterCount(value string) int {
	count := 0
	for _, item := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n'
	}) {
		if strings.TrimSpace(item) != "" {
			count++
		}
	}
	return count
}

func splitPromptTerms(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == '\n'
	})
}

func uniquePromptTerms(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func buildStoryboardVideoPrompt(project *model.DramaProject, storyboard *model.DramaStoryboard, assets []model.DramaAsset, segments []model.DramaStoryboardSegment) string {
	assetLines := make([]string, 0, len(assets))
	for _, asset := range assets {
		name := strings.TrimSpace(asset.Name)
		if name == "" {
			continue
		}
		kind := "scene"
		if asset.Type == "character" {
			kind = "character"
		}
		line := fmt.Sprintf("%s %s: %s", kind, name, compactText(strings.TrimSpace(asset.Description+" "+asset.ReferencePrompt), 180))
		if asset.ReferenceFileID != 0 {
			line += " (approved global reference image exists; preserve this identity/layout)"
		}
		assetLines = append(assetLines, line)
	}
	timeline := make([]string, 0, len(segments))
	for index, segment := range segments {
		label := strings.TrimSpace(segment.Title)
		if label == "" {
			label = fmt.Sprintf("segment %d", segment.Seq)
		}
		duration := segment.DurationSec
		if duration <= 0 {
			duration = 3
		}
		items := []string{
			fmt.Sprintf("%d. %s, about %d seconds", index+1, label, duration),
			"characters: " + strings.TrimSpace(segment.Characters),
			"scene: " + strings.TrimSpace(segment.Scene),
			"action: " + compactText(segment.Action, 220),
			"shot: " + compactText(segment.Shot, 180),
			"composition: " + compactText(segment.CompositionPrompt, 220),
			"motion: " + compactText(segment.VideoPrompt, 220),
			"dialogue mood: " + compactText(segment.Dialogue, 160),
		}
		keptItems := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item != "" && !strings.HasSuffix(item, ":") {
				keptItems = append(keptItems, item)
			}
		}
		timeline = append(timeline, strings.Join(keptItems, "; "))
	}
	if len(timeline) == 0 {
		timeline = append(timeline, "single continuous segment: "+compactText(strings.TrimSpace(storyboard.Plot+" "+storyboard.Content), 900))
	}
	parts := []string{
		"10-second single-shot cinematic image-to-video.",
		"Use the input image as the first frame. Preserve the approved global character identities, clothing, hairstyles, body shapes, and approved scene layout throughout the whole shot.",
		"Use the storyboard segment timeline below as the motion plan. Show all required characters from the segments when they are listed. Do not turn the shot into a solo portrait unless the segment explicitly has only one character.",
		"Keep one continuous camera shot for the whole storyboard, smooth natural motion only, no hard scene cut, no identity change, no clothing change, no room redesign.",
		"Project context: " + compactText(project.Preface, 260),
		"Global continuity assets: " + compactText(strings.Join(assetLines, " | "), 900),
		"Shot title: " + strings.TrimSpace(storyboard.Title),
		"Storyboard content: " + compactText(storyboard.Content, 900),
		"Dialogue context: " + compactText(storyboard.Dialogue, 400),
		"Visual action: " + compactText(storyboard.Plot, 500),
		"Segment timeline: " + strings.Join(timeline, "\n"),
		"Camera motion: subtle handheld stillness or slow push-in, realistic breathing, small eye movements, natural hand movements, preserve composition from the first frame.",
	}
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && !strings.HasSuffix(part, ":") {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "\n")
}

func buildStoryboardVideoNegativePrompt(segments []model.DramaStoryboardSegment) string {
	parts := []string{
		"text, subtitles, watermark, logo, UI, flicker, jitter, distorted face, deformed hands, identity change, clothing change, hairstyle change, scene change, room redesign, extra people, missing character, solo portrait, close-up portrait, cropped character, wrong gaze direction, wrong hand position, foreground obstruction",
	}
	for _, segment := range segments {
		if strings.TrimSpace(segment.NegativePrompt) != "" {
			parts = append(parts, strings.TrimSpace(segment.NegativePrompt))
		}
	}
	return strings.Join(parts, ", ")
}

func compactText(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if limit > 0 && len([]rune(value)) > limit {
		runes := []rune(value)
		return string(runes[:limit])
	}
	return value
}

func buildAssetReferencePrompt(asset *model.DramaAsset, raw string) string {
	if strings.TrimSpace(asset.ReferencePrompt) != "" {
		raw = asset.ReferencePrompt
	}
	referencePrompt := extractPromptField(raw, []string{"参考图提示词", "reference_prompt", "Reference prompt"})
	if referencePrompt == "" {
		referencePrompt = cleanPromptText(raw)
	}
	englishHints := inferAssetEnglishHints(asset, raw+" "+asset.Description+" "+referencePrompt)
	prefix := "realistic cinematic reference photo, clear subject, complete composition, high detail, natural skin texture, not concept art"
	if asset.Type == "character" {
		prefix = "realistic cinematic character reference photo, single real person, full body or medium full shot, clear face, natural skin texture, modern realistic clothing, not fantasy, not armor"
	} else if asset.Type == "scene" {
		prefix = "realistic cinematic environment reference photo, wide establishing shot, clear spatial layout, coherent lighting, high detail"
	}
	parts := []string{prefix, englishHints, referencePrompt, "no text, no watermark, no logo, no UI, no extra people unless requested"}
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, ", ")
}

func assetReferenceSettings(settings ImageGenerationSettings) ImageGenerationSettings {
	settings.NegativePrompt = strings.TrimSpace(strings.Join([]string{
		settings.NegativePrompt,
		"armor, cuirass, breastplate, knight, medieval, fantasy costume, robe, cloak, cape, game character, concept art, illustration, anime, painting, sketch, plastic skin, doll",
	}, ", "))
	return settings
}

func storyboardImageSettings(settings ImageGenerationSettings, segment *model.DramaStoryboardSegment) ImageGenerationSettings {
	if segment != nil && settings.Height >= settings.Width {
		settings.Width = 1344
		settings.Height = 768
	}
	negativeParts := []string{settings.NegativePrompt}
	if segment == nil {
		negativeParts = append(negativeParts, "wrong pose, wrong hand position, wrong gaze direction, different room, foreground obstruction, vertical poster composition")
	} else {
		negativeParts = append(negativeParts, buildSegmentImageNegativePrompt(segment))
	}
	settings.NegativePrompt = strings.TrimSpace(strings.Join(negativeParts, ", "))
	return settings
}

func buildSegmentImageNegativePrompt(segment *model.DramaStoryboardSegment) string {
	characterCount := segmentCharacterCount(segment.Characters)
	characters := strings.ToLower(segment.Characters)
	closeUpRequested := strings.Contains(strings.ToLower(segment.Shot), "close-up") ||
		strings.Contains(segment.Shot, "特写") || strings.Contains(segment.Shot, "近景")
	parts := []string{
		"low quality", "blurry", "deformed face", "deformed hands", "extra fingers",
		"wrong identity", "different face", "different clothing", "different room",
		"wrong pose", "wrong hand position", "wrong gaze direction", "extra people",
		"foreground obstruction", "vertical poster composition", "text", "watermark", "logo", "UI",
	}
	if characterCount >= 2 {
		parts = append(parts, "solo portrait", "cropped second person", "missing character")
	}
	for _, rawPart := range splitPromptTerms(segment.NegativePrompt) {
		part := strings.TrimSpace(rawPart)
		lower := strings.ToLower(part)
		if part == "" {
			continue
		}
		if characterCount <= 1 && (lower == "single person" || lower == "solo portrait" || lower == "cropped second person" || lower == "missing character") {
			continue
		}
		if !strings.Contains(characters, "丈夫") && lower == "missing husband" {
			continue
		}
		if !strings.Contains(characters, "妻子") && lower == "missing wife" {
			continue
		}
		if closeUpRequested && (lower == "close-up" || lower == "close-up portrait" || lower == "extreme close-up" || lower == "headshot" || lower == "bust shot") {
			continue
		}
		parts = append(parts, part)
	}
	return strings.Join(uniquePromptTerms(parts), ", ")
}

func inferAssetEnglishHints(asset *model.DramaAsset, text string) string {
	if asset.Type != "character" {
		return ""
	}
	lower := strings.ToLower(text)
	hints := []string{}
	if containsAny(text, []string{"中国", "華人", "华人", "Chinese"}) {
		hints = append(hints, "Chinese")
	}
	if containsAny(text, []string{"男性", "男人", "丈夫", "男声", "男"}) {
		hints = append(hints, "man")
	}
	if containsAny(text, []string{"女性", "女人", "妻子", "女声", "女"}) {
		hints = append(hints, "woman")
	}
	if strings.Contains(text, "32") {
		hints = append(hints, "32 years old")
	}
	if containsAny(text, []string{"国字脸", "方脸"}) {
		hints = append(hints, "square face")
	}
	if strings.Contains(text, "剑眉") {
		hints = append(hints, "strong straight eyebrows")
	}
	if strings.Contains(text, "星目") {
		hints = append(hints, "bright eyes")
	}
	if strings.Contains(text, "鼻梁高") {
		hints = append(hints, "high nose bridge")
	}
	if containsAny(text, []string{"短发", "黑发"}) {
		hints = append(hints, "short black hair")
	}
	if strings.Contains(text, "西装") || strings.Contains(lower, "suit") {
		hints = append(hints, "modern tailored business suit")
	}
	if strings.Contains(text, "深蓝") || strings.Contains(lower, "navy") {
		hints = append(hints, "navy blue")
	}
	if strings.Contains(text, "白衬衫") || strings.Contains(lower, "white shirt") {
		hints = append(hints, "white dress shirt")
	}
	if strings.Contains(text, "皮鞋") {
		hints = append(hints, "black leather shoes")
	}
	if strings.Contains(text, "手表") || strings.Contains(text, "腕表") {
		hints = append(hints, "silver wristwatch")
	}
	if containsAny(text, []string{"室内", "暖光"}) {
		hints = append(hints, "warm indoor lighting")
	}
	if len(hints) == 0 {
		return ""
	}
	hints = append([]string{"modern realistic photo of"}, hints...)
	return strings.Join(hints, ", ")
}

func extractPromptField(raw string, labels []string) string {
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		for _, label := range labels {
			if !strings.Contains(lower, strings.ToLower(label)) {
				continue
			}
			if value := textAfterSeparator(trimmed); value != "" {
				return value
			}
		}
	}
	return ""
}

func cleanPromptText(raw string) string {
	lines := make([]string, 0)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if value := textAfterSeparator(line); value != "" {
			line = value
		}
		lines = append(lines, line)
	}
	return compactText(strings.Join(lines, ", "), 900)
}

func textAfterSeparator(line string) string {
	separators := []string{"：", ":", "=", " - "}
	for _, separator := range separators {
		if index := strings.Index(line, separator); index >= 0 {
			return strings.TrimSpace(line[index+len(separator):])
		}
	}
	return ""
}

func (s *DramaService) clearTaskGeneratedPayload(task *model.DramaTask) {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil || payload == nil {
		return
	}
	delete(payload, "results")
	delete(payload, "prompt_log")
	s.ensureTaskSourceFields(task, payload)
	if data, err := json.Marshal(payload); err == nil {
		task.Payload = string(data)
	}
}

func (s *DramaService) appendTaskPromptLog(task *model.DramaTask, target, prompt string) {
	payload := decodeTaskPayloadMap(task)
	var promptLog []generationPromptLog
	if raw, ok := payload["prompt_log"]; ok {
		if data, err := json.Marshal(raw); err == nil {
			_ = json.Unmarshal(data, &promptLog)
		}
	}
	promptLog = append(promptLog, generationPromptLog{Target: target, Prompt: prompt})
	payload["prompt_log"] = promptLog
	s.ensureTaskSourceFields(task, payload)
	if data, err := json.Marshal(payload); err == nil {
		task.Payload = string(data)
		_ = s.repo.UpdateTask(task)
	}
}

func (s *DramaService) appendTaskResult(task *model.DramaTask, result generationTaskResult) {
	payload := decodeTaskPayloadMap(task)
	var results []generationTaskResult
	if raw, ok := payload["results"]; ok {
		if data, err := json.Marshal(raw); err == nil {
			_ = json.Unmarshal(data, &results)
		}
	}
	results = append(results, result)
	payload["results"] = results
	s.ensureTaskSourceFields(task, payload)
	if data, err := json.Marshal(payload); err == nil {
		task.Payload = string(data)
		_ = s.repo.UpdateTask(task)
	}
}

func decodeTaskPayloadMap(task *model.DramaTask) map[string]interface{} {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil || payload == nil {
		return make(map[string]interface{})
	}
	return payload
}

func (s *DramaService) ensureTaskSourceFields(task *model.DramaTask, payload map[string]interface{}) {
	if label, ok := payload["source_label"].(string); ok && strings.TrimSpace(label) != "" {
		return
	}
	ids := payloadStringSlice(payload["storyboard_ids"])
	if len(ids) == 0 {
		return
	}
	storyboards, err := s.repo.ListStoryboards(task.OwnerID, task.ProjectID)
	if err != nil || len(storyboards) == 0 {
		return
	}
	payload["storyboard_count"] = len(ids)
	if len(ids) == len(storyboards) {
		payload["source_label"] = fmt.Sprintf("全部分镜（%d 镜）", len(storyboards))
		return
	}
	selected := make(map[uint64]bool, len(ids))
	for _, rawID := range ids {
		if id, err := strconv.ParseUint(rawID, 10, 64); err == nil {
			selected[id] = true
		}
	}
	matched := make([]model.DramaStoryboard, 0, len(ids))
	for _, storyboard := range storyboards {
		if selected[storyboard.ID] {
			matched = append(matched, storyboard)
		}
	}
	if len(matched) == 1 {
		payload["source_label"] = fmt.Sprintf("分镜 %d：%s", matched[0].Seq, matched[0].Title)
		return
	}
	if len(matched) > 1 {
		seqs := make([]string, 0, len(matched))
		for _, storyboard := range matched {
			seqs = append(seqs, strconv.Itoa(storyboard.Seq))
		}
		payload["source_label"] = "分镜 " + strings.Join(seqs, "、")
	}
}

func payloadStringSlice(value interface{}) []string {
	items, ok := value.([]interface{})
	if ok {
		result := make([]string, 0, len(items))
		for _, item := range items {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" {
				result = append(result, text)
			}
		}
		return result
	}
	if items, ok := value.([]string); ok {
		return items
	}
	return nil
}

func (s *DramaService) saveGeneratedImage(ownerID, projectID uint64, title, folderKey, label, originalFilename string, data []byte) (uint64, error) {
	return s.saveGeneratedFile(ownerID, projectID, title, folderKey, label, originalFilename, data, "")
}

func (s *DramaService) saveGeneratedFile(ownerID, projectID uint64, title, folderKey, label, originalFilename string, data []byte, fallbackContentType string) (uint64, error) {
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
	if fallbackContentType != "" && (contentType == "application/octet-stream" || strings.HasPrefix(contentType, "text/plain")) {
		contentType = fallbackContentType
	}
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

func (s *DramaService) readCloudFile(ctx context.Context, ownerID, fileID uint64) ([]byte, string, error) {
	files, err := s.repo.ListFilesByIDs(ownerID, []uint64{fileID})
	if err != nil {
		return nil, "", err
	}
	if len(files) == 0 {
		return nil, "", fmt.Errorf("云盘文件不存在或已删除")
	}
	object, err := s.minioClient.GetObject(ctx, s.bucket, files[0].StorageKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	data, readErr := io.ReadAll(object)
	closeErr := object.Close()
	if readErr != nil {
		return nil, "", readErr
	}
	if closeErr != nil {
		return nil, "", closeErr
	}
	return data, files[0].Name, nil
}

func assetTypeLabel(assetType string) string {
	if assetType == "character" {
		return "character reference"
	}
	return "scene reference"
}
