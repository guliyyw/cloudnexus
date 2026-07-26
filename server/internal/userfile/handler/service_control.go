package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type managedServiceDef struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Service     string `json:"service"`
	ComposeName string `json:"compose_name"`
	Profile     string `json:"profile"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type ServiceControlHandler struct {
	client *http.Client
}

func NewServiceControlHandler() *ServiceControlHandler {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			return d.DialContext(ctx, "unix", "/var/run/docker.sock")
		},
	}
	return &ServiceControlHandler{
		client: &http.Client{Transport: transport, Timeout: 10 * time.Second},
	}
}

var managedServices = []managedServiceDef{
	{Key: "files", Name: "Files", Service: "user-file-svc", ComposeName: "user-file-svc", Required: true, Description: "基础必开：登录、云盘、权限、管理后台、系统状态、音乐、相册、分享、回收站。"},
	{Key: "im", Name: "IM", Service: "im-svc", ComposeName: "im-svc", Profile: "im", Description: "需要聊天、好友私聊、实时消息、会话未读提醒时开启。"},
	{Key: "docker", Name: "Docker", Service: "docker-svc", ComposeName: "docker-svc", Profile: "docker", Description: "需要 Docker 管理页面、容器列表、日志、统计、镜像和远程 Docker 节点时开启。"},
	{Key: "camera", Name: "Camera", Service: "camera-svc", ComposeName: "camera-svc", Profile: "camera", Description: "需要视频监控、摄像头管理、人脸库、考勤签到、图片/视频识别时开启。"},
	{Key: "mediamtx", Name: "MediaMTX", Service: "mediamtx", ComposeName: "mediamtx", Profile: "camera", Description: "摄像头直播依赖：把 RTSP 摄像头流转换为浏览器可播放的 HLS。"},
	{Key: "ai-inference", Name: "AI Inference", Service: "ai-inference", ComposeName: "ai-inference", Profile: "camera", Description: "摄像头 AI 识别依赖：用于目标检测、图片/视频分析等视觉推理。"},
	{Key: "collab", Name: "Collab", Service: "collab-svc", ComposeName: "collab-svc", Profile: "collab", Description: "需要在线文档多人实时协作编辑时开启。"},
	{Key: "drama", Name: "Drama", Service: "drama-svc", ComposeName: "drama-svc", Profile: "drama", Description: "需要短剧工坊、项目/分镜/资产管理、TTS、图片任务、视频合成时开启。"},
	{Key: "comfyui", Name: "ComfyUI", Service: "comfyui", ComposeName: "comfyui", Profile: "comfyui", Description: "短剧/图片生成的 GPU 引擎；需要 Stable Diffusion、IP-Adapter、参考图生成时开启。"},
}

type dockerContainerSummary struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	State  string            `json:"State"`
	Status string            `json:"Status"`
	Labels map[string]string `json:"Labels"`
}

type managedServiceView struct {
	managedServiceDef
	ContainerID string `json:"container_id"`
	State       string `json:"state"`
	StatusText  string `json:"status_text"`
	Created     bool   `json:"created"`
	Startable   bool   `json:"startable"`
}

func (h *ServiceControlHandler) HandleListServices(c *gin.Context) {
	containers, err := h.listComposeContainers()
	if err != nil {
		c.JSON(503, response.Error(503, "Docker daemon is not available"))
		return
	}

	views := make([]managedServiceView, 0, len(managedServices))
	for _, def := range managedServices {
		container := findContainer(containers, def.ComposeName)
		view := managedServiceView{managedServiceDef: def}
		if container != nil {
			view.ContainerID = container.ID
			view.State = container.State
			view.StatusText = container.Status
			view.Created = true
			view.Startable = container.State != "running"
		}
		views = append(views, view)
	}
	c.JSON(200, response.OKWithData(gin.H{"services": views}))
}

func (h *ServiceControlHandler) HandleStartService(c *gin.Context) {
	serviceName := c.Param("service")
	def, ok := findManagedService(serviceName)
	if !ok {
		c.JSON(404, response.Error(404, "Unknown service"))
		return
	}
	if def.Required {
		c.JSON(409, response.Error(409, "Required service is managed by the minimal stack"))
		return
	}

	containers, err := h.listComposeContainers()
	if err != nil {
		c.JSON(503, response.Error(503, "Docker daemon is not available"))
		return
	}
	container := findContainer(containers, def.ComposeName)
	if container == nil {
		cmd := fmt.Sprintf("docker compose -f docker-compose.single.yml --profile %s up -d", def.Profile)
		c.JSON(409, response.Error(409, "Service container has not been created yet. Run once on the host: "+cmd))
		return
	}
	if container.State == "running" {
		c.JSON(200, response.OK("Service is already running"))
		return
	}

	req, _ := http.NewRequest(http.MethodPost, "http://docker/containers/"+container.ID+"/start", nil)
	resp, err := h.client.Do(req)
	if err != nil {
		c.JSON(500, response.Error(500, "Failed to start service"))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		c.JSON(resp.StatusCode, response.Error(resp.StatusCode, "Docker refused to start service"))
		return
	}
	c.JSON(200, response.OK("Service start requested"))
}

func (h *ServiceControlHandler) listComposeContainers() ([]dockerContainerSummary, error) {
	filters := map[string][]string{
		"label": {"com.docker.compose.project=deploy"},
	}
	rawFilters, _ := json.Marshal(filters)
	u := "http://docker/containers/json?all=true&filters=" + url.QueryEscape(string(rawFilters))
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker list returned %d", resp.StatusCode)
	}
	var containers []dockerContainerSummary
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, err
	}
	sort.Slice(containers, func(i, j int) bool {
		return strings.Join(containers[i].Names, ",") < strings.Join(containers[j].Names, ",")
	})
	return containers, nil
}

func findManagedService(name string) (managedServiceDef, bool) {
	for _, def := range managedServices {
		if def.Service == name || def.ComposeName == name || def.Key == name {
			return def, true
		}
	}
	return managedServiceDef{}, false
}

func findContainer(containers []dockerContainerSummary, composeName string) *dockerContainerSummary {
	for i := range containers {
		if containers[i].Labels["com.docker.compose.service"] == composeName {
			return &containers[i]
		}
	}
	return nil
}
