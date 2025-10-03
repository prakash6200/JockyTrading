package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ProfileImage          string    `gorm:"default:''"`
	Name                  string    `gorm:"default:''"`
	Email                 string    `gorm:"unique;not null"`
	Mobile                string    `gorm:"default:''"`
	Role                  string    `gorm:"default:'USER'"` // Default role is USER, AMC, ADMIN,
	Password              string    `gorm:"not null"`
	BankDetails           uint      `gorm:"foreignKey:BankID"` // Corrected foreign key reference
	UserKYC               uint      `gorm:"foreignKey:KycID"`  // Corrected foreign key reference
	IsMobileVerified      bool      `gorm:"default:false"`
	IsEmailVerified       bool      `gorm:"default:false"`
	MainBalance           uint      `gorm:"default:0"`
	LastLogin             time.Time `gorm:"default:NULL"`
	PanNumber             string
	IsAdharVerified       bool `gorm:"default:false"`
	IsPanVerified         bool `gorm:"default:false"`
	Address               string
	City                  string
	State                 string
	PinCode               string
	ContactPersonName     string
	ContactPerDesignation string
	FundName              string
	EquityPer             float32
	DebtPer               float32
	CashSplit             float32
	FailedLoginAttempts   int        `gorm:"default:0"`
	LastFailedLogin       *time.Time `json:"last_failed_login"`
	IsBlocked             bool       `gorm:"default:false"`
	BlockedUntil          *time.Time `json:"blocked_until"`
	IsDeleted             bool       `gorm:"default:false"`
}
