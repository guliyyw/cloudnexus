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
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudnexus/server/pkg/model"
	_ "golang.org/x/image/webp"
)

const imageGenerationProjectID uint64 = 0

type ImageGenerationInput struct {
	Prompt          string
	NegativePrompt  string
	Width           int
	Height          int
	Steps           int
	CFG             float64
	ImageCount      int
	ReferenceWeight float64
	References      []*multipart.FileHeader
}

func (s *DramaService) ListImageGenerationTasks(ownerID uint64) ([]model.DramaTask, error) {
	return s.repo.ListTasks(ownerID, imageGenerationProjectID)
}

func (s *DramaService) CreateImageGenerationTask(ownerID uint64, input ImageGenerationInput) (*model.DramaTask, error) {
	prompt := strings.TrimSpace(input.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("请输入图片提示词")
	}
	if input.ImageCount < 1 {
		input.ImageCount = 1
	}
	if input.ImageCount > 4 {
		input.ImageCount = 4
	}
	if len(input.References) > 4 {
		return nil, fmt.Errorf("最多上传 4 张参考图")
	}

	referenceIDs := make([]string, 0, len(input.References))
	for index, header := range input.References {
		fileID, sourceWidth, sourceHeight, err := s.saveImageGenerationReference(ownerID, header)
		if err != nil {
			return nil, err
		}
		if index == 0 && sourceWidth > 0 && sourceHeight > 0 {
			input.Width, input.Height = fitReferenceDimensions(sourceWidth, sourceHeight)
		}
		referenceIDs = append(referenceIDs, strconv.FormatUint(fileID, 10))
	}

	payload, _ := json.Marshal(generationTaskPayload{
		Prompt: prompt, NegativePrompt: strings.TrimSpace(input.NegativePrompt),
		Width: input.Width, Height: input.Height, Steps: input.Steps, CFG: input.CFG,
		ImageCount: input.ImageCount, ReferenceFileIDs: referenceIDs, ReferenceWeight: input.ReferenceWeight,
	})
	task := &model.DramaTask{
		ProjectID: imageGenerationProjectID, OwnerID: ownerID, Type: "image_generation",
		Status: "pending", Progress: 0, Message: "任务已创建，等待执行", Payload: string(payload),
	}
	if err := s.repo.CreateTask(task); err != nil {
		return nil, err
	}
	if s.taskRunner != nil {
		if err := s.taskRunner.Enqueue(task.ID); err != nil {
			now := time.Now()
			task.Status, task.Message, task.FinishedAt = "failed", "任务入队失败："+err.Error(), &now
			_ = s.repo.UpdateTask(task)
			return task, nil
		}
		s.taskRunner.Publish(*task)
	}
	return task, nil
}

func (s *DramaService) CancelImageGenerationTask(ownerID, taskID uint64) (*model.DramaTask, error) {
	return s.CancelTask(ownerID, imageGenerationProjectID, taskID)
}

func (s *DramaService) RetryImageGenerationTask(ownerID, taskID uint64) (*model.DramaTask, error) {
	return s.RetryTask(ownerID, imageGenerationProjectID, taskID)
}

