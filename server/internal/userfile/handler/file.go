package handler

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	versionMessage := c.DefaultPostForm("version_message", "")

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
		file, err := h.svc.Upload(userID, parentID, fh, versionMessage)
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

type saveTextReq struct {
	Content        string `json:"content"`
	VersionMessage string `json:"version_message"`
}

type saveWordReq struct {
	HTML           string `json:"html"`
	VersionMessage string `json:"version_message"`
}

func (h *FileHandler) HandleSaveText(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "invalid file id"))
		return
	}
	var req saveTextReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "invalid request"))
		return
	}
	file, err := h.svc.SaveTextContent(userID, fileID, req.Content, req.VersionMessage)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(file))
}

func (h *FileHandler) HandleSaveContent(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "invalid file id"))
		return
	}
	uploaded, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "请选择要保存的文件内容"))
		return
	}
	defer uploaded.Close()
	content, err := io.ReadAll(uploaded)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "读取文件内容失败"))
		return
	}
	contentType := header.Header.Get("Content-Type")
	versionMessage := c.DefaultPostForm("version_message", "online edit")
	file, err := h.svc.SaveBinaryContent(userID, fileID, content, contentType, versionMessage)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(file))
}

func (h *FileHandler) HandleSaveWord(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "invalid file id"))
		return
	}
	var req saveWordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "invalid request"))
		return
	}
	file, err := h.svc.SaveWordHTML(userID, fileID, req.HTML, req.VersionMessage)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(file))
}

func (h *FileHandler) HandleExportWord(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "invalid file id"))
		return
	}
	var req saveWordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "invalid request"))
		return
	}
	content, file, err := h.svc.ConvertHTMLToWord(userID, fileID, req.HTML)
	if err != nil {
		handleError(c, err)
		return
	}
	name := strings.TrimSuffix(file.Name, filepath.Ext(file.Name)) + ".docx"
	c.Header("Content-Disposition", "attachment; filename=\""+name+"\"")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", content)
}

func (h *FileHandler) HandleConvertWordToPDF(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "invalid file id"))
		return
	}
	stream, name, size, err := h.svc.ConvertWordToPDF(userID, fileID)
	if err != nil {
		handleError(c, err)
		return
	}
	defer stream.Close()
	c.Header("Content-Disposition", "inline; filename=\""+name+"\"")
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.DataFromReader(http.StatusOK, size, "application/pdf", stream, nil)
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
	ParentID uint64 `json:"parent_id,string"`
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

type batchReq struct {
	IDs []string `json:"ids" binding:"required"`
}

func parseIDs(strs []string) []uint64 {
	ids := make([]uint64, 0, len(strs))
	for _, s := range strs {
		id, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func (h *FileHandler) HandleBatchDelete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req batchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误：请提供 ids 数组"))
		return
	}

	deleted, errs := h.svc.BatchDelete(userID, parseIDs(req.IDs))
	if errs == nil {
		errs = []string{}
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"deleted": deleted,
		"errors":  errs,
	}))
}

func (h *FileHandler) HandleBatchDownload(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req batchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误：请提供 ids 数组"))
		return
	}

	ids := parseIDs(req.IDs)
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, response.Error(400, "请选择至少一个文件"))
		return
	}

	pr, pw := io.Pipe()
	go func() {
		zw := zip.NewWriter(pw)
		seen := make(map[string]int)
		var copyErr error

		for _, id := range ids {
			stream, file, err := h.svc.Download(userID, id)
			if err != nil {
				continue
			}

			name := file.Name
			if cnt, ok := seen[name]; ok {
				ext := filepath.Ext(name)
				base := name[:len(name)-len(ext)]
				name = fmt.Sprintf("%s_%d%s", base, cnt, ext)
			}
			seen[file.Name]++

			fw, err := zw.Create(name)
			if err != nil {
				stream.Close()
				continue
			}

			if _, err := io.Copy(fw, stream); err != nil {
				copyErr = err
			}
			stream.Close()
		}

		if err := zw.Close(); err != nil && copyErr == nil {
			copyErr = err
		}
		if copyErr != nil {
			pw.CloseWithError(copyErr)
		} else {
			pw.Close()
		}
	}()

	filename := fmt.Sprintf("files-%s.zip", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.DataFromReader(http.StatusOK, -1, "application/zip", pr, nil)
}

