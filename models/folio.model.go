package models

import (
	"gorm.io/gorm"
)

type Folio struct {
	gorm.Model      // Auto includes ID, CreatedAt, UpdatedAt, DeletedAt
	UserID     uint `gorm:"foreignKey:UserID"` //AMC id
	FolioNo    string
	IsDeleted  bool `gorm:"default:false"`
}
