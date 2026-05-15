package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"

	"github.com/gin-gonic/gin"
)

type AlbumHandler struct {
	svc *service.AlbumService
}

func NewAlbumHandler(svc *service.AlbumService) *AlbumHandler {
	return &AlbumHandler{svc: svc}
}

func (h *AlbumHandler) HandleCreate(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req service.CreateAlbumReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	album, err := h.svc.Create(userID, req)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(album))
}

func (h *AlbumHandler) HandleList(c *gin.Context) {
	userID := c.GetUint64("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	albums, total, err := h.svc.ListByOwner(userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"albums": albums,
		"total":  total,
	}))
}

func (h *AlbumHandler) HandleGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的相册 ID"))
		return
	}
	album, err := h.svc.GetByID(id)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(album))
}

func (h *AlbumHandler) HandleUpdate(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的相册 ID"))
		return
	}
	var req service.UpdateAlbumReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	album, err := h.svc.Update(id, userID, req)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(album))
}

func (h *AlbumHandler) HandleDelete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的相册 ID"))
		return
	}
	if err := h.svc.Delete(id, userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("删除成功"))
}

func (h *AlbumHandler) HandleAddFiles(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的相册 ID"))
		return
	}
	var req service.AddFilesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if err := h.svc.AddFiles(id, userID, req); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("添加成功"))
}

func (h *AlbumHandler) HandleRemoveFile(c *gin.Context) {
	userID := c.GetUint64("user_id")
	albumID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的相册 ID"))
		return
	}
	fileID, err := strconv.ParseUint(c.Param("fileId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的文件 ID"))
		return
	}
	if err := h.svc.RemoveFile(albumID, fileID, userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("移除成功"))
}

func (h *AlbumHandler) HandleGetFiles(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的相册 ID"))
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	files, total, err := h.svc.GetFiles(id, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"files": files,
		"total": total,
	}))
}
