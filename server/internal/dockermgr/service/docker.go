package service

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type DockerService struct {
	mgr *EndpointManager
}

func NewDockerService(mgr *EndpointManager) *DockerService {
	return &DockerService{mgr: mgr}
}

func (s *DockerService) clientFor(endpoint string) (*endpointClient, error) {
	return s.mgr.GetClient(endpoint)
}

type ContainerInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Status  string   `json:"status"`
	Ports   []string `json:"ports"`
	Created string   `json:"created"`
}

type dockerContainer struct {
	ID      string   `json:"Id"`
	Names   []string `json:"Names"`
	Image   string   `json:"Image"`
	State   string   `json:"State"`
	Status  string   `json:"Status"`
	Created int64    `json:"Created"`
	Ports   []struct {
		IP          string `json:"IP"`
		PublicPort  int    `json:"PublicPort"`
		PrivatePort int    `json:"PrivatePort"`
		Type        string `json:"Type"`
	} `json:"Ports"`
}

type ContainerInspect struct {
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
}

func (s *DockerService) InspectContainer(endpoint, id string) (*ContainerInspect, error) {
	ec, err := s.clientFor(endpoint)
	if err != nil {
		return nil, err
	}
	resp, err := ec.client.Get(fmt.Sprintf("%s/containers/%s/json", ec.baseURL, id))
	if err != nil {
		return nil, fmt.Errorf("inspect container failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("container not found: %s", string(body))
	}
	var info ContainerInspect
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *DockerService) checkOwnership(endpoint, id string, userID uint64, isAdmin bool) error {
	if isAdmin {
		return nil
	}
	info, err := s.InspectContainer(endpoint, id)
	if err != nil {
		return err
	}
	creatorStr := info.Config.Labels["cloudnexus.creator"]
	if creatorStr == "" {
		return fmt.Errorf("容器无所有权信息")
	}
	creatorID, err := strconv.ParseUint(creatorStr, 10, 64)
	if err != nil || creatorID != userID {
		return fmt.Errorf("无权操作该容器")
	}
	return nil
}

func (s *DockerService) ListContainers(endpoint string, all bool, userID uint64, isAdmin bool) ([]ContainerInfo, error) {
	ec, err := s.clientFor(endpoint)
	if err != nil {
		return nil, err
	}
	apiURL := fmt.Sprintf("%s/containers/json?all=%v", ec.baseURL, all)
	if !isAdmin {
		labelFilter := fmt.Sprintf(`{"label":["cloudnexus.creator=%d"]}`, userID)
		apiURL += "&filters=" + url.QueryEscape(labelFilter)
	}
	resp, err := ec.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("连接 Docker 失败: %w", err)
	}
	defer resp.Body.Close()

	var containers []dockerContainer
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, err
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		ports := make([]string, 0)
		result = append(result, ContainerInfo{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			Status:  c.Status,
			Ports:   ports,
			Created: time.Unix(c.Created, 0).Format(time.RFC3339),
		})
	}
	return result, nil
}

func (s *DockerService) doAction(endpoint, id, action string) error {
	ec, err := s.clientFor(endpoint)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/containers/%s/%s", ec.baseURL, id, action)
	req, _ := http.NewRequest("POST", url, nil)
	resp, err := ec.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("docker error: %s", string(body))
	}
	return nil
}

func (s *DockerService) StartContainer(endpoint, id string, userID uint64, isAdmin bool) error {
	if err := s.checkOwnership(endpoint, id, userID, isAdmin); err != nil {
		return err
	}
	return s.doAction(endpoint, id, "start")
}
func (s *DockerService) StopContainer(endpoint, id string, userID uint64, isAdmin bool) error {
	if err := s.checkOwnership(endpoint, id, userID, isAdmin); err != nil {
		return err
	}
	return s.doAction(endpoint, id, "stop")
}
func (s *DockerService) RestartContainer(endpoint, id string, userID uint64, isAdmin bool) error {
	if err := s.checkOwnership(endpoint, id, userID, isAdmin); err != nil {
		return err
	}
	return s.doAction(endpoint, id, "restart")
}

