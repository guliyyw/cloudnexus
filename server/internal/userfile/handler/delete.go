package handler

import (
	"net/http"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type DeleteHandler struct {
	svc *service.DeleteService
}

func NewDeleteHandler(svc *service.DeleteService) *DeleteHandler {
	return &DeleteHandler{svc: svc}
}

func (h *DeleteHandler) HandleRequestDelete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	if err := h.svc.RequestDelete(userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("账号注销申请已提交，30 天后将永久删除"))
}

func (h *DeleteHandler) HandleCancelDelete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	if err := h.svc.CancelDelete(userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("账号注销已取消"))
}

func (h *DeleteHandler) HandleConfirmDelete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	if err := h.svc.ConfirmDelete(userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("账号已删除"))
}
