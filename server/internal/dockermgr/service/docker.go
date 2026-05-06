package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

type DockerService struct {
	httpClient *http.Client
	baseURL    string
}

func NewDockerService() (*DockerService, error) {
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		if runtime.GOOS == "windows" {
			host = "tcp://localhost:2375"
		} else {
			host = "unix:///var/run/docker.sock"
		}
	}

	var transport *http.Transport
	if strings.HasPrefix(host, "unix://") {
		sock := strings.TrimPrefix(host, "unix://")
		transport = &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", sock)
			},
		}
	} else {
		transport = &http.Transport{}
	}

	baseURL := "http://localhost"
	if strings.HasPrefix(host, "tcp://") {
		baseURL = "http://" + strings.TrimPrefix(host, "tcp://")
	}

	return &DockerService{
		httpClient: &http.Client{Transport: transport, Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}, nil
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

func (s *DockerService) ListContainers(all bool) ([]ContainerInfo, error) {
	url := fmt.Sprintf("%s/containers/json?all=%v", s.baseURL, all)
	resp, err := s.httpClient.Get(url)
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

func (s *DockerService) doAction(id, action string) error {
	url := fmt.Sprintf("%s/containers/%s/%s", s.baseURL, id, action)
	req, _ := http.NewRequest("POST", url, nil)
	resp, err := s.httpClient.Do(req)
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

func (s *DockerService) StartContainer(id string) error  { return s.doAction(id, "start") }
func (s *DockerService) StopContainer(id string) error   { return s.doAction(id, "stop") }
func (s *DockerService) RestartContainer(id string) error { return s.doAction(id, "restart") }

func (s *DockerService) RemoveContainer(id string, force bool) error {
	url := fmt.Sprintf("%s/containers/%s?force=%v", s.baseURL, id, force)
	req, _ := http.NewRequest("DELETE", url, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *DockerService) GetLogs(id, tail string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/containers/%s/logs?stdout=true&stderr=true&tail=%s", s.baseURL, id, tail)
	resp, err := s.httpClient.Get(url)
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

func (s *DockerService) ListImages() ([]ImageInfo, error) {
	url := fmt.Sprintf("%s/images/json", s.baseURL)
	resp, err := s.httpClient.Get(url)
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

func (s *DockerService) PullImage(image string) error {
	url := fmt.Sprintf("%s/images/create?fromImage=%s", s.baseURL, image)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("X-Registry-Auth", "{}")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("拉取镜像失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("拉取镜像失败: %s", string(body))
	}
	// Drain the stream (Docker returns JSON lines during pull)
	io.Copy(io.Discard, resp.Body)
	return nil
}

func (s *DockerService) RemoveImage(image string, force bool) error {
	url := fmt.Sprintf("%s/images/%s?force=%v", s.baseURL, image, force)
	req, _ := http.NewRequest("DELETE", url, nil)
	resp, err := s.httpClient.Do(req)
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

func (s *DockerService) GetStats(id string) (*ContainerStats, error) {
	url := fmt.Sprintf("%s/containers/%s/stats?stream=false", s.baseURL, id)
	resp, err := s.httpClient.Get(url)
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

func (s *DockerService) CreateContainer(image, name string) (string, error) {
	body := fmt.Sprintf(`{"Image":"%s"}`, image)
	createURL := fmt.Sprintf("%s/containers/create?name=%s", s.baseURL, name)
	resp, err := s.httpClient.Post(createURL, "application/json", strings.NewReader(body))
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

	return result.ID, s.doAction(result.ID, "start")
}
