package models

import (
	"gorm.io/gorm"
)

type AmcStocks struct {
	gorm.Model      // Auto includes ID, CreatedAt, UpdatedAt, DeletedAt
	UserId     uint // AMC id
	StockId    uint
	IsDeleted  bool `gorm:"default:false"`
}
