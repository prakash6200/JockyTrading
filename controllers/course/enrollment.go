package controllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	courseModels "fib/models/course"
	"fib/utils"

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

	// Check if course exists and is published
	var course courseModels.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ? AND is_published = ?", courseID, false, true).First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found or not available!", nil)
	}

	// Check if user is already enrolled
	var existingEnrollment courseModels.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&existingEnrollment).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "User already enrolled in this course!", nil)
	}

	// Get total published content count
	var totalContents int64
	database.Database.Db.Model(&courseModels.CourseContent{}).Where("course_id = ? AND is_deleted = ? AND is_published = ?", courseID, false, true).Count(&totalContents)

	// Create enrollment
	enrollment := courseModels.Enrollment{
		UserID:        userID,
		CourseID:      uint(courseID),
		Status:        "ENROLLED",
		TotalContents: int(totalContents),
	}

	// Save to database with transaction
	tx := database.Database.Db.Begin()
	if err := tx.Create(&enrollment).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to enroll in course!", nil)
	}
	tx.Commit()

	// Send enrollment email asynchronously
	go utils.SendEnrollmentEmail(user.Email, user.Name, course.Title)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Enrolled in course successfully!", enrollment)
}

func GetEnrollments(c *fiber.Ctx) error {
	// Retrieve validated course ID
	courseID := c.Locals("courseID").(int)

	// Pagination
	reqData, ok := c.Locals("validatedEnrollmentList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})

	page := 1
	limit := 10
	if ok && reqData.Page != nil && *reqData.Page > 0 {
		page = *reqData.Page
	}
	if ok && reqData.Limit != nil && *reqData.Limit > 0 {
		limit = *reqData.Limit
	}
	offset := (page - 1) * limit

	var enrollments []courseModels.Enrollment
	var total int64

	db := database.Database.Db.Model(&courseModels.Enrollment{}).Where("course_id = ? AND is_deleted = ?", courseID, false)
	db.Count(&total)

	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&enrollments).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch enrollments!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Enrollments fetched successfully!", fiber.Map{
		"enrollments": enrollments,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}
