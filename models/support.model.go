package models

import "gorm.io/gorm"

type SupportTicket struct {
	gorm.Model
	UserID      uint   `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status" gorm:"default:'open'"`
	Priority    string `json:"priority" gorm:"default:'medium'"`
	Category    string `json:"category" gorm:"default:'general'"`
	IsDeleted   bool   `json:"is_deleted" gorm:"default:false"`
}
