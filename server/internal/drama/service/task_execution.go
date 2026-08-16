package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
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
	case "video":
		return s.executeVideoTask(ctx, task, update)
	default:
		return fmt.Errorf("unsupported task type: %s", task.Type)
	}
}

func (s *DramaService) comfyClientForTask(ctx context.Context, ownerID uint64) (*ComfyClient, *model.DramaSetting, ComfyStatus, error) {
	setting, err := s.repo.GetSetting(ownerID)
	if err != nil {
		return nil, nil, ComfyStatus{}, err
	}
	client := NewComfyClient(setting.ComfyUIURL)
	status := client.Status(ctx)
	if !status.Connected {
		return nil, nil, status, fmt.Errorf("ComfyUI is not reachable: %s", status.Error)
	}
	return client, setting, status, nil
}

func decodeGenerationTaskPayload(task *model.DramaTask) (generationTaskPayload, error) {
	var payload generationTaskPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return generationTaskPayload{}, fmt.Errorf("invalid generation payload: %w", err)
	}
	return payload, nil
}

func (s *DramaService) executeVideoTask(ctx context.Context, task *model.DramaTask, update func(int, string)) error {
	project, err := s.repo.GetProject(task.OwnerID, task.ProjectID)
	if err != nil {
		return err
	}
	client, setting, _, err := s.comfyClientForTask(ctx, task.OwnerID)
	if err != nil {
		return err
	}
	payload, err := decodeGenerationTaskPayload(task)
	if err != nil {
		return err
	}
	videoSettings := defaultVideoGenerationSettings(setting.VideoSettings)
	return s.generateStoryboardSegmentVideos(ctx, task, client, project, payload, videoSettings, update)
}

