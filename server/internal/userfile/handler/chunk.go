package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type ChunkHandler struct {
	svc *service.ChunkService
}

func NewChunkHandler(svc *service.ChunkService) *ChunkHandler {
	return &ChunkHandler{svc: svc}
}

type initReq struct {
	FileName string `json:"file_name" binding:"required"`
	FileSize int64  `json:"file_size" binding:"required"`
	ParentID uint64 `json:"parent_id,string"`
	MimeType string `json:"mime_type"`
}

func (h *ChunkHandler) HandleInitUpload(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req initReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误：请提供 file_name 和 file_size"))
		return
	}

	chunk, err := h.svc.InitUpload(userID, req.ParentID, req.FileName, req.FileSize, req.MimeType)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"upload_id":    chunk.UploadID,
		"chunk_size":   chunk.ChunkSize,
		"total_chunks": chunk.TotalChunks,
	}))
}

func (h *ChunkHandler) HandleUploadChunk(c *gin.Context) {
	userID := c.GetUint64("user_id")
	uploadID := c.PostForm("upload_id")
	chunkIdxStr := c.PostForm("chunk_index")

	if uploadID == "" || chunkIdxStr == "" {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误：需要 upload_id 和 chunk_index"))
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIdxStr)
	if err != nil || chunkIndex < 0 {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的 chunk_index"))
		return
	}

	file, fh, err := c.Request.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "请选择分片文件"))
		return
	}
	file.Close()

	completed, totalChunks, err := h.svc.UploadChunk(userID, uploadID, int32(chunkIndex), fh)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"chunk_index":  chunkIndex,
		"completed":    completed,
		"total_chunks": totalChunks,
		"received":     true,
	}))
}

func (h *ChunkHandler) HandleGetStatus(c *gin.Context) {
	userID := c.GetUint64("user_id")
	uploadID := c.Param("uploadId")
	if uploadID == "" {
		c.JSON(http.StatusBadRequest, response.Error(400, "需要 upload_id"))
		return
	}

	chunk, err := h.svc.GetStatus(userID, uploadID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"upload_id":    chunk.UploadID,
		"file_name":    chunk.FileName,
		"file_size":    chunk.FileSize,
		"total_chunks": chunk.TotalChunks,
		"completed":    chunk.Completed,
		"chunk_size":   chunk.ChunkSize,
		"status":       chunk.Status,
		"created_at":   chunk.CreatedAt,
	}))
}

type completeReq struct {
	UploadID       string `json:"upload_id" binding:"required"`
	VersionMessage string `json:"version_message"`
}

func (h *ChunkHandler) HandleComplete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req completeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误：需要 upload_id"))
		return
	}

	file, err := h.svc.CompleteUpload(userID, req.UploadID, req.VersionMessage)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.OKWithData(file))
}

func (h *ChunkHandler) HandleCancel(c *gin.Context) {
	userID := c.GetUint64("user_id")
	uploadID := c.Param("uploadId")
	if uploadID == "" {
		c.JSON(http.StatusBadRequest, response.Error(400, "需要 upload_id"))
		return
	}

	if err := h.svc.CancelUpload(userID, uploadID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已取消"))
}

func (h *ChunkHandler) HandleListIncomplete(c *gin.Context) {
	userID := c.GetUint64("user_id")

	uploads, err := h.svc.ListIncomplete(userID)
	if err != nil {
		handleError(c, err)
		return
	}

	type item struct {
		UploadID       string `json:"upload_id"`
		FileName       string `json:"file_name"`
		FileSize       int64  `json:"file_size"`
		TotalChunks    int    `json:"total_chunks"`
		CompletedCount int    `json:"completed_count"`
		ChunkSize      int    `json:"chunk_size"`
		Status         string `json:"status"`
		CreatedAt      string `json:"created_at"`
	}

	items := make([]item, 0, len(uploads))
	for _, u := range uploads {
		items = append(items, item{
			UploadID:       u.UploadID,
			FileName:       u.FileName,
			FileSize:       u.FileSize,
			TotalChunks:    u.TotalChunks,
			CompletedCount: len(u.Completed),
			ChunkSize:      u.ChunkSize,
			Status:         u.Status,
			CreatedAt:      u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	if items == nil {
		items = []item{}
	}

	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"uploads": items,
	}))
}
