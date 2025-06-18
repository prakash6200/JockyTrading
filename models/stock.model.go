package models

import (
	"gorm.io/gorm"
)

type Stocks struct {
	gorm.Model        // Auto includes ID, CreatedAt, UpdatedAt, DeletedAt
	Symbol     string `gorm:"unique;not null"`
	Name       string `gorm:"not null"`
	Exchange   string `gorm:"not null"`
	Isin       string `gorm:"unique;not null"`
	Series     string `gorm:"not null"`
	IsDeleted  bool   `gorm:"default:false"`
}
