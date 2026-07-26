package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type DashboardService struct {
	db *gorm.DB
}

func NewDashboardService(db *gorm.DB) *DashboardService {
	return &DashboardService{db: db}
}

type ModuleInfo struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type DashboardSummary struct {
	Total   int `json:"total"`
	Healthy int `json:"healthy"`
	Warning int `json:"warning"`
	Error   int `json:"error"`
}

type DashboardStatus struct {
	Modules []ModuleInfo     `json:"modules"`
	Summary DashboardSummary `json:"summary"`
}

var moduleDefs = []struct {
	Key     string
	Name    string
	Icon    string
	Service string
}{
	{Key: "files", Name: "云存储", Icon: "CloudOutlined", Service: "user-file-svc"},
	{Key: "im", Name: "即时通讯", Icon: "MessageOutlined", Service: "im-svc"},
	{Key: "docker", Name: "Docker 管理", Icon: "ContainerOutlined", Service: "docker-svc"},
	{Key: "cameras", Name: "视频监控", Icon: "VideoCameraOutlined", Service: "camera-svc"},
	{Key: "collab", Name: "在线文档", Icon: "FileTextOutlined", Service: "collab-svc"},
	{Key: "drama", Name: "短剧工坊", Icon: "PlaySquareOutlined", Service: "drama-svc"},
	{Key: "infra", Name: "基础设施", Icon: "ClusterOutlined", Service: ""},
}

func (s *DashboardService) GetStatus() (*DashboardStatus, error) {
	var nodes []model.DockerNode
	if err := s.db.Find(&nodes).Error; err != nil {
		return nil, err
	}

	serviceNodes := make(map[string][]model.DockerNode)
	infraNodes := make([]model.DockerNode, 0)
	for _, n := range nodes {
		if n.NodeType == "infrastructure" {
			infraNodes = append(infraNodes, n)
		} else {
			serviceNodes[n.Service] = append(serviceNodes[n.Service], n)
		}
	}

	modules := make([]ModuleInfo, 0, len(moduleDefs))
	summary := DashboardSummary{}
	for _, def := range moduleDefs {
		var mod ModuleInfo
		if def.Key == "infra" {
			mod = s.buildModule(def, infraNodes)
		} else {
			mod = s.buildModule(def, serviceNodes[def.Service])
		}
		modules = append(modules, mod)
		switch mod.Status {
		case "green":
			summary.Healthy++
		case "yellow":
			summary.Warning++
		case "red":
			summary.Error++
		}
	}
	summary.Total = len(modules)

	return &DashboardStatus{Modules: modules, Summary: summary}, nil
}

func (s *DashboardService) buildModule(def struct {
	Key     string
	Name    string
	Icon    string
	Service string
}, nodes []model.DockerNode) ModuleInfo {
	mod := ModuleInfo{Key: def.Key, Name: def.Name, Icon: def.Icon, Status: "green"}
	if len(nodes) == 0 {
		mod.Status = "red"
		mod.Detail = "无节点"
		return mod
	}

	healthyCount, unresponsiveCount, offlineCount := 0, 0, 0
	for _, n := range nodes {
		switch effectiveNodeStatus(n) {
		case "healthy":
			healthyCount++
		case "unresponsive":
			unresponsiveCount++
		case "offline":
			offlineCount++
		default:
			offlineCount++
		}
	}

	// A module stays available while at least one current instance is healthy.
	// Replaced container records remain visible without failing the module.
	if healthyCount > 0 {
		mod.Status = "green"
	} else if unresponsiveCount > 0 {
		mod.Status = "yellow"
	} else {
		mod.Status = "red"
	}

	mod.Detail = formatNodeDetail(healthyCount, unresponsiveCount, offlineCount, len(nodes))
	return mod
}

func effectiveNodeStatus(n model.DockerNode) string {
	if n.LastHeartbeat != nil && time.Since(*n.LastHeartbeat) > 45*time.Second {
		return "offline"
	}
	return n.Status
}

func formatNodeDetail(healthy, unresponsive, offline, total int) string {
	if total == 0 {
		return "无节点"
	}
	parts := make([]string, 0, 3)
	if healthy > 0 {
		parts = append(parts, pluralize(healthy, "正常"))
	}
	if unresponsive > 0 {
		parts = append(parts, pluralize(unresponsive, "无响应"))
	}
	if offline > 0 {
		parts = append(parts, pluralize(offline, "离线"))
	}
	if len(parts) == 1 && healthy == total {
		return pluralize(total, "正常")
	}
	return strings.Join(parts, "；")
}

func pluralize(count int, state string) string {
	return fmt.Sprintf("%d 个节点%s", count, state)
}
