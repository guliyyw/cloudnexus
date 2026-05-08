package system

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

// InfraNode defines an infrastructure node to monitor.
type InfraNode struct {
	Name    string
	Host    string
	Port    int
	ProbeFn func() bool
}

// HealthAggregator periodically probes all registered nodes' /healthz endpoints
// and manages online session tracking.
type HealthAggregator struct {
	db         *gorm.DB
	interval   time.Duration
	client     *http.Client
	stopCh     chan struct{}
	stopped    bool
	mu         sync.Mutex
	infraNodes []InfraNode
	failures   map[string]int
	failuresMu sync.Mutex
	alerter    *AlertEvaluator
}

// NewHealthAggregator creates a new health aggregator.
// Probes every 15s with a 5s HTTP timeout per node.
func NewHealthAggregator(db *gorm.DB, alerter *AlertEvaluator) *HealthAggregator {
	return &HealthAggregator{
		db:       db,
		interval: 15 * time.Second,
		client:   &http.Client{Timeout: 5 * time.Second},
		stopCh:   make(chan struct{}),
		failures: make(map[string]int),
		alerter:  alerter,
	}
}

// RegisterInfra adds an infrastructure node for health monitoring.
// The node record is created in the DB if it does not exist.
func (a *HealthAggregator) RegisterInfra(node InfraNode) {
	a.infraNodes = append(a.infraNodes, node)
	var existing model.DockerNode
	if err := a.db.Where("name = ?", node.Name).First(&existing).Error; err == nil {
		return
	}
	now := time.Now()
	rec := model.DockerNode{
		Name:          node.Name,
		Host:          node.Host,
		Port:          node.Port,
		NodeType:      "infrastructure",
		Service:       node.Name,
		Status:        "healthy",
		FirstSeenAt:   &now,
		LastHeartbeat: &now,
	}
	a.db.Create(&rec)
}

// Start begins periodic health probing.
func (a *HealthAggregator) Start() {
	go a.probeAll()
	go a.loop()
}

// Stop halts the health aggregator.
func (a *HealthAggregator) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stopped {
		return
	}
	a.stopped = true
	close(a.stopCh)
}

func (a *HealthAggregator) loop() {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.probeAll()
		}
	}
}

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

func (a *HealthAggregator) probeAll() {
	var nodes []model.DockerNode
	if err := a.db.Find(&nodes).Error; err != nil {
		fmt.Fprintf(os.Stderr, "health aggregator: query nodes failed: %v\n", err)
		return
	}

	for _, node := range nodes {
		if node.NodeType == "infrastructure" {
			a.probeInfraNode(node)
		} else {
			a.probeServiceNode(node)
		}
	}
}

// Consecutive failure thresholds.
const unresponsiveAfter = 2 // ~30s of failures → unresponsive
const offlineAfter = 5      // ~75s of failures → offline

func (a *HealthAggregator) recordFailure(name string) int {
	a.failuresMu.Lock()
	defer a.failuresMu.Unlock()
	a.failures[name]++
	return a.failures[name]
}

func (a *HealthAggregator) resetFailures(name string) {
	a.failuresMu.Lock()
	defer a.failuresMu.Unlock()
	delete(a.failures, name)
}

func (a *HealthAggregator) probeInfraNode(node model.DockerNode) {
	now := time.Now()

	// First try the registered probe function (for local infrastructure nodes).
	for _, infra := range a.infraNodes {
		if infra.Name == node.Name {
			if infra.ProbeFn() {
				a.resetFailures(node.Name)
				a.markHealthy(&node, node.Service, node.Version, now)
			} else {
				a.handleFailure(&node, now)
			}
			return
		}
	}

	// Fall back to generic TCP probe for manually added infra nodes.
	if TCPProbe(node.Host, node.Port)() {
		a.resetFailures(node.Name)
		a.markHealthy(&node, node.Service, node.Version, now)
	} else {
		a.handleFailure(&node, now)
	}
}

func (a *HealthAggregator) probeServiceNode(node model.DockerNode) {
	now := time.Now()
	if a.tryProbe(node.Host, node.Port, &node, now) {
		return
	}

	// Fall back to node name as hostname (Docker container ID resolves via Docker DNS).
	if a.tryProbe(node.Name, node.Port, &node, now) {
		return
	}

	a.handleFailure(&node, now)
}

// tryProbe attempts to probe a node at the given host:port. Returns true on success.
func (a *HealthAggregator) tryProbe(host string, port int, node *model.DockerNode, now time.Time) bool {
	url := fmt.Sprintf("http://%s:%d/healthz", host, port)
	resp, err := a.client.Get(url)
	if err != nil || resp == nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var hr healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil || hr.Status != "ok" {
		return false
	}

	a.resetFailures(node.Name)
	a.markHealthy(node, hr.Service, hr.Version, now)
	return true
}

func (a *HealthAggregator) handleFailure(node *model.DockerNode, now time.Time) {
	count := a.recordFailure(node.Name)

	if count >= offlineAfter {
		a.markOffline(node, now)
	} else if count >= unresponsiveAfter {
		a.markUnresponsive(node, now)
	}
	// If count < unresponsiveAfter, keep current status (allow transient failures)
}