func (s *DramaService) executeImageTask(ctx context.Context, task *model.DramaTask, update func(int, string)) error {
	project, err := s.repo.GetProject(task.OwnerID, task.ProjectID)
	if err != nil {
		return err
	}
	client, setting, status, err := s.comfyClientForTask(ctx, task.OwnerID)
	if err != nil {
		return err
	}
	imageSettings := defaultImageGenerationSettings(setting.ImageSettings)
	imageSettings.Checkpoint = selectImageCheckpoint(
		imageSettings.Checkpoint,
		status.Checkpoints,
		project.Description+" "+project.Settings+" "+project.Preface,
	)
	if imageSettings.Checkpoint == "" {
		return fmt.Errorf("ComfyUI is connected, but no checkpoint model was detected")
	}
	imageSettings.UseFaceID = status.Models["ipadapter_faceid_plusv2_sdxl"] &&
		status.Models["ipadapter_faceid_lora_sdxl"] &&
		status.Models["faceid_nodes"]

	payload, err := decodeGenerationTaskPayload(task)
	if err != nil {
		return err
	}
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
	styleHint := project.Description + " " + project.Settings + " " + project.Preface + " " + prompt
	update(15, fmt.Sprintf("Generating reference image: %s", asset.Name))
	assetSettings := assetReferenceSettings(settings, styleHint)
	var data []byte
	var filename string
	if asset.Type == "character" {
		viewPrompts := buildCharacterViewPrompts(asset, prompt, styleHint)
		for index, viewPrompt := range viewPrompts {
			s.appendTaskPromptLog(task, fmt.Sprintf("%s %s", asset.Name, characterViewLabel(index)), viewPrompt)
		}
		data, filename, err = generateCharacterTurnaround(ctx, client, settings, styleHint, viewPrompts, update)
		prompt = strings.Join(viewPrompts, "\n")
	} else {
		prompt = buildAssetReferencePrompt(asset, prompt, styleHint)
		s.appendTaskPromptLog(task, asset.Name, prompt)
		data, filename, err = client.Generate(ctx, prompt, assetSettings, update)
	}
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

func (s *DramaService) generateStoryboardSegmentVideos(ctx context.Context, task *model.DramaTask, client *ComfyClient, project *model.DramaProject, payload generationTaskPayload, videoSettings VideoGenerationSettings, update func(int, string)) error {
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
		width, height := videoDimensionsForSettings(imageData, videoSettings)
		targetTitle := storyboardVideoTargetTitle(storyboard, target.Segment)
		if target.Segment != nil && videoSettings.Continuity {
			if previous := adjacentSegmentTarget(target.Segment.ID, storyboards, segmentsByStoryboard, -1); previous.Segment != nil {
				if tail := selectSegmentTailFrame(previous.Segment.ID, mediaByStoryboard[previous.Storyboard.ID]); tail.FileID != 0 {
					tailData, tailName, tailErr := s.readCloudFile(ctx, project.OwnerID, tail.FileID)
					if tailErr != nil {
						return fmt.Errorf("read previous segment tail frame: %w", tailErr)
					}
					imageData, imageName, startFrame = tailData, tailName, tail
					width, height = videoDimensionsForSettings(imageData, videoSettings)
				}
			}
		}
		s.appendTaskPromptLog(task, targetTitle+" start frame", fmt.Sprintf("%s (file_id=%d)", startFrame.Label, startFrame.FileID))
		var lastFrameData []byte
		lastFrameName := ""
		if target.Segment != nil && videoSettings.Model == "minimax_h3" && videoSettings.H3Mode == "fl2va" {
			if next := adjacentSegmentTarget(target.Segment.ID, storyboards, segmentsByStoryboard, 1); next.Segment != nil {
				nextAssets := filterAssetsForSegment(assets, next.Segment)
				endFrame := selectSegmentPlannedFrame(next.Storyboard, next.Segment, mediaByStoryboard[next.Storyboard.ID], nextAssets)
				if endFrame.FileID != 0 {
					lastFrameData, lastFrameName, err = s.readCloudFile(ctx, project.OwnerID, endFrame.FileID)
					if err != nil {
						return fmt.Errorf("read H3 FL2VA last frame: %w", err)
					}
					prompt += "\nUse the supplied last image as the exact final frame. Preserve identity, wardrobe, props, screen direction, and scene geometry while creating one physically plausible continuous motion path from the first frame to the last frame."
					s.appendTaskPromptLog(task, targetTitle+" last frame", fmt.Sprintf("%s (file_id=%d)", endFrame.Label, endFrame.FileID))
				}
			}
		}
		s.appendTaskPromptLog(task, targetTitle, prompt)
		r2vReferences := make([]ComfyReferenceImage, 0)
		if videoSettings.Model == "minimax_h3" && videoSettings.H3Mode == "r2v" {
			for _, asset := range relevantAssets {
				if asset.ReferenceFileID == 0 || asset.ReferenceFileID == startFrame.FileID {
					continue
				}
				refData, refName, refErr := s.readCloudFile(ctx, project.OwnerID, asset.ReferenceFileID)
				if refErr != nil {
					return fmt.Errorf("读取 R2V 资产参考图失败（%s）：%w", asset.Name, refErr)
				}
				r2vReferences = append(r2vReferences, ComfyReferenceImage{Name: refName, Data: refData, Kind: asset.Type})
				if len(r2vReferences) >= 8 {
					break
				}
			}
		}

		baseProgress := index * 90 / len(targets)
		span := 90 / len(targets)
		update(5+baseProgress, fmt.Sprintf("Generating video %d/%d: %s", index+1, len(targets), targetTitle))
		data, filename, err := client.GenerateVideoBetweenFramesSizedWithReferences(ctx, imageData, imageName, lastFrameData, lastFrameName, prompt, negative, durationSec, width, height, videoSettings, r2vReferences, func(localProgress int, message string) {
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
		if target.Segment != nil && videoSettings.Continuity {
			tailData, tailErr := extractVideoTailFrame(ctx, data)
			if tailErr != nil {
				return fmt.Errorf("extract %s tail frame: %w", targetTitle, tailErr)
			}
			tailID, tailErr := s.saveGeneratedImage(project.OwnerID, project.ID, project.Title, "images", fmt.Sprintf("storyboard-%03d-segment-%02d-tail", storyboard.Seq, target.Segment.Seq), "tail.png", tailData)
			if tailErr != nil {
				return tailErr
			}
			tailSortOrder, tailErr := s.repo.NextStoryboardMediaSort(project.OwnerID, project.ID, storyboard.ID)
			if tailErr != nil {
				return tailErr
			}
			tailMedia := model.DramaStoryboardMedia{
				ProjectID: project.ID, StoryboardID: storyboard.ID, SegmentID: target.Segment.ID, OwnerID: project.OwnerID,
				Kind: "image", FileID: tailID, Source: "video_tail", Prompt: "Automatically extracted final video frame", SortOrder: tailSortOrder,
			}
			if tailErr = s.repo.CreateStoryboardMedia(&tailMedia); tailErr != nil {
				return tailErr
			}
			mediaByStoryboard[storyboard.ID] = append(mediaByStoryboard[storyboard.ID], tailMedia)
			s.appendTaskPromptLog(task, targetTitle+" extracted tail frame", fmt.Sprintf("file_id=%d", tailID))
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
	visibleState := plot
	if segment != nil {
		segmentTitle = strings.TrimSpace(segment.Title)
		compositionPrompt = strings.TrimSpace(segment.CompositionPrompt)
		if strings.TrimSpace(segment.ReferencePrompt) != "" {
			basePrompt = strings.TrimSpace(segment.ReferencePrompt)
			// A segment reference prompt describes the final static frame. Once it
			// exists, do not mix the action timeline (or the video prompt) back into
			// the image request; doing so turns a still-image prompt into motion.
			visibleState = strings.TrimSpace(segment.ReferencePrompt)
		} else {
			visibleState = strings.TrimSpace(segment.Action)
		}
		if strings.TrimSpace(segment.Scene) != "" {
			scene = strings.TrimSpace(segment.Scene)
		}
	}
	relevantAssets := assets
	if segment != nil {
		relevantAssets = filterAssetsForSegment(assets, segment)
	}
	characters := make([]string, 0)
	characterNames := make([]string, 0)
	characterLabels := make([]string, 0)
	scenes := make([]string, 0)
	for _, asset := range relevantAssets {
		name := strings.TrimSpace(asset.Name)
		description := compactText(storyboardAssetPrompt(asset), 140)
		if name == "" && description == "" {
			continue
		}
		line := strings.TrimSpace(name + ": " + description)
		if asset.ReferenceFileID != 0 {
			line += fmt.Sprintf(" (approved reference image file_id=%d, keep identity and styling consistent)", asset.ReferenceFileID)
		}
		if asset.Type == "character" {
			characters = append(characters, line)
			characterNames = append(characterNames, name)
			characterLabels = append(characterLabels, characterSemanticLabel(name, storyboardAssetPrompt(asset)))
		} else if asset.Type == "scene" {
			scenes = append(scenes, line)
		}
	}
	parts := []string{
		"Mandatory frame constraints: " + mandatoryStoryboardConstraints(characterLabels, basePrompt+" "+scene),
		"Composition priority: " + compactText(compositionPrompt, 260),
		projectVisualStyleDirective(project, basePrompt+" "+compositionPrompt) + ", wide horizontal 16:9, coherent lighting, clear subjects and details.",
		"Visible static frame state: " + compactText(visibleState, 380),
		"Scene: " + compactText(scene, 180),
		"Visual details: " + compactText(basePrompt, 380),
		"Continuity rule: preserve identities, age, hairstyle, clothing, body shape, props, and room layout from approved references. References control identity and environment only; the composition priority controls pose and framing.",
		"Shot: " + strings.TrimSpace(storyboard.Title) + " / " + segmentTitle,
	}
	if len(characters) > 0 {
		parts = append(parts, "Characters: "+compactText(strings.Join(characters, "; "), 420))
		bindings := make([]string, 0, len(characterNames))
		for index, name := range characterNames {
			bindings = append(bindings, name+"="+characterRegionLabel(index, len(characterNames)))
		}
		parts = append(parts, "Character reference binding: "+strings.Join(bindings, ", ")+". Keep each identity inside its assigned region; never blend or swap faces between regions.")
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

func projectVisualStyleDirective(project *model.DramaProject, localPrompt string) string {
	styleText := localPrompt
	if project != nil {
		styleText = strings.Join([]string{
			project.Description,
			project.Settings,
			project.Preface,
			localPrompt,
		}, " ")
	}
	switch detectDramaVisualStyle(styleText) {
	case "anime":
		return "high-quality anime cinematic still frame, consistent character design, polished linework"
	case "3d":
		return "high-quality stylized 3D cinematic still frame, consistent character models and materials"
	case "illustration":
		return "high-quality illustrated cinematic still frame, consistent art direction and character design"
	case "realistic":
		return "photorealistic cinematic still frame, natural skin and material detail"
	default:
		return "high-quality cinematic still frame matching the approved reference style"
	}
}

func detectDramaVisualStyle(text string) string {
	lower := strings.ToLower(text)
	switch {
	case containsAny(lower, []string{"anime", "manga", "二次元", "动漫", "日漫", "赛璐璐"}):
		return "anime"
	case containsAny(lower, []string{"3d animation", "3d render", "三维动画", "3d动画", "皮克斯", "pixar"}):
		return "3d"
	case containsAny(lower, []string{"illustration", "watercolor", "comic", "插画", "水彩", "绘本", "漫画"}):
		return "illustration"
	case containsAny(lower, []string{"photorealistic", "realistic photo", "live action", "真人", "写实", "实拍", "摄影"}):
		return "realistic"
	default:
		return "reference"
	}
}

func mandatoryStoryboardConstraints(characterLabels []string, visualPrompt string) string {
	constraints := make([]string, 0, len(characterLabels)+3)
	if len(characterLabels) > 0 {
		constraints = append(constraints, fmt.Sprintf("exactly %d people/main characters visible in one coherent scene", len(characterLabels)))
		for index, label := range characterLabels {
			if !isSemanticCharacterLabel(label) {
				label = characterSemanticLabel(label, "")
			}
			constraints = append(constraints, fmt.Sprintf("%s on the %s", label, characterRegionLabel(index, len(characterLabels))))
		}
	}
	lower := strings.ToLower(visualPrompt)
	if strings.Contains(lower, "暖") || strings.Contains(lower, "warm") {
		constraints = append(constraints, "warm natural interior lighting")
	}
	if strings.Contains(lower, "明亮") || strings.Contains(lower, "充足") || strings.Contains(lower, "bright") {
		constraints = append(constraints, "bright high-key evenly lit image, faces clearly illuminated")
	}
	if strings.Contains(lower, "无遮挡") || strings.Contains(lower, "无前景") ||
		strings.Contains(lower, "unobstructed") || strings.Contains(lower, "no foreground") {
		constraints = append(constraints, "unobstructed view, no foreground object crossing any face")
	}
	return strings.Join(constraints, ", ")
}

func isSemanticCharacterLabel(label string) bool {
	switch strings.TrimSpace(strings.ToLower(label)) {
	case "character", "adult woman", "adult man", "elderly woman", "elderly man", "girl", "boy", "dog", "cat", "robot character":
		return true
	default:
		return false
	}
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
		characterOrder := make(map[string]int)
		for _, asset := range matched {
			if asset.Type == "character" {
				characterOrder[asset.Name] = strings.Index(characters, asset.Name)
			}
		}
		sort.SliceStable(matched, func(i, j int) bool {
			if matched[i].Type != matched[j].Type {
				return matched[i].Type == "character"
			}
			if matched[i].Type != "character" {
				return false
			}
			left := characterRegionRank(segment, matched[i].Name, characterOrder[matched[i].Name])
			right := characterRegionRank(segment, matched[j].Name, characterOrder[matched[j].Name])
			if left < 0 {
				left = len(characters)
			}
			if right < 0 {
				right = len(characters)
			}
			return left < right
		})
		return matched
	}
	return assets
}

func characterRegionRank(segment *model.DramaStoryboardSegment, name string, fallback int) int {
	texts := []string{
		segment.ReferencePrompt,
		segment.CompositionPrompt,
		segment.Action,
		segment.Shot,
	}
	for _, text := range texts {
		if position, ok := characterSpatialPosition(text, name); ok {
			return position*1000 + fallback
		}
	}
	return 1000 + fallback
}

func characterSpatialPosition(text, name string) (int, bool) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return 0, false
	}
	clauses := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		switch r {
		case ',', ';', '，', '；', '。', '\n', '\r':
			return true
		default:
			return false
		}
	})
	for _, clause := range clauses {
		if !strings.Contains(clause, name) {
			continue
		}
		hasLeft := strings.Contains(clause, "左侧") || strings.Contains(clause, "左边") ||
			strings.Contains(clause, "left side") || strings.Contains(clause, "on the left")
		hasCenter := strings.Contains(clause, "中央") || strings.Contains(clause, "中间") ||
			strings.Contains(clause, "center")
		hasRight := strings.Contains(clause, "右侧") || strings.Contains(clause, "右边") ||
			strings.Contains(clause, "right side") || strings.Contains(clause, "on the right")
		switch {
		case hasLeft && !hasCenter && !hasRight:
			return 0, true
		case hasCenter && !hasLeft && !hasRight:
			return 1, true
		case hasRight && !hasLeft && !hasCenter:
			return 2, true
		}
	}
	return 0, false
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
			return false
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
			Prompt: storyboardAssetPrompt(asset),
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

func characterRegionLabel(index, count int) string {
	if count <= 1 {
		return "full frame"
	}
	if count == 2 {
		if index == 0 {
			return "left side"
		}
		return "right side"
	}
	if count == 3 {
		switch index {
		case 0:
			return "left side"
		case 1:
			return "center"
		default:
			return "right side"
		}
	}
	return fmt.Sprintf("horizontal region %d of %d", index+1, count)
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
		if item.Kind == "image" && item.Source != "video_tail" && item.FileID != 0 && item.SegmentID == segment.ID {
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

func selectSegmentTailFrame(segmentID uint64, media []model.DramaStoryboardMedia) storyboardVideoStartFrame {
	for index := len(media) - 1; index >= 0; index-- {
		item := media[index]
		if item.SegmentID == segmentID && item.Kind == "image" && item.Source == "video_tail" && item.FileID != 0 {
			return storyboardVideoStartFrame{FileID: item.FileID, Label: "previous segment extracted tail frame"}
		}
	}
	return storyboardVideoStartFrame{}
}

// selectSegmentPlannedFrame deliberately excludes extracted video tails. It is
// used as the FL2VA destination, while tails are reserved for continuity starts.
func selectSegmentPlannedFrame(storyboard *model.DramaStoryboard, segment *model.DramaStoryboardSegment, media []model.DramaStoryboardMedia, assets []model.DramaAsset) storyboardVideoStartFrame {
	for index := len(media) - 1; index >= 0; index-- {
		item := media[index]
		if item.Kind == "image" && item.Source != "video_tail" && item.FileID != 0 && item.SegmentID == segment.ID {
			return storyboardVideoStartFrame{FileID: item.FileID, Label: fmt.Sprintf("segment %d planned image", segment.Seq)}
		}
	}
	if segment.ReferenceFileID != 0 {
		return storyboardVideoStartFrame{FileID: segment.ReferenceFileID, Label: fmt.Sprintf("segment %d reference image", segment.Seq)}
	}
	return storyboardVideoStartFrame{}
}

func adjacentSegmentTarget(segmentID uint64, storyboards []model.DramaStoryboard, segmentsByStoryboard map[uint64][]model.DramaStoryboardSegment, offset int) storyboardVideoTarget {
	ordered := make([]storyboardVideoTarget, 0)
	for storyboardIndex := range storyboards {
		storyboard := &storyboards[storyboardIndex]
		segments := segmentsByStoryboard[storyboard.ID]
		for segmentIndex := range segments {
			ordered = append(ordered, storyboardVideoTarget{Storyboard: storyboard, Segment: &segments[segmentIndex]})
		}
	}
	for index := range ordered {
		if ordered[index].Segment.ID == segmentID {
			targetIndex := index + offset
			if targetIndex >= 0 && targetIndex < len(ordered) {
				return ordered[targetIndex]
			}
			break
		}
	}
	return storyboardVideoTarget{}
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

func videoDimensionsForSettings(data []byte, settings VideoGenerationSettings) (int, int) {
	if settings.Model == "minimax_h3" {
		orientationWidth, orientationHeight := videoDimensionsFromImage(data)
		switch settings.SizePreset {
		case "landscape":
			orientationWidth, orientationHeight = 16, 9
		case "portrait":
			orientationWidth, orientationHeight = 9, 16
		case "square":
			orientationWidth, orientationHeight = 1, 1
		}
		return normalizeH3VideoDimensions(orientationWidth, orientationHeight, settings.QualityPreset)
	}
	switch settings.SizePreset {
	case "landscape":
		return 832, 480
	case "portrait":
		return 480, 832
	case "square":
		return 832, 832
	default:
		return videoDimensionsFromImage(data)
	}
}

func extractVideoTailFrame(ctx context.Context, video []byte) ([]byte, error) {
	input, err := os.CreateTemp("", "cloudnexus-drama-tail-*.mp4")
	if err != nil {
		return nil, err
	}
	inputName := input.Name()
	defer os.Remove(inputName)
	if _, err = input.Write(video); err != nil {
		input.Close()
		return nil, err
	}
	if err = input.Close(); err != nil {
		return nil, err
	}
	output, err := os.CreateTemp("", "cloudnexus-drama-tail-*.png")
	if err != nil {
		return nil, err
	}
	outputName := output.Name()
	output.Close()
	defer os.Remove(outputName)

	command := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-sseof", "-0.08", "-i", inputName, "-frames:v", "1", outputName)
	if output, commandErr := command.CombinedOutput(); commandErr != nil {
		return nil, fmt.Errorf("ffmpeg: %w (%s)", commandErr, strings.TrimSpace(string(output)))
	}
	frame, err := os.ReadFile(outputName)
	if err != nil {
		return nil, err
	}
	if len(frame) == 0 {
		return nil, fmt.Errorf("ffmpeg produced an empty tail frame")
	}
	return frame, nil
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

func buildAssetReferencePrompt(asset *model.DramaAsset, raw, styleHint string) string {
	if strings.TrimSpace(asset.ReferencePrompt) != "" {
		raw = asset.ReferencePrompt
	}
	referencePrompt := extractPromptField(raw, []string{"参考图提示词", "reference_prompt", "Reference prompt"})
	if referencePrompt == "" {
		referencePrompt = cleanPromptText(raw)
	}
	englishHints := inferAssetEnglishHints(asset, raw+" "+asset.Description+" "+referencePrompt)
	prefix := "high-quality reference image matching the requested visual style, clear subject, complete composition, consistent design"
	style := detectDramaVisualStyle(styleHint + " " + referencePrompt)
	if style == "realistic" {
		prefix = "photorealistic cinematic reference photo, clear subject, complete composition, high detail, natural materials"
	} else if style == "anime" {
		prefix = "high-quality anime character design reference, polished linework and coloring, consistent design"
	} else if style == "3d" {
		prefix = "high-quality stylized 3D character model reference, consistent materials and proportions"
	} else if style == "illustration" {
		prefix = "high-quality illustrated character reference, consistent art direction and proportions"
	}
	if asset.Type == "scene" {
		prefix += ", environment only, wide establishing view, clear spatial layout, coherent lighting, no foreground character"
	}
	ending := "no text, no watermark, no logo, no UI"
	if asset.Type == "scene" {
		ending += ", no people"
	}
	parts := []string{prefix, englishHints, referencePrompt, ending}
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, ", ")
}

func buildCharacterViewPrompts(asset *model.DramaAsset, raw, styleHint string) []string {
	if strings.TrimSpace(asset.ReferencePrompt) != "" {
		raw = asset.ReferencePrompt
	}
	appearance := extractPromptField(raw, []string{"参考图提示词", "reference_prompt", "Reference prompt"})
	if appearance == "" {
		appearance = cleanPromptText(raw)
	}
	appearance = characterAppearancePrompt(asset, appearance)
	style := detectDramaVisualStyle(styleHint + " " + appearance)
	stylePrefix := "photorealistic full-body studio character reference, natural skin and fabric texture"
	switch style {
	case "anime":
		stylePrefix = "polished anime full-body studio character reference, clean linework, flat consistent coloring"
	case "3d":
		stylePrefix = "high-quality stylized 3D full-body studio character reference, consistent materials"
	case "illustration":
		stylePrefix = "high-quality illustrated full-body studio character reference, consistent art direction"
	}
	common := strings.Join([]string{
		stylePrefix,
		"one single person, exactly one full-body depiction, centered",
		"head and both feet fully visible with comfortable margin",
		"neutral expression, closed mouth, relaxed A-pose, arms slightly away from torso, straight legs",
		"eye-level orthographic projection, no perspective distortion",
		"seamless light-gray studio background, flat even shadowless lighting",
		appearance,
		"no text, no watermark, no logo, no UI",
	}, ", ")
	return []string{
		"STRICT ORIENTATION: FRONT VIEW ONLY, exact 0-degree front-facing orthographic view, face torso knees and toes point directly toward camera, symmetrical shoulders, both eyes visible, " + common,
		"STRICT ORIENTATION: LEFT PROFILE ONLY, exact 90-degree left-facing orthographic side view, head torso hips knees and feet all point left, nose silhouette points left, exactly one eye visible, shoulders and hips seen from the side, " + common,
		"STRICT ORIENTATION: BACK VIEW ONLY, exact 180-degree rear-facing orthographic view, head torso hips knees and heels all point directly away from camera, back of head and rear of outfit visible, face and eyes completely hidden, " + common,
	}
}

func characterViewLabel(index int) string {
	labels := []string{"front view", "left profile view", "rear view"}
	if index < 0 || index >= len(labels) {
		return "view"
	}
	return labels[index]
}

func generateCharacterTurnaround(ctx context.Context, client *ComfyClient, settings ImageGenerationSettings, styleHint string, prompts []string, update func(int, string)) ([]byte, string, error) {
	if len(prompts) != 3 {
		return nil, "", fmt.Errorf("character turnaround requires exactly three view prompts")
	}
	viewSettings := characterViewReferenceSettings(settings, styleHint)
	viewSettings.Width = 512
	viewSettings.Height = 768
	views := make([][]byte, 0, len(prompts))
	for index, prompt := range prompts {
		label := characterViewLabel(index)
		currentSettings := viewSettings
		currentSettings.Seed = 0 // Each orientation needs independent composition noise.
		currentSettings.NegativePrompt = strings.TrimSpace(strings.Join([]string{currentSettings.NegativePrompt, characterViewOrientationNegative(index)}, ", "))
		update(15+index*22, fmt.Sprintf("Generating character %s %d/3", label, index+1))
		data, _, err := client.Generate(ctx, prompt, currentSettings, func(localProgress int, message string) {
			update(15+index*22+localProgress*20/100, fmt.Sprintf("Character %s: %s", label, message))
		})
		if err != nil {
			return nil, "", fmt.Errorf("generate character %s: %w", label, err)
		}
		views = append(views, data)
	}
	update(88, "Combining front, profile, and rear views")
	combined, err := stitchCharacterViews(views)
	if err != nil {
		return nil, "", err
	}
	return combined, "character-turnaround.png", nil
}

func characterViewOrientationNegative(index int) string {
	switch index {
	case 0:
		return "side view, profile view, three-quarter view, rear view, back view, turned body, asymmetrical shoulders"
	case 1:
		return "front view, front-facing, rear view, back view, three-quarter view, looking at camera, both eyes visible, chest facing camera, symmetrical shoulders"
	case 2:
		return "front view, front-facing, side view, profile view, three-quarter view, visible face, visible eyes, looking at camera, nose visible, chest visible, toes visible"
	default:
		return ""
	}
}

func stitchCharacterViews(encodedViews [][]byte) ([]byte, error) {
	if len(encodedViews) != 3 {
		return nil, fmt.Errorf("stitch character views: expected 3 images, got %d", len(encodedViews))
	}
	decoded := make([]image.Image, 0, len(encodedViews))
	panelWidth, panelHeight := 0, 0
	for index, encoded := range encodedViews {
		view, _, err := image.Decode(bytes.NewReader(encoded))
		if err != nil {
			return nil, fmt.Errorf("decode character %s: %w", characterViewLabel(index), err)
		}
		width, height := view.Bounds().Dx(), view.Bounds().Dy()
		if index == 0 {
			panelWidth, panelHeight = width, height
		} else if width != panelWidth || height != panelHeight {
			return nil, fmt.Errorf("stitch character views: inconsistent image sizes %dx%d and %dx%d", panelWidth, panelHeight, width, height)
		}
		decoded = append(decoded, view)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, panelWidth*3, panelHeight))
	for index, view := range decoded {
		target := image.Rect(index*panelWidth, 0, (index+1)*panelWidth, panelHeight)
		draw.Draw(canvas, target, view, view.Bounds().Min, draw.Src)
	}
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, fmt.Errorf("encode character turnaround: %w", err)
	}
	return output.Bytes(), nil
}

func assetReferenceSettings(settings ImageGenerationSettings, styleHint string) ImageGenerationSettings {
	styleNegative := "text, watermark, logo, UI, cropped subject"
	switch detectDramaVisualStyle(styleHint) {
	case "realistic":
		styleNegative += ", anime, illustration, painting, sketch, plastic skin, doll"
	case "anime":
		styleNegative += ", photorealistic, live action, 3d render"
	case "3d":
		styleNegative += ", flat 2d drawing, photorealistic live action"
	}
	settings.NegativePrompt = strings.TrimSpace(strings.Join([]string{settings.NegativePrompt, styleNegative}, ", "))
	return settings
}

func characterViewReferenceSettings(settings ImageGenerationSettings, styleHint string) ImageGenerationSettings {
	settings = assetReferenceSettings(settings, styleHint)
	viewNegative := "multiple people, two people, three people, duplicate person, repeated person, extra body, extra head, extra limbs, collage, triptych, contact sheet, model sheet, split screen, multiple panels, panel border, cropped head, cropped feet, close-up, bust shot, action pose, bent legs, props, scenery, labels, annotations"
	settings.NegativePrompt = strings.TrimSpace(strings.Join([]string{settings.NegativePrompt, viewNegative}, ", "))
	return settings
}

func characterAppearancePrompt(asset *model.DramaAsset, referencePrompt string) string {
	referencePrompt = strings.TrimSpace(referencePrompt)
	lower := strings.ToLower(referencePrompt)
	if strings.Contains(lower, "character turnaround") || strings.Contains(lower, "model sheet") || strings.Contains(lower, "triptych") || strings.Contains(lower, "contact sheet") || strings.Contains(lower, "three views") || strings.Contains(lower, "exactly 3 panels") || strings.Contains(lower, "exactly three full-body") {
		// The layout portion is regenerated canonically above. Asset Description
		// contains the stable age/face/hair/clothing fields and is safer than
		// attempting to preserve arbitrary, possibly contradictory sheet syntax.
		referencePrompt = ""
	}
	if referencePrompt != "" {
		return compactText(referencePrompt, 700)
	}
	return compactText(characterVisualDescription(asset.Description), 700)
}

func characterVisualDescription(description string) string {
	visualLabels := []string{"年龄", "外貌", "服装", "age", "appearance", "clothing"}
	lines := strings.Split(strings.ReplaceAll(description, "；", "\n"), "\n")
	visual := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		for _, label := range visualLabels {
			if strings.HasPrefix(lower, strings.ToLower(label)+"：") || strings.HasPrefix(lower, strings.ToLower(label)+":") {
				visual = append(visual, line)
				break
			}
		}
	}
	if len(visual) > 0 {
		return strings.Join(visual, ", ")
	}
	return strings.TrimSpace(description)
}

func storyboardAssetPrompt(asset model.DramaAsset) string {
	if asset.Type == "character" {
		// Never leak the character-sheet layout into a cinematic storyboard.
		return compactText(asset.Description, 320)
	}
	return compactText(strings.TrimSpace(asset.Description+" "+asset.ReferencePrompt), 360)
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
