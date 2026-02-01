package course

import (
	"time"

	"gorm.io/gorm"
)

// CertificateRequest represents a student's request for course completion certificate
type CertificateRequest struct {
	gorm.Model
	UserID          uint       `json:"user_id" gorm:"index;not null"`
	CourseID        uint       `json:"course_id" gorm:"index;not null"`
	EnrollmentID    uint       `json:"enrollment_id" gorm:"index;not null"`
	Status          string     `json:"status" gorm:"default:'PENDING'"` // PENDING, APPROVED, REJECTED
	RequestedAt     time.Time  `json:"requested_at"`
	ApprovedAt      *time.Time `json:"approved_at"`
	ApprovedBy      *uint      `json:"approved_by"`
	RejectionReason string     `json:"rejection_reason"`
	IsDeleted       bool       `gorm:"default:false"`
}

// Certificate represents an issued certificate for course completion
type Certificate struct {
	gorm.Model
	UserID            uint      `json:"user_id" gorm:"index;not null"`
	CourseID          uint      `json:"course_id" gorm:"index;not null"`
	CertificateURL    string    `json:"certificate_url"`
	CertificateNumber string    `json:"certificate_number" gorm:"unique"`
	IssuedAt          time.Time `json:"issued_at"`
	IsDeleted         bool      `gorm:"default:false"`
}
