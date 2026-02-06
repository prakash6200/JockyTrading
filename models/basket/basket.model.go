package basket

import (
	"gorm.io/gorm"
)

// BasketType enum values
const (
	BasketTypeIntraHour = "INTRA_HOUR"
	BasketTypeIntraday  = "INTRADAY"
	BasketTypeDelivery  = "DELIVERY"
)

// Basket is the master entity for investment baskets
type Basket struct {
	gorm.Model
	Name             string  `gorm:"not null" json:"name"`
	Description      string  `gorm:"type:text" json:"description"`
	AMCID            uint    `gorm:"not null;index" json:"amcId"`
	BasketType       string  `gorm:"not null;type:varchar(20)" json:"basketType"` // INTRA_HOUR, INTRADAY, DELIVERY
	CurrentVersionID *uint   `json:"currentVersionId"`
	SubscriptionFee  float64 `gorm:"default:0" json:"subscriptionFee"`
	IsFeeBased       bool    `gorm:"default:false" json:"isFeeBased"`
	IsDeleted        bool    `gorm:"default:false" json:"isDeleted"`

	// Relations
	Versions       []BasketVersion `gorm:"foreignKey:BasketID" json:"versions,omitempty"`
	CurrentVersion *BasketVersion  `gorm:"foreignKey:CurrentVersionID" json:"currentVersion,omitempty"`
}

func (Basket) TableName() string {
	return "baskets"
}