type moveCopyReq struct {
	ID             uint64 `json:"id,string" binding:"required"`
	TargetParentID uint64 `json:"target_parent_id,string"`
}

func (h *FileHandler) HandleMove(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req moveCopyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	file, err := h.svc.Move(userID, req.ID, req.TargetParentID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(file))
}

func (h *FileHandler) HandleCopy(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req moveCopyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	file, err := h.svc.Copy(userID, req.ID, req.TargetParentID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.OKWithData(file))
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

// ── 协作文档 ──

type createCollabReq struct {
	Title      string `json:"title" binding:"required"`
	ParentID   uint64 `json:"parent_id,string"`
	CollabType string `json:"collab_type"`
}

type createOfficeReq struct {
	Title    string `json:"title" binding:"required"`
	ParentID uint64 `json:"parent_id,string"`
	Kind     string `json:"kind" binding:"required"`
}

func (h *FileHandler) HandleCreateCollab(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req createCollabReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误：请提供 title"))
		return
	}
	if req.CollabType == "" {
		req.CollabType = "doc"
	}
	file, err := h.svc.CreateCollab(userID, req.ParentID, req.Title, req.CollabType)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.OKWithData(file))
}

func (h *FileHandler) HandleCreateOfficeDoc(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req createOfficeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误：请提供 title 和 kind"))
		return
	}
	file, err := h.svc.CreateOfficeDoc(userID, req.ParentID, req.Title, req.Kind)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.OKWithData(file))
}

func (h *FileHandler) HandleListCollabDocs(c *gin.Context) {
	userID := c.GetUint64("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("q")

	files, total, err := h.svc.ListCollabDocs(userID, page, pageSize, keyword)
	if err != nil {
		handleError(c, err)
		return
	}

	type item struct {
		ID         uint64 `json:"id,string"`
		Title      string `json:"title"`
		OwnerID    uint64 `json:"owner_id,string"`
		LastEditor string `json:"last_editor"`
		Version    int64  `json:"version"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at"`
	}
	items := make([]item, 0, len(files))
	for _, f := range files {
		items = append(items, item{
			ID:         f.ID,
			Title:      f.Name,
			OwnerID:    f.UserID,
			LastEditor: "",
			Version:    1,
			CreatedAt:  f.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:  f.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	if items == nil {
		items = []item{}
	}

	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

func (h *FileHandler) HandleGetFileMeta(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的文件 ID"))
		return
	}
	file, err := h.svc.GetFileMeta(userID, fileID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(file))
}

// ── 文件版本 ──

func (h *FileHandler) HandleListVersions(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的文件 ID"))
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	versions, total, err := h.svc.ListVersions(userID, fileID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"items":     versions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

func (h *FileHandler) HandleRestoreVersion(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的文件 ID"))
		return
	}
	versionID, err := strconv.ParseUint(c.Param("versionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的版本 ID"))
		return
	}

	file, err := h.svc.RestoreVersion(userID, fileID, versionID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(file))
}

func (h *FileHandler) HandleDownloadVersion(c *gin.Context) {
	userID := c.GetUint64("user_id")
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的文件 ID"))
		return
	}
	versionID, err := strconv.ParseUint(c.Param("versionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的版本 ID"))
		return
	}

	stream, ver, file, err := h.svc.DownloadVersion(userID, fileID, versionID)
	if err != nil {
		handleError(c, err)
		return
	}
	defer stream.Close()

	filename := fmt.Sprintf("%s_v%d%s", file.Name[:len(file.Name)-len(filepath.Ext(file.Name))], ver.VersionNum, filepath.Ext(file.Name))
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Length", strconv.FormatInt(ver.Size, 10))
	c.DataFromReader(http.StatusOK, ver.Size, file.MimeType, stream, nil)
}
