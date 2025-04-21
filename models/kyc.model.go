package models

import (
	"gorm.io/gorm"
)

type UserKYC struct {
	gorm.Model
	UserID     uint          `gorm:"not null;index"` // Foreign key to User table
	AdharID    uint          `gorm:"index"`          // Foreign key to AadharDetails table
	PanID      uint          `gorm:"index"`          // Foreign key to PanDetails table
	Aadhar     AadharDetails `gorm:"foreignKey:AdharID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Pan        PanDetails    `gorm:"foreignKey:PanID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	IsVerified bool          `gorm:"default:false"` // KYC verification status
	IsDeleted  bool          `gorm:"default:false"` // Soft delete flag
}

type AadharDetails struct {
	gorm.Model
	AadharNumber string `gorm:"unique;not null"` // Aadhar number must be unique and not null
	Name         string `gorm:"default:''"`      // Name on the Aadhar card
	ProfileImage string `gorm:"default:''"`      // Profile image
	DOB          string `gorm:"default:''"`      // Date of Birth
	Address      string `gorm:"default:''"`      // Address on the Aadhar card
	IsVerified   bool   `gorm:"default:false"`   // Verification status, default is false
	RefID        string `gorm:"default:''"`      // Reference ID
}

type PanDetails struct {
	gorm.Model
	PanNumber  string `gorm:"unique;not null"` // PAN number must be unique and not null
	Name       string `gorm:"default:''"`      // Name on the PAN card
	IsVerified bool   `gorm:"default:false"`   // Verification status, default is false
}
