package handler

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/cloudnexus/server/pkg/logger"
	"github.com/cloudnexus/server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"gorm.io/gorm"
)

var startTime = time.Now()

type SystemHandler struct {
	db    *gorm.DB
	minio *minio.Client
}

func NewSystemHandler(db *gorm.DB, minioClient *minio.Client) *SystemHandler {
	return &SystemHandler{db: db, minio: minioClient}
}

func (h *SystemHandler) HandleHealthz(c *gin.Context) {
	components := make(map[string]string)

	sqlDB, err := h.db.DB()
	if err != nil {
		components["database"] = "error: " + err.Error()
	} else if err := sqlDB.Ping(); err != nil {
		components["database"] = "error: " + err.Error()
	} else {
		components["database"] = "ok"
	}

	if _, err := h.minio.ListBuckets(context.Background()); err != nil {
		components["minio"] = "error: " + err.Error()
	} else {
		components["minio"] = "ok"
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.JSON(200, gin.H{
		"status":     "ok",
		"service":    "user-file-svc",
		"uptime":     time.Since(startTime).String(),
		"go_version": runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
		"memory_mb":  memStats.Alloc / 1024 / 1024,
		"components": components,
	})
}

func (h *SystemHandler) HandleMetrics(c *gin.Context) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.JSON(200, response.OKWithData(gin.H{
		"uptime_seconds": int(time.Since(startTime).Seconds()),
		"goroutines":     runtime.NumGoroutine(),
		"heap_alloc_mb":  memStats.HeapAlloc / 1024 / 1024,
		"heap_sys_mb":    memStats.HeapSys / 1024 / 1024,
		"stack_inuse_kb": memStats.StackInuse / 1024,
		"num_gc":         memStats.NumGC,
		"go_version":     runtime.Version(),
		"num_cpu":        runtime.NumCPU(),
	}))
}

func (h *SystemHandler) HandleLogs(c *gin.Context) {
	level := c.DefaultQuery("level", "")
	requestID := c.DefaultQuery("request_id", "")
	userID := c.DefaultQuery("user_id", "")
	service := c.DefaultQuery("service", "")
	limit := 200

	var logs []logger.LogEntry
	if service != "" {
		date := c.DefaultQuery("date", "")
		logs = logger.ReadLogFile(service, date, limit)
	} else {
		logs = logger.QueryLogs(level, requestID, userID, limit)
	}
	if logs == nil {
		logs = []logger.LogEntry{}
	}
	c.JSON(200, response.OKWithData(gin.H{"logs": logs, "total": len(logs)}))
}

// HandleLogServices returns available service names from log files.
func (h *SystemHandler) HandleLogServices(c *gin.Context) {
	services := logger.ListLogServices()
	c.JSON(200, response.OKWithData(gin.H{"services": services}))
}

// HandleLogFiles returns a list of available log date directories with sizes.
func (h *SystemHandler) HandleLogFiles(c *gin.Context) {
	files := logger.ListLogFiles()
	c.JSON(200, response.OKWithData(gin.H{"files": files}))
}

// HandleLogDownload serves a log file for a given date as a downloadable attachment.
func (h *SystemHandler) HandleLogDownload(c *gin.Context) {
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	if _, err := time.Parse("2006-01-02", date); err != nil {
		c.JSON(400, response.Error(400, "无效的日期格式，请使用 YYYY-MM-DD"))
		return
	}

	filePath := logger.GetLogFilePath(date, "user-file-svc")
	if filePath == "" {
		c.JSON(404, response.Error(404, "日志系统未配置"))
		return
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, response.Error(404, "该日期没有日志文件"))
		return
	}
	c.FileAttachment(filePath, fmt.Sprintf("cloudnexus-logs-%s.log", date))
}

type resourceMetrics struct {
	CPUPercent     float64 `json:"cpu_percent"`
	MemTotalMB     uint64  `json:"mem_total_mb"`
	MemUsedMB      uint64  `json:"mem_used_mb"`
	MemPercent     float64 `json:"mem_percent"`
	DiskTotalMB    uint64  `json:"disk_total_mb"`
	DiskUsedMB     uint64  `json:"disk_used_mb"`
	DiskPercent    float64 `json:"disk_percent"`
	DiskPath       string  `json:"disk_path"`
	NetBytesRecv   uint64  `json:"net_bytes_recv"`
	NetBytesSent   uint64  `json:"net_bytes_sent"`
	NetPacketsRecv uint64  `json:"net_packets_recv"`
	NetPacketsSent uint64  `json:"net_packets_sent"`
}

var (
	prevNetRecv  uint64
	prevNetSent  uint64
	netMu        sync.Mutex
	lastNetCheck time.Time
)

