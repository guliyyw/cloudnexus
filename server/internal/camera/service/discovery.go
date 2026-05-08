package service

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	apperrors "github.com/cloudnexus/server/pkg/errors"
)

// DiscoverRequest is the JSON body for POST /cameras/discover.
type DiscoverRequest struct {
	Subnet string `json:"subnet"` // e.g. "192.168.1.0/24"
	Ports  []int  `json:"ports"`  // defaults to [554, 8554, 80, 8080]
}

// DiscoveredCamera is one confirmed camera found on the network.
type DiscoveredCamera struct {
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	RTSPURL string `json:"rtsp_url"`
	Source  string `json:"source"` // "rtsp_probe" or "onvif"
}

// DiscoverResponse is the API response.
type DiscoverResponse struct {
	Cameras        []DiscoveredCamera `json:"cameras"`
	ScanDurationMs int64              `json:"scan_duration_ms"`
	TotalScanned   int                `json:"total_scanned"`
	OpenPorts      int                `json:"open_ports"`
}

var defaultPorts = []int{554, 8554, 80, 8080}

// DiscoverCameras scans the given subnet for RTSP/ONVIF cameras.
func (s *CameraService) DiscoverCameras(req DiscoverRequest) (*DiscoverResponse, error) {
	if req.Subnet == "" {
		return nil, apperrors.NewAppError(400, "子网不能为空，示例: 192.168.1.0/24", apperrors.ErrBadRequest)
	}
	if len(req.Ports) == 0 {
		req.Ports = defaultPorts
	}

	prefix, err := netip.ParsePrefix(req.Subnet)
	if err != nil {
		return nil, apperrors.NewAppError(400, "子网格式无效，示例: 192.168.1.0/24", apperrors.ErrBadRequest)
	}
	if !prefix.Addr().Is4() {
		return nil, apperrors.NewAppError(400, "仅支持 IPv4 子网", apperrors.ErrBadRequest)
	}
	ones := prefix.Bits()
	if 32-ones > 8 {
		return nil, apperrors.NewAppError(400, "子网过大，最多扫描 256 个地址 (/24)", apperrors.ErrBadRequest)
	}

	start := time.Now()

	addrs := expandPrefix(prefix)
	totalScanned := len(addrs)

	// Phase 1: TCP port scan
	openResults := runTCPScan(addrs, req.Ports, 50, 2*time.Second)

	// Phase 2: RTSP probe on open ports
	var discovered []DiscoveredCamera
	for _, r := range openResults {
		if probeRTSP(r.ip, r.port, 3*time.Second) {
			discovered = append(discovered, DiscoveredCamera{
				IP:      r.ip.String(),
				Port:    r.port,
				RTSPURL: fmt.Sprintf("rtsp://%s:%d/", r.ip.String(), r.port),
				Source:  "rtsp_probe",
			})
		} else {
			// If port is open but RTSP probe failed, still report as candidate
			// (could be an ONVIF camera with RTSP on a different port)
			discovered = append(discovered, DiscoveredCamera{
				IP:      r.ip.String(),
				Port:    r.port,
				RTSPURL: fmt.Sprintf("rtsp://%s:%d/", r.ip.String(), r.port),
				Source:  "tcp_open",
			})
		}
	}

	// Phase 3: ONVIF WS-Discovery (best-effort)
	onvifCameras := probeONVIF(5 * time.Second)
	discovered = mergeDiscovered(discovered, onvifCameras)

	return &DiscoverResponse{
		Cameras:        discovered,
		ScanDurationMs: time.Since(start).Milliseconds(),
		TotalScanned:   totalScanned,
		OpenPorts:      len(openResults),
	}, nil
}

// expandPrefix returns all IPv4 addresses in a prefix.
func expandPrefix(prefix netip.Prefix) []netip.Addr {
	addr := prefix.Addr()
	var addrs []netip.Addr
	for {
		if !prefix.Contains(addr) {
			break
		}
		addrs = append(addrs, addr)
		addr = addr.Next()
	}
	return addrs
}

type scanResult struct {
	ip   netip.Addr
	port int
	open bool
}

// runTCPScan concurrently dials IP:port pairs.
func runTCPScan(addrs []netip.Addr, ports []int, workers int, timeout time.Duration) []scanResult {
	jobs := make(chan struct {
		ip   netip.Addr
		port int
	}, len(addrs)*len(ports))
	results := make(chan scanResult, len(addrs)*len(ports))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				addr := net.JoinHostPort(job.ip.String(), strconv.Itoa(job.port))
				conn, err := net.DialTimeout("tcp", addr, timeout)
				if err == nil {
					conn.Close()
					results <- scanResult{ip: job.ip, port: job.port, open: true}
				}
			}
		}()
	}

	for _, addr := range addrs {
		for _, port := range ports {
			jobs <- struct {
				ip   netip.Addr
				port int
			}{addr, port}
		}
	}
	close(jobs)
	wg.Wait()
	close(results)

	var open []scanResult
	for r := range results {
		if r.open {
			open = append(open, r)
		}
	}
	return open
}

