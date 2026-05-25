// Package httputil provides HTTP-related utility functions for handlers.
package httputil

import (
	"net/http"

	"github.com/cloudnexus/server/pkg/middleware"
	"github.com/gorilla/websocket"
)

// DefaultUpgrader is a shared WebSocket upgrader with recommended settings.
// It checks the Origin header against ALLOWED_ORIGINS environment variable.
var DefaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     middleware.CheckWebSocketOrigin,
}

// UpgradeWebSocket upgrades an HTTP connection to WebSocket using the default upgrader.
// Returns the WebSocket connection or an error.
func UpgradeWebSocket(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return DefaultUpgrader.Upgrade(w, r, nil)
}
