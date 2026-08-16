package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

const defaultComfyUIURL = "http://comfyui:8188"

type ComfyStatus struct {
	Connected   bool            `json:"connected"`
	URL         string          `json:"url"`
	Checkpoints []string        `json:"checkpoints"`
	IPAdapter   bool            `json:"ip_adapter"`
	ReActor     bool            `json:"reactor"`
	Models      map[string]bool `json:"models"`
	Missing     []string        `json:"missing"`
	Error       string          `json:"error,omitempty"`
}

type ImageGenerationSettings struct {
	Checkpoint     string  `json:"checkpoint"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Steps          int     `json:"steps"`
	CFG            float64 `json:"cfg"`
	Sampler        string  `json:"sampler"`
	Scheduler      string  `json:"scheduler"`
	NegativePrompt string  `json:"negative_prompt"`
	UseFaceID      bool    `json:"-"`
	Seed           int64   `json:"-"`
}

// VideoGenerationSettings is persisted in DramaSetting.VideoSettings.  The
// JSON field is kept for backwards compatibility with existing installations.
type VideoGenerationSettings struct {
	Model          string `json:"model"`
	H3Mode         string `json:"h3_mode"`
	AudioMode      string `json:"audio_mode"`
	H3RefImageSize string `json:"h3_ref_image_size"`
	QualityPreset  string `json:"quality_preset"`
	SizePreset     string `json:"size_preset"`
	Continuity     bool   `json:"continuity"`
}

func defaultVideoGenerationSettings(raw string) VideoGenerationSettings {
	settings := VideoGenerationSettings{Model: "wan22", H3Mode: "i2v", AudioMode: "external", H3RefImageSize: "match", QualityPreset: "standard", SizePreset: "auto", Continuity: true}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &settings)
	}
	if settings.Model != "minimax_h3" && settings.Model != "wan22" {
		settings.Model = "wan22"
	}
	if settings.H3Mode != "i2v" && settings.H3Mode != "fl2va" && settings.H3Mode != "r2v" {
		settings.H3Mode = "i2v"
	}
	if settings.AudioMode != "native" && settings.AudioMode != "external" && settings.AudioMode != "none" {
		settings.AudioMode = "external"
	}
	if settings.H3RefImageSize != "max" {
		settings.H3RefImageSize = "match"
	}
	settings.QualityPreset = normalizeVideoQualityPreset(settings.QualityPreset)
	if settings.SizePreset != "landscape" && settings.SizePreset != "portrait" && settings.SizePreset != "square" {
		settings.SizePreset = "auto"
	}
	return settings
}

type ComfyImage struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

type ComfyVideo struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

type ComfyReferenceImage struct {
	Name   string
	Data   []byte
	Kind   string
	Weight float64
	Prompt string
}

type ComfyClient struct {
	baseURL string
	http    *http.Client
	media   *http.Client
}

func NewComfyClient(rawURL string) *ComfyClient {
	rawURL = strings.TrimRight(strings.TrimSpace(rawURL), "/")
	if rawURL == "" {
		rawURL = defaultComfyUIURL
	}
	return &ComfyClient{
		baseURL: rawURL,
		http:    &http.Client{Timeout: 5 * time.Second},
		media:   &http.Client{Timeout: 10 * time.Minute},
	}
}

func (c *ComfyClient) Status(ctx context.Context) ComfyStatus {
	status := ComfyStatus{URL: c.baseURL}
	if err := c.getJSON(ctx, "/system_stats", nil); err != nil {
		status.Error = err.Error()
		status.Missing = []string{"ComfyUI 服务连接"}
		return status
	}
	status.Connected = true
	status.Models = make(map[string]bool)

	var objectInfo map[string]interface{}
	if err := c.getJSON(ctx, "/object_info", &objectInfo); err != nil {
		status.Error = "读取节点信息失败：" + err.Error()
		status.Missing = []string{"节点信息"}
		return status
	}
	status.Checkpoints = extractCheckpoints(objectInfo)
	status.Models["photorealistic_sdxl"] = checkpointNameContains(status.Checkpoints, "realvisxl", "juggernaut")
	status.Models["clip_vision_sdxl"] = objectOptionContains(objectInfo, "CLIPVisionLoader", "clip_name", "CLIP-ViT-H-14-laion2B-s32B-b79K.safetensors")
	status.Models["ipadapter_plus_sdxl"] = objectOptionContains(objectInfo, "IPAdapterModelLoader", "ipadapter_file", "ip-adapter-plus_sdxl_vit-h.safetensors")
	status.Models["ipadapter_plus_face_sdxl"] = objectOptionContains(objectInfo, "IPAdapterModelLoader", "ipadapter_file", "ip-adapter-plus-face_sdxl_vit-h.safetensors")
	status.Models["ipadapter_faceid_plusv2_sdxl"] = objectOptionContains(objectInfo, "IPAdapterModelLoader", "ipadapter_file", "ip-adapter-faceid-plusv2_sdxl.bin")
	status.Models["ipadapter_faceid_lora_sdxl"] = objectOptionContains(objectInfo, "LoraLoader", "lora_name", "ip-adapter-faceid-plusv2_sdxl_lora.safetensors")
	status.Models["faceid_nodes"] = objectInfoHasNodes(objectInfo, "IPAdapterUnifiedLoaderFaceID", "IPAdapterFaceID")
	status.Models["regional_ipadapter_nodes"] = objectInfoHasNodes(objectInfo,
		"IPAdapterAdvanced", "IPAdapterUnifiedLoader", "SolidMask", "MaskComposite", "FeatherMask")
	status.Models["wan22_high_noise"] = objectOptionContains(objectInfo, "UNETLoader", "unet_name", "wan2.2_i2v_high_noise_14B_fp8_scaled.safetensors")
	status.Models["wan22_low_noise"] = objectOptionContains(objectInfo, "UNETLoader", "unet_name", "wan2.2_i2v_low_noise_14B_fp8_scaled.safetensors")
	status.Models["wan22_text_encoder"] = objectOptionContains(objectInfo, "CLIPLoader", "clip_name", "umt5_xxl_fp8_e4m3fn_scaled.safetensors")
	status.Models["wan_vae"] = objectOptionContains(objectInfo, "VAELoader", "vae_name", "wan_2.1_vae.safetensors")
	status.Models["minimax_h3_nodes"] = objectInfoHasNodes(objectInfo, "MiniMaxH3ImageToVideo", "MiniMaxH3ReferenceToVideo", "MiniMaxH3SigmaShift")
	status.Models["minimax_h3_fl2va"] = objectOptionContains(objectInfo, "UNETLoader", "unet_name", "minimax_h3_fl2va_pruned_int8_convrot.safetensors")
	status.Models["minimax_h3_ref2va"] = objectOptionContains(objectInfo, "UNETLoader", "unet_name", "minimax_h3_ref2va_pruned_int8_convrot.safetensors")
	status.Models["minimax_h3_text_encoder"] = objectOptionContains(objectInfo, "CLIPLoader", "clip_name", "qwen3vl_32b_minimax_h3_nvfp4_awq.safetensors")
	status.Models["minimax_h3_video_vae"] = objectOptionContains(objectInfo, "VAELoader", "vae_name", "minimax_h3_video_vae_fp16.safetensors")
	status.Models["minimax_h3_audio_vae"] = objectOptionContains(objectInfo, "VAELoader", "vae_name", "minimax_h3_audio_vae_fp32.safetensors")
	for name := range objectInfo {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "ipadapter") || strings.Contains(lower, "ip_adapter") {
			status.IPAdapter = true
		}
		if strings.Contains(lower, "reactor") {
			status.ReActor = true
		}
	}
	if len(status.Checkpoints) == 0 {
		status.Missing = append(status.Missing, "Checkpoint 模型")
	}
	if !status.IPAdapter {
		status.Missing = append(status.Missing, "IP-Adapter 节点（角色一致性阶段使用）")
	}
	requiredModels := map[string]string{
		"clip_vision_sdxl":         "CLIP-ViT-H-14-laion2B-s32B-b79K.safetensors",
		"ipadapter_plus_sdxl":      "ip-adapter-plus_sdxl_vit-h.safetensors",
		"ipadapter_plus_face_sdxl": "ip-adapter-plus-face_sdxl_vit-h.safetensors",
		"regional_ipadapter_nodes": "IPAdapter regional mask nodes",
		"wan22_high_noise":         "wan2.2_i2v_high_noise_14B_fp8_scaled.safetensors",
		"wan22_low_noise":          "wan2.2_i2v_low_noise_14B_fp8_scaled.safetensors",
		"wan22_text_encoder":       "umt5_xxl_fp8_e4m3fn_scaled.safetensors",
		"wan_vae":                  "wan_2.1_vae.safetensors",
	}
	for key, label := range requiredModels {
		if !status.Models[key] {
			status.Missing = append(status.Missing, label)
		}
	}
	if status.Models["minimax_h3_nodes"] {
		for key, label := range map[string]string{
			"minimax_h3_fl2va":        "minimax_h3_fl2va_pruned_int8_convrot.safetensors",
			"minimax_h3_ref2va":       "minimax_h3_ref2va_pruned_int8_convrot.safetensors",
			"minimax_h3_text_encoder": "qwen3vl_32b_minimax_h3_nvfp4_awq.safetensors",
			"minimax_h3_video_vae":    "minimax_h3_video_vae_fp16.safetensors",
			"minimax_h3_audio_vae":    "minimax_h3_audio_vae_fp32.safetensors",
		} {
			if !status.Models[key] {
				status.Missing = append(status.Missing, label)
			}
		}
	} else {
		status.Missing = append(status.Missing, "MiniMax H3 ComfyUI 节点（需 ComfyUI 0.30.0+）")
	}
	return status
}

func objectInfoHasNodes(objectInfo map[string]interface{}, names ...string) bool {
	for _, name := range names {
		if _, ok := objectInfo[name]; !ok {
			return false
		}
	}
	return true
}

func (c *ComfyClient) Generate(ctx context.Context, prompt string, settings ImageGenerationSettings, progress func(int, string)) ([]byte, string, error) {
	workflow := systemTextToImageWorkflow(prompt, settings)
	return c.generateWithWorkflow(ctx, workflow, progress)
}

func (c *ComfyClient) GenerateWithReferences(ctx context.Context, prompt string, settings ImageGenerationSettings, references []ComfyReferenceImage, progress func(int, string)) ([]byte, string, error) {
	uploaded := make([]string, 0, len(references))
	for index, reference := range references {
		if len(reference.Data) == 0 {
			continue
		}
		name := reference.Name
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("reference-%d.png", index+1)
		}
		uploadedName, err := c.UploadImage(ctx, reference.Data, name)
		if err != nil {
			return nil, "", fmt.Errorf("上传参考图到 ComfyUI 失败：%w", err)
		}
		uploaded = append(uploaded, uploadedName)
	}
	if len(uploaded) == 0 {
		return c.Generate(ctx, prompt, settings, progress)
	}
	workflow := systemTextToImageIPAdapterWorkflow(prompt, settings, uploaded, references)
	data, filename, err := c.generateWithWorkflow(ctx, workflow, progress)
	if err == nil || !settings.UseFaceID || !isFaceIDNoFaceError(err.Error()) {
		return data, filename, err
	}

	// FaceID is an optional identity-strengthening path. A generated character
	// reference may be a full-body shot, have an occluded/small face, or depict a
	// non-human character. InsightFace treats all of those as fatal, so retry the
	// same prompt and uploaded references with the regular portrait IPAdapter.
	// This preserves character appearance without failing the entire storyboard.
	progress(20, "FaceID 未在角色参考图中检测到清晰人脸，正在使用普通角色参考模式重试")
	fallbackSettings := settings
	fallbackSettings.UseFaceID = false
	fallbackWorkflow := systemTextToImageIPAdapterWorkflow(prompt, fallbackSettings, uploaded, references)
	return c.generateWithWorkflow(ctx, fallbackWorkflow, progress)
}

func isFaceIDNoFaceError(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "insightface") &&
		(strings.Contains(lower, "no face detected") || strings.Contains(lower, "no face found"))
}

func (c *ComfyClient) GenerateVideoFromImage(ctx context.Context, image []byte, imageName, prompt, negativePrompt string, durationSec int, progress func(int, string)) ([]byte, string, error) {
	return c.GenerateVideoFromImageSized(ctx, image, imageName, prompt, negativePrompt, durationSec, 832, 480, progress)
}

func (c *ComfyClient) GenerateVideoFromImageSized(ctx context.Context, image []byte, imageName, prompt, negativePrompt string, durationSec, width, height int, progress func(int, string)) ([]byte, string, error) {
	return c.GenerateVideoFromImageSizedWithSettings(ctx, image, imageName, prompt, negativePrompt, durationSec, width, height, defaultVideoGenerationSettings(""), progress)
}

func (c *ComfyClient) GenerateVideoFromImageSizedWithSettings(ctx context.Context, image []byte, imageName, prompt, negativePrompt string, durationSec, width, height int, settings VideoGenerationSettings, progress func(int, string)) ([]byte, string, error) {
	return c.GenerateVideoFromImageSizedWithReferences(ctx, image, imageName, prompt, negativePrompt, durationSec, width, height, settings, nil, progress)
}

func (c *ComfyClient) GenerateVideoFromImageSizedWithReferences(ctx context.Context, image []byte, imageName, prompt, negativePrompt string, durationSec, width, height int, settings VideoGenerationSettings, references []ComfyReferenceImage, progress func(int, string)) ([]byte, string, error) {
	return c.GenerateVideoBetweenFramesSizedWithReferences(ctx, image, imageName, nil, "", prompt, negativePrompt, durationSec, width, height, settings, references, progress)
}

// GenerateVideoBetweenFramesSizedWithReferences optionally constrains the H3
// FL2VA output with a final frame. Other modes safely ignore the final frame.
func (c *ComfyClient) GenerateVideoBetweenFramesSizedWithReferences(ctx context.Context, image []byte, imageName string, lastImage []byte, lastImageName, prompt, negativePrompt string, durationSec, width, height int, settings VideoGenerationSettings, references []ComfyReferenceImage, progress func(int, string)) ([]byte, string, error) {
	uploadedName, err := c.UploadImage(ctx, image, imageName)
	if err != nil {
		return nil, "", fmt.Errorf("上传视频首帧到 ComfyUI 失败：%w", err)
	}
	uploadedReferences := []string{uploadedName}
	if settings.Model == "minimax_h3" && settings.H3Mode == "r2v" {
		for index, reference := range references {
			if len(uploadedReferences) >= 9 || len(reference.Data) == 0 {
				break
			}
			name := reference.Name
			if strings.TrimSpace(name) == "" {
				name = fmt.Sprintf("h3-reference-%d.png", index+1)
			}
			uploadedReference, uploadErr := c.UploadImage(ctx, reference.Data, name)
			if uploadErr != nil {
				return nil, "", fmt.Errorf("上传 H3 R2V 参考图失败：%w", uploadErr)
			}
			uploadedReferences = append(uploadedReferences, uploadedReference)
		}
	}
	uploadedLastName := ""
	if settings.Model == "minimax_h3" && settings.H3Mode == "fl2va" && len(lastImage) > 0 {
		if strings.TrimSpace(lastImageName) == "" {
			lastImageName = "h3-last-frame.png"
		}
		uploadedLastName, err = c.UploadImage(ctx, lastImage, lastImageName)
		if err != nil {
			return nil, "", fmt.Errorf("upload H3 FL2VA last frame: %w", err)
		}
	}
	workflow, err := c.imageToVideoWorkflow(ctx, uploadedName, uploadedLastName, prompt, negativePrompt, durationSec, width, height, settings, uploadedReferences)
	if err != nil {
		return nil, "", err
	}
	return c.generateVideoWithWorkflow(ctx, workflow, progress)
}

func (c *ComfyClient) imageToVideoWorkflow(ctx context.Context, uploadedName, lastFrameName, prompt, negativePrompt string, durationSec, width, height int, settings VideoGenerationSettings, referenceNames []string) (map[string]interface{}, error) {
	if settings.Model == "minimax_h3" {
		settings.QualityPreset = normalizeVideoQualityPreset(settings.QualityPreset)
		if settings.QualityPreset == "fast" {
			settings.AudioMode = "external"
		}
		key := "minimax_h3_fl2va"
		if settings.H3Mode == "r2v" {
			key = "minimax_h3_ref2va"
		}
		status := c.Status(ctx)
		if !status.Models["minimax_h3_nodes"] || !status.Models[key] || !status.Models["minimax_h3_text_encoder"] || !status.Models["minimax_h3_video_vae"] || ((settings.H3Mode == "r2v" || settings.AudioMode == "native") && !status.Models["minimax_h3_audio_vae"]) {
			return nil, fmt.Errorf("MiniMax H3 组件或模型未准备完整，请检查 ComfyUI 状态: %s", strings.Join(status.Missing, ", "))
		}
		return systemMiniMaxH3ImageToVideoWorkflow(uploadedName, lastFrameName, prompt, negativePrompt, durationSec, width, height, settings, referenceNames), nil
	}
	ok, err := c.hasLocalWan22I2V(ctx)
	if err == nil && ok {
		return systemWan22LocalImageToVideoWorkflow(uploadedName, prompt, negativePrompt, durationSec, width, height), nil
	}
	return nil, fmt.Errorf("未检测到完整的本地 Wan2.2 图生视频模型，请确认 high_noise、low_noise、umt5 文本编码器和 wan_2.1_vae.safetensors 已放入 ComfyUI models 目录并重启 ComfyUI")
}

func (c *ComfyClient) hasLocalWan22I2V(ctx context.Context) (bool, error) {
	var objectInfo map[string]interface{}
	if err := c.getJSON(ctx, "/object_info", &objectInfo); err != nil {
		return false, err
	}
	return objectOptionContains(objectInfo, "UNETLoader", "unet_name", "wan2.2_i2v_high_noise_14B_fp8_scaled.safetensors") &&
		objectOptionContains(objectInfo, "UNETLoader", "unet_name", "wan2.2_i2v_low_noise_14B_fp8_scaled.safetensors") &&
		objectOptionContains(objectInfo, "CLIPLoader", "clip_name", "umt5_xxl_fp8_e4m3fn_scaled.safetensors") &&
		objectOptionContains(objectInfo, "VAELoader", "vae_name", "wan_2.1_vae.safetensors"), nil
}

func (c *ComfyClient) generateWithWorkflow(ctx context.Context, workflow map[string]interface{}, progress func(int, string)) ([]byte, string, error) {
	body, err := json.Marshal(map[string]interface{}{
		"prompt":    workflow,
		"client_id": fmt.Sprintf("cloudnexus-drama-%d", time.Now().UnixNano()),
	})
	if err != nil {
		return nil, "", err
	}
	var queued struct {
		PromptID string `json:"prompt_id"`
		Error    string `json:"error"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/prompt", body, &queued); err != nil {
		return nil, "", fmt.Errorf("提交 ComfyUI 工作流失败：%w", err)
	}
	if queued.PromptID == "" {
		return nil, "", fmt.Errorf("ComfyUI 未返回任务编号%s", messageSuffix(queued.Error))
	}
	progress(25, "工作流已提交到 ComfyUI")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = c.Interrupt(context.Background(), queued.PromptID)
			return nil, "", ctx.Err()
		case <-ticker.C:
			var history map[string]json.RawMessage
			if err := c.getJSON(ctx, "/history/"+url.PathEscape(queued.PromptID), &history); err != nil {
				continue
			}
			raw, ok := history[queued.PromptID]
			if !ok {
				progress(45, "ComfyUI 正在生成图片")
				continue
			}
			image, err := firstHistoryImage(raw)
			if err != nil {
				return nil, "", err
			}
			progress(80, "图片生成完成，正在保存到云盘")
			data, err := c.DownloadImage(ctx, image)
			if err != nil {
				return nil, "", err
			}
			return data, image.Filename, nil
		}
	}
}

