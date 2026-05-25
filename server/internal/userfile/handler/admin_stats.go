package handler

import (
	"net/http"

	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminStatsHandler struct {
	db *gorm.DB
}

func NewAdminStatsHandler(db *gorm.DB) *AdminStatsHandler {
	return &AdminStatsHandler{db: db}
}

type AdminStats struct {
	UserCount              int64  `json:"user_count"`
	FileCount              int64  `json:"file_count"`
	StorageUsedBytes       int64  `json:"storage_used_bytes"`
	AlbumCount             int64  `json:"album_count"`
	AlbumFileCount         int64  `json:"album_file_count"`
	MusicTrackCount        int64  `json:"music_track_count"`
	PlaylistCount          int64  `json:"playlist_count"`
	DashboardLastUpdated   string `json:"dashboard_last_updated"`
}

func (h *AdminStatsHandler) HandleAdminStats(c *gin.Context) {
	var stats AdminStats

	h.db.Table("users").Count(&stats.UserCount)
	h.db.Table("files").Where("deleted_at IS NULL").Count(&stats.FileCount)
	h.db.Table("files").Where("deleted_at IS NULL AND is_dir = false").Select("COALESCE(SUM(size), 0)").Row().Scan(&stats.StorageUsedBytes)
	h.db.Table("albums").Count(&stats.AlbumCount)
	h.db.Table("album_files").Count(&stats.AlbumFileCount)
	h.db.Table("public_tracks").Count(&stats.MusicTrackCount)
	h.db.Table("playlists").Count(&stats.PlaylistCount)

	var lastUpdated *string
	h.db.Table("dashboard_health_snapshots").Select("MAX(timestamp)::text").Row().Scan(&lastUpdated)
	if lastUpdated != nil {
		stats.DashboardLastUpdated = *lastUpdated
	}

	c.JSON(http.StatusOK, response.OKWithData(stats))
}