func (s *DramaService) executeStandaloneImageTask(ctx context.Context, task *model.DramaTask, update func(int, string)) error {
	setting, err := s.repo.GetSetting(task.OwnerID)
	if err != nil {
		return err
	}
	client := NewComfyClient(setting.ComfyUIURL)
	status := client.Status(ctx)
	if !status.Connected {
		return fmt.Errorf("ComfyUI 不可用：%s", status.Error)
	}
	settings := defaultImageGenerationSettings(setting.ImageSettings)
	if settings.Checkpoint == "" && len(status.Checkpoints) > 0 {
		settings.Checkpoint = status.Checkpoints[0]
	}
	if settings.Checkpoint == "" {
		return fmt.Errorf("ComfyUI 未检测到可用的 Checkpoint 模型")
	}

	var payload generationTaskPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("生成参数无效")
	}
	if payload.Width >= 256 && payload.Width <= 2048 {
		settings.Width = payload.Width
	}
	if payload.Height >= 256 && payload.Height <= 2048 {
		settings.Height = payload.Height
	}
	if payload.Steps >= 1 && payload.Steps <= 100 {
		settings.Steps = payload.Steps
	}
	if payload.CFG > 0 && payload.CFG <= 30 {
		settings.CFG = payload.CFG
	}
	if strings.TrimSpace(payload.NegativePrompt) != "" {
		settings.NegativePrompt = strings.TrimSpace(settings.NegativePrompt + ", " + payload.NegativePrompt)
	}

	references := make([]ComfyReferenceImage, 0, len(payload.ReferenceFileIDs))
	for _, rawID := range payload.ReferenceFileIDs {
		fileID, parseErr := strconv.ParseUint(rawID, 10, 64)
		if parseErr != nil || fileID == 0 {
			continue
		}
		data, name, readErr := s.readCloudFile(ctx, task.OwnerID, fileID)
		if readErr != nil {
			return readErr
		}
		weight := payload.ReferenceWeight
		if weight <= 0 || weight > 1.5 {
			weight = 0.65
		}
		references = append(references, ComfyReferenceImage{Name: name, Data: data, Kind: "reference", Weight: weight})
	}
	generationPrompt := buildStandaloneImagePrompt(payload.Prompt, len(references) > 0)
	s.appendTaskPromptLog(task, "图片生成", generationPrompt)

	count := payload.ImageCount
	if count < 1 {
		count = 1
	}
	if count > 4 {
		count = 4
	}
	for i := 0; i < count; i++ {
		base := 10 + (i * 80 / count)
		progress := func(value int, message string) {
			update(base+(value*80/count/100), message)
		}
		var data []byte
		var filename string
		if len(references) > 0 {
			data, filename, err = client.GenerateWithReferences(ctx, generationPrompt, settings, references, progress)
		} else {
			data, filename, err = client.Generate(ctx, generationPrompt, settings, progress)
		}
		if err != nil {
			return err
		}
		fileID, saveErr := s.saveGeneratedImage(task.OwnerID, imageGenerationProjectID, "图片生成", "images", fmt.Sprintf("生成图片-%d", i+1), filename, data)
		if saveErr != nil {
			return saveErr
		}
		s.appendTaskResult(task, generationTaskResult{Kind: "image_generation", FileID: strconv.FormatUint(fileID, 10), Title: fmt.Sprintf("生成图片 %d", i+1), Prompt: payload.Prompt})
	}
	return nil
}

func (s *DramaService) saveImageGenerationReference(ownerID uint64, header *multipart.FileHeader) (uint64, int, int, error) {
	if header == nil || header.Size <= 0 {
		return 0, 0, 0, fmt.Errorf("参考图片无效")
	}
	if header.Size > 10<<20 {
		return 0, 0, 0, fmt.Errorf("单张参考图不能超过 10 MB")
	}
	file, err := header.Open()
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (10<<20)+1))
	if err != nil {
		return 0, 0, 0, err
	}
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return 0, 0, 0, fmt.Errorf("参考文件必须是图片")
	}
	config, _, decodeErr := image.DecodeConfig(bytes.NewReader(data))
	if decodeErr != nil {
		return 0, 0, 0, fmt.Errorf("无法读取参考图片：%w", decodeErr)
	}
	fileID, err := s.saveGeneratedFile(ownerID, imageGenerationProjectID, "图片生成", "assets", "参考图", header.Filename, data, contentType)
	return fileID, config.Width, config.Height, err
}

func fitReferenceDimensions(sourceWidth, sourceHeight int) (int, int) {
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return 1024, 1024
	}
	ratio := float64(sourceWidth) / float64(sourceHeight)
	var width, height int
	if ratio >= 1 {
		width = 1024
		if ratio >= 1.45 {
			width = 1344
		}
		height = roundToMultiple(float64(width)/ratio, 64)
	} else {
		height = 1024
		if ratio <= 0.69 {
			height = 1344
		}
		width = roundToMultiple(float64(height)*ratio, 64)
	}
	if width < 64 {
		width = 64
	}
	if height < 64 {
		height = 64
	}
	if width > 1536 {
		width = 1536
	}
	if height > 1536 {
		height = 1536
	}
	return width, height
}

func roundToMultiple(value float64, multiple int) int {
	return int(value/float64(multiple)+0.5) * multiple
}

func buildStandaloneImagePrompt(raw string, hasReference bool) string {
	parts := []string{
		"high quality image, highly detailed, coherent composition, professional lighting, clean visual structure",
		"Follow the user prompt precisely. Preserve the requested subject count, appearance, environment, camera angle, style, and mood.",
		"User prompt: " + compactText(raw, 1800),
	}
	if hasReference {
		parts = append(parts,
			"Reference lock: use the first reference image as the strict composition and content foundation. Preserve its aspect ratio, subject identity, silhouette, spatial layout, camera angle, dominant colors, and recognizable visual features. Apply the user prompt as a controlled edit, not as an unrelated redesign.",
			"Additional reference images provide secondary identity, material, and style guidance.",
		)
	}
	parts = append(parts, "single complete image, no text, no watermark, no logo, no UI, no collage")
	return strings.Join(parts, "\n")
}
