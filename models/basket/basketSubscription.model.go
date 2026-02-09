package basket

import (
	"time"

	"gorm.io/gorm"
)

// SubscriptionStatus enum values
const (
	SubscriptionActive    = "ACTIVE"
	SubscriptionExpired   = "EXPIRED"
	SubscriptionCancelled = "CANCELLED"
)

// SubscriptionPeriod enum values
const (
	PeriodMonthly = "MONTHLY"
	PeriodYearly  = "YEARLY"
)

// BasketSubscription tracks user subscriptions to baskets
type BasketSubscription struct {
	gorm.Model
	UserID             uint       `gorm:"not null;index" json:"userId"`
	BasketID           uint       `gorm:"not null;index" json:"basketId"`
	BasketVersionID    uint       `gorm:"not null" json:"basketVersionId"`
	SubscribedAt       time.Time  `gorm:"not null" json:"subscribedAt"`
	SubscriptionPrice  float64    `gorm:"not null;default:0" json:"subscriptionPrice"`
	BasketPrice        float64    `gorm:"not null;default:0" json:"basketPrice"` // Price of basket at subscription
	Status             string     `gorm:"not null;type:varchar(20);default:'ACTIVE'" json:"status"`
	SubscriptionPeriod string     `gorm:"type:varchar(20);default:'MONTHLY'" json:"subscriptionPeriod"` // MONTHLY or YEARLY
	ExpiresAt          *time.Time `json:"expiresAt"`
	ReminderSent       bool       `gorm:"default:false" json:"reminderSent"` // Track if expiry reminder was sent
	PaymentID          string     `json:"paymentId"`
	IsDeleted          bool       `gorm:"default:false" json:"isDeleted"`

	// Relations
	Basket        Basket        `gorm:"foreignKey:BasketID" json:"basket,omitempty"`
	BasketVersion BasketVersion `gorm:"foreignKey:BasketVersionID" json:"basketVersion,omitempty"`
}

func (BasketSubscription) TableName() string {
	return "basket_subscriptions"
}
