package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

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
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "请上传音频文件"))
		return
	}
	result, err := h.svc.UploadPublicTrack(
		userID,
		fileHeader,
		c.PostForm("title"),
		c.PostForm("artist"),
		c.PostForm("album"),
	)
	if err != nil {
		handleError(c, err)
		return
	}
	if result.Duplicated {
		c.JSON(http.StatusConflict, response.Error(409, "曲目已存在"))
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(result.Track))
}

func (h *MusicHandler) HandleDeleteTrack(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "閺冪姵鏅ラ惃?ID"))
		return
	}
	if err := h.svc.DeletePublicTrack(id, userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("閸掔娀娅庨幋鎰"))
}

func (h *MusicHandler) HandleListLikes(c *gin.Context) {
	userID := c.GetUint64("user_id")
	res, err := h.svc.ListLikedTracks(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(res))
}

func (h *MusicHandler) HandleAddLike(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req service.TrackRefReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "invalid request"))
		return
	}
	if err := h.svc.SetTrackLike(userID, req); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("liked"))
}

func (h *MusicHandler) HandleRemoveLike(c *gin.Context) {
	userID := c.GetUint64("user_id")
	trackID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "invalid id"))
		return
	}
	source := c.DefaultQuery("source", "public")
	if err := h.svc.RemoveTrackLike(userID, trackID, source); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("unliked"))
}

func (h *MusicHandler) HandleListRecent(c *gin.Context) {
	userID := c.GetUint64("user_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	res, err := h.svc.ListRecentTracks(userID, limit)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(res))
}

func (h *MusicHandler) HandleRecordRecent(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req service.TrackRefReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "invalid request"))
		return
	}
	if err := h.svc.RecordRecentTrack(userID, req); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("recent play recorded"))
}

func (h *MusicHandler) HandleStream(c *gin.Context) {
	userID := c.GetUint64("user_id")
	trackID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "鏃犳晥鐨?ID"))
		return
	}
	source := c.DefaultQuery("source", "public")

	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		start, end, ok := parseByteRange(rangeHeader)
		if !ok {
			c.JSON(http.StatusBadRequest, response.Error(400, "无效的 Range 请求头"))
			return
		}

		reader, mimeType, totalSize, err := h.svc.StreamRange(trackID, source, userID, start, end)
		if err != nil {
			handleError(c, err)
			return
		}
		defer reader.Close()

		if start < 0 || start >= totalSize {
			c.Header("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		if end < 0 || end >= totalSize {
			end = totalSize - 1
		}
		if end < start {
			c.Header("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		contentLength := end - start + 1

		c.DataFromReader(http.StatusPartialContent, contentLength, mimeType, io.LimitReader(reader, contentLength), map[string]string{
			"Accept-Ranges":  "bytes",
			"Content-Range":  fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize),
			"Content-Length": fmt.Sprintf("%d", contentLength),
		})
		return
	}

	reader, mimeType, size, err := h.svc.StreamAudio(trackID, source, userID)
	if err != nil {
		handleError(c, err)
		return
	}
	defer reader.Close()

	c.DataFromReader(http.StatusOK, size, mimeType, reader, map[string]string{
		"Accept-Ranges":  "bytes",
		"Content-Length": fmt.Sprintf("%d", size),
	})
}

func parseByteRange(header string) (int64, int64, bool) {
	if !strings.HasPrefix(header, "bytes=") {
		return 0, 0, false
	}
	spec := strings.TrimSpace(strings.TrimPrefix(header, "bytes="))
	if strings.Contains(spec, ",") {
		return 0, 0, false
	}
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return 0, 0, false
	}
	start, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || start < 0 {
		return 0, 0, false
	}
	end := int64(-1)
	if strings.TrimSpace(parts[1]) != "" {
		end, err = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil {
			return 0, 0, false
		}
	}
	return start, end, true
}
func (h *MusicHandler) HandleCreatePlaylist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req service.CreatePlaylistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "閸欏倹鏆熼柨娆掝嚖"))
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
		c.JSON(http.StatusBadRequest, response.Error(400, "閺冪姵鏅ラ惃?ID"))
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
		c.JSON(http.StatusBadRequest, response.Error(400, "閺冪姵鏅ラ惃?ID"))
		return
	}
	var req service.UpdatePlaylistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "閸欏倹鏆熼柨娆掝嚖"))
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
		c.JSON(http.StatusBadRequest, response.Error(400, "閺冪姵鏅ラ惃?ID"))
		return
	}
	if err := h.svc.DeletePlaylist(id, userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("閸掔娀娅庨幋鎰"))
}

func (h *MusicHandler) HandleAddTrackToPlaylist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	playlistID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "閺冪姵鏅ラ惃?ID"))
		return
	}
	var req service.AddTrackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "閸欏倹鏆熼柨娆掝嚖"))
		return
	}
	if err := h.svc.AddTrackToPlaylist(playlistID, userID, req); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("濞ｈ濮為幋鎰"))
}

func (h *MusicHandler) HandleRemoveTrackFromPlaylist(c *gin.Context) {
	userID := c.GetUint64("user_id")
	playlistID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "閺冪姵鏅ラ惃?ID"))
		return
	}
	trackID, err := strconv.ParseUint(c.Param("trackId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "閺冪姵鏅ラ惃鍕锤閻?ID"))
		return
	}
	if err := h.svc.RemoveTrackFromPlaylist(playlistID, trackID, userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("缁夊娅庨幋鎰"))
}

func (h *MusicHandler) HandleGetLyrics(c *gin.Context) {
	userID := c.GetUint64("user_id")
	trackID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "閺冪姵鏅ラ惃?ID"))
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
		c.JSON(http.StatusBadRequest, response.Error(400, "閺冪姵鏅ラ惃?ID"))
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
		c.JSON(http.StatusBadRequest, response.Error(400, "閺冪姵鏅ラ惃?ID"))
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
	c.JSON(http.StatusOK, response.OK("鐎电厧鍙嗛幋鎰"))
}
