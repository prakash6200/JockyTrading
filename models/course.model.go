package models

import "gorm.io/gorm"

type Course struct {
	gorm.Model
	Title       string `json:"title"`
	Description string `json:"description"`
	IsDeleted   bool   `gorm:"default:false"`
}
