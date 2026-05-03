package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type FileHandler struct {
	svc *service.FileService
}

func NewFileHandler(svc *service.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

func (h *FileHandler) HandleUpload(c *gin.Context) {
	userID := c.GetUint64("user_id")
	parentID, _ := strconv.ParseUint(c.DefaultPostForm("parent_id", "0"), 10, 64)

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "不是有效的文件上传请求"))
		return
	}
	files := form.File["file"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, response.Error(400, "请选择文件"))
		return
	}

	results := make([]*model.File, 0, len(files))
	errMsgs := make([]string, 0)

	for _, fh := range files {
		file, err := h.svc.Upload(userID, parentID, fh)
		if err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", fh.Filename, err.Error()))
			continue
		}
		results = append(results, file)
	}

	c.JSON(http.StatusCreated, response.OKWithData(gin.H{
		"files":  results,
		"errors": errMsgs,
		"total":  len(files),
		"ok":     len(results),
	}))
}

func (h *FileHandler) HandleDownload(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的文件 ID"))
		return
	}

	stream, file, err := h.svc.Download(userID, fileID)
	if err != nil {
		handleError(c, err)
		return
	}
	defer stream.Close()

	disposition := "attachment"
	if c.Query("inline") == "true" {
		disposition = "inline"
	}
	c.Header("Content-Disposition", disposition+"; filename=\""+file.Name+"\"")
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Length", strconv.FormatInt(file.Size, 10))
	c.DataFromReader(http.StatusOK, file.Size, file.MimeType, stream, nil)
}

func (h *FileHandler) HandleList(c *gin.Context) {
	userID := c.GetUint64("user_id")
	parentID, _ := strconv.ParseUint(c.DefaultQuery("parent_id", "0"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	files, total, err := h.svc.ListFiles(userID, parentID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"items":     files,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

func (h *FileHandler) HandleDelete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的文件 ID"))
		return
	}

	if err := h.svc.DeleteFile(userID, fileID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("deleted"))
}

type mkdirReq struct {
	Name     string `json:"name" binding:"required"`
	ParentID uint64 `json:"parent_id"`
}

func (h *FileHandler) HandleMkdir(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req mkdirReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}

	dir, err := h.svc.Mkdir(userID, req.ParentID, req.Name)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.OKWithData(dir))
}

func (h *FileHandler) HandleSearch(c *gin.Context) {
	userID := c.GetUint64("user_id")
	keyword := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	files, total, err := h.svc.Search(userID, keyword, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"items":     files,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}
