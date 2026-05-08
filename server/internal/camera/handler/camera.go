package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/camera/service"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/cloudnexus/server/pkg/snowflake"

	"github.com/gin-gonic/gin"
)

type CameraHandler struct {
	svc *service.CameraService
	rec *service.RecognitionService
}

func NewCameraHandler(svc *service.CameraService, rec *service.RecognitionService) *CameraHandler {
	return &CameraHandler{svc: svc, rec: rec}
}

func getUserID(c *gin.Context) uint64 {
	v, _ := c.Get("user_id")
	id, _ := v.(uint64)
	return id
}

// --- Camera CRUD ---

func (h *CameraHandler) HandleListCameras(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	cameras, total, err := h.svc.ListCameras(getUserID(c), offset, pageSize)
	if err != nil {
		c.JSON(500, response.Error(500, "查询摄像头列表失败"))
		return
	}
	if cameras == nil {
		cameras = []model.Camera{}
	}
	c.JSON(200, response.OKWithData(gin.H{
		"items":     cameras,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

func (h *CameraHandler) HandleCreateCamera(c *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		StreamURL string `json:"stream_url"`
		Protocol  string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误: name 和 stream_url 必填"))
		return
	}
	if req.Protocol == "" {
		req.Protocol = "rtsp"
	}
	cam := &model.Camera{
		OwnerID:   getUserID(c),
		Name:      req.Name,
		StreamURL: req.StreamURL,
		Protocol:  req.Protocol,
	}
	cam.ID = snowflake.Uint64()
	if err := h.svc.CreateCamera(cam); err != nil {
		c.JSON(400, response.Error(400, err.Error()))
		return
	}
	c.JSON(201, response.OKWithData(gin.H{"camera": cam}))
}

func (h *CameraHandler) HandleUpdateCamera(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的摄像头 ID"))
		return
	}
	var req struct {
		Name      string `json:"name"`
		StreamURL string `json:"stream_url"`
		Protocol  string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误"))
		return
	}
	cam := &model.Camera{
		BaseModel: model.BaseModel{ID: id},
		Name:      req.Name,
		StreamURL: req.StreamURL,
		Protocol:  req.Protocol,
	}
	if err := h.svc.UpdateCamera(cam); err != nil {
		c.JSON(400, response.Error(400, err.Error()))
		return
	}
	c.JSON(200, response.OK("更新成功"))
}

func (h *CameraHandler) HandleDeleteCamera(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的摄像头 ID"))
		return
	}
	if err := h.svc.DeleteCamera(id, getUserID(c)); err != nil {
		c.JSON(400, response.Error(400, err.Error()))
		return
	}
	c.JSON(200, response.OK("删除成功"))
}

// --- Stream ---

func (h *CameraHandler) HandleStartStream(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的摄像头 ID"))
		return
	}
	hlsURL, webrtcURL, err := h.svc.StartStream(id, getUserID(c))
	if err != nil {
		c.JSON(400, response.Error(400, err.Error()))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{
		"hls_url":    hlsURL,
		"webrtc_url": webrtcURL,
	}))
}

func (h *CameraHandler) HandleStopStream(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的摄像头 ID"))
		return
	}
	if err := h.svc.StopStream(id, getUserID(c)); err != nil {
		c.JSON(400, response.Error(400, err.Error()))
		return
	}
	c.JSON(200, response.OK("视频流已停止"))
}

// --- AI Recognition ---

func (h *CameraHandler) HandleStartRecognition(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的摄像头 ID"))
		return
	}
	if err := h.rec.StartRecognition(id); err != nil {
		c.JSON(400, response.Error(400, err.Error()))
		return
	}
	c.JSON(200, response.OK("AI识别已开启"))
}

func (h *CameraHandler) HandleStopRecognition(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的摄像头 ID"))
		return
	}
	if err := h.rec.StopRecognition(id); err != nil {
		c.JSON(400, response.Error(400, err.Error()))
		return
	}
	c.JSON(200, response.OK("AI识别已停止"))
}

func (h *CameraHandler) HandleListEvents(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的摄像头 ID"))
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
	offset := (page - 1) * pageSize

	events, total, err := h.svc.ListEvents(id, offset, pageSize)
	if err != nil {
		c.JSON(500, response.Error(500, "查询识别事件失败"))
		return
	}
	if events == nil {
		events = []model.RecognitionEvent{}
	}
	c.JSON(200, response.OKWithData(gin.H{
		"items":     events,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

// HandleDiscoverCameras scans the local network for RTSP/ONVIF cameras.
func (h *CameraHandler) HandleDiscoverCameras(c *gin.Context) {
	var req service.DiscoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误: 请提供 subnet 字段"))
		return
	}
	resp, err := h.svc.DiscoverCameras(req)
	if err != nil {
		if appErr, ok := err.(interface{ Code() int }); ok {
			c.JSON(appErr.Code(), response.Error(appErr.Code(), err.Error()))
			return
		}
		c.JSON(500, response.Error(500, "扫描失败: "+err.Error()))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{
		"cameras":          resp.Cameras,
		"scan_duration_ms": resp.ScanDurationMs,
		"total_scanned":    resp.TotalScanned,
		"open_ports":       resp.OpenPorts,
	}))
}

// HandleDetectImage performs one-shot AI detection on an uploaded image.
func (h *CameraHandler) HandleDetectImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "请上传图片文件"))
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(500, response.Error(500, "读取图片失败"))
		return
	}
	defer f.Close()
	buf := make([]byte, file.Size)
	f.Read(buf)
	objects, err := h.rec.DetectImage(buf)
	if err != nil {
		c.JSON(500, response.Error(500, "AI识别失败: "+err.Error()))
		return
	}
	if objects == nil {
		objects = []service.DetectedObject{}
	}
	c.JSON(200, response.OKWithData(gin.H{"objects": objects}))
}
