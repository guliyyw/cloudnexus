package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MusicHandler struct {
	svc *service.MusicService
}

func NewMusicHandler(svc *service.MusicService) *MusicHandler {
	return &MusicHandler{svc: svc}
}

func (h *MusicHandler) HandleGetLibrary(c *gin.Context) {
	userID := c.GetUint64("user_id")
	source := c.DefaultQuery("source", "all")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	res, err := h.svc.GetLibrary(userID, source, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(res))
}

func (h *MusicHandler) HandleUploadTrack(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req struct {
		Title      string `json:"title"`
		Artist     string `json:"artist"`
		Album      string `json:"album"`
		Duration   int    `json:"duration"`
		StorageKey string `json:"storage_key"`
		MimeType   string `json:"mime_type"`
		FileSize   int64  `json:"file_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	track, err := h.svc.UploadPublicTrack(userID, req.Title, req.Artist, req.Album, req.Duration, req.StorageKey, req.MimeType, req.FileSize)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(track))
}

func (h *MusicHandler) HandleDeleteTrack(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的 ID"))
		return
	}
	if err := h.svc.DeletePublicTrack(id, userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("删除成功"))
}

func (h *MusicHandler) HandleStream(c *gin.Context) {
	userID := c.GetUint64("user_id")
	trackID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的 ID"))
		return
	}
	source := c.DefaultQuery("source", "public")

	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		var start, end int64
		end = -1
		if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err != nil {
			if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); err != nil {
				c.JSON(http.StatusBadRequest, response.Error(400, "无效的 Range 头"))
				return
			}
		}

		reader, mimeType, totalSize, err := h.svc.StreamRange(trackID, source, userID, start, end)
		if err != nil {
			handleError(c, err)
			return
		}
		defer reader.Close()

		if end == -1 || end >= totalSize {
			end = totalSize - 1
		}
		contentLength := end - start + 1

		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
		c.Header("Content-Length", fmt.Sprintf("%d", contentLength))
		c.Header("Content-Type", mimeType)
		c.Header("Accept-Ranges", "bytes")
		c.Status(http.StatusPartialContent)
		c.Writer.WriteHeaderNow()
		io.Copy(c.Writer, reader)
		return
	}

	reader, mimeType, size, err := h.svc.StreamAudio(trackID, source, userID)
	if err != nil {
		handleError(c, err)
		return
	}
	defer reader.Close()

	c.Header("Content-Type", mimeType)
	c.Header("Content-Length", fmt.Sprintf("%d", size))
	c.Header("Accept-Ranges", "bytes")
	c.Status(http.StatusOK)
	io.Copy(c.Writer, reader)
}

func (h *MusicHandler) HandleCreatePlaylist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req service.CreatePlaylistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	pl, err := h.svc.CreatePlaylist(userID, req)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(pl))
}

func (h *MusicHandler) HandleListPlaylists(c *gin.Context) {
	userID := c.GetUint64("user_id")
	pls, err := h.svc.ListPlaylists(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(pls))
}

func (h *MusicHandler) HandleGetPlaylist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的 ID"))
		return
	}
	pl, tracks, err := h.svc.GetPlaylist(id, userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"playlist": pl,
		"tracks":   tracks,
	}))
}

func (h *MusicHandler) HandleUpdatePlaylist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的 ID"))
		return
	}
	var req service.UpdatePlaylistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	pl, err := h.svc.UpdatePlaylist(id, userID, req)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(pl))
}

func (h *MusicHandler) HandleDeletePlaylist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的 ID"))
		return
	}
	if err := h.svc.DeletePlaylist(id, userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("删除成功"))
}

func (h *MusicHandler) HandleAddTrackToPlaylist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	playlistID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的 ID"))
		return
	}
	var req service.AddTrackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if err := h.svc.AddTrackToPlaylist(playlistID, userID, req); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("添加成功"))
}

func (h *MusicHandler) HandleRemoveTrackFromPlaylist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	playlistID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的 ID"))
		return
	}
	trackID, err := strconv.ParseUint(c.Param("trackId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的曲目 ID"))
		return
	}
	if err := h.svc.RemoveTrackFromPlaylist(playlistID, trackID, userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("移除成功"))
}

func (h *MusicHandler) HandleGetLyrics(c *gin.Context) {
	userID := c.GetUint64("user_id")
	trackID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的 ID"))
		return
	}
	source := c.DefaultQuery("source", "public")

	lyrics, err := h.svc.GetLyrics(trackID, source, userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"lyrics": lyrics,
	}))
}

func (h *MusicHandler) HandleExportPlaylist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的 ID"))
		return
	}
	format := c.DefaultQuery("format", "json")

	content, err := h.svc.ExportPlaylist(id, userID, format)
	if err != nil {
		handleError(c, err)
		return
	}

	filename := uuid.New().String()
	ext := "json"
	if format == "m3u" {
		ext = "m3u"
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.%s\"", filename, ext))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(content))
}

func (h *MusicHandler) HandleImportPlaylist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的 ID"))
		return
	}
	format := c.PostForm("format")
	if format == "" {
		format = "json"
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "请上传文件"))
		return
	}
	defer file.Close()

	data := make([]byte, header.Size)
	if _, err := io.ReadFull(file, data); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "读取文件失败"))
		return
	}

	if err := h.svc.ImportPlaylist(id, userID, data, format); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("导入成功"))
}
