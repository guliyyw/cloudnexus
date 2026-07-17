package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultComfyUIURL = "http://comfyui:8188"

type ComfyStatus struct {
	Connected   bool     `json:"connected"`
	URL         string   `json:"url"`
	Checkpoints []string `json:"checkpoints"`
	IPAdapter   bool     `json:"ip_adapter"`
	ReActor     bool     `json:"reactor"`
	Missing     []string `json:"missing"`
	Error       string   `json:"error,omitempty"`
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
}

type ComfyImage struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

type ComfyClient struct {
	baseURL string
	http    *http.Client
}

func NewComfyClient(rawURL string) *ComfyClient {
	rawURL = strings.TrimRight(strings.TrimSpace(rawURL), "/")
	if rawURL == "" {
		rawURL = defaultComfyUIURL
	}
	return &ComfyClient{
		baseURL: rawURL,
		http:    &http.Client{Timeout: 5 * time.Second},
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

	var objectInfo map[string]interface{}
	if err := c.getJSON(ctx, "/object_info", &objectInfo); err != nil {
		status.Error = "读取节点信息失败：" + err.Error()
		status.Missing = []string{"节点信息"}
		return status
	}
	status.Checkpoints = extractCheckpoints(objectInfo)
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
	return status
}

func (c *ComfyClient) Generate(ctx context.Context, prompt string, settings ImageGenerationSettings, progress func(int, string)) ([]byte, string, error) {
	workflow := systemTextToImageWorkflow(prompt, settings)
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

func (c *ComfyClient) DownloadImage(ctx context.Context, image ComfyImage) ([]byte, error) {
	query := url.Values{}
	query.Set("filename", image.Filename)
	query.Set("subfolder", image.Subfolder)
	query.Set("type", image.Type)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/view?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 ComfyUI 图片失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载 ComfyUI 图片失败：HTTP %d", resp.StatusCode)
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
	seed := time.Now().UnixNano() & 0x7fffffffffffffff
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

func messageSuffix(message string) string {
	if strings.TrimSpace(message) == "" {
		return ""
	}
	return "：" + strings.TrimSpace(message)
}
