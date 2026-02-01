package course

import "gorm.io/gorm"

// Module represents a section/module within a course
type Module struct {
	gorm.Model
	CourseID    uint   `json:"course_id" gorm:"index;not null"`
	Title       string `json:"title"`
	Description string `json:"description"`
	OrderIndex  int    `json:"order_index" gorm:"default:0"` // Module order in course
	IsDeleted   bool   `gorm:"default:false"`
}
