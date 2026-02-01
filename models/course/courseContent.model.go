package course

import "gorm.io/gorm"

// CourseContent represents content within a module, organized by day
type CourseContent struct {
	gorm.Model
	CourseID    uint   `json:"course_id" gorm:"index;not null"`
	ModuleID    uint   `json:"module_id" gorm:"index;not null"`
	Day         int    `json:"day" gorm:"default:1"` // Day number within module
	Title       string `json:"title"`
	Description string `json:"description"`
	ContentType string `json:"content_type" gorm:"default:'TEXT'"` // TEXT, MCQ, VIDEO, IMAGE
	TextContent string `json:"text_content" gorm:"type:text"`      // For TEXT type
	VideoURL    string `json:"video_url"`                          // For VIDEO type
	ImageURL    string `json:"image_url"`                          // For IMAGE type
	OrderIndex  int    `json:"order_index" gorm:"default:0"`       // Order within day
	IsPublished bool   `json:"is_published" gorm:"default:false"`
	IsDeleted   bool   `gorm:"default:false"`
}

// ContentCompletion tracks user's completion of course content
type ContentCompletion struct {
	gorm.Model
	UserID          uint   `json:"user_id" gorm:"index;not null"`
	CourseID        uint   `json:"course_id" gorm:"index;not null"`
	CourseContentID uint   `json:"course_content_id" gorm:"index;not null"`
	Status          string `json:"status" gorm:"default:'COMPLETED'"`
	IsDeleted       bool   `gorm:"default:false"`
}
