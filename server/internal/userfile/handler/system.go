package handler

import (
	"context"
	"runtime"
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
	limit := 200
	logs := logger.QueryLogs(level, limit)
	if logs == nil {
		logs = []logger.LogEntry{}
	}
	c.JSON(200, response.OKWithData(gin.H{"logs": logs, "total": len(logs)}))
}

type resourceMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemTotalMB    uint64  `json:"mem_total_mb"`
	MemUsedMB     uint64  `json:"mem_used_mb"`
	MemPercent    float64 `json:"mem_percent"`
	DiskTotalMB   uint64  `json:"disk_total_mb"`
	DiskUsedMB    uint64  `json:"disk_used_mb"`
	DiskPercent   float64 `json:"disk_percent"`
	DiskPath      string  `json:"disk_path"`
	NetBytesRecv  uint64  `json:"net_bytes_recv"`
	NetBytesSent  uint64  `json:"net_bytes_sent"`
	NetPacketsRecv uint64 `json:"net_packets_recv"`
	NetPacketsSent uint64 `json:"net_packets_sent"`
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

func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}
