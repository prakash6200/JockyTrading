package models

import (
	"gorm.io/gorm"
)

type Enrollment struct {
	gorm.Model
	UserID    uint   `json:"user_id" gorm:"index;not null"`
	CourseID  uint   `json:"course_id" gorm:"index;not null"`
	Status    string `json:"status" gorm:"default:'ENROLLED'"`
	IsDeleted bool   `gorm:"default:false"`
	User      User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Course    Course `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE"`
}