func (h *SystemHandler) HandleResourceMetrics(c *gin.Context) {
	rm := resourceMetrics{}

	// CPU
	if percents, err := cpu.PercentWithContext(context.Background(), 0, false); err == nil && len(percents) > 0 {
		rm.CPUPercent = round1(percents[0])
	}

	// Memory
	if vmem, err := mem.VirtualMemoryWithContext(context.Background()); err == nil {
		rm.MemTotalMB = vmem.Total / 1024 / 1024
		rm.MemUsedMB = vmem.Used / 1024 / 1024
		rm.MemPercent = round1(vmem.UsedPercent)
	}

	// Disk (root partition)
	rm.DiskPath = "/"
	if du, err := disk.UsageWithContext(context.Background(), "/"); err == nil {
		rm.DiskTotalMB = du.Total / 1024 / 1024
		rm.DiskUsedMB = du.Used / 1024 / 1024
		rm.DiskPercent = round1(du.UsedPercent)
	}

	// Network (cumulative, report since last check as delta-per-second)
	netMu.Lock()
	if counters, err := net.IOCountersWithContext(context.Background(), false); err == nil && len(counters) > 0 {
		c := counters[0]
		if prevNetRecv > 0 {
			elapsed := time.Since(lastNetCheck).Seconds()
			if elapsed > 0 {
				rm.NetBytesRecv = uint64(float64(c.BytesRecv-prevNetRecv) / elapsed)
				rm.NetBytesSent = uint64(float64(c.BytesSent-prevNetSent) / elapsed)
				rm.NetPacketsRecv = uint64(float64(c.PacketsRecv) / elapsed)
				rm.NetPacketsSent = uint64(float64(c.PacketsSent) / elapsed)
			}
		}
		prevNetRecv = c.BytesRecv
		prevNetSent = c.BytesSent
	}
	lastNetCheck = time.Now()
	netMu.Unlock()

	c.JSON(200, response.OKWithData(rm))
}

// --- historical metrics ---

// MetricSnapshot represents a point-in-time system metrics reading.
type MetricSnapshot struct {
	Timestamp   time.Time `json:"timestamp"`
	UptimeSec   int       `json:"uptime_seconds"`
	Goroutines  int       `json:"goroutines"`
	HeapAllocMB float64   `json:"heap_alloc_mb"`
	CPUPercent  float64   `json:"cpu_percent"`
	MemPercent  float64   `json:"mem_percent"`
}

const maxMetricsHistory = 300

var (
	metricsHistoryMu sync.Mutex
	metricsHistory   = make([]MetricSnapshot, 0, maxMetricsHistory)
)

// collectMetricsSnapshot reads current metrics and appends to the ring buffer.
func collectMetricsSnapshot() {
	snap := MetricSnapshot{
		Timestamp:  time.Now(),
		UptimeSec:  int(time.Since(startTime).Seconds()),
		Goroutines: runtime.NumGoroutine(),
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	snap.HeapAllocMB = float64(memStats.HeapAlloc) / 1024 / 1024

	if percents, err := cpu.PercentWithContext(context.Background(), 0, false); err == nil && len(percents) > 0 {
		snap.CPUPercent = round1(percents[0])
	}
	if vmem, err := mem.VirtualMemoryWithContext(context.Background()); err == nil {
		snap.MemPercent = round1(vmem.UsedPercent)
	}

	metricsHistoryMu.Lock()
	if len(metricsHistory) >= maxMetricsHistory {
		metricsHistory = metricsHistory[1:]
	}
	metricsHistory = append(metricsHistory, snap)
	metricsHistoryMu.Unlock()
}

// StartMetricsCollector launches a background goroutine that collects
// metric snapshots every 10 seconds.
func (h *SystemHandler) StartMetricsCollector() {
	collectMetricsSnapshot()
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			collectMetricsSnapshot()
		}
	}()
}

// HandleMetricsHistory returns the last N metric snapshots.
func (h *SystemHandler) HandleMetricsHistory(c *gin.Context) {
	n := 60
	if qn, err := strconv.Atoi(c.DefaultQuery("n", "60")); err == nil && qn > 0 && qn <= maxMetricsHistory {
		n = qn
	}

	metricsHistoryMu.Lock()
	start := 0
	if len(metricsHistory) > n {
		start = len(metricsHistory) - n
	}
	result := make([]MetricSnapshot, len(metricsHistory)-start)
	copy(result, metricsHistory[start:])
	metricsHistoryMu.Unlock()

	if result == nil {
		result = []MetricSnapshot{}
	}
	c.JSON(200, response.OKWithData(gin.H{"snapshots": result}))
}

func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}
