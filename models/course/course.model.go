package course

import "gorm.io/gorm"

// Course represents a learning course
type Course struct {
	gorm.Model
	Title        string `json:"title"`
	Description  string `json:"description"`
	Author       string `json:"author"`
	Duration     int64  `json:"duration" gorm:"default:0"`     // duration in hours
	Status       string `json:"status" gorm:"default:'DRAFT'"` // DRAFT, ACTIVE, INACTIVE
	Rating       uint   `json:"rating" gorm:"default:0"`
	ThumbnailURL string `json:"thumbnail_url"`
	IsPublished  bool   `json:"is_published" gorm:"default:false"`
	IsDeleted    bool   `gorm:"default:false"`
}
