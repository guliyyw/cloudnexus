package model

type Playlist struct {
	BaseModel
	OwnerID    uint64 `json:"owner_id,string" gorm:"index:idx_owner"`
	Name       string `json:"name" gorm:"not null;size:200"`
	IsPublic   bool   `json:"is_public" gorm:"default:false"`
	TrackCount int    `json:"track_count" gorm:"-"`
}

type PlaylistTrack struct {
	PlaylistID uint64 `json:"playlist_id,string" gorm:"primaryKey"`
	TrackID    uint64 `json:"track_id,string" gorm:"primaryKey"`
	Source     string `json:"source" gorm:"size:10"`
	SortOrder  int    `json:"sort_order"`
}
