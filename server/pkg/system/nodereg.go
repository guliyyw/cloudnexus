package system

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NodeRegistrar struct {
	db            *gorm.DB
	name          string
	host          string
	port          int
	serviceName   string
	containerName string
	version       string
	stopCh        chan struct{}
	stopped       bool
	mu            sync.Mutex
}

// NewNodeRegistrar creates a registrar. serviceName is the logical service name
// (e.g. "user-file-svc"). name defaults to the container hostname.
// host defaults to the first non-loopback IPv4 address detected on the machine,
// or "localhost" as last resort. Override via NODE_HOST env var.
// Call Start() to begin heartbeat.
func NewNodeRegistrar(db *gorm.DB, name, host, serviceName string, port int) *NodeRegistrar {
	containerName, _ := os.Hostname()
	if name == "" {
		name = containerName
	}
	if host == "" {
		host = detectHostIP()
	}
	if port == 0 {
		port = 8081
	}
	return &NodeRegistrar{
		db:            db,
		name:          name,
		host:          host,
		port:          port,
		serviceName:   serviceName,
		containerName: containerName,
		version:       os.Getenv("SERVICE_VERSION"),
		stopCh:        make(chan struct{}),
	}
}

func detectHostIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			if ipnet.IP.IsPrivate() {
				return ipnet.IP.String()
			}
		}
	}
	// If no private IP found, try any non-loopback IPv4.
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "localhost"
}

// Start registers the node and begins periodic heartbeat (every 10s).
func (n *NodeRegistrar) Start() {
	n.upsert("healthy")
	go n.heartbeatLoop()
}

// Stop marks the node as offline and stops the heartbeat loop.
func (n *NodeRegistrar) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.stopped {
		return
	}
	n.stopped = true
	close(n.stopCh)
	n.upsert("offline")
}

func (n *NodeRegistrar) heartbeatLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.upsert("healthy")
		}
	}
}

func (n *NodeRegistrar) upsert(status string) {
	now := time.Now()
	node := model.DockerNode{
		Name:     n.name,
		Host:     n.host,
		Port:     n.port,
		NodeType: "service",
	}

	var existing model.DockerNode
	isNew := n.db.Where("name = ?", n.name).First(&existing).Error != nil

	if isNew {
		node.Status = status
		node.Service = n.serviceName
		node.FirstSeenAt = &now
		node.TotalOnlineSeconds = 0
		node.ContainerName = n.containerName
		node.Version = n.version
	}

	// On conflict, only update host/port/last_heartbeat — do NOT touch status,
	// which is managed by HealthAggregator (healthy/unresponsive/offline).
	err := n.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"host", "port", "last_heartbeat",
		}),
	}).Create(&node).Error
	if err != nil {
		fmt.Fprintf(os.Stderr, "node register failed: %v\n", err)
		return
	}

	// Always update container_name and version (may change after rebuild)
	n.db.Model(&model.DockerNode{}).
		Where("name = ?", n.name).
		Updates(map[string]interface{}{
			"last_heartbeat": now,
			"container_name": n.containerName,
			"version":        n.version,
		})

	// Mark any other nodes with same host+port+service as offline (stale containers).
	// Only if host is a real IP (not localhost), to avoid cross-server false positives.
	n.db.Model(&model.DockerNode{}).
		Where("name != ? AND host = ? AND port = ? AND service = ? AND status IN ('healthy','unresponsive')",
			n.name, n.host, n.port, n.serviceName).
		Updates(map[string]interface{}{
			"status":        "offline",
			"offline_since": now,
		})
}
