package controllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	courseModels "fib/models/course"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// ContentWithMCQ represents content with MCQ options
type ContentWithMCQ struct {
	courseModels.CourseContent
	MCQOptions  []courseModels.MCQOption `json:"mcq_options,omitempty"`
	IsCompleted bool                     `json:"is_completed"`
}

func GetCourseContent(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	// Retrieve validated course ID
	courseID, _ := strconv.Atoi(c.Locals("courseID").(string))

	// Get optional filters from query params
	moduleIDStr := c.Query("module_id")
	dayStr := c.Query("day")
	contentType := c.Query("content_type")

	// Retrieve validated pagination request
	reqData, _ := c.Locals("validatedCourseContentList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})

	// Set default pagination
	page := 1
	limit := 10
	if reqData != nil && reqData.Page != nil {
		page = *reqData.Page
	}
	if reqData != nil && reqData.Limit != nil {
		limit = *reqData.Limit
	}
	offset := (page - 1) * limit

	// Build query with filters
	db := database.Database.Db.Model(&courseModels.CourseContent{}).Where("course_id = ? AND is_deleted = ? AND is_published = ?", courseID, false, true)

	// Apply optional filters
	if moduleIDStr != "" {
		if moduleID, err := strconv.Atoi(moduleIDStr); err == nil && moduleID > 0 {
			db = db.Where("module_id = ?", moduleID)
		}
	}
	if dayStr != "" {
		if day, err := strconv.Atoi(dayStr); err == nil && day > 0 {
			db = db.Where("day = ?", day)
		}
	}
	if contentType != "" {
		db = db.Where("content_type = ?", contentType)
	}

	// Get total count
	var total int64
	db.Count(&total)

	// Fetch paginated data
	var contents []courseModels.CourseContent
	if err := db.Offset(offset).Limit(limit).Order("module_id asc, day asc, order_index asc").Find(&contents).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch course content!", nil)
	}

	// Enrich contents with MCQ options and completion status
	result := make([]ContentWithMCQ, len(contents))
	for i, content := range contents {
		result[i] = ContentWithMCQ{
			CourseContent: content,
		}

		// Check if completed by user
		var completion courseModels.ContentCompletion
		if err := database.Database.Db.Where("user_id = ? AND course_content_id = ? AND is_deleted = ?", userId, content.ID, false).First(&completion).Error; err == nil {
			result[i].IsCompleted = true
		}

		// Get MCQ options if content is MCQ type
		if content.ContentType == "MCQ" {
			var options []courseModels.MCQOption
			database.Database.Db.Where("content_id = ? AND is_deleted = ?", content.ID, false).Order("order_index asc").Find(&options)
			// Remove IsCorrect from options for users (don't show answers)
			for j := range options {
				options[j].IsCorrect = false
			}
			result[i].MCQOptions = options
		}
	}

	// Prepare response
	response := map[string]interface{}{
		"contents": result,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Course content fetched successfully!", response)
}

func CreateCourseContent(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	// Retrieve validated course ID
	courseID, err := strconv.Atoi(c.Locals("courseID").(string))
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid course ID!", nil)
	}

	// Check if course exists
	var course courseModels.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", courseID, false).First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found!", nil)
	}

	// Retrieve validated content data
	reqData, ok := c.Locals("validatedCourseContent").(*struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Create new course content
	content := courseModels.CourseContent{
		CourseID:    uint(courseID),
		Title:       reqData.Title,
		Description: reqData.Description,
	}

	// Save to database
	if err := database.Database.Db.Create(&content).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create course content!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Course content created successfully!", content)
}

func MarkContentComplete(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	// Retrieve validated IDs
	courseID := c.Locals("courseID").(int)
	contentID := c.Locals("contentID").(int)

	// Check if course exists and is active
	var course courseModels.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ? AND is_published = ?", courseID, false, true).First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found or not published!", nil)
	}

	// Check if course content exists
	var content courseModels.CourseContent
	if err := database.Database.Db.Where("id = ? AND course_id = ? AND is_deleted = ? AND is_published = ?", contentID, courseID, false, true).First(&content).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course content not found!", nil)
	}

	// Check if user is enrolled in the course
	var enrollment courseModels.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&enrollment).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "User not enrolled in this course!", nil)
	}

	// Check if content is already marked as completed
	var existingCompletion courseModels.ContentCompletion
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND course_content_id = ? AND is_deleted = ?", userID, courseID, contentID, false).First(&existingCompletion).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Content already marked as completed!", nil)
	}

	// Create completion record
	completion := courseModels.ContentCompletion{
		UserID:          userID,
		CourseID:        uint(courseID),
		CourseContentID: uint(contentID),
		Status:          "COMPLETED",
	}

	// Save to database with transaction
	tx := database.Database.Db.Begin()
	if err := tx.Create(&completion).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to mark content as completed!", nil)
	}
	tx.Commit()

	// Update enrollment progress
	updateEnrollmentProgress(userID, uint(courseID))

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Content marked as completed successfully!", completion)
}

func GetContentCompletions(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	// Retrieve validated course ID
	courseID := c.Locals("courseID").(int)

	// Check if course exists and is published
	var course courseModels.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ? AND is_published = ?", courseID, false, true).First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found or not published!", nil)
	}

	// Check if user is enrolled
	var enrollment courseModels.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&enrollment).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "User not enrolled in this course!", nil)
	}

	// Retrieve validated pagination request
	reqData, ok := c.Locals("validatedCompletionList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		// Fetch all completions without pagination
		var completions []courseModels.ContentCompletion
		if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).Find(&completions).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch completions!", nil)
		}

		response := map[string]interface{}{
			"completions": completions,
			"pagination": map[string]interface{}{
				"total": int64(len(completions)),
				"page":  1,
				"limit": len(completions),
			},
		}
		return middleware.JsonResponse(c, fiber.StatusOK, true, "Completions fetched successfully!", response)
	}

	// Set default pagination
	page := *reqData.Page
	limit := *reqData.Limit
	offset := (page - 1) * limit

	// Fetch completions with pagination
	var completions []courseModels.ContentCompletion
	db := database.Database.Db.Model(&courseModels.ContentCompletion{}).Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false)

	// Get total count
	var total int64
	db.Count(&total)

	// Fetch paginated data
	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&completions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch completions!", nil)
	}

	// Prepare response
	response := map[string]interface{}{
		"completions": completions,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Completions fetched successfully!", response)
}
