package models

import (
	"gorm.io/gorm"
)

type AmcStocks struct {
	gorm.Model      // Auto includes ID, CreatedAt, UpdatedAt, DeletedAt
	UserID     uint `gorm:"foreignKey:UserID"` //AMC id
	StockId    uint
	HoldingPer float32
	IsDeleted  bool `gorm:"default:false"`
}
