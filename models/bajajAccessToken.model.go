package models

import "gorm.io/gorm"

// BajajAccessToken stores the access token for Bajaj broking API
type BajajAccessToken struct {
	gorm.Model
	Token     string `gorm:"type:text;not null" json:"token"`
	IsDeleted bool   `gorm:"default:false" json:"isDeleted"`
}

// TableName sets the table name for GORM
func (BajajAccessToken) TableName() string {
	return "bajaj_access_tokens"
}
