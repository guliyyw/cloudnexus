package handler

import (
	"strings"

	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NodeHandler struct {
	db *gorm.DB
}

func NewNodeHandler(db *gorm.DB) *NodeHandler {
	return &NodeHandler{db: db}
}

type addNodeRequest struct {
	Name    string `json:"name" binding:"required"`
	Host    string `json:"host" binding:"required"`
	Port    int    `json:"port"`
	TLSCert string `json:"tls_cert"`
	TLSKey  string `json:"tls_key"`
	CACert  string `json:"ca_cert"`
}

// HandleListNodes returns all registered nodes, with optional filters:
// ?service=user-file-svc,im-svc &host=localhost&type=service,infrastructure&status=healthy,offline
func (h *NodeHandler) HandleListNodes(c *gin.Context) {
	query := h.db.Model(&model.DockerNode{})

	if svc := c.Query("service"); svc != "" {
		query = query.Where("service IN ?", parseCSV(svc))
	}
	if host := c.Query("host"); host != "" {
		query = query.Where("host = ?", host)
	}
	if typ := c.Query("type"); typ != "" {
		query = query.Where("node_type IN ?", parseCSV(typ))
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status IN ?", parseCSV(status))
	}

	var nodes []model.DockerNode
	if err := query.Order("name ASC").Find(&nodes).Error; err != nil {
		c.JSON(500, response.Error(500, "查询节点列表失败"))
		return
	}
	if nodes == nil {
		nodes = []model.DockerNode{}
	}
	c.JSON(200, response.OKWithData(gin.H{"nodes": nodes}))
}

func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	var res []string
	for _, v := range parts {
		if t := strings.TrimSpace(v); t != "" {
			res = append(res, t)
		}
	}
	return res
}

// HandleGetNode returns a single node by name.
func (h *NodeHandler) HandleGetNode(c *gin.Context) {
	name := c.Param("name")
	var node model.DockerNode
	if err := h.db.Where("name = ?", name).First(&node).Error; err != nil {
		c.JSON(404, response.Error(404, "节点不存在"))
		return
	}
	c.JSON(200, response.OKWithData(gin.H{"node": node}))
}

// HandleAddNode registers a new Docker node manually.
func (h *NodeHandler) HandleAddNode(c *gin.Context) {
	var req addNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误: name 和 host 必填"))
		return
	}
	if req.Port == 0 {
		req.Port = 2376
	}

	node := model.DockerNode{
		Name:    req.Name,
		Host:    req.Host,
		Port:    req.Port,
		TLSCert: req.TLSCert,
		TLSKey:  req.TLSKey,
		CACert:  req.CACert,
		Status:  "offline",
	}

	if err := h.db.Create(&node).Error; err != nil {
		c.JSON(409, response.Error(409, "节点已存在或创建失败"))
		return
	}
	c.JSON(201, response.OKWithData(gin.H{"node": node}))
}

// HandleGetNodeSessions returns the online session history for a node.
func (h *NodeHandler) HandleGetNodeSessions(c *gin.Context) {
	name := c.Param("name")
	var sessions []model.NodeOnlineSession
	if err := h.db.Where("node_name = ?", name).Order("start_time DESC").Find(&sessions).Error; err != nil {
		c.JSON(500, response.Error(500, "查询会话历史失败"))
		return
	}
	if sessions == nil {
		sessions = []model.NodeOnlineSession{}
	}
	c.JSON(200, response.OKWithData(gin.H{"sessions": sessions}))
}

// HandleDeleteNode removes a node.
func (h *NodeHandler) HandleDeleteNode(c *gin.Context) {
	name := c.Param("name")
	result := h.db.Where("name = ?", name).Delete(&model.DockerNode{})
	if result.Error != nil {
		c.JSON(500, response.Error(500, "删除节点失败"))
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(404, response.Error(404, "节点不存在"))
		return
	}
	c.JSON(200, response.OK("节点已删除"))
}