func (s *DockerService) RemoveContainer(endpoint, id string, force bool, userID uint64, isAdmin bool) error {
	if err := s.checkOwnership(endpoint, id, userID, isAdmin); err != nil {
		return err
	}
	ec, err := s.clientFor(endpoint)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/containers/%s?force=%v", ec.baseURL, id, force)
	req, _ := http.NewRequest("DELETE", url, nil)
	resp, err := ec.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *DockerService) GetLogs(endpoint, id, tail string, userID uint64, isAdmin bool) (io.ReadCloser, error) {
	if err := s.checkOwnership(endpoint, id, userID, isAdmin); err != nil {
		return nil, err
	}
	ec, err := s.clientFor(endpoint)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/containers/%s/logs?stdout=true&stderr=true&tail=%s", ec.baseURL, id, tail)
	resp, err := ec.client.Get(url)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// --- Image management ---

type ImageInfo struct {
	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
	Size    int64    `json:"size"`
	Created string   `json:"created"`
}

type dockerImage struct {
	ID      string   `json:"Id"`
	RepoTags []string `json:"RepoTags"`
	Size    int64    `json:"Size"`
	Created int64    `json:"Created"`
}

func (s *DockerService) ListImages(endpoint string) ([]ImageInfo, error) {
	ec, err := s.clientFor(endpoint)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/images/json", ec.baseURL)
	resp, err := ec.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取镜像列表失败: %w", err)
	}
	defer resp.Body.Close()

	var raw []dockerImage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	result := make([]ImageInfo, 0, len(raw))
	for _, img := range raw {
		id := img.ID
		if strings.HasPrefix(id, "sha256:") {
			id = id[7:19]
		}
		if len(id) > 12 {
			id = id[:12]
		}
		tags := img.RepoTags
		if tags == nil {
			tags = []string{"<none>:<none>"}
		}
		result = append(result, ImageInfo{
			ID:      id,
			Tags:    tags,
			Size:    img.Size,
			Created: time.Unix(img.Created, 0).Format(time.RFC3339),
		})
	}
	return result, nil
}

func (s *DockerService) PullImage(endpoint, image string) error {
	ec, err := s.clientFor(endpoint)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/images/create?fromImage=%s", ec.baseURL, image)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("X-Registry-Auth", "{}")
	resp, err := ec.client.Do(req)
	if err != nil {
		return fmt.Errorf("拉取镜像失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("拉取镜像失败: %s", string(body))
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

func (s *DockerService) RemoveImage(endpoint, image string, force bool) error {
	ec, err := s.clientFor(endpoint)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/images/%s?force=%v", ec.baseURL, image, force)
	req, _ := http.NewRequest("DELETE", url, nil)
	resp, err := ec.client.Do(req)
	if err != nil {
		return fmt.Errorf("删除镜像失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("删除镜像失败: %s", string(body))
	}
	return nil
}

// --- Container stats ---

type ContainerStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   int64   `json:"memory_usage"`
	MemoryLimit   int64   `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
}

type dockerStats struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage int64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		OnlineCPUs     int   `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage int64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage int64 `json:"usage"`
		Limit int64 `json:"limit"`
	} `json:"memory_stats"`
}

func (s *DockerService) GetStats(endpoint, id string, userID uint64, isAdmin bool) (*ContainerStats, error) {
	if err := s.checkOwnership(endpoint, id, userID, isAdmin); err != nil {
		return nil, err
	}
	ec, err := s.clientFor(endpoint)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/containers/%s/stats?stream=false", ec.baseURL, id)
	resp, err := ec.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取容器状态失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取容器状态失败: %s", string(body))
	}

	var raw dockerStats
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(raw.CPUStats.SystemCPUUsage - raw.PreCPUStats.SystemCPUUsage)
	cpuPercent := 0.0
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(raw.CPUStats.OnlineCPUs) * 100
	}

	memPercent := 0.0
	if raw.MemoryStats.Limit > 0 {
		memPercent = float64(raw.MemoryStats.Usage) / float64(raw.MemoryStats.Limit) * 100
	}

	return &ContainerStats{
		CPUPercent:    math.Round(cpuPercent*100) / 100,
		MemoryUsage:   raw.MemoryStats.Usage,
		MemoryLimit:   raw.MemoryStats.Limit,
		MemoryPercent: math.Round(memPercent*100) / 100,
	}, nil
}

func (s *DockerService) CreateContainer(endpoint, image, name string, userID uint64, username string) (string, error) {
	ec, err := s.clientFor(endpoint)
	if err != nil {
		return "", err
	}
	labels := fmt.Sprintf(`"cloudnexus.creator":"%d","cloudnexus.creator_name":"%s"`, userID, username)
	body := fmt.Sprintf(`{"Image":"%s","Labels":{%s}}`, image, labels)
	createURL := fmt.Sprintf("%s/containers/create?name=%s", ec.baseURL, name)
	resp, err := ec.client.Post(createURL, "application/json", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("create container failed: %w", err)
	}

	return result.ID, s.doAction(endpoint, result.ID, "start")
}

func (s *DockerService) Ping() error {
	ec, _ := s.mgr.GetClient("local")
	if ec == nil {
		return fmt.Errorf("local docker endpoint not available")
	}
	resp, err := ec.client.Get(ec.baseURL + "/_ping")
	if err != nil {
		return fmt.Errorf("docker daemon unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker daemon returned status %d", resp.StatusCode)
	}
	return nil
}

// ListEndpoints returns all available Docker endpoints.
func (s *DockerService) ListEndpoints() []EndpointInfo {
	return s.mgr.ListEndpoints()
}

// PingEndpoint pings a specific endpoint.
func (s *DockerService) PingEndpoint(name string) error {
	return s.mgr.PingEndpoint(name)
}
