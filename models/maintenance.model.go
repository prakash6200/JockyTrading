package models

import (
	"gorm.io/gorm"
)

type Maintenance struct {
	gorm.Model                // Auto includes ID, CreatedAt, UpdatedAt, DeletedAt
	AppMaintenance       bool `gorm:"default:false"`
	ForceUpdate          bool `gorm:"default:false"`
	IosLatestVersion     string
	AndroidLatestVersion string `gorm:"not null"`
	IsDeleted            bool   `gorm:"default:false"`
}
