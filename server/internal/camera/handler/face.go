package handler

import (
	"strconv"

	"github.com/cloudnexus/server/internal/camera/service"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/response"

	"github.com/gin-gonic/gin"
)

type FaceHandler struct {
	svc *service.FaceService
}

func NewFaceHandler(svc *service.FaceService) *FaceHandler {
	return &FaceHandler{svc: svc}
}

// HandleGetThumbnail serves a face profile's thumbnail image from MinIO.
// Uses query token auth for <img> tag compatibility.
func (h *FaceHandler) HandleGetThumbnail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的人脸 ID"))
		return
	}
	data, contentType, err := h.svc.GetThumbnail(id)
	if err != nil {
		c.JSON(404, response.Error(404, "缩略图不存在"))
		return
	}
	c.Data(200, contentType, data)
}

// HandleListProfiles returns all face profiles for the current user.
func (h *FaceHandler) HandleListProfiles(c *gin.Context) {
	profiles, err := h.svc.ListProfiles(getUserID(c))
	if err != nil {
		c.JSON(500, response.Error(500, "查询人脸库失败"))
		return
	}
	if profiles == nil {
		profiles = []model.FaceProfile{}
	}
	c.JSON(200, response.OKWithData(gin.H{"profiles": profiles}))
}

// HandleCreateProfile registers a new face profile.
func (h *FaceHandler) HandleCreateProfile(c *gin.Context) {
	var req struct {
		Name         string    `json:"name"`
		Embedding    []float64 `json:"embedding"`
		ThumbnailURL string    `json:"thumbnail_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误: name 和 embedding 必填"))
		return
	}
	if len(req.Embedding) == 0 {
		c.JSON(400, response.Error(400, "embedding 不能为空"))
		return
	}
	p, err := h.svc.CreateProfile(getUserID(c), req.Name, req.Embedding, req.ThumbnailURL)
	if err != nil {
		c.JSON(400, response.Error(400, err.Error()))
		return
	}
	c.JSON(201, response.OKWithData(gin.H{"profile": p}))
}

// HandleUpdateProfile renames a face profile.
func (h *FaceHandler) HandleUpdateProfile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的人脸 ID"))
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(400, response.Error(400, "参数错误: name 必填"))
		return
	}
	if err := h.svc.UpdateProfile(id, getUserID(c), req.Name); err != nil {
		if appErr, ok := err.(interface{ Code() int }); ok {
			c.JSON(appErr.Code(), response.Error(appErr.Code(), err.Error()))
			return
		}
		c.JSON(500, response.Error(500, err.Error()))
		return
	}
	c.JSON(200, response.OK("更新成功"))
}

// HandleDeleteProfile deletes a face profile.
func (h *FaceHandler) HandleDeleteProfile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的人脸 ID"))
		return
	}
	if err := h.svc.DeleteProfile(id, getUserID(c)); err != nil {
		if appErr, ok := err.(interface{ Code() int }); ok {
			c.JSON(appErr.Code(), response.Error(appErr.Code(), err.Error()))
			return
		}
		c.JSON(500, response.Error(500, err.Error()))
		return
	}
	c.JSON(200, response.OK("删除成功"))
}

// HandleMatchFace compares a face embedding against the library.
func (h *FaceHandler) HandleMatchFace(c *gin.Context) {
	var req struct {
		Embedding []float64 `json:"embedding"`
		CameraID  string    `json:"camera_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Embedding) == 0 {
		c.JSON(400, response.Error(400, "参数错误: embedding 必填"))
		return
	}
	result, err := h.svc.MatchEmbedding(getUserID(c), req.Embedding)
	if err != nil {
		c.JSON(500, response.Error(500, "匹配失败"))
		return
	}

	// Record event if camera_id is provided
	if req.CameraID != "" {
		camID, _ := strconv.ParseUint(req.CameraID, 10, 64)
		if camID > 0 {
			h.svc.RecordEvent(camID, result, nil)
			// Also record attendance session
			h.svc.RecordAttendance(camID, result)
		}
	}

	c.JSON(200, response.OKWithData(result))
}

// HandleGetDailyAttendance returns daily check-in/check-out summary.
func (h *FaceHandler) HandleGetDailyAttendance(c *gin.Context) {
	date := c.DefaultQuery("date", "")
	if date == "" {
		c.JSON(400, response.Error(400, "date 参数必填"))
		return
	}
	attendance, err := h.svc.GetDailyAttendance(date)
	if err != nil {
		c.JSON(500, response.Error(500, "查询考勤失败"))
		return
	}
	if attendance == nil {
		attendance = []service.DailyAttendance{}
	}
	c.JSON(200, response.OKWithData(gin.H{
		"items": attendance,
		"date":  date,
	}))
}

// HandleGetAttendanceByFace returns attendance sessions for a specific face.
func (h *FaceHandler) HandleGetAttendanceByFace(c *gin.Context) {
	faceID, err := strconv.ParseUint(c.Query("face_id"), 10, 64)
	if err != nil || faceID == 0 {
		c.JSON(400, response.Error(400, "face_id 参数必填"))
		return
	}
	dateFrom := c.DefaultQuery("date_from", "")
	dateTo := c.DefaultQuery("date_to", "")
	sessions, err := h.svc.GetAttendanceByFace(faceID, dateFrom, dateTo)
	if err != nil {
		c.JSON(500, response.Error(500, "查询考勤失败"))
		return
	}
	if sessions == nil {
		sessions = []model.FaceAttendanceSession{}
	}
	c.JSON(200, response.OKWithData(gin.H{
		"items": sessions,
	}))
}

// HandleListFaceEvents returns face recognition events for a camera.
func (h *FaceHandler) HandleListFaceEvents(c *gin.Context) {
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
		c.JSON(500, response.Error(500, "查询人脸识别事件失败"))
		return
	}
	if events == nil {
		events = []model.FaceRecognitionEvent{}
	}
	c.JSON(200, response.OKWithData(gin.H{
		"items":     events,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}
