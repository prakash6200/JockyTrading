package models

import (
	"time"

	"gorm.io/gorm"
)

type LoginTracking struct {
	gorm.Model
	UserID    uint      `json:"user_id"`
	IPAddress string    `json:"ip_address"`
	Device    string    `json:"device"`
	Timestamp time.Time `json:"timestamp"`
	IsDeleted bool      `gorm:"default:false"`
}
