package models

import (
	"gorm.io/gorm"
)

type StockPrices struct {
	gorm.Model
	StockID uint    `gorm:"index"` // Link to Stocks table
	Date    string  `gorm:"index"` // YYYY-MM-DD format
	Close   float64 // Closing price
}
