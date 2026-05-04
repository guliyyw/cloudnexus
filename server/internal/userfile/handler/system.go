package handler

import (
	"context"
	"runtime"
	"time"

	"github.com/cloudnexus/server/pkg/logger"
	"github.com/cloudnexus/server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
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
