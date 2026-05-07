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

	// Merge by logical identity (host+port+service) — same logical service
	// on the same host after rebuild gets the same node record.
	var existing model.DockerNode
	err := n.db.Where("host = ? AND port = ? AND service = ? AND node_type = 'service'",
		n.host, n.port, n.serviceName).First(&existing).Error

	if err == nil && existing.Name != n.name {
		// Same logical service, new container (rebuild) — merge into existing record.
		oldName := existing.Name
		wasOnline := existing.OfflineSince == nil
		n.db.Model(&existing).Updates(map[string]interface{}{
			"name":           n.name,
			"host":           n.host,
			"port":           n.port,
			"last_heartbeat": now,
			"container_name": n.containerName,
			"version":        n.version,
		})
		// Close the old container's open session if any.
		// New session will be created by HealthAggregator on next probe.
		if wasOnline {
			n.db.Model(&model.NodeOnlineSession{}).
				Where("node_name = ? AND end_time IS NULL", oldName).
				Updates(map[string]interface{}{
					"end_time": now,
					"duration": gorm.Expr("EXTRACT(EPOCH FROM (? - start_time))", now),
				})
		}
		// Update all old sessions to point to the new node name so
		// session history is preserved under the current node identity.
		n.db.Model(&model.NodeOnlineSession{}).
			Where("node_name = ?", oldName).
			Update("node_name", n.name)
		return
	}

	if err == nil {
		// Same name — heartbeat update only, status managed by HealthAggregator.
		n.db.Model(&existing).Updates(map[string]interface{}{
			"host":           n.host,
			"port":           n.port,
			"last_heartbeat": now,
			"container_name": n.containerName,
			"version":        n.version,
		})
		return
	}

	// New logical service — insert.
	node := model.DockerNode{
		Name:          n.name,
		Host:          n.host,
		Port:          n.port,
		NodeType:      "service",
		Status:        status,
		Service:       n.serviceName,
		FirstSeenAt:   &now,
		ContainerName: n.containerName,
		Version:       n.version,
	}

	// ON CONFLICT (name) as safety net for concurrent inserts.
	err = n.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"host", "port", "last_heartbeat",
		}),
	}).Create(&node).Error
	if err != nil {
		fmt.Fprintf(os.Stderr, "node register failed: %v\n", err)
	}
}
