package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

// endpointClient wraps an HTTP client configured for one Docker daemon.
type endpointClient struct {
	client  *http.Client
	baseURL string
	name    string
	host    string
	port    int
	tls     bool
}

// EndpointManager holds multiple Docker endpoint clients.
// The "local" endpoint is always available from DOCKER_HOST env or platform default.
type EndpointManager struct {
	db        *gorm.DB
	mu        sync.RWMutex
	endpoints map[string]*endpointClient
}

// EndpointInfo is the public view of a Docker endpoint.
type EndpointInfo struct {
	Name   string `json:"name"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Status string `json:"status"`
	TLS    bool   `json:"tls"`
}

// NewEndpointManager creates an EndpointManager with the "local" endpoint.
func NewEndpointManager(db *gorm.DB) *EndpointManager {
	mgr := &EndpointManager{
		db:        db,
		endpoints: make(map[string]*endpointClient),
	}
	mgr.endpoints["local"] = mgr.buildLocalClient()
	return mgr
}

func (m *EndpointManager) buildLocalClient() *endpointClient {
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		// Windows 默认使用命名管道而非明文 TCP
		if runtime.GOOS == "windows" {
			// Windows 使用命名管道 npipe:////./pipe/docker_engine
			// 需要 github.com/Microsoft/go-winio 支持，此处暂时返回错误
			host = "npipe:////./pipe/docker_engine"
		} else {
			host = "unix:///var/run/docker.sock"
		}
	}

	ec := &endpointClient{name: "local", host: host}

	if strings.HasPrefix(host, "unix://") {
		sock := strings.TrimPrefix(host, "unix://")
		ec.baseURL = "http://localhost"
		ec.port = 0
		ec.client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", sock)
				},
			},
			Timeout: 30 * time.Second,
		}
	} else if strings.HasPrefix(host, "npipe://") {
		// Windows 命名管道 - 需要额外依赖，当前标记为不支持
		// TODO: 使用 github.com/Microsoft/go-winio 实现命名管道支持
		ec.baseURL = "http://localhost"
		ec.port = 0
		ec.client = &http.Client{
			Timeout: 30 * time.Second,
		}
	} else {
		addr := host
		if strings.HasPrefix(addr, "tcp://") {
			addr = strings.TrimPrefix(addr, "tcp://")
		}
		ec.baseURL = "http://" + addr
		ec.host = addr
		if idx := strings.LastIndex(addr, ":"); idx > 0 {
			ec.host = addr[:idx]
			fmt.Sscanf(addr[idx+1:], "%d", &ec.port)
		}
		if ec.port == 0 {
			ec.port = 2375
		}
		// SECURITY: 明文 TCP 连接不安全，应使用 TLS (端口 2376)
		// 生产环境必须配置 TLS 证书
		ec.client = &http.Client{
			Transport: &http.Transport{},
			Timeout:   30 * time.Second,
		}
	}
	return ec
}

// GetClient returns the endpoint client for the given name.
// Empty or "local" returns the built-in local client.
func (m *EndpointManager) GetClient(name string) (*endpointClient, error) {
	if name == "" {
		name = "local"
	}

	m.mu.RLock()
	ec, ok := m.endpoints[name]
	m.mu.RUnlock()
	if ok {
		return ec, nil
	}

	// Look up in DB
	if m.db == nil {
		return nil, fmt.Errorf("Docker 端点 %s 不存在（数据库不可用）", name)
	}
	var node model.DockerNode
	if err := m.db.Where("name = ? AND node_type = ?", name, "docker_endpoint").First(&node).Error; err != nil {
		return nil, fmt.Errorf("Docker 端点 %s 不存在", name)
	}

	ec, err := m.buildTLSClient(&node)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.endpoints[name] = ec
	m.mu.Unlock()
	return ec, nil
}

func (m *EndpointManager) buildTLSClient(node *model.DockerNode) (*endpointClient, error) {
	hasTLS := node.TLSCert != "" && node.TLSKey != ""
	scheme := "http"
	port := node.Port
	if port == 0 {
		if hasTLS {
			port = 2376
		} else {
			port = 2375
		}
	}

	var transport *http.Transport
	if hasTLS {
		scheme = "https"
		caPool, err := x509.SystemCertPool()
		if err != nil {
			caPool = x509.NewCertPool()
		}
		if node.CACert != "" {
			if !caPool.AppendCertsFromPEM([]byte(node.CACert)) {
				return nil, fmt.Errorf("端点 %s CA 证书解析失败", node.Name)
			}
		}
		cert, err := tls.X509KeyPair([]byte(node.TLSCert), []byte(node.TLSKey))
		if err != nil {
			return nil, fmt.Errorf("端点 %s TLS 证书/密钥无效: %w", node.Name, err)
		}
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caPool,
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			},
		}
	} else {
		transport = &http.Transport{}
	}

	baseURL := fmt.Sprintf("%s://%s:%d", scheme, node.Host, port)
	return &endpointClient{
		client:  &http.Client{Transport: transport, Timeout: 30 * time.Second},
		baseURL: baseURL,
		name:    node.Name,
		host:    node.Host,
		port:    port,
		tls:     hasTLS,
	}, nil
}

// ListEndpoints returns all available Docker endpoints.
func (m *EndpointManager) ListEndpoints() []EndpointInfo {
	result := []EndpointInfo{}

	// Local endpoint always first
	m.mu.RLock()
	local := m.endpoints["local"]
	m.mu.RUnlock()
	if local != nil {
		status := "unknown"
		if err := m.pingClient(local); err == nil {
			status = "healthy"
		} else {
			status = "offline"
		}
		result = append(result, EndpointInfo{
			Name:   "local",
			Host:   local.host,
			Port:   local.port,
			Status: status,
			TLS:    local.tls,
		})
	}

	// DB endpoints
	if m.db != nil {
		var nodes []model.DockerNode
		if err := m.db.Where("node_type = ?", "docker_endpoint").Order("name ASC").Find(&nodes).Error; err == nil {
			for _, node := range nodes {
				result = append(result, EndpointInfo{
					Name:   node.Name,
					Host:   node.Host,
					Port:   node.Port,
					Status: node.Status,
					TLS:    node.TLSCert != "" && node.TLSKey != "",
				})
			}
		}
	}
	return result
}

// PingEndpoint pings a named Docker endpoint via /_ping.
func (m *EndpointManager) PingEndpoint(name string) error {
	ec, err := m.GetClient(name)
	if err != nil {
		return err
	}
	return m.pingClient(ec)
}

func (m *EndpointManager) pingClient(ec *endpointClient) error {
	resp, err := ec.client.Get(ec.baseURL + "/_ping")
	if err != nil {
		return fmt.Errorf("Docker 端点 %s 不可达: %w", ec.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Docker 端点 %s 返回状态 %d", ec.name, resp.StatusCode)
	}
	return nil
}

// RefreshEndpoints clears the cached remote clients so they are rebuilt on next use.
func (m *EndpointManager) RefreshEndpoints() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name := range m.endpoints {
		if name != "local" {
			delete(m.endpoints, name)
		}
	}
}

// PingAll pings all registered endpoints and updates their status in the DB.
func (m *EndpointManager) PingAll() {
	eps := m.ListEndpoints()
	for _, ep := range eps {
		if ep.Name == "local" {
			continue
		}
		err := m.PingEndpoint(ep.Name)
		now := time.Now()
		status := "healthy"
		if err != nil {
			status = "offline"
		}
		if m.db != nil {
			m.db.Model(&model.DockerNode{}).Where("name = ?", ep.Name).Updates(map[string]interface{}{
				"status":         status,
				"last_heartbeat": now,
			})
		}
	}
}