func (a *HealthAggregator) markUnresponsive(node *model.DockerNode, now time.Time) {
	if node.Status == "unresponsive" {
		return // already unresponsive
	}

	wasOnline := node.Status == "healthy" && node.OfflineSince == nil
	updates := map[string]interface{}{
		"status":         "unresponsive",
		"last_heartbeat": now,
	}

	// Close the current session only if transitioning from healthy
	if wasOnline {
		a.db.Model(&model.NodeOnlineSession{}).
			Where("node_name = ? AND end_time IS NULL", node.Name).
			Updates(map[string]interface{}{
				"end_time": now,
				"duration": gorm.Expr("EXTRACT(EPOCH FROM (? - start_time))", now),
			})
		var total int64
		a.db.Model(&model.NodeOnlineSession{}).
			Where("node_name = ?", node.Name).
			Select("COALESCE(SUM(duration), 0)").
			Scan(&total)
		updates["total_online_seconds"] = total
		updates["offline_since"] = now
	}

	a.db.Model(&model.DockerNode{}).Where("name = ?", node.Name).Updates(updates)

	// Update the local copy so subsequent checks see correct state
	node.Status = "unresponsive"
	if wasOnline {
		node.OfflineSince = &now
	}

	if a.alerter != nil {
		a.alerter.Evaluate(*node, "unresponsive")
	}
}

func (a *HealthAggregator) markHealthy(node *model.DockerNode, service, version string, now time.Time) {
	updates := map[string]interface{}{
		"status":         "healthy",
		"last_heartbeat": now,
	}

	if service != "" && node.Service == "" {
		updates["service"] = service
	}
	if version != "" && version != node.Version {
		updates["version"] = version
	}
	if node.FirstSeenAt == nil {
		updates["first_seen_at"] = now
	}

	wasOffline := node.OfflineSince != nil

	if wasOffline {
		a.db.Model(&model.NodeOnlineSession{}).
			Where("node_name = ? AND end_time IS NULL", node.Name).
			Updates(map[string]interface{}{
				"end_time": now,
				"duration": gorm.Expr("EXTRACT(EPOCH FROM (? - start_time))", now),
			})

		session := model.NodeOnlineSession{
			NodeName:  node.Name,
			StartTime: now,
		}
		if err := a.db.Create(&session).Error; err != nil {
			fmt.Fprintf(os.Stderr, "health aggregator: create session failed: %v\n", err)
		}

		updates["offline_since"] = nil
	} else {
		a.db.Model(&model.NodeOnlineSession{}).
			Where("node_name = ? AND end_time IS NULL", node.Name).
			Update("duration", gorm.Expr("EXTRACT(EPOCH FROM (? - start_time))", now))

		var count int64
		a.db.Model(&model.NodeOnlineSession{}).
			Where("node_name = ? AND end_time IS NULL", node.Name).
			Count(&count)
		if count == 0 {
			session := model.NodeOnlineSession{
				NodeName:      node.Name,
				StartTime:     now,
				ContainerName: node.ContainerName,
				Version:       node.Version,
			}
			a.db.Create(&session)
		}
	}

	a.db.Model(&model.DockerNode{}).Where("name = ?", node.Name).Updates(updates)

	var total int64
	a.db.Model(&model.NodeOnlineSession{}).
		Where("node_name = ?", node.Name).
		Select("COALESCE(SUM(duration), 0)").
		Scan(&total)
	a.db.Model(&model.DockerNode{}).Where("name = ?", node.Name).
		Update("total_online_seconds", total)

	if a.alerter != nil && wasOffline {
		go a.alerter.ResolveAlert(node.Name)
		a.alerter.Evaluate(*node, "recovery")
	}
}

func (a *HealthAggregator) markOffline(node *model.DockerNode, now time.Time) {
	wasOnline := (node.Status == "healthy" || node.Status == "unresponsive") && node.OfflineSince == nil

	if wasOnline {
		a.db.Model(&model.NodeOnlineSession{}).
			Where("node_name = ? AND end_time IS NULL", node.Name).
			Updates(map[string]interface{}{
				"end_time": now,
				"duration": gorm.Expr("EXTRACT(EPOCH FROM (? - start_time))", now),
			})

		var total int64
		a.db.Model(&model.NodeOnlineSession{}).
			Where("node_name = ?", node.Name).
			Select("COALESCE(SUM(duration), 0)").
			Scan(&total)
		a.db.Model(&model.DockerNode{}).Where("name = ?", node.Name).
			Updates(map[string]interface{}{
				"status":               "offline",
				"offline_since":        now,
				"total_online_seconds": total,
			})
	} else {
		a.db.Model(&model.DockerNode{}).Where("name = ?", node.Name).
			Update("status", "offline")
	}

	if a.alerter != nil {
		a.alerter.Evaluate(*node, "offline")
	}
}

// TCPProbe returns an InfraNode probe function that checks TCP connectivity.
func TCPProbe(host string, port int) func() bool {
	return func() bool {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 3*time.Second)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}
}

// HTTPProbe returns an InfraNode probe function that checks an HTTP endpoint.
func HTTPProbe(url string) func() bool {
	return func() bool {
		resp, err := http.Get(url)
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}
}
