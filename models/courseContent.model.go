package models

import "gorm.io/gorm"

type CourseContent struct {
	gorm.Model
	CourseID    uint   `json:"course_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IsDeleted   bool   `gorm:"default:false"`
}
