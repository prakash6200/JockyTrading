package course

import "gorm.io/gorm"

// MCQOption represents an option for a multiple choice question content
type MCQOption struct {
	gorm.Model
	ContentID  uint   `json:"content_id" gorm:"index;not null"`
	OptionText string `json:"option_text"`
	IsCorrect  bool   `json:"is_correct" gorm:"default:false"`
	OrderIndex int    `json:"order_index" gorm:"default:0"`
	IsDeleted  bool   `gorm:"default:false"`
}

// MCQAttempt represents a student's attempt at answering an MCQ
type MCQAttempt struct {
	gorm.Model
	UserID          uint   `json:"user_id" gorm:"index;not null"`
	ContentID       uint   `json:"content_id" gorm:"index;not null"`
	SelectedOptions string `json:"selected_options"` // JSON array of selected option IDs
	Score           int    `json:"score"`            // Score achieved
	MaxScore        int    `json:"max_score"`        // Maximum possible score
	IsCorrect       bool   `json:"is_correct" gorm:"default:false"`
	AttemptNumber   int    `json:"attempt_number" gorm:"default:1"`
	IsDeleted       bool   `gorm:"default:false"`
}
