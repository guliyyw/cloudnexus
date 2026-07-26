package handler

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/cloudnexus/server/internal/camera/service"
	"github.com/cloudnexus/server/pkg/httputil"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type RecordingHandler struct {
	svc *service.RecordingService
}

func NewRecordingHandler(svc *service.RecordingService) *RecordingHandler {
	return &RecordingHandler{svc: svc}
}

func (h *RecordingHandler) HandleStart(c *gin.Context) {
	cameraID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req service.RecordingOptions
	_ = c.ShouldBindJSON(&req)
	status, err := h.svc.Start(cameraID, httputil.GetUserID(c), req)
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(200, response.OKWithData(status))
}

func (h *RecordingHandler) HandleStop(c *gin.Context) {
	cameraID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Stop(cameraID, httputil.GetUserID(c)); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(200, response.OK("recording stopped"))
}

func (h *RecordingHandler) HandleStatus(c *gin.Context) {
	cameraID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	status, err := h.svc.Status(cameraID, httputil.GetUserID(c))
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(200, response.OKWithData(status))
}

func (h *RecordingHandler) HandleList(c *gin.Context) {
	cameraID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	recordings, total, err := h.svc.List(cameraID, httputil.GetUserID(c), (page-1)*pageSize, pageSize)
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	if recordings == nil {
		recordings = []model.CameraRecording{}
	}
	c.JSON(200, response.OKWithData(gin.H{
		"items":     recordings,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

func (h *RecordingHandler) HandleDelete(c *gin.Context) {
	recordingID, ok := parseUintParam(c, "recording_id")
	if !ok {
		return
	}
	if err := h.svc.Delete(recordingID, httputil.GetUserID(c)); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(200, response.OK("recording deleted"))
}

func (h *RecordingHandler) HandlePlayback(c *gin.Context) {
	recordingID, ok := parseUintParam(c, "recording_id")
	if !ok {
		return
	}
	rec, err := h.svc.Get(recordingID, httputil.GetUserID(c))
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.Header("Content-Type", "video/mp4")
	c.Header("Content-Disposition", `inline; filename="`+filepath.Base(rec.FileName)+`"`)
	http.ServeFile(c.Writer, c.Request, rec.FilePath)
}

func parseUintParam(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "invalid id"))
		return 0, false
	}
	return id, true
}
