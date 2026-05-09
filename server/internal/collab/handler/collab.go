package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/collab/service"
	"github.com/cloudnexus/server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { return true },
}

type CollabHandler struct {
	hub *service.DocHub
}

func NewCollabHandler(hub *service.DocHub) *CollabHandler {
	return &CollabHandler{hub: hub}
}

func getUID(c *gin.Context) uint64 {
	v, _ := c.Get("user_id")
	id, _ := v.(uint64)
	return id
}

func (h *CollabHandler) HandleWebSocket(c *gin.Context) {
	userID := getUID(c)
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "参数错误"))
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	if err := h.hub.Join(docID, userID, conn); err != nil {
		log.Printf("collab: join doc %d user %d failed: %v", docID, userID, err)
		conn.Close()
		return
	}
}
