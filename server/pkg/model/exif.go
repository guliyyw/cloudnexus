package model

import "time"

type ExifMetadata struct {
	FileID        uint64     `json:"file_id,string" gorm:"primaryKey"`
	Make          string     `json:"make" gorm:"size:100"`
	Model         string     `json:"model" gorm:"size:100"`
	DateTimeTaken *time.Time `json:"datetime_taken"`
	Latitude      *float64   `json:"latitude"`
	Longitude     *float64   `json:"longitude"`
	ISO           *int       `json:"iso"`
	FNumber       *float64   `json:"f_number"`
	ExposureTime  string     `json:"exposure_time" gorm:"size:20"`
	FocalLength   *float64   `json:"focal_length"`
}
