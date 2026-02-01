package controllers

import (
	"encoding/json"
	"fib/database"
	"fib/middleware"
	"fib/models"
	courseModels "fib/models/course"

	"github.com/gofiber/fiber/v2"
)

// SubmitMCQAnswer submits and evaluates an MCQ answer
func SubmitMCQAnswer(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	courseID := c.Locals("courseID").(int)
	contentID := c.Locals("contentID").(int)

	// Check enrollment
	var enrollment courseModels.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&enrollment).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "User not enrolled in this course!", nil)
	}

	// Check content exists and is MCQ
	var content courseModels.CourseContent
	if err := database.Database.Db.Where("id = ? AND course_id = ? AND is_deleted = ? AND is_published = ?", contentID, courseID, false, true).First(&content).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Content not found!", nil)
	}

	if content.ContentType != "MCQ" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Content is not an MCQ!", nil)
	}

	reqData := new(struct {
		SelectedOptionIDs []uint `json:"selected_option_ids"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	if len(reqData.SelectedOptionIDs) == 0 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Please select at least one option!", nil)
	}

	// Get correct options
	var correctOptions []courseModels.MCQOption
	database.Database.Db.Where("content_id = ? AND is_correct = ? AND is_deleted = ?", contentID, true, false).Find(&correctOptions)

	// Calculate score
	correctOptionIDs := make(map[uint]bool)
	for _, opt := range correctOptions {
		correctOptionIDs[opt.ID] = true
	}

	correctCount := 0
	for _, selectedID := range reqData.SelectedOptionIDs {
		if correctOptionIDs[selectedID] {
			correctCount++
		}
	}

	isCorrect := correctCount == len(correctOptions) && len(reqData.SelectedOptionIDs) == len(correctOptions)

	// Get attempt number
	var attemptCount int64
	database.Database.Db.Model(&courseModels.MCQAttempt{}).Where("user_id = ? AND content_id = ? AND is_deleted = ?", userID, contentID, false).Count(&attemptCount)

	// Store selected options as JSON
	selectedJSON, _ := json.Marshal(reqData.SelectedOptionIDs)

	attempt := courseModels.MCQAttempt{
		UserID:          userID,
		ContentID:       uint(contentID),
		SelectedOptions: string(selectedJSON),
		Score:           correctCount,
		MaxScore:        len(correctOptions),
		IsCorrect:       isCorrect,
		AttemptNumber:   int(attemptCount) + 1,
	}

	if err := database.Database.Db.Create(&attempt).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to submit answer!", nil)
	}

	// If correct, mark content as completed
	if isCorrect {
		var existingCompletion courseModels.ContentCompletion
		if err := database.Database.Db.Where("user_id = ? AND course_content_id = ? AND is_deleted = ?", userID, contentID, false).First(&existingCompletion).Error; err != nil {
			completion := courseModels.ContentCompletion{
				UserID:          userID,
				CourseID:        uint(courseID),
				CourseContentID: uint(contentID),
				Status:          "COMPLETED",
			}
			database.Database.Db.Create(&completion)

			// Update enrollment progress
			updateEnrollmentProgress(userID, uint(courseID))
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Answer submitted!", fiber.Map{
		"attempt":    attempt,
		"is_correct": isCorrect,
		"score":      correctCount,
		"max_score":  len(correctOptions),
	})
}

// GetUserProgress gets the user's progress in a course
func GetUserProgress(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	courseID := c.Locals("courseID").(int)

	var enrollment courseModels.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&enrollment).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "User not enrolled in this course!", nil)
	}

	// Get completed content IDs
	var completedContents []courseModels.ContentCompletion
	database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).Find(&completedContents)

	completedIDs := make([]uint, len(completedContents))
	for i, cc := range completedContents {
		completedIDs[i] = cc.CourseContentID
	}

	// Get module-wise progress
	var modules []courseModels.Module
	database.Database.Db.Where("course_id = ? AND is_deleted = ?", courseID, false).Order("order_index asc").Find(&modules)

	type ModuleProgress struct {
		ModuleID          uint    `json:"module_id"`
		ModuleName        string  `json:"module_name"`
		TotalContents     int64   `json:"total_contents"`
		CompletedContents int64   `json:"completed_contents"`
		Progress          float64 `json:"progress"`
	}

	moduleProgress := make([]ModuleProgress, len(modules))
	for i, mod := range modules {
		var totalContent int64
		var completedContent int64

		database.Database.Db.Model(&courseModels.CourseContent{}).Where("module_id = ? AND is_deleted = ? AND is_published = ?", mod.ID, false, true).Count(&totalContent)
		database.Database.Db.Model(&courseModels.ContentCompletion{}).
			Joins("JOIN course_contents ON content_completions.course_content_id = course_contents.id").
			Where("content_completions.user_id = ? AND course_contents.module_id = ? AND content_completions.is_deleted = ?", userID, mod.ID, false).
			Count(&completedContent)

		progress := float64(0)
		if totalContent > 0 {
			progress = float64(completedContent) / float64(totalContent) * 100
		}

		moduleProgress[i] = ModuleProgress{
			ModuleID:          mod.ID,
			ModuleName:        mod.Title,
			TotalContents:     totalContent,
			CompletedContents: completedContent,
			Progress:          progress,
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Progress fetched successfully!", fiber.Map{
		"enrollment":      enrollment,
		"completed_ids":   completedIDs,
		"module_progress": moduleProgress,
	})
}

// updateEnrollmentProgress updates the enrollment progress after content completion
func updateEnrollmentProgress(userID uint, courseID uint) {
	var totalContent int64
	var completedContent int64

	database.Database.Db.Model(&courseModels.CourseContent{}).Where("course_id = ? AND is_deleted = ? AND is_published = ?", courseID, false, true).Count(&totalContent)
	database.Database.Db.Model(&courseModels.ContentCompletion{}).Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).Count(&completedContent)

	var enrollment courseModels.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&enrollment).Error; err != nil {
		return
	}

	enrollment.CompletedContents = int(completedContent)
	enrollment.TotalContents = int(totalContent)

	if totalContent > 0 {
		enrollment.Progress = float64(completedContent) / float64(totalContent) * 100
	}

	if enrollment.Progress >= 100 {
		enrollment.Status = "COMPLETED"
		now := enrollment.UpdatedAt
		enrollment.CompletedAt = &now
	} else if enrollment.Progress > 0 {
		enrollment.Status = "IN_PROGRESS"
	}

	database.Database.Db.Save(&enrollment)
}
