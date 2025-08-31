package models

import "gorm.io/gorm"

type Review struct {
	gorm.Model
	UserID    uint   `gorm:"not null"`                                   // Who gave the review
	AmcId     uint   `gorm:"not null"`                                   // AMC ID
	Rating    int    `gorm:"not null;check:rating >= 1 AND rating <= 5"` // 1–5 rating
	Comment   string `gorm:"type:text;default:''"`                       // Optional comment
	IsDeleted bool   `gorm:"default:false"`
}
