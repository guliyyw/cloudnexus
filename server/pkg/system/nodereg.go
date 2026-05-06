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
	db       *gorm.DB
	name     string
	host     string
	port     int
	stopCh   chan struct{}
	stopped  bool
	mu       sync.Mutex
}

// NewNodeRegistrar creates a registrar. Call Start() to begin heartbeat.
func NewNodeRegistrar(db *gorm.DB, name, host string, port int) *NodeRegistrar {
	if name == "" {
		name, _ = os.Hostname()
	}
	if host == "" {
		host = "localhost"
	}
	if port == 0 {
		port = 8081
	}
	return &NodeRegistrar{
		db:     db,
		name:   name,
		host:   host,
		port:   port,
		stopCh: make(chan struct{}),
	}
}

// Start registers the node and begins periodic heartbeat (every 10s).
// Call Stop() on shutdown.
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
	// Use ON CONFLICT to upsert; also update last_heartbeat
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
	// Update last_heartbeat separately (Create with OnConflict sets it on insert only)
	n.db.Model(&model.DockerNode{}).
		Where("name = ?", n.name).
		Update("last_heartbeat", now)
}