func (c *ComfyClient) generateVideoWithWorkflow(ctx context.Context, workflow map[string]interface{}, progress func(int, string)) ([]byte, string, error) {
	body, err := json.Marshal(map[string]interface{}{
		"prompt":    workflow,
		"client_id": fmt.Sprintf("cloudnexus-drama-video-%d", time.Now().UnixNano()),
	})
	if err != nil {
		return nil, "", err
	}
	var queued struct {
		PromptID string `json:"prompt_id"`
		Error    string `json:"error"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/prompt", body, &queued); err != nil {
		return nil, "", fmt.Errorf("提交 ComfyUI 视频工作流失败：%w", err)
	}
	if queued.PromptID == "" {
		return nil, "", fmt.Errorf("ComfyUI 未返回视频任务编号%s", messageSuffix(queued.Error))
	}
	progress(25, "视频工作流已提交到 ComfyUI")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = c.Interrupt(context.Background(), queued.PromptID)
			return nil, "", ctx.Err()
		case <-ticker.C:
			var history map[string]json.RawMessage
			if err := c.getJSON(ctx, "/history/"+url.PathEscape(queued.PromptID), &history); err != nil {
				continue
			}
			raw, ok := history[queued.PromptID]
			if !ok {
				progress(45, "ComfyUI 正在生成视频")
				continue
			}
			video, err := firstHistoryVideo(raw)
			if err != nil {
				return nil, "", friendlyComfyVideoError(err)
			}
			progress(85, "视频生成完成，正在保存到云盘")
			data, err := c.DownloadVideo(ctx, video)
			if err != nil {
				return nil, "", err
			}
			return data, video.Filename, nil
		}
	}
}

func (c *ComfyClient) UploadImage(ctx context.Context, data []byte, filename string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	filename = "cloudnexus_ref_" + fmt.Sprintf("%d_", time.Now().UnixNano()) + filepath.Base(filename)
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	_ = writer.WriteField("type", "input")
	_ = writer.WriteField("overwrite", "true")
	if err := writer.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/upload/image", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := c.media.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respData, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respData)))
	}
	var uploaded struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(respData, &uploaded); err != nil || uploaded.Name == "" {
		return filename, nil
	}
	return uploaded.Name, nil
}

func (c *ComfyClient) DownloadImage(ctx context.Context, image ComfyImage) ([]byte, error) {
	query := url.Values{}
	query.Set("filename", image.Filename)
	query.Set("subfolder", image.Subfolder)
	query.Set("type", image.Type)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/view?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.media.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 ComfyUI 图片失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载 ComfyUI 图片失败：HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *ComfyClient) DownloadVideo(ctx context.Context, video ComfyVideo) ([]byte, error) {
	query := url.Values{}
	query.Set("filename", video.Filename)
	query.Set("subfolder", video.Subfolder)
	query.Set("type", video.Type)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/view?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.media.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 ComfyUI 视频失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载 ComfyUI 视频失败：HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *ComfyClient) Interrupt(ctx context.Context, promptID string) error {
	deleteBody, _ := json.Marshal(map[string]interface{}{"delete": []string{promptID}})
	_ = c.doJSON(ctx, http.MethodPost, "/queue", deleteBody, nil)
	return c.doJSON(ctx, http.MethodPost, "/interrupt", []byte(`{}`), nil)
}

func (c *ComfyClient) getJSON(ctx context.Context, endpoint string, target interface{}) error {
	return c.doJSON(ctx, http.MethodGet, endpoint, nil, target)
}

func (c *ComfyClient) doJSON(ctx context.Context, method, endpoint string, body []byte, target interface{}) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if target != nil && len(data) > 0 {
		return json.Unmarshal(data, target)
	}
	return nil
}

func systemTextToImageWorkflow(prompt string, settings ImageGenerationSettings) map[string]interface{} {
	seed := imageGenerationSeed(settings)
	return map[string]interface{}{
		"1": map[string]interface{}{"class_type": "CheckpointLoaderSimple", "inputs": map[string]interface{}{"ckpt_name": settings.Checkpoint}},
		"2": map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": prompt, "clip": []interface{}{"1", 1}}},
		"3": map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": settings.NegativePrompt, "clip": []interface{}{"1", 1}}},
		"4": map[string]interface{}{"class_type": "EmptyLatentImage", "inputs": map[string]interface{}{"width": settings.Width, "height": settings.Height, "batch_size": 1}},
		"5": map[string]interface{}{"class_type": "KSampler", "inputs": map[string]interface{}{
			"seed": seed, "steps": settings.Steps, "cfg": settings.CFG, "sampler_name": settings.Sampler,
			"scheduler": settings.Scheduler, "denoise": 1, "model": []interface{}{"1", 0},
			"positive": []interface{}{"2", 0}, "negative": []interface{}{"3", 0}, "latent_image": []interface{}{"4", 0},
		}},
		"6": map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"5", 0}, "vae": []interface{}{"1", 2}}},
		"7": map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "cloudnexus_drama", "images": []interface{}{"6", 0}}},
	}
}

func systemTextToImageIPAdapterWorkflow(prompt string, settings ImageGenerationSettings, uploaded []string, references []ComfyReferenceImage) map[string]interface{} {
	if workflow, err := systemRegionalStoryboardWorkflow(prompt, settings, uploaded, references); err == nil {
		return workflow
	}
	seed := imageGenerationSeed(settings)
	workflow := map[string]interface{}{
		"1": map[string]interface{}{"class_type": "CheckpointLoaderSimple", "inputs": map[string]interface{}{"ckpt_name": settings.Checkpoint}},
		"2": map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": prompt, "clip": []interface{}{"1", 1}}},
		"3": map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": settings.NegativePrompt, "clip": []interface{}{"1", 1}}},
		"4": map[string]interface{}{"class_type": "EmptyLatentImage", "inputs": map[string]interface{}{"width": settings.Width, "height": settings.Height, "batch_size": 1}},
		"8": map[string]interface{}{"class_type": "CLIPVisionLoader", "inputs": map[string]interface{}{"clip_name": "CLIP-ViT-H-14-laion2B-s32B-b79K.safetensors"}},
		"9": map[string]interface{}{"class_type": "IPAdapterUnifiedLoader", "inputs": map[string]interface{}{"model": []interface{}{"1", 0}, "preset": "PLUS (high strength)"}},
	}
	modelRef := []interface{}{"9", 0}
	nextID := 10
	firstLoadID := ""
	for index, imageName := range uploaded {
		loadID := fmt.Sprintf("%d", nextID)
		nextID++
		if firstLoadID == "" {
			firstLoadID = loadID
		}
		prepID := fmt.Sprintf("%d", nextID)
		nextID++
		adapterID := fmt.Sprintf("%d", nextID)
		nextID++
		weight := referenceWeight(index, references)
		weightType := referenceWeightType(index, references)
		endAt := referenceEndAt(index, references)
		workflow[loadID] = map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": imageName}}
		workflow[prepID] = map[string]interface{}{"class_type": "PrepImageForClipVision", "inputs": map[string]interface{}{
			"image": []interface{}{loadID, 0}, "interpolation": "LANCZOS", "crop_position": "center", "sharpening": 0.05,
		}}
		workflow[adapterID] = map[string]interface{}{"class_type": "IPAdapterAdvanced", "inputs": map[string]interface{}{
			"model":          modelRef,
			"ipadapter":      []interface{}{"9", 1},
			"image":          []interface{}{prepID, 0},
			"weight":         weight,
			"weight_type":    weightType,
			"combine_embeds": "average",
			"start_at":       0.0,
			"end_at":         endAt,
			"embeds_scaling": "V only",
			"clip_vision":    []interface{}{"8", 0},
		}}
		modelRef = []interface{}{adapterID, 0}
	}
	latentRef := []interface{}{"4", 0}
	denoise := 1.0
	if firstLoadID != "" && len(references) > 0 && references[0].Kind == "reference" {
		scaleID := fmt.Sprintf("%d", nextID)
		nextID++
		encodeID := fmt.Sprintf("%d", nextID)
		workflow[scaleID] = map[string]interface{}{"class_type": "ImageScale", "inputs": map[string]interface{}{
			"image": []interface{}{firstLoadID, 0}, "upscale_method": "lanczos",
			"width": settings.Width, "height": settings.Height, "crop": "center",
		}}
		workflow[encodeID] = map[string]interface{}{"class_type": "VAEEncode", "inputs": map[string]interface{}{
			"pixels": []interface{}{scaleID, 0}, "vae": []interface{}{"1", 2},
		}}
		latentRef = []interface{}{encodeID, 0}
		denoise = referenceDenoise(references[0].Weight)
	}
	workflow["5"] = map[string]interface{}{"class_type": "KSampler", "inputs": map[string]interface{}{
		"seed": seed, "steps": settings.Steps, "cfg": settings.CFG, "sampler_name": settings.Sampler,
		"scheduler": settings.Scheduler, "denoise": denoise, "model": modelRef,
		"positive": []interface{}{"2", 0}, "negative": []interface{}{"3", 0}, "latent_image": latentRef,
	}}
	workflow["6"] = map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"5", 0}, "vae": []interface{}{"1", 2}}}
	workflow["7"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"filename_prefix": "cloudnexus_drama", "images": []interface{}{"6", 0}}}
	return workflow
}

func imageGenerationSeed(settings ImageGenerationSettings) int64 {
	if settings.Seed > 0 {
		return settings.Seed
	}
	return time.Now().UnixNano() & 0x7fffffffffffffff
}

func systemWanImageToVideoWorkflow(imageName, prompt, negativePrompt string, durationSec int) map[string]interface{} {
	if durationSec < 5 {
		durationSec = 5
	}
	if durationSec > 15 {
		durationSec = 15
	}
	if durationSec > 5 && durationSec < 10 {
		durationSec = 10
	}
	if durationSec > 10 {
		durationSec = 15
	}
	seed := time.Now().UnixNano() & 0x7fffffff
	return map[string]interface{}{
		"1": map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": imageName}},
		"2": map[string]interface{}{"class_type": "WanImageToVideoApi", "inputs": map[string]interface{}{
			"model":           "wan2.6-i2v",
			"image":           []interface{}{"1", 0},
			"prompt":          prompt,
			"negative_prompt": negativePrompt,
			"resolution":      "720P",
			"duration":        durationSec,
			"seed":            seed,
			"generate_audio":  false,
			"prompt_extend":   true,
			"watermark":       false,
			"shot_type":       "single",
		}},
		"3": map[string]interface{}{"class_type": "SaveVideo", "inputs": map[string]interface{}{
			"video": []interface{}{"2", 0}, "filename_prefix": "cloudnexus_drama_video", "format": "mp4", "codec": "h264",
		}},
	}
}

func systemWan22LocalImageToVideoWorkflow(imageName, prompt, negativePrompt string, durationSec, width, height int) map[string]interface{} {
	if durationSec < 2 {
		durationSec = 2
	}
	if durationSec > 10 {
		durationSec = 10
	}
	width, height = normalizeWanVideoDimensions(width, height)
	seed := time.Now().UnixNano() & 0x7fffffffffffffff
	frameLength := 81
	fps := 8.0
	if durationSec <= 5 {
		frameLength = durationSec*16 + 1
		fps = 16.0
	}
	negative := strings.TrimSpace(strings.Join([]string{
		negativePrompt,
		"色调艳丽，过曝，静态，细节模糊不清，字幕，作品，画作，画面，静止，整体发灰，最差质量，低质量，JPEG压缩残留，丑陋的，残缺的，多余的手指，画得不好的手部，画得不好的脸部，畸形的，毁容的，形态畸形的肢体，手指融合，静止不动的画面，杂乱的背景，背景人很多",
	}, ", "))
	return map[string]interface{}{
		"1":  map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": imageName}},
		"2":  map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": "umt5_xxl_fp8_e4m3fn_scaled.safetensors", "type": "wan", "device": "default"}},
		"3":  map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": "wan_2.1_vae.safetensors"}},
		"4":  map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": "wan2.2_i2v_high_noise_14B_fp8_scaled.safetensors", "weight_dtype": "default"}},
		"5":  map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": "wan2.2_i2v_low_noise_14B_fp8_scaled.safetensors", "weight_dtype": "default"}},
		"6":  map[string]interface{}{"class_type": "ModelSamplingSD3", "inputs": map[string]interface{}{"model": []interface{}{"4", 0}, "shift": 5.0}},
		"7":  map[string]interface{}{"class_type": "ModelSamplingSD3", "inputs": map[string]interface{}{"model": []interface{}{"5", 0}, "shift": 5.0}},
		"8":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": prompt, "clip": []interface{}{"2", 0}}},
		"9":  map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"text": negative, "clip": []interface{}{"2", 0}}},
		"10": map[string]interface{}{"class_type": "WanImageToVideo", "inputs": map[string]interface{}{"positive": []interface{}{"8", 0}, "negative": []interface{}{"9", 0}, "vae": []interface{}{"3", 0}, "start_image": []interface{}{"1", 0}, "width": width, "height": height, "length": frameLength, "batch_size": 1}},
		"11": map[string]interface{}{"class_type": "KSamplerAdvanced", "inputs": map[string]interface{}{"model": []interface{}{"6", 0}, "positive": []interface{}{"10", 0}, "negative": []interface{}{"10", 1}, "latent_image": []interface{}{"10", 2}, "add_noise": "enable", "noise_seed": seed, "steps": 20, "cfg": 1.0, "sampler_name": "euler", "scheduler": "simple", "start_at_step": 0, "end_at_step": 10, "return_with_leftover_noise": "enable"}},
		"12": map[string]interface{}{"class_type": "KSamplerAdvanced", "inputs": map[string]interface{}{"model": []interface{}{"7", 0}, "positive": []interface{}{"10", 0}, "negative": []interface{}{"10", 1}, "latent_image": []interface{}{"11", 0}, "add_noise": "disable", "noise_seed": seed, "steps": 20, "cfg": 1.0, "sampler_name": "euler", "scheduler": "simple", "start_at_step": 10, "end_at_step": 20, "return_with_leftover_noise": "disable"}},
		"13": map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"12", 0}, "vae": []interface{}{"3", 0}}},
		"14": map[string]interface{}{"class_type": "CreateVideo", "inputs": map[string]interface{}{"images": []interface{}{"13", 0}, "fps": fps}},
		"15": map[string]interface{}{"class_type": "SaveVideo", "inputs": map[string]interface{}{"video": []interface{}{"14", 0}, "filename_prefix": "cloudnexus_drama_video", "format": "mp4", "codec": "h264"}},
	}
}

// systemMiniMaxH3ImageToVideoWorkflow follows ComfyUI's native AV workflow:
// H3 produces video and audio latents together, then the audio branch is
// optionally discarded so the existing drama pipeline can keep external TTS.
func systemMiniMaxH3ImageToVideoWorkflow(imageName, lastFrameName, prompt, negativePrompt string, durationSec, width, height int, settings VideoGenerationSettings, referenceNames []string) map[string]interface{} {
	settings.QualityPreset = normalizeVideoQualityPreset(settings.QualityPreset)
	width, height = normalizeH3VideoDimensions(width, height, settings.QualityPreset)
	length := normalizeH3VideoLength(durationSec)
	steps := h3VideoSteps(settings.QualityPreset)
	if settings.QualityPreset == "fast" {
		settings.AudioMode = "external"
	}
	modelName := "minimax_h3_fl2va_pruned_int8_convrot.safetensors"
	conditioning := "MiniMaxH3ImageToVideo"
	if settings.H3Mode == "r2v" {
		modelName = "minimax_h3_ref2va_pruned_int8_convrot.safetensors"
		conditioning = "MiniMaxH3ReferenceToVideo"
	}
	workflow := map[string]interface{}{
		"1":  map[string]interface{}{"class_type": "UNETLoader", "inputs": map[string]interface{}{"unet_name": modelName, "weight_dtype": "default"}},
		"2":  map[string]interface{}{"class_type": "CLIPLoader", "inputs": map[string]interface{}{"clip_name": "qwen3vl_32b_minimax_h3_nvfp4_awq.safetensors", "type": "minimax", "device": "default"}},
		"3":  map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": "minimax_h3_video_vae_fp16.safetensors"}},
		"4":  map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": "minimax_h3_audio_vae_fp32.safetensors"}},
		"5":  map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": imageName}},
		"6":  map[string]interface{}{"class_type": "MiniMaxH3SigmaShift", "inputs": map[string]interface{}{"model": []interface{}{"1", 0}, "shift_video": 12.0, "shift_audio": 3.0}},
		"7":  map[string]interface{}{"class_type": conditioning, "inputs": map[string]interface{}{"clip": []interface{}{"2", 0}, "vae": []interface{}{"3", 0}, "prompt": strings.TrimSpace(prompt), "width": width, "height": height, "length": length}},
		"8":  map[string]interface{}{"class_type": "RandomNoise", "inputs": map[string]interface{}{"noise_seed": time.Now().UnixNano() & 0x7fffffffffffffff}},
		"9":  map[string]interface{}{"class_type": "KSamplerSelect", "inputs": map[string]interface{}{"sampler_name": "res_multistep"}},
		"10": map[string]interface{}{"class_type": "BasicScheduler", "inputs": map[string]interface{}{"model": []interface{}{"6", 0}, "scheduler": "simple", "steps": steps, "denoise": 1.0}},
		"11": map[string]interface{}{"class_type": "BasicGuider", "inputs": map[string]interface{}{"model": []interface{}{"6", 0}, "conditioning": []interface{}{"7", 0}}},
		"12": map[string]interface{}{"class_type": "SamplerCustomAdvanced", "inputs": map[string]interface{}{"noise": []interface{}{"8", 0}, "guider": []interface{}{"11", 0}, "sampler": []interface{}{"9", 0}, "sigmas": []interface{}{"10", 0}, "latent_image": []interface{}{"7", 1}}},
		"13": map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"12", 0}, "vae": []interface{}{"3", 0}}},
		"14": map[string]interface{}{"class_type": "CreateVideo", "inputs": map[string]interface{}{"images": []interface{}{"13", 0}, "fps": 24.0}},
		"15": map[string]interface{}{"class_type": "SaveVideo", "inputs": map[string]interface{}{"video": []interface{}{"14", 0}, "filename_prefix": "cloudnexus_h3_video", "format": "mp4", "codec": "h264"}},
	}
	// I2V needs the uploaded frame; R2V presents it as Picture 1.  The
	// conditioning node's dynamic reference input is an API input name.
	if settings.H3Mode == "r2v" {
		workflow["7"].(map[string]interface{})["inputs"].(map[string]interface{})["audio_vae"] = []interface{}{"4", 0}
		workflow["7"].(map[string]interface{})["inputs"].(map[string]interface{})["ref_image_size"] = settings.H3RefImageSize
		if len(referenceNames) == 0 {
			referenceNames = []string{imageName}
		}
		workflow["7"].(map[string]interface{})["inputs"].(map[string]interface{})["prompt"] = "Use <Picture 1> as the primary shot composition and use additional Picture references for identity and scene consistency. " + strings.TrimSpace(prompt)
		for index, referenceName := range referenceNames {
			if index >= 9 {
				break
			}
			if index == 0 && referenceName == imageName {
				workflow["7"].(map[string]interface{})["inputs"].(map[string]interface{})[fmt.Sprintf("ref_image_%d", index+1)] = []interface{}{"5", 0}
				continue
			}
			loaderID := fmt.Sprintf("18%d", index+1)
			workflow[loaderID] = map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": referenceName}}
			workflow["7"].(map[string]interface{})["inputs"].(map[string]interface{})[fmt.Sprintf("ref_image_%d", index+1)] = []interface{}{loaderID, 0}
		}
	} else {
		workflow["7"].(map[string]interface{})["inputs"].(map[string]interface{})["first_frame"] = []interface{}{"5", 0}
		if settings.H3Mode == "fl2va" && strings.TrimSpace(lastFrameName) != "" {
			workflow["18"] = map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": lastFrameName}}
			workflow["7"].(map[string]interface{})["inputs"].(map[string]interface{})["last_frame"] = []interface{}{"18", 0}
		}
	}
	if settings.AudioMode == "native" {
		workflow["16"] = map[string]interface{}{"class_type": "VAEDecodeAudio", "inputs": map[string]interface{}{"samples": []interface{}{"12", 0}, "vae": []interface{}{"4", 0}}}
		workflow["17"] = map[string]interface{}{"class_type": "CreateVideo", "inputs": map[string]interface{}{"images": []interface{}{"13", 0}, "audio": []interface{}{"16", 0}, "fps": 24.0}}
		workflow["15"].(map[string]interface{})["inputs"].(map[string]interface{})["video"] = []interface{}{"17", 0}
	}
	return workflow
}

func normalizeVideoQualityPreset(value string) string {
	switch value {
	case "fast", "standard", "final":
		return value
	default:
		return "standard"
	}
}

func h3VideoSteps(quality string) int {
	switch normalizeVideoQualityPreset(quality) {
	case "fast":
		return 8
	case "final":
		return 20
	default:
		return 12
	}
}

func normalizeH3VideoDimensions(width, height int, quality string) (int, int) {
	if width <= 0 || height <= 0 {
		width, height = 9, 16
	}
	shortSide, longSide, squareSide := 576, 1024, 768
	switch normalizeVideoQualityPreset(quality) {
	case "fast":
		shortSide, longSide, squareSide = 512, 896, 512
	case "final":
		shortSide, longSide, squareSide = 768, 1344, 1024
	}
	if width == height {
		return squareSide, squareSide
	}
	if width > height {
		return longSide, shortSide
	}
	return shortSide, longSide
}

func normalizeH3VideoLength(durationSec int) int {
	if durationSec < 1 {
		durationSec = 1
	}
	if durationSec > 15 {
		durationSec = 15
	}
	frames := int(math.Round(float64(durationSec) * 24))
	if frames < 5 {
		frames = 5
	}
	for frames%17 != 5 {
		frames++
	}
	return frames
}

func normalizeWanVideoDimensions(width, height int) (int, int) {
	if width <= 0 || height <= 0 {
		return 832, 480
	}
	width = ((width + 8) / 16) * 16
	height = ((height + 8) / 16) * 16
	if width < 256 {
		width = 256
	}
	if height < 256 {
		height = 256
	}
	if width > 832 {
		width = 832
	}
	if height > 832 {
		height = 832
	}
	return width, height
}

func referenceWeight(index int, references []ComfyReferenceImage) float64 {
	if index < len(references) && references[index].Weight > 0 {
		return references[index].Weight
	}
	if index < len(references) && references[index].Kind == "scene" {
		return 0.55
	}
	return 0.62
}

func referenceWeightType(index int, references []ComfyReferenceImage) string {
	if index < len(references) && (references[index].Kind == "scene" || references[index].Kind == "reference") {
		return "composition precise"
	}
	return "style transfer precise"
}

func referenceEndAt(index int, references []ComfyReferenceImage) float64 {
	if index < len(references) && references[index].Kind == "reference" {
		return 0.92
	}
	if index < len(references) && references[index].Kind == "scene" {
		return 0.72
	}
	return 0.82
}

func referenceDenoise(weight float64) float64 {
	if weight <= 0 {
		weight = 0.65
	}
	denoise := 0.9 - 0.55*weight
	if denoise < 0.3 {
		return 0.3
	}
	if denoise > 0.75 {
		return 0.75
	}
	return denoise
}

func defaultImageGenerationSettings(raw string) ImageGenerationSettings {
	settings := ImageGenerationSettings{
		Width: 768, Height: 1024, Steps: 24, CFG: 7,
		Sampler: "euler", Scheduler: "normal",
		NegativePrompt: "低质量，模糊，变形，多余手指，文字，水印",
	}
	_ = json.Unmarshal([]byte(raw), &settings)
	if settings.Width < 64 {
		settings.Width = 768
	}
	if settings.Height < 64 {
		settings.Height = 1024
	}
	if settings.Steps <= 0 {
		settings.Steps = 24
	}
	if settings.CFG <= 0 {
		settings.CFG = 7
	}
	if settings.Sampler == "" {
		settings.Sampler = "euler"
	}
	if settings.Scheduler == "" {
		settings.Scheduler = "normal"
	}
	return settings
}

func selectImageCheckpoint(current string, available []string, styleHints ...string) string {
	current = strings.TrimSpace(current)
	lowerCurrent := strings.ToLower(current)
	if current != "" && !strings.Contains(lowerCurrent, "sd_xl_base") {
		return current
	}
	style := detectDramaVisualStyle(strings.Join(styleHints, " "))
	patterns := []string{"realvisxl", "juggernaut"}
	if style == "anime" {
		patterns = []string{"animagine", "anime", "pony"}
	} else if style == "3d" || style == "illustration" {
		patterns = []string{"dreamshaper", "art", "illustration"}
	}
	for _, pattern := range patterns {
		for _, checkpoint := range available {
			if strings.Contains(strings.ToLower(checkpoint), pattern) {
				return checkpoint
			}
		}
	}
	if style == "anime" || style == "3d" || style == "illustration" {
		for _, checkpoint := range available {
			if strings.Contains(strings.ToLower(checkpoint), "sd_xl_base") {
				return checkpoint
			}
		}
	}
	if current != "" {
		return current
	}
	if len(available) > 0 {
		return available[0]
	}
	return ""
}

func checkpointNameContains(checkpoints []string, patterns ...string) bool {
	for _, checkpoint := range checkpoints {
		lower := strings.ToLower(checkpoint)
		for _, pattern := range patterns {
			if strings.Contains(lower, strings.ToLower(pattern)) {
				return true
			}
		}
	}
	return false
}

func extractCheckpoints(info map[string]interface{}) []string {
	node, ok := info["CheckpointLoaderSimple"].(map[string]interface{})
	if !ok {
		return nil
	}
	input, _ := node["input"].(map[string]interface{})
	required, _ := input["required"].(map[string]interface{})
	ckpt, _ := required["ckpt_name"].([]interface{})
	if len(ckpt) == 0 {
		return nil
	}
	values, _ := ckpt[0].([]interface{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		if name, ok := value.(string); ok && name != "" {
			result = append(result, name)
		}
	}
	return result
}

func objectOptionContains(info map[string]interface{}, nodeName, inputName, expected string) bool {
	node, ok := info[nodeName].(map[string]interface{})
	if !ok {
		return false
	}
	input, _ := node["input"].(map[string]interface{})
	required, _ := input["required"].(map[string]interface{})
	rawOptions, _ := required[inputName].([]interface{})
	if len(rawOptions) == 0 {
		return false
	}
	options, _ := rawOptions[0].([]interface{})
	for _, option := range options {
		if name, ok := option.(string); ok && name == expected {
			return true
		}
	}
	return false
}

func firstHistoryImage(raw json.RawMessage) (ComfyImage, error) {
	var history struct {
		Outputs map[string]struct {
			Images []ComfyImage `json:"images"`
		} `json:"outputs"`
		Status json.RawMessage `json:"status"`
	}
	if err := json.Unmarshal(raw, &history); err != nil {
		return ComfyImage{}, err
	}
	for _, output := range history.Outputs {
		if len(output.Images) > 0 {
			return output.Images[0], nil
		}
	}
	statusText := strings.TrimSpace(string(history.Status))
	if len(statusText) > 800 {
		statusText = statusText[:800]
	}
	return ComfyImage{}, fmt.Errorf("ComfyUI 任务结束但没有生成图片%s", messageSuffix(statusText))
}

func firstHistoryVideo(raw json.RawMessage) (ComfyVideo, error) {
	var history struct {
		Outputs map[string]struct {
			Videos []ComfyVideo `json:"videos"`
			GIFs   []ComfyVideo `json:"gifs"`
			Images []ComfyVideo `json:"images"`
		} `json:"outputs"`
		Status json.RawMessage `json:"status"`
	}
	if err := json.Unmarshal(raw, &history); err != nil {
		return ComfyVideo{}, err
	}
	for _, output := range history.Outputs {
		if len(output.Videos) > 0 {
			return output.Videos[0], nil
		}
		if len(output.GIFs) > 0 {
			return output.GIFs[0], nil
		}
		for _, image := range output.Images {
			lower := strings.ToLower(image.Filename)
			if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".webm") || strings.HasSuffix(lower, ".mov") || strings.HasSuffix(lower, ".gif") {
				return image, nil
			}
		}
	}
	statusText := strings.TrimSpace(string(history.Status))
	if len(statusText) > 800 {
		statusText = statusText[:800]
	}
	if isComfyUnauthorized(statusText) {
		return ComfyVideo{}, fmt.Errorf("ComfyUI 视频节点需要登录授权。当前使用的是 WanImageToVideoApi 云端节点，请先在 ComfyUI 中登录账号，或改用已安装本地模型的 Wan/LTX/Hunyuan 图生视频工作流")
	}
	return ComfyVideo{}, fmt.Errorf("ComfyUI 视频任务结束但没有生成视频%s", messageSuffix(statusText))
}

func messageSuffix(message string) string {
	if strings.TrimSpace(message) == "" {
		return ""
	}
	return "：" + strings.TrimSpace(message)
}

func friendlyComfyVideoError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if isComfyUnauthorized(message) {
		return fmt.Errorf("ComfyUI 视频节点需要登录授权。当前使用的是 WanImageToVideoApi 云端节点，请先在 ComfyUI 中登录账号，或改用已安装本地模型的 Wan/LTX/Hunyuan 图生视频工作流")
	}
	return err
}

func isComfyUnauthorized(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "unauthorized") || strings.Contains(lower, "please login first") || strings.Contains(lower, "auth_token_comfy_org")
}
