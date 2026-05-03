package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
