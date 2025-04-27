package models

import (
	"gorm.io/gorm"
)

type Stocks struct {
	gorm.Model        // Auto includes ID, CreatedAt, UpdatedAt, DeletedAt
	Symbol     string `gorm:"unique"`
	Name       string `gorm:"not null"`
	Sector     string `gorm:"not null"`
	Exchange   string `gorm:"not null"` // New field to store the exchange
	IsDeleted  bool   `gorm:"default:false"`
}
