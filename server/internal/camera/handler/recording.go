package handler

import (
	"net/http"
	"path/filepath"
	"strconv"
	"time"

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
	if pageSize < 1 || pageSize > 500 {
		pageSize = 20
	}
	var from, to *time.Time
	if date := c.Query("date"); date != "" {
		offsetMinute, err := strconv.Atoi(c.DefaultQuery("timezone_offset_minutes", "0"))
		if err != nil || offsetMinute < -840 || offsetMinute > 840 {
			c.JSON(400, response.Error(400, "invalid timezone offset"))
			return
		}
		location := time.FixedZone("recording-client", -offsetMinute*60)
		day, err := time.ParseInLocation("2006-01-02", date, location)
		if err != nil {
			c.JSON(400, response.Error(400, "invalid date, expected YYYY-MM-DD"))
			return
		}
		nextDay := day.AddDate(0, 0, 1)
		from, to = &day, &nextDay
	}
	recordings, total, err := h.svc.List(
		cameraID,
		httputil.GetUserID(c),
		from,
		to,
		(page-1)*pageSize,
		pageSize,
	)
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
	stream, rec, err := h.svc.OpenPlayback(recordingID, httputil.GetUserID(c))
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	defer stream.Close()
	c.Header("Content-Type", "video/mp4")
	c.Header("Content-Disposition", `inline; filename="`+filepath.Base(rec.FileName)+`"`)
	http.ServeContent(c.Writer, c.Request, rec.FileName, rec.StartedAt, stream)
}

func parseUintParam(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "invalid id"))
		return 0, false
	}
	return id, true
}
