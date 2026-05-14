package handler

import (
	"net/http"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) HandleGetStatus(c *gin.Context) {
	status, err := h.svc.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取仪表盘状态失败"))
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(status))
}
