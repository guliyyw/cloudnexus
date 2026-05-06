package system

import (
	"fmt"
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
// (e.g. "user-file-svc"). name defaults to the container hostname. Call Start() to begin heartbeat.
func NewNodeRegistrar(db *gorm.DB, name, host, serviceName string, port int) *NodeRegistrar {
	containerName, _ := os.Hostname()
	if name == "" {
		name = containerName
	}
	if host == "" {
		host = "localhost"
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
		Name:   n.name,
		Host:   n.host,
		Port:   n.port,
		Status: status,
	}

	var existing model.DockerNode
	isNew := n.db.Where("name = ?", n.name).First(&existing).Error != nil

	if isNew {
		node.Service = n.serviceName
		node.FirstSeenAt = &now
		node.TotalOnlineSeconds = 0
		node.ContainerName = n.containerName
		node.Version = n.version
	}

	err := n.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"host", "port", "status", "last_heartbeat",
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
	// The new container takes over.
	n.db.Model(&model.DockerNode{}).
		Where("name != ? AND host = ? AND port = ? AND service = ? AND status = 'healthy'",
			n.name, n.host, n.port, n.serviceName).
		Updates(map[string]interface{}{
			"status":        "offline",
			"offline_since": now,
		})
}
