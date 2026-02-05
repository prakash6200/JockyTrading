package models

import (
	"gorm.io/gorm"
)

// Stocks represents scrip master data for trading
type Stocks struct {
	gorm.Model
	ExchID         string  `gorm:"column:exch_id;type:varchar(20)" json:"exchId"`
	Token          int     `gorm:"column:token;index" json:"token"` // Exchange token for Bajaj API
	Symbol         string  `gorm:"column:symbol;index;not null" json:"symbol"`
	Series         string  `gorm:"column:series;type:varchar(20)" json:"series"`
	FullName       string  `gorm:"column:full_name;type:text" json:"fullName"`
	Expiry         string  `gorm:"column:expiry;type:varchar(50)" json:"expiry"`
	StrikePrice    float64 `gorm:"column:strike_price" json:"strikePrice"`
	MarketLot      int     `gorm:"column:mkt_lot" json:"marketLot"`
	InstrumentType string  `gorm:"column:inst_type;type:varchar(50)" json:"instrumentType"`
	ISIN           string  `gorm:"column:isin;type:varchar(50);index" json:"isin"`
	FaceValue      float64 `gorm:"column:face_value" json:"faceValue"`
	TickSize       float64 `gorm:"column:tick_size" json:"tickSize"`
	Sector         string  `gorm:"column:sector;type:text" json:"sector"`
	Industry       string  `gorm:"column:industry;type:text" json:"industry"`
	MarketCap      float64 `gorm:"column:mkt_cap" json:"marketCap"`
	MarketCapType  string  `gorm:"column:mkt_cap_type;type:varchar(20)" json:"marketCapType"`
	IndexSymbol    string  `gorm:"column:index_symbol;type:text" json:"indexSymbol"`
	// Legacy fields (kept for compatibility)
	Name      string `gorm:"column:name" json:"name"`
	Exchange  string `gorm:"column:exchange" json:"exchange"`
	IsDeleted bool   `gorm:"default:false" json:"isDeleted"`
}

func (Stocks) TableName() string {
	return "stocks"
}
