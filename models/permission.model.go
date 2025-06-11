package models

import (
	"gorm.io/gorm"
)

type Permission struct {
	gorm.Model
	ID         uint `gorm:"primaryKey"`
	UserID     uint `gorm:"not null"`          // Foreign key
	User       User `gorm:"foreignKey:UserID"` // Association with User
	Role       string
	Permission string `gorm:"type:varchar(255)"` // e.g., "add-member"
	IsDeleted  bool   `gorm:"default:false"`
}
