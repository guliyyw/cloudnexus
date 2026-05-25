package handler

import (
	"log"
	"strconv"

	"github.com/cloudnexus/server/internal/collab/service"
	"github.com/cloudnexus/server/pkg/httputil"
	"github.com/cloudnexus/server/pkg/response"

	"github.com/gin-gonic/gin"
)

type CollabHandler struct {
	hub *service.DocHub
}

func NewCollabHandler(hub *service.DocHub) *CollabHandler {
	return &CollabHandler{hub: hub}
}

func (h *CollabHandler) HandleWebSocket(c *gin.Context) {
	userID := httputil.GetUserID(c)
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "参数错误"))
		return
	}

	conn, err := httputil.DefaultUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	if err := h.hub.Join(docID, userID, conn); err != nil {
		log.Printf("collab: join doc %d user %d failed: %v", docID, userID, err)
		conn.Close()
		return
	}
}
