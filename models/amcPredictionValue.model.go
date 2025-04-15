package models

import "gorm.io/gorm"

type AMCPredictionValue struct {
	gorm.Model
	UserID      uint   `gorm:"not null"`     // AMC user ID
	Title       string `gorm:"not null"`     // Title for the prediction
	Prediction  int    `gorm:"not null"`     // Predicted value (e.g., 23, 10)
	Achieved    *int   `gorm:"default:NULL"` // Achieved value (nullable)
	Description string `gorm:"type:text"`    // Detailed description
	IsDeleted   bool   `gorm:"default:false"`

	// Relationship
	User User `gorm:"foreignKey:UserID"`
}
