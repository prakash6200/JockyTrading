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
