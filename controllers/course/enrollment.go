package controllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"

	"github.com/gofiber/fiber/v2"
)

func EnrollInCourse(c *fiber.Ctx) error {
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

	// Check if user is already enrolled
	var existingEnrollment models.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&existingEnrollment).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "User already enrolled in this course!", nil)
	}

	// Create enrollment
	enrollment := models.Enrollment{
		UserID:   userID,
		CourseID: uint(courseID),
		Status:   "ENROLLED",
	}

	// Save to database with transaction
	tx := database.Database.Db.Begin()
	if err := tx.Create(&enrollment).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to enroll in course!", nil)
	}
	tx.Commit()

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Enrolled in course successfully!", enrollment)
}

func GetEnrollments(c *fiber.Ctx) error {
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

	// Retrieve validated pagination request
	reqData, ok := c.Locals("validatedEnrollmentList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		// Fetch all enrollments without pagination
		var enrollments []models.Enrollment
		if err := database.Database.Db.Where("user_id = ? AND is_deleted = ?", userID, false).Preload("Course").Find(&enrollments).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch enrollments!", nil)
		}
		response := map[string]interface{}{
			"enrollments": enrollments,
			"pagination": map[string]interface{}{
				"total": int64(len(enrollments)),
				"page":  1,
				"limit": len(enrollments),
			},
		}
		return middleware.JsonResponse(c, fiber.StatusOK, true, "Enrollments fetched successfully!", response)
	}

	// Set default pagination
	page := *reqData.Page
	limit := *reqData.Limit
	offset := (page - 1) * limit

	// Fetch enrollments with pagination
	var enrollments []models.Enrollment
	db := database.Database.Db.Model(&models.Enrollment{}).Where("user_id = ? AND is_deleted = ?", userID, false).Preload("Course")

	// Get total count
	var total int64
	db.Count(&total)

	// Fetch paginated data
	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&enrollments).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch enrollments!", nil)
	}

	// Prepare response
	response := map[string]interface{}{
		"enrollments": enrollments,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Enrollments fetched successfully!", response)
}
