package handler

import (
	"net/http"
	"time"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type HealthHistoryHandler struct {
	svc *service.HealthHistoryService
}

func NewHealthHistoryHandler(svc *service.HealthHistoryService) *HealthHistoryHandler {
	return &HealthHistoryHandler{svc: svc}
}

func (h *HealthHistoryHandler) HandleGetHistory(c *gin.Context) {
	since := parseSince(c.DefaultQuery("range", "24h"))
	snapshots, err := h.svc.GetHealthHistory(since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取健康历史失败"))
		return
	}
	if snapshots == nil {
		snapshots = []service.HealthSnapshot{}
	}
	c.JSON(http.StatusOK, response.OKWithData(map[string]interface{}{
		"snapshots": snapshots,
	}))
}

func (h *HealthHistoryHandler) HandleGetResources(c *gin.Context) {
	since := parseSince(c.DefaultQuery("range", "24h"))
	svc := c.DefaultQuery("service", "all")
	data, err := h.svc.GetResourceHistory(since, svc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取资源历史失败"))
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(data))
}

func parseSince(rangeStr string) time.Time {
	switch rangeStr {
	case "1h":
		return time.Now().Add(-1 * time.Hour)
	case "6h":
		return time.Now().Add(-6 * time.Hour)
	case "12h":
		return time.Now().Add(-12 * time.Hour)
	case "7d":
		return time.Now().Add(-7 * 24 * time.Hour)
	default:
		return time.Now().Add(-24 * time.Hour)
	}
}
