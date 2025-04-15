package models

import (
	"gorm.io/gorm"
)

type AMCProfile struct {
	gorm.Model
	UserID uint `gorm:"uniqueIndex"` // One-to-one relationship
	User   User `gorm:"foreignKey:UserID"`

	AmcName           string `gorm:"not null"` // e.g., "AMC Asset Management"
	AccountManager    string `gorm:"not null"`
	Email             string `gorm:"not null"`
	PhoneNumber       string `gorm:"not null"`
	ManagerExperience int    `gorm:"default:0"` // In years
	AboutManager      string `gorm:"type:text"`

	RequiredDocument  string `gorm:"default:''"` // Paths to uploaded files (as string or file refs)
	OptionalDocument1 string `gorm:"default:''"`
	OptionalDocument2 string `gorm:"default:''"`
	AmcLogo           string `gorm:"default:''"`
	ManagerImage      string `gorm:"default:''"`
}
