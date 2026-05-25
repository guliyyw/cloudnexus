package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/dockermgr/service"
	"github.com/cloudnexus/server/pkg/httputil"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type DockerHandler struct {
	svc *service.DockerService
}

func NewDockerHandler(svc *service.DockerService) *DockerHandler {
	return &DockerHandler{svc: svc}
}

// 使用 httputil 包中的共享函数
func getUserID(c *gin.Context) uint64       { return httputil.GetUserID(c) }
func isAdmin(c *gin.Context) bool            { return httputil.IsAdmin(c) }
func getUsername(c *gin.Context) string      { return httputil.GetUsername(c) }
func getEndpoint(c *gin.Context) string      { return c.DefaultQuery("endpoint", "local") }

func (h *DockerHandler) HandleListContainers(c *gin.Context) {
	endpoint := getEndpoint(c)
	all, _ := strconv.ParseBool(c.DefaultQuery("all", "false"))
	containers, err := h.svc.ListContainers(endpoint, all, getUserID(c), isAdmin(c))
	if err != nil {
		// SECURITY: 使用共享错误处理，不暴露内部错误详情
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(containers))
}

type createContainerReq struct {
	Image string `json:"image" binding:"required"`
	Name  string `json:"name"`
}

func (h *DockerHandler) HandleCreateContainer(c *gin.Context) {
	endpoint := getEndpoint(c)
	var req createContainerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	id, err := h.svc.CreateContainer(endpoint, req.Image, req.Name, getUserID(c), getUsername(c))
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.OKWithData(gin.H{"id": id}))
}

func (h *DockerHandler) HandleStartContainer(c *gin.Context) {
	endpoint := getEndpoint(c)
	id := c.Param("id")
	if err := h.svc.StartContainer(endpoint, id, getUserID(c), isAdmin(c)); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("started"))
}

func (h *DockerHandler) HandleStopContainer(c *gin.Context) {
	endpoint := getEndpoint(c)
	id := c.Param("id")
	if err := h.svc.StopContainer(endpoint, id, getUserID(c), isAdmin(c)); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("stopped"))
}

func (h *DockerHandler) HandleRestartContainer(c *gin.Context) {
	endpoint := getEndpoint(c)
	id := c.Param("id")
	if err := h.svc.RestartContainer(endpoint, id, getUserID(c), isAdmin(c)); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("restarted"))
}

func (h *DockerHandler) HandleRemoveContainer(c *gin.Context) {
	endpoint := getEndpoint(c)
	id := c.Param("id")
	force, _ := strconv.ParseBool(c.DefaultQuery("force", "false"))
	if err := h.svc.RemoveContainer(endpoint, id, force, getUserID(c), isAdmin(c)); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("removed"))
}

func (h *DockerHandler) HandleGetLogs(c *gin.Context) {
	endpoint := getEndpoint(c)
	id := c.Param("id")
	tail := c.DefaultQuery("tail", "100")
	reader, err := h.svc.GetLogs(endpoint, id, tail, getUserID(c), isAdmin(c))
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	defer reader.Close()

	c.Header("Content-Type", "text/plain; charset=utf-8")
	io.Copy(c.Writer, reader)
}

// --- Image management ---

func (h *DockerHandler) HandleListImages(c *gin.Context) {
	endpoint := getEndpoint(c)
	images, err := h.svc.ListImages(endpoint)
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(images))
}

type pullImageReq struct {
	Image string `json:"image" binding:"required"`
}

func (h *DockerHandler) HandlePullImage(c *gin.Context) {
	endpoint := getEndpoint(c)
	var req pullImageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.PullImage(endpoint, req.Image); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("image pulled"))
}

func (h *DockerHandler) HandleRemoveImage(c *gin.Context) {
	endpoint := getEndpoint(c)
	image := c.Param("image")
	force, _ := strconv.ParseBool(c.DefaultQuery("force", "false"))
	if err := h.svc.RemoveImage(endpoint, image, force); err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("image removed"))
}

// --- Container stats ---

func (h *DockerHandler) HandleGetStats(c *gin.Context) {
	endpoint := getEndpoint(c)
	id := c.Param("id")
	stats, err := h.svc.GetStats(endpoint, id, getUserID(c), isAdmin(c))
	if err != nil {
		httputil.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(stats))
}

// --- Endpoint management ---

func (h *DockerHandler) HandleListEndpoints(c *gin.Context) {
	endpoints := h.svc.ListEndpoints()
	c.JSON(http.StatusOK, response.OKWithData(endpoints))
}

func (h *DockerHandler) HandlePingEndpoint(c *gin.Context) {
	endpoint := getEndpoint(c)
	if err := h.svc.PingEndpoint(endpoint); err != nil {
		// 对于 ping 端点，返回具体的错误信息是可以接受的
		c.JSON(http.StatusServiceUnavailable, response.Error(503, err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.OK("pong"))
}
