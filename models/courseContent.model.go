package models

import "gorm.io/gorm"

type CourseContent struct {
	gorm.Model
	CourseID    uint   `json:"course_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IsDeleted   bool   `gorm:"default:false"`
}

type ContentCompletion struct {
	gorm.Model
	UserID          uint          `json:"user_id" gorm:"index;not null"`
	CourseID        uint          `json:"course_id" gorm:"index;not null"`
	CourseContentID uint          `json:"course_content_id" gorm:"index;not null"`
	Status          string        `json:"status" gorm:"default:'COMPLETED'"`
	IsDeleted       bool          `gorm:"default:false"`
	User            User          `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Course          Course        `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE"`
	CourseContent   CourseContent `gorm:"foreignKey:CourseContentID;constraint:OnDelete:CASCADE"`
}
