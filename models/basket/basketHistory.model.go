package basket

import (
	"gorm.io/gorm"
)

// HistoryAction enum values
const (
	ActionCreated      = "CREATED"
	ActionSubmitted    = "SUBMITTED"
	ActionApproved     = "APPROVED"
	ActionRejected     = "REJECTED"
	ActionTimeSlotSet  = "TIME_SLOT_SET"
	ActionWentLive     = "WENT_LIVE"
	ActionExpired      = "EXPIRED"
	ActionUnpublished  = "UNPUBLISHED"
	ActionStockAdded   = "STOCK_ADDED"
	ActionStockRemoved = "STOCK_REMOVED"
)

// ActorType enum values
const (
	ActorAMC    = "AMC"
	ActorAdmin  = "ADMIN"
	ActorSystem = "SYSTEM"
	ActorUser   = "USER"
)

// BasketHistory is the audit log for all basket actions
type BasketHistory struct {
	gorm.Model
	BasketVersionID uint   `gorm:"not null;index" json:"basketVersionId"`
	Action          string `gorm:"not null;type:varchar(30)" json:"action"`
	ActorID         uint   `gorm:"not null" json:"actorId"`
	ActorType       string `gorm:"not null;type:varchar(10)" json:"actorType"` // AMC, ADMIN, SYSTEM, USER
	Comments        string `gorm:"type:text" json:"comments"`
	Metadata        string `gorm:"type:jsonb" json:"metadata"` // JSON for additional info

	// Relations
	BasketVersion BasketVersion `gorm:"foreignKey:BasketVersionID" json:"-"`
}

func (BasketHistory) TableName() string {
	return "basket_history"
}
