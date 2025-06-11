package models

import (
	"time"

	"gorm.io/gorm"
)

// BankDetails model
type BankDetails struct {
	gorm.Model             // Auto includes ID, CreatedAt, UpdatedAt, DeletedAt
	BankName    string     `gorm:"default:''"`
	AccountNo   string     `gorm:"default:''"`
	HolderName  string     `gorm:"default:''"`
	IFSCCode    string     `gorm:"default:''"`
	BranchName  string     `gorm:"default:''"`
	AccountType string     `gorm:"type:text;default:'savings'"`
	UserID      uint       `gorm:"foreignKey:UserID"`
	Image       string     `gorm:"default:''"`
	IsVerified  bool       `gorm:"default:false" json:"isVerified"`
	VerifiedAt  *time.Time `json:"verifiedAt"`
	IsDeleted   bool       `gorm:"default:false"`
}
