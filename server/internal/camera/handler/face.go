package handler

import (
	"strconv"

	"github.com/cloudnexus/server/internal/camera/service"
	"github.com/cloudnexus/server/pkg/httputil"
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
	profiles, err := h.svc.ListProfiles(httputil.GetUserID(c))
	if err != nil {
		httputil.HandleError(c, err)
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
		httputil.BadRequest(c, "参数错误: name 和 embedding 必填")
		return
	}
	if len(req.Embedding) == 0 {
		httputil.BadRequest(c, "embedding 不能为空")
		return
	}
	p, err := h.svc.CreateProfile(httputil.GetUserID(c), req.Name, req.Embedding, req.ThumbnailURL)
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(201, response.OKWithData(gin.H{"profile": p}))
}

// HandleUpdateProfile renames a face profile.
func (h *FaceHandler) HandleUpdateProfile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httputil.BadRequest(c, "无效的人脸 ID")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		httputil.BadRequest(c, "参数错误: name 必填")
		return
	}
	if err := h.svc.UpdateProfile(id, httputil.GetUserID(c), req.Name); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(200, response.OK("更新成功"))
}

// HandleDeleteProfile deletes a face profile.
func (h *FaceHandler) HandleDeleteProfile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httputil.BadRequest(c, "无效的人脸 ID")
		return
	}
	if err := h.svc.DeleteProfile(id, httputil.GetUserID(c)); err != nil {
		httputil.HandleError(c, err)
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
		httputil.BadRequest(c, "参数错误: embedding 必填")
		return
	}
	result, err := h.svc.MatchEmbedding(httputil.GetUserID(c), req.Embedding)
	if err != nil {
		httputil.HandleError(c, err)
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
		httputil.BadRequest(c, "date 参数必填")
		return
	}
	attendance, err := h.svc.GetDailyAttendance(date)
	if err != nil {
		httputil.HandleError(c, err)
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
		httputil.BadRequest(c, "face_id 参数必填")
		return
	}
	dateFrom := c.DefaultQuery("date_from", "")
	dateTo := c.DefaultQuery("date_to", "")
	sessions, err := h.svc.GetAttendanceByFace(faceID, dateFrom, dateTo)
	if err != nil {
		httputil.HandleError(c, err)
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
		httputil.BadRequest(c, "无效的摄像头 ID")
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
		httputil.HandleError(c, err)
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

// HandleClearFaceEvents deletes all face recognition events for a camera.
// Only clears recognition records, NOT attendance data.
func (h *FaceHandler) HandleClearFaceEvents(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httputil.BadRequest(c, "无效的摄像头 ID")
		return
	}
	count, err := h.svc.ClearFaceEvents(id)
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"deleted": count}))
}

// HandleDeleteAttendanceSession deletes a single attendance session.
func (h *FaceHandler) HandleDeleteAttendanceSession(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httputil.BadRequest(c, "无效的考勤记录 ID")
		return
	}
	if err := h.svc.DeleteSession(id, httputil.GetUserID(c)); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(200, response.OK("删除成功"))
}

// HandleClearAttendance deletes all attendance sessions for a face on a date.
func (h *FaceHandler) HandleClearAttendance(c *gin.Context) {
	faceID, err := strconv.ParseUint(c.Query("face_id"), 10, 64)
	if err != nil || faceID == 0 {
		httputil.BadRequest(c, "face_id 参数必填")
		return
	}
	date := c.Query("date")
	if date == "" {
		httputil.BadRequest(c, "date 参数必填")
		return
	}
	count, err := h.svc.ClearAttendanceByFaceDate(faceID, httputil.GetUserID(c), date)
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"deleted": count}))
}

// HandleGetAttendanceStatus returns all personnel with check-in status for a date.
func (h *FaceHandler) HandleGetAttendanceStatus(c *gin.Context) {
	date := c.DefaultQuery("date", "")
	if date == "" {
		httputil.BadRequest(c, "date 参数必填")
		return
	}
	items, signedCount, unsignedCount, err := h.svc.GetAttendanceStatus(httputil.GetUserID(c), date)
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	if items == nil {
		items = []service.AttendanceStatusItem{}
	}
	c.JSON(200, response.OKWithData(gin.H{
		"items":         items,
		"date":          date,
		"total":         len(items),
		"signed_count":  signedCount,
		"unsigned_count": unsignedCount,
	}))
}
