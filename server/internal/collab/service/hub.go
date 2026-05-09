package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/cloudnexus/server/pkg/model"
	"github.com/gorilla/websocket"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

// y-websocket 线路协议
// 每条 Yjs 消息格式：[messageSync(0), 子类型(0/1/2), ...数据]
const (
	msgSync        = 0 // 外层：sync 消息
	yjsSyncStep1   = 0 // 子类型：客户端请求同步 (state vector)
	yjsSyncStep2   = 1 // 子类型：服务端响应 (完整 update)
	yjsSyncUpdate  = 2 // 子类型：增量更新
)

// DocClient 文档房间中的客户端
type DocClient struct {
	UserID   uint64
	Username string
	docID    uint64
	conn     *websocket.Conn
	room     *DocRoom
	send     chan []byte
}

func newDocClient(userID uint64, username string, docID uint64, conn *websocket.Conn, room *DocRoom) *DocClient {
	return &DocClient{
		UserID:   userID,
		Username: username,
		docID:    docID,
		conn:     conn,
		room:     room,
		send:     make(chan []byte, 256),
	}
}

func (c *DocClient) readPump() {
	defer func() {
		c.room.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		if len(data) == 0 {
			continue
		}

		// SyncStep1: 客户端请求同步 — 子类型为 0
		if msgType == websocket.BinaryMessage && len(data) >= 2 && data[0] == msgSync && data[1] == yjsSyncStep1 {
			c.room.relay(c, msgType, data)
			c.room.respondSyncStep1(c)
			continue
		}

		c.room.relay(c, msgType, data)
	}
}

