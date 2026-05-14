package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type TrashHandler struct {
	svc *service.TrashService
}

func NewTrashHandler(svc *service.TrashService) *TrashHandler {
	return &TrashHandler{svc: svc}
}

func (h *TrashHandler) HandleListTrash(c *gin.Context) {
	userID := c.GetUint64("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	files, total, err := h.svc.ListTrash(userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"items":     files,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

func (h *TrashHandler) HandleRestore(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的文件 ID"))
		return
	}

	if err := h.svc.Restore(userID, fileID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已恢复"))
}

func (h *TrashHandler) HandlePermanentDelete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的文件 ID"))
		return
	}

	if err := h.svc.PermanentDelete(userID, fileID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已彻底删除"))
}

func (h *TrashHandler) HandleEmptyTrash(c *gin.Context) {
	userID := c.GetUint64("user_id")

	count, err := h.svc.EmptyTrash(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"deleted": count,
	}))
}
