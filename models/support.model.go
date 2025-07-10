package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SupportTicket struct {
	gorm.Model
	UserID    uint           `json:"user_id"`
	Title     string         `json:"title"`
	Subject   string         `json:"subject"`
	Message   datatypes.JSON `json:"message" gorm:"type:json"`
	Status    string         `json:"status" gorm:"default:'open'"`
	Priority  string         `json:"priority" gorm:"default:'medium'"`
	Category  string         `json:"category" gorm:"default:'general'"`
	IsDeleted bool           `json:"is_deleted" gorm:"default:false"`
}
