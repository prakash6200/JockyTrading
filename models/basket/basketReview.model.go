package basket

import (
	"fib/models"
	"time"

	"gorm.io/gorm"
)

// ReviewStatus defines the status of a review
type ReviewStatus string

const (
	ReviewStatusPending  ReviewStatus = "PENDING"
	ReviewStatusApproved ReviewStatus = "APPROVED"
	ReviewStatusRejected ReviewStatus = "REJECTED"
)

type BasketReview struct {
	gorm.Model
	BasketID uint         `gorm:"not null;index" json:"basketId"`
	UserID   uint         `gorm:"not null;index" json:"userId"`
	Rating   int          `gorm:"not null;check:rating >= 1 AND rating <= 5" json:"rating"`
	Review   string       `gorm:"type:text" json:"review"`
	Status   ReviewStatus `gorm:"type:varchar(20);default:'PENDING'" json:"status"`

	// AMC Reply
	Reply     string     `gorm:"type:text" json:"reply"`
	RepliedAt *time.Time `json:"repliedAt"`

	// Associations - omit in JSON list unless Preloaded
	User models.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
