package model

import "time"

type DramaProject struct {
	BaseModel
	OwnerID     uint64 `json:"owner_id,string" gorm:"not null;index"`
	Title       string `json:"title" gorm:"not null;size:200"`
	Description string `json:"description" gorm:"size:1000"`
	Preface     string `json:"preface" gorm:"type:text"`
	RawScript   string `json:"raw_script,omitempty" gorm:"type:text"`
	Settings    string `json:"settings" gorm:"type:jsonb;default:'{}'"`
}

type DramaStoryboard struct {
	BaseModel
	ProjectID       uint64 `json:"project_id,string" gorm:"not null;index"`
	OwnerID         uint64 `json:"owner_id,string" gorm:"not null;index"`
	Seq             int    `json:"seq" gorm:"not null;index"`
	Title           string `json:"title" gorm:"size:200"`
	Content         string `json:"content" gorm:"type:text"`
	Original        string `json:"original" gorm:"type:text"`
	Prompt          string `json:"prompt" gorm:"type:text"`
	Dialogue        string `json:"dialogue" gorm:"type:text"`
	SceneAnchor     string `json:"scene_anchor" gorm:"type:text"`
	Plot            string `json:"plot" gorm:"type:text"`
	Modified        bool   `json:"modified" gorm:"default:false;index"`
	ImageFileID     uint64 `json:"image_file_id,string" gorm:"default:0"`
	AudioFileID     uint64 `json:"audio_file_id,string" gorm:"default:0"`
	AudioDurationMS int    `json:"audio_duration_ms" gorm:"default:0"`
	SubtitleASS     string `json:"subtitle_ass" gorm:"type:text"`
	VideoFileID     uint64 `json:"video_file_id,string" gorm:"default:0"`
}

type DramaStoryboardMedia struct {
	BaseModel
	ProjectID    uint64 `json:"project_id,string" gorm:"not null;index"`
	StoryboardID uint64 `json:"storyboard_id,string" gorm:"not null;index"`
	SegmentID    uint64 `json:"segment_id,string" gorm:"default:0;index"`
	OwnerID      uint64 `json:"owner_id,string" gorm:"not null;index"`
	Kind         string `json:"kind" gorm:"not null;size:20;index"`
	FileID       uint64 `json:"file_id,string" gorm:"not null;default:0"`
	Source       string `json:"source" gorm:"size:40;default:'generated'"`
	Prompt       string `json:"prompt" gorm:"type:text"`
	SortOrder    int    `json:"sort_order" gorm:"default:0;index"`
	Selected     bool   `json:"selected" gorm:"default:false;index"`
}

type DramaStoryboardSegment struct {
	BaseModel
	ProjectID         uint64 `json:"project_id,string" gorm:"not null;index"`
	StoryboardID      uint64 `json:"storyboard_id,string" gorm:"not null;index"`
	OwnerID           uint64 `json:"owner_id,string" gorm:"not null;index"`
	Seq               int    `json:"seq" gorm:"not null;index"`
	Title             string `json:"title" gorm:"size:200"`
	DurationSec       int    `json:"duration_sec" gorm:"default:3"`
	Purpose           string `json:"purpose" gorm:"type:text"`
	Characters        string `json:"characters" gorm:"type:text"`
	Scene             string `json:"scene" gorm:"size:200"`
	Dialogue          string `json:"dialogue" gorm:"type:text"`
	Action            string `json:"action" gorm:"type:text"`
	Shot              string `json:"shot" gorm:"type:text"`
	CompositionPrompt string `json:"composition_prompt" gorm:"type:text"`
	ReferencePrompt   string `json:"reference_prompt" gorm:"type:text"`
	VideoPrompt       string `json:"video_prompt" gorm:"type:text"`
	NegativePrompt    string `json:"negative_prompt" gorm:"type:text"`
	ReferenceFileID   uint64 `json:"reference_file_id,string" gorm:"default:0"`
	VideoFileID       uint64 `json:"video_file_id,string" gorm:"default:0"`
}

type DramaAsset struct {
	BaseModel
	ProjectID       uint64 `json:"project_id,string" gorm:"not null;index"`
	OwnerID         uint64 `json:"owner_id,string" gorm:"not null;index"`
	Type            string `json:"type" gorm:"not null;size:20;index"`
	Name            string `json:"name" gorm:"not null;size:120"`
	Description     string `json:"description" gorm:"type:text"`
	ReferencePrompt string `json:"reference_prompt" gorm:"type:text"`
	VoiceName       string `json:"voice_name" gorm:"size:100"`
	ReferenceFileID uint64 `json:"reference_file_id,string" gorm:"default:0"`
}

type DramaTask struct {
	BaseModel
	ProjectID    uint64     `json:"project_id,string" gorm:"not null;index"`
	OwnerID      uint64     `json:"owner_id,string" gorm:"not null;index"`
	Type         string     `json:"type" gorm:"not null;size:30;index"`
	Status       string     `json:"status" gorm:"not null;size:30;index"`
	Progress     int        `json:"progress" gorm:"default:0"`
	Message      string     `json:"message" gorm:"size:1000"`
	Payload      string     `json:"payload" gorm:"type:jsonb;default:'{}'"`
	StoryboardID uint64     `json:"storyboard_id,string" gorm:"default:0"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
}

type DramaSetting struct {
	BaseModel
	OwnerID       uint64 `json:"owner_id,string" gorm:"not null;uniqueIndex"`
	ComfyUIURL    string `json:"comfyui_url" gorm:"size:500"`
	ImageSettings string `json:"image_settings" gorm:"type:jsonb;default:'{}'"`
	TTSEngine     string `json:"tts_engine" gorm:"size:50;default:'edge-tts'"`
	TTSConfig     string `json:"tts_config" gorm:"type:jsonb;default:'{}'"`
	VideoSettings string `json:"video_settings" gorm:"type:jsonb;default:'{}'"`
	StorageRoot   string `json:"storage_root" gorm:"size:200;default:'短剧工坊'"`
}