func (c *DocClient) writePump() {
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
			wsType := int(msg[0])
			if err := c.conn.WriteMessage(wsType, msg[1:]); err != nil {
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

func (c *DocClient) sendMsg(wsType int, data []byte) {
	msg := append([]byte{byte(wsType)}, data...)
	select {
	case c.send <- msg:
	default:
	}
}

// --- DocRoom ---

type DocRoom struct {
	docID       uint64
	clients     map[uint64]*DocClient
	updateCount int
	lastPersist time.Time
	register    chan *DocClient
	unregister  chan *DocClient
	mu          sync.Mutex
	hub         *DocHub
}

func newDocRoom(docID uint64, hub *DocHub) *DocRoom {
	return &DocRoom{
		docID:       docID,
		clients:     make(map[uint64]*DocClient),
		lastPersist: time.Now(),
		register:    make(chan *DocClient),
		unregister:  make(chan *DocClient),
		hub:         hub,
	}
}

func (r *DocRoom) run() {
	for {
		select {
		case client := <-r.register:
			r.mu.Lock()
			if old, ok := r.clients[client.UserID]; ok {
				close(old.send)
			}
			r.clients[client.UserID] = client
			r.mu.Unlock()
			log.Printf("collab: user %d joined doc %d", client.UserID, r.docID)

		case client := <-r.unregister:
			r.mu.Lock()
			if c, ok := r.clients[client.UserID]; ok && c == client {
				delete(r.clients, client.UserID)
				close(client.send)
			}
			remaining := len(r.clients)
			r.mu.Unlock()
			log.Printf("collab: user %d left doc %d (%d remaining)", client.UserID, r.docID, remaining)
			if remaining == 0 {
				r.mu.Lock()
				r.updateCount = 0
				r.mu.Unlock()
			}
		}
	}
}

// respondSyncStep1 当房间只有请求者时，用已存文档回复 SyncStep1
func (r *DocRoom) respondSyncStep1(client *DocClient) {
	r.mu.Lock()
	count := len(r.clients)
	r.mu.Unlock()

	if count > 1 {
		return // 有其他客户端会回复
	}

	data, err := r.hub.loadDocument(r.docID)
	if err != nil || len(data) == 0 {
		return
	}

	// 发送 SyncStep2: [msgSync, yjsSyncStep2, ...data]
	reply := append([]byte{msgSync, yjsSyncStep2}, data...)
	client.sendMsg(websocket.BinaryMessage, reply)
	log.Printf("collab: sync response to user %d for doc %d (%d bytes)", client.UserID, r.docID, len(reply))
}

func (r *DocRoom) relay(sender *DocClient, wsType int, data []byte) {
	if wsType == websocket.BinaryMessage {
		r.mu.Lock()
		r.updateCount++
		r.mu.Unlock()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.clients {
		if c.UserID != sender.UserID {
			c.sendMsg(wsType, data)
		}
	}

	if wsType == websocket.BinaryMessage {
		r.hub.relayUpdate(r.docID, sender.UserID, data)
	}
}

func (r *DocRoom) broadcastAll(wsType int, data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.clients {
		c.sendMsg(wsType, data)
	}
}

func (r *DocRoom) maybePersist(lastEditor uint64, data []byte) {
	r.mu.Lock()
	count := r.updateCount
	elapsed := time.Since(r.lastPersist)
	r.mu.Unlock()

	if count >= 50 || elapsed > 5*time.Second {
		r.mu.Lock()
		if r.updateCount > 0 {
			r.hub.persist(r.docID, data, lastEditor)
			r.updateCount = 0
			r.lastPersist = time.Now()
		}
		r.mu.Unlock()
	}
}

// --- DocHub ---

type DocHub struct {
	rooms  map[uint64]*DocRoom
	mu     sync.RWMutex
	redis  *redis.Client
	db     *gorm.DB
	minio  *minio.Client
	bucket string
}

func NewDocHub(db *gorm.DB, minioClient *minio.Client, bucket string) *DocHub {
	return &DocHub{
		rooms:  make(map[uint64]*DocRoom),
		db:     db,
		minio:  minioClient,
		bucket: bucket,
	}
}

func (h *DocHub) verifyAccess(docID, userID uint64) (bool, error) {
	var count int64
	err := h.db.Model(&model.File{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", docID, userID).
		Count(&count).Error
	return count > 0, err
}

func (h *DocHub) loadDocument(docID uint64) ([]byte, error) {
	storageKey := fmt.Sprintf("collab/%d/ydoc.bin", docID)
	obj, err := h.minio.GetObject(context.Background(), h.bucket, storageKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

func (h *DocHub) Join(docID, userID uint64, conn *websocket.Conn) error {
	ok, err := h.verifyAccess(docID, userID)
	if err != nil {
		return fmt.Errorf("校验权限失败: %w", err)
	}
	if !ok {
		return fmt.Errorf("无权访问此文档")
	}

	username := fmt.Sprintf("user_%d", userID)

	h.mu.Lock()
	room, exists := h.rooms[docID]
	if !exists {
		room = newDocRoom(docID, h)
		h.rooms[docID] = room
		go room.run()
		log.Printf("collab: created room for doc %d", docID)
	}
	h.mu.Unlock()

	client := newDocClient(userID, username, docID, conn, room)
	room.register <- client
	go client.writePump()
	go client.readPump()

	return nil
}

func (h *DocHub) EnableRedisRelay(rdb *redis.Client) {
	h.redis = rdb
	go h.runRedisRelay()
}

type collabRelayMsg struct {
	DocID   uint64 `json:"doc_id"`
	UserID  uint64 `json:"user_id"`
	DataB64 string `json:"data_b64"`
}

func (h *DocHub) relayUpdate(docID, userID uint64, data []byte) {
	if h.redis == nil {
		return
	}
	payload := collabRelayMsg{
		DocID:   docID,
		UserID:  userID,
		DataB64: base64.StdEncoding.EncodeToString(data),
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	channel := fmt.Sprintf("collab:doc:%d", docID)
	if err := h.redis.Publish(ctx, channel, jsonBytes).Err(); err != nil {
		log.Printf("collab: redis publish error: %v", err)
	}
}

func (h *DocHub) runRedisRelay() {
	ctx := context.Background()
	pubsub := h.redis.PSubscribe(ctx, "collab:doc:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	log.Printf("collab: redis relay listening on collab:doc:*")

	for msg := range ch {
		var relay collabRelayMsg
		if err := json.Unmarshal([]byte(msg.Payload), &relay); err != nil {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(relay.DataB64)
		if err != nil {
			continue
		}
		h.mu.RLock()
		room, ok := h.rooms[relay.DocID]
		h.mu.RUnlock()
		if !ok {
			continue
		}
		room.broadcastAll(websocket.BinaryMessage, data)
	}
}

func (h *DocHub) persist(docID uint64, content []byte, lastEditor uint64) {
	storageKey := fmt.Sprintf("collab/%d/ydoc.bin", docID)
	_, err := h.minio.PutObject(context.Background(), h.bucket, storageKey,
		bytes.NewReader(content), int64(len(content)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		log.Printf("collab: persist doc %d to minio: %v", docID, err)
	}
}
