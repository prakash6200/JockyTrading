package controllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

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

	// Retrieve validated pagination request
	reqData, ok := c.Locals("validatedCourseContentList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		// If no pagination validator is set, proceed without pagination
		var contents []models.CourseContent
		if err := database.Database.Db.Where("course_id = ? AND is_deleted = ?", courseID, false).Find(&contents).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch course content!", nil)
		}
		return middleware.JsonResponse(c, fiber.StatusOK, true, "Course content fetched successfully!", fiber.Map{
			"contents": contents,
		})
	}

	// Set default pagination
	page := 1
	limit := 10
	if reqData.Page != nil {
		page = *reqData.Page
	}
	if reqData.Limit != nil {
		limit = *reqData.Limit
	}
	offset := (page - 1) * limit

	// Fetch course content with pagination
	var contents []models.CourseContent
	db := database.Database.Db.Model(&models.CourseContent{}).Where("course_id = ? AND is_deleted = ?", courseID, false)

	// Get total count
	var total int64
	db.Count(&total)

	// Fetch paginated data
	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&contents).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch course content!", nil)
	}

	// Prepare response
	response := map[string]interface{}{
		"contents": contents,
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
	var course models.Course
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
	content := models.CourseContent{
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
	var course models.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ? AND status = ?", courseID, false, "ACTIVE").First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found or not active!", nil)
	}

	// Check if course content exists
	var content models.CourseContent
	if err := database.Database.Db.Where("id = ? AND course_id = ? AND is_deleted = ?", contentID, courseID, false).First(&content).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course content not found!", nil)
	}

	// Check if user is enrolled in the course
	var enrollment models.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&enrollment).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "User not enrolled in this course!", nil)
	}

	// Check if content is already marked as completed
	var existingCompletion models.ContentCompletion
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND course_content_id = ? AND is_deleted = ?", userID, courseID, contentID, false).First(&existingCompletion).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Content already marked as completed!", nil)
	}

	// Create completion record
	completion := models.ContentCompletion{
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

	// Check if course exists and is active
	var course models.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ? AND status = ?", courseID, false, "ACTIVE").First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found or not active!", nil)
	}

	// Check if user is enrolled
	var enrollment models.Enrollment
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
		var completions []models.ContentCompletion
		if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).Preload("CourseContent").Find(&completions).Error; err != nil {
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
	var completions []models.ContentCompletion
	db := database.Database.Db.Model(&models.ContentCompletion{}).Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).Preload("CourseContent")

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
