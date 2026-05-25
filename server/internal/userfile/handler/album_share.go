package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *AlbumHandler) HandleCreateAlbumShare(c *gin.Context) {
	userID := c.GetUint64("user_id")
	albumID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的相册 ID"))
		return
	}

	var req service.CreateAlbumShareReq
	if err := c.ShouldBindJSON(&req); err != nil {
		req = service.CreateAlbumShareReq{}
	}

	share, err := h.shareSvc.CreateAlbumShare(userID, albumID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.OKWithData(share))
}
