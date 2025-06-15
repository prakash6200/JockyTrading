package controllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"

	"github.com/gofiber/fiber/v2"
)

func GetAllCourses(c *fiber.Ctx) error {
	// Assuming user authentication is in place, retrieve userId from JWT middleware
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists (assuming a User model exists)
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Retrieve validated pagination request (we'll add pagination similar to amcController)
	reqData, ok := c.Locals("validatedList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		// If no pagination validator is set, proceed without pagination
		var courses []models.Course
		if err := database.Database.Db.Where("is_deleted = ?", false).Find(&courses).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch courses!", nil)
		}
		return middleware.JsonResponse(c, fiber.StatusOK, true, "Courses fetched successfully!", fiber.Map{
			"courses": courses,
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

	// Fetch courses with pagination
	var courses []models.Course
	db := database.Database.Db.Model(&models.Course{}).Where("is_deleted = ?", false)

	// Get total count
	var total int64
	db.Count(&total)

	// Fetch paginated data
	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&courses).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch courses!", nil)
	}

	// Prepare response
	response := map[string]interface{}{
		"courses": courses,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Courses fetched successfully!", response)
}

func CreateCourse(c *fiber.Ctx) error {
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

	// Get validated request data
	reqData, ok := c.Locals("validatedCourse").(*struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Author      string `json:"author"`
		Duration    int64  `json:"duration"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Create new course
	course := models.Course{
		Title:       reqData.Title,
		Description: reqData.Description,
		Author:      reqData.Author,
		Duration:    reqData.Duration,
		Status:      "ACTIVE",
	}

	// Save to database
	if err := database.Database.Db.Create(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create course!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Course created successfully!", course)
}
