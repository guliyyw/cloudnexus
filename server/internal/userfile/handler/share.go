package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"

	"github.com/gin-gonic/gin"
)

type ShareHandler struct {
	svc     *service.ShareService
	fileSvc *service.FileService
}

func NewShareHandler(svc *service.ShareService, fileSvc *service.FileService) *ShareHandler {
	return &ShareHandler{svc: svc, fileSvc: fileSvc}
}

func (h *ShareHandler) HandleCreateShare(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的文件 ID"))
		return
	}

	var req service.CreateShareReq
	if err := c.ShouldBindJSON(&req); err != nil {
		req = service.CreateShareReq{}
	}

	share, err := h.svc.CreateShare(userID, fileID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.OKWithData(share))
}

func (h *ShareHandler) HandleGetShareByCode(c *gin.Context) {
	code := c.Param("code")
	share, err := h.svc.GetShareByCode(code)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.OKWithData(share))
}

func (h *ShareHandler) HandleVerifyPassword(c *gin.Context) {
	code := c.Param("code")
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "请输入密码"))
		return
	}

	if _, err := h.svc.VerifyPassword(code, req.Password); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.OK("验证成功"))
}

func (h *ShareHandler) HandleDownloadShare(c *gin.Context) {
	code := c.Param("code")
	password := c.Query("password")

	info, err := h.svc.GetShareByCode(code)
	if err != nil {
		handleError(c, err)
		return
	}

	if info.Password != "" {
		if password == "" {
			c.JSON(http.StatusForbidden, response.Error(403, "需要密码"))
			return
		}
		if _, err := h.svc.VerifyPassword(code, password); err != nil {
			handleError(c, err)
			return
		}
	}

	stream, file, err := h.fileSvc.Download(info.OwnerID, info.FileID)
	if err != nil {
		handleError(c, err)
		return
	}
	defer stream.Close()

	h.svc.RecordDownload(info.ID)

	c.Header("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	c.DataFromReader(http.StatusOK, file.Size, file.MimeType, stream, nil)
}

func (h *ShareHandler) HandleListSharesByFile(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的文件 ID"))
		return
	}

	shares, err := h.svc.ListSharesByFile(userID, fileID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.OKWithData(shares))
}

func (h *ShareHandler) HandleListMyShares(c *gin.Context) {
	userID := c.GetUint64("user_id")
	shares, err := h.svc.ListMyShares(userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.OKWithData(shares))
}

func (h *ShareHandler) HandleDeleteShare(c *gin.Context) {
	userID := c.GetUint64("user_id")
	shareID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的分享 ID"))
		return
	}

	if err := h.svc.DeleteShare(userID, shareID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.OK("已取消分享"))
}
