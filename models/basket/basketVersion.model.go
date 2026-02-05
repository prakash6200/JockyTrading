package basket

import (
	"time"

	"gorm.io/gorm"
)

// VersionStatus enum values
const (
	StatusDraft           = "DRAFT"
	StatusPendingApproval = "PENDING_APPROVAL"
	StatusApproved        = "APPROVED"
	StatusScheduled       = "SCHEDULED"
	StatusPublished       = "PUBLISHED"
	StatusExpired         = "EXPIRED"
	StatusUnpublished     = "UNPUBLISHED"
	StatusRejected        = "REJECTED"
)

// BasketVersion tracks each version of a basket
type BasketVersion struct {
	gorm.Model
	BasketID        uint       `gorm:"not null;index" json:"basketId"`
	VersionNumber   int        `gorm:"not null" json:"versionNumber"`
	Status          string     `gorm:"not null;type:varchar(20);default:'DRAFT'" json:"status"`
	SubmittedAt     *time.Time `json:"submittedAt"`
	ApprovedAt      *time.Time `json:"approvedAt"`
	ApprovedBy      *uint      `json:"approvedBy"`
	RejectionReason string     `gorm:"type:text" json:"rejectionReason"`
	PriceAtApproval float64    `gorm:"default:0" json:"priceAtApproval"`
	TradingDate     *time.Time `json:"tradingDate"` // For INTRADAY: specific trading date
	IsDeleted       bool       `gorm:"default:false" json:"isDeleted"`

	// Relations
	Basket   Basket          `gorm:"foreignKey:BasketID" json:"basket,omitempty"`
	Stocks   []BasketStock   `gorm:"foreignKey:BasketVersionID" json:"stocks,omitempty"`
	TimeSlot *BasketTimeSlot `gorm:"foreignKey:BasketVersionID" json:"timeSlot,omitempty"`
	History  []BasketHistory `gorm:"foreignKey:BasketVersionID" json:"history,omitempty"`
}

func (BasketVersion) TableName() string {
	return "basket_versions"
}
