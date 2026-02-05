package basket

import (
	"time"

	"gorm.io/gorm"
)

// BasketTimeSlot manages time slots for INTRA_HOUR baskets (admin-managed)
type BasketTimeSlot struct {
	gorm.Model
	BasketVersionID   uint       `gorm:"not null;uniqueIndex" json:"basketVersionId"`
	ScheduledDate     time.Time  `gorm:"not null;type:date" json:"scheduledDate"`
	StartTime         time.Time  `gorm:"not null" json:"startTime"`
	EndTime           time.Time  `gorm:"not null" json:"endTime"`
	DurationMinutes   int        `gorm:"not null" json:"durationMinutes"`
	Timezone          string     `gorm:"default:'Asia/Kolkata'" json:"timezone"`
	SetByAdminID      uint       `gorm:"not null" json:"setByAdminId"`
	ActualPublishTime *time.Time `json:"actualPublishTime"`
	ActualExpireTime  *time.Time `json:"actualExpireTime"`

	// Relations
	BasketVersion BasketVersion `gorm:"foreignKey:BasketVersionID" json:"-"`
}

func (BasketTimeSlot) TableName() string {
	return "basket_time_slots"
}
