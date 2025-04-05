package models

import (
	"time"

	"gorm.io/gorm"
)

type OTP struct {
	gorm.Model
	UserID      uint      `gorm:"not null" json:"user_id"`               // Foreign key to the user (optional if OTP is for specific users)
	Email       string    `gorm:"size:100;index" json:"email,omitempty"` // Email for OTP, if applicable
	Mobile      string    `gorm:"size:15;index" json:"mobile,omitempty"` // Mobile for OTP, if applicable
	Code        string    `gorm:"size:6;not null" json:"code"`           // The OTP code
	ExpiresAt   time.Time `gorm:"not null" json:"expires_at"`            // Expiry time for the OTP
	IsUsed      bool      `gorm:"default:false" json:"is_used"`
	Description string    `gorm:"size:255" json:"description,omitempty"` // Description of the OTP
	IsDeleted   bool      `gorm:"default:false"`                         // Flag to indicate if the OTP has been used
}
