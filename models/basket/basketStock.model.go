package basket

import (
	"gorm.io/gorm"
)

// BasketStock represents stocks in each basket version
type BasketStock struct {
	gorm.Model
	BasketVersionID uint    `gorm:"not null;index" json:"basketVersionId"`
	StockID         uint    `gorm:"not null;index" json:"stockId"`
	Quantity        int     `gorm:"not null;default:1" json:"quantity"`
	Weightage       float64 `gorm:"not null;default:0" json:"weightage"` // Percentage weight in basket
	PriceAtCreation float64 `gorm:"default:0" json:"priceAtCreation"`
	PriceAtApproval float64 `gorm:"default:0" json:"priceAtApproval"`
	OrderType       string  `gorm:"type:varchar(10);default:'MARKET'" json:"orderType"` // MARKET, LIMIT
	TargetPrice     float64 `gorm:"default:0" json:"targetPrice"`
	StopLossPrice   float64 `gorm:"default:0" json:"stopLossPrice"`
	Token           int     `gorm:"default:0" json:"token"`                    // Exchange token for Bajaj API
	Symbol          string  `gorm:"type:varchar(50);default:''" json:"symbol"` // Stock symbol
	Units           int     `gorm:"default:1" json:"units"`                    // Number of units
	StockName       string  `gorm:"-" json:"stockName"`                        // Stock full name (populated via join)
	IsDeleted       bool    `gorm:"default:false" json:"isDeleted"`

	// Relations
	BasketVersion BasketVersion `gorm:"foreignKey:BasketVersionID" json:"-"`
}

func (BasketStock) TableName() string {
	return "basket_stocks"
}
