package basket

import (
	"gorm.io/gorm"
)

// MessageAction enum
const (
	ActionBuy     = "BUY"
	ActionSell    = "SELL"
	ActionHold    = "HOLD"
	ActionGeneral = "GENERAL"
)

// MessageSender enum
const (
	SenderAMC  = "AMC"
	SenderUser = "USER"
)

type BasketMessage struct {
	gorm.Model
	BasketID     uint   `gorm:"not null;index" json:"basketId"`
	SenderID     uint   `gorm:"not null" json:"senderId"`                    // UserID of sender (AMC or User)
	SenderType   string `gorm:"type:varchar(10);not null" json:"senderType"` // AMC or USER
	TargetUserID uint   `gorm:"default:0" json:"targetUserId"`               // For direct replies (0 if broadcast or generic to AMC)
	Action       string `gorm:"type:varchar(20);default:'GENERAL'" json:"action"`
	Message      string `gorm:"type:text;not null" json:"message"`

	// If true, it's a broadcast to all subscribers (from AMC)
	IsBroadcast bool `gorm:"default:false" json:"isBroadcast"`

	// Relations
	Basket Basket `gorm:"foreignKey:BasketID" json:"basket,omitempty"`
}

func (BasketMessage) TableName() string {
	return "basket_messages"
}
