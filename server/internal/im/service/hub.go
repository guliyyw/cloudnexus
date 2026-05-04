package service

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536
)

type WSMessage struct {
	Type           string `json:"type"`
	ID             uint64 `json:"id,omitempty,string"`
	ConversationID uint64 `json:"conversation_id,omitempty,string"`
	SenderID       uint64 `json:"sender_id,omitempty,string"`
	Content        string `json:"content,omitempty"`
	MsgType        string `json:"msg_type,omitempty"`
	Status         string `json:"status,omitempty"`
	UserID         uint64 `json:"user_id,omitempty,string"`
	LastReadMsgID  uint64 `json:"last_read_msg_id,omitempty,string"`
	MsgID          uint64 `json:"msg_id,omitempty,string"`
	CreatedAt      string `json:"created_at,omitempty"`
}

type Client struct {
	UserID uint64
	conn   *websocket.Conn
	hub    *Hub
	send   chan []byte
}

func newClient(userID uint64, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		UserID: userID,
		conn:   conn,
		hub:    hub,
		send:   make(chan []byte, 256),
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		var wsMsg WSMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			continue
		}
		wsMsg.SenderID = c.UserID
		c.hub.onMessage <- &wsMsg
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type Hub struct {
	clients    map[uint64]*Client
	register   chan *Client
	unregister chan *Client
	onMessage  chan *WSMessage
	mu         sync.RWMutex
	msgHandler func(*WSMessage)
}

func NewHub(msgHandler func(*WSMessage)) *Hub {
	return &Hub{
		clients:    make(map[uint64]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		onMessage:  make(chan *WSMessage, 256),
		msgHandler: msgHandler,
	}
}

func (h *Hub) Register(userID uint64, conn *websocket.Conn) {
	client := newClient(userID, conn, h)
	h.register <- client
	go client.writePump()
	go client.readPump()
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// Close existing connection if user already has one
			if old, ok := h.clients[client.UserID]; ok {
				close(old.send)
			}
			h.clients[client.UserID] = client
			h.mu.Unlock()
			h.broadcastPresence(client.UserID, "online")
			log.Printf("ws: user %d connected", client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if c, ok := h.clients[client.UserID]; ok && c == client {
				delete(h.clients, client.UserID)
				close(client.send)
			}
			h.mu.Unlock()
			h.broadcastPresence(client.UserID, "offline")
			log.Printf("ws: user %d disconnected", client.UserID)

		case msg := <-h.onMessage:
			if h.msgHandler != nil {
				h.msgHandler(msg)
			}
		}
	}
}

func (h *Hub) SendToUser(userID uint64, msg interface{}) {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case client.send <- data:
	default:
	}
}

func (h *Hub) broadcastPresence(userID uint64, status string) {
	msg := WSMessage{
		Type:   "presence",
		UserID: userID,
		Status: status,
	}
	data, _ := json.Marshal(msg)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		if c.UserID != userID {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}
