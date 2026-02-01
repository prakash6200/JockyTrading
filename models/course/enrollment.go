package course

import (
	"time"

	"gorm.io/gorm"
)

// Enrollment tracks a user's enrollment in a course with progress
type Enrollment struct {
	gorm.Model
	UserID            uint       `json:"user_id" gorm:"index;not null"`
	CourseID          uint       `json:"course_id" gorm:"index;not null"`
	Status            string     `json:"status" gorm:"default:'ENROLLED'"` // ENROLLED, IN_PROGRESS, COMPLETED
	Progress          float64    `json:"progress" gorm:"default:0"`        // Completion percentage (0-100)
	CompletedContents int        `json:"completed_contents" gorm:"default:0"`
	TotalContents     int        `json:"total_contents" gorm:"default:0"`
	CompletedAt       *time.Time `json:"completed_at"`
	IsDeleted         bool       `gorm:"default:false"`
}