// probeRTSP sends an RTSP OPTIONS request to verify the target is an RTSP server.
func probeRTSP(ip netip.Addr, port int, timeout time.Duration) bool {
	addr := net.JoinHostPort(ip.String(), strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	// Try OPTIONS * first (most compatible), then OPTIONS with URL
	reqs := []string{
		fmt.Sprintf("OPTIONS * RTSP/1.0\r\nCSeq: 1\r\nUser-Agent: CloudNexus/1.0\r\n\r\n"),
		fmt.Sprintf("OPTIONS rtsp://%s/ RTSP/1.0\r\nCSeq: 1\r\nUser-Agent: CloudNexus/1.0\r\n\r\n", addr),
	}

	for _, req := range reqs {
		conn.Write([]byte(req))
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			continue
		}
		resp := string(buf[:n])
		if strings.HasPrefix(resp, "RTSP/1.0 200") || strings.HasPrefix(resp, "RTSP/2.0 200") {
			return true
		}
		// Some cameras (Dahua) return "RTSP/1.0 401" (auth required) — still a camera.
		if strings.HasPrefix(resp, "RTSP/1.0 401") || strings.HasPrefix(resp, "RTSP/1.0 400") {
			return true
		}
	}
	return false
}

// --- ONVIF WS-Discovery ---

var xaddrsRe = regexp.MustCompile(`<[^>]*XAddrs[^>]*>([^<]+)</[^>]*XAddrs[^>]*>`)

func probeONVIF(timeout time.Duration) []DiscoveredCamera {
	// WS-Discovery multicast address
	maddr := &net.UDPAddr{
		IP:   net.ParseIP("239.255.255.250"),
		Port: 3702,
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, maddr)
	if err != nil {
		return nil
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	// Build WS-Discovery Probe SOAP message
	soap := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<e:Envelope xmlns:e="http://www.w3.org/2003/05/soap-envelope"
            xmlns:w="http://schemas.xmlsoap.org/ws/2004/08/addressing"
            xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery"
            xmlns:dn="http://www.onvif.org/ver10/network/wsdl">
  <e:Header>
    <w:MessageID>urn:uuid:cloudnexus-discover</w:MessageID>
    <w:To>urn:schemas-xmlsoap-org:ws:2005:04:discovery</w:To>
    <w:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</w:Action>
  </e:Header>
  <e:Body>
    <d:Probe>
      <d:Types>dn:NetworkVideoTransmitter</d:Types>
    </d:Probe>
  </e:Body>
</e:Envelope>`)

	conn.WriteTo([]byte(soap), maddr)

	var cameras []DiscoveredCamera
	buf := make([]byte, 65507)
	seen := make(map[string]bool)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			break // timeout or error — stop collecting
		}
		resp := string(buf[:n])
		addrs := extractXAddrs(resp)
		for _, xaddr := range addrs {
			ip := extractIPFromURL(xaddr)
			if ip == "" {
				ip = remoteAddr.IP.String()
			}
			key := ip
			if seen[key] {
				continue
			}
			seen[key] = true
			cameras = append(cameras, DiscoveredCamera{
				IP:      ip,
				Port:    554,
				RTSPURL: fmt.Sprintf("rtsp://%s:554/", ip),
				Source:  "onvif",
			})
		}
	}
	return cameras
}

func extractXAddrs(soapResponse string) []string {
	matches := xaddrsRe.FindStringSubmatch(soapResponse)
	if len(matches) < 2 {
		return nil
	}
	return strings.Fields(matches[1])
}

func extractIPFromURL(rawURL string) string {
	// rawURL may be "http://192.168.1.100/onvif/device_service" or just a bare IP
	s := rawURL
	if idx := strings.Index(s, "://"); idx != -1 {
		s = s[idx+3:]
	}
	if idx := strings.IndexByte(s, '/'); idx != -1 {
		s = s[:idx]
	}
	if idx := strings.LastIndexByte(s, ':'); idx != -1 {
		s = s[:idx]
	}
	return s
}

func mergeDiscovered(rtsp, onvif []DiscoveredCamera) []DiscoveredCamera {
	seen := make(map[string]DiscoveredCamera)
	for _, c := range rtsp {
		key := fmt.Sprintf("%s:%d", c.IP, c.Port)
		seen[key] = c
	}
	for _, c := range onvif {
		key := fmt.Sprintf("%s:%d", c.IP, c.Port)
		if existing, ok := seen[key]; ok {
			if existing.Source == "tcp_open" && c.Source == "onvif" {
				existing.Source = "onvif"
			}
			seen[key] = existing
		} else {
			seen[key] = c
		}
	}
	result := make([]DiscoveredCamera, 0, len(seen))
	for _, c := range seen {
		result = append(result, c)
	}
	return result
}

// _ ensures context import is used (for future cancellation support)
var _ = context.Background
