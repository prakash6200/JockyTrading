package models

import (
	"gorm.io/gorm"
)

type Course struct {
	gorm.Model
	Title       string `json:"title"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Duration    int64  `json:"duration"` // duration of the course
	Status      string `json:"status"`
	Rating      uint   `json:"default:0"`
	IsDeleted   bool   `gorm:"default:false"`
}
