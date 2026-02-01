package controllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	courseModels "fib/models/course"

	"github.com/gofiber/fiber/v2"
)

// GetCourseDetails gets course details with modules for users
func GetCourseDetails(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	courseID := c.Locals("courseID").(int)

	var course courseModels.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ? AND is_published = ?", courseID, false, true).First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found!", nil)
	}

	// Get modules
	var modules []courseModels.Module
	database.Database.Db.Where("course_id = ? AND is_deleted = ?", courseID, false).Order("order_index asc").Find(&modules)

	// Check if user is enrolled
	var enrollment courseModels.Enrollment
	isEnrolled := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&enrollment).Error == nil

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Course details fetched successfully!", fiber.Map{
		"course":      course,
		"modules":     modules,
		"is_enrolled": isEnrolled,
		"enrollment":  enrollment,
	})
}

// GetDayContent gets content for a specific day in a module
func GetDayContent(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	courseID := c.Locals("courseID").(int)
	moduleID := c.Locals("moduleID").(int)
	day := c.Locals("day").(int)

	// Check enrollment
	var enrollment courseModels.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&enrollment).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Please enroll in this course first!", nil)
	}

	// Check module exists
	var module courseModels.Module
	if err := database.Database.Db.Where("id = ? AND course_id = ? AND is_deleted = ?", moduleID, courseID, false).First(&module).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Module not found!", nil)
	}

	// Get content for the day
	var contents []courseModels.CourseContent
	if err := database.Database.Db.Where("module_id = ? AND day = ? AND is_deleted = ? AND is_published = ?", moduleID, day, false, true).
		Order("order_index asc").Find(&contents).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch content!", nil)
	}

	// Get content with MCQ options for MCQ type
	type ContentWithOptions struct {
		courseModels.CourseContent
		MCQOptions  []courseModels.MCQOption `json:"mcq_options,omitempty"`
		IsCompleted bool                     `json:"is_completed"`
	}

	result := make([]ContentWithOptions, len(contents))
	for i, content := range contents {
		result[i] = ContentWithOptions{
			CourseContent: content,
		}

		// Check if completed
		var completion courseModels.ContentCompletion
		if err := database.Database.Db.Where("user_id = ? AND course_content_id = ? AND is_deleted = ?", userID, content.ID, false).First(&completion).Error; err == nil {
			result[i].IsCompleted = true
		}

		// Get MCQ options if content is MCQ type
		if content.ContentType == "MCQ" {
			var options []courseModels.MCQOption
			database.Database.Db.Where("content_id = ? AND is_deleted = ?", content.ID, false).Order("order_index asc").Find(&options)
			// Remove IsCorrect from options for users
			for j := range options {
				options[j].IsCorrect = false
			}
			result[i].MCQOptions = options
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Day content fetched successfully!", fiber.Map{
		"module":   module,
		"day":      day,
		"contents": result,
	})
}
