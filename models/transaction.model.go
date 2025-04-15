package models

import (
	"gorm.io/gorm"
)

// Transactions model
type Transactions struct {
	gorm.Model             // Auto includes ID, CreatedAt, UpdatedAt, DeletedAt
	UserID          uint   `gorm:"foreignKey:UserID"`
	TransactionType string `gorm:"not null"` // DEPOSIT/WITHDRAW
	Amount          uint   `gorm:"not null"`
	Status          string `gorm:"not null"` // pending/completed
	IsDeleted       bool   `gorm:"default:false"`
}
