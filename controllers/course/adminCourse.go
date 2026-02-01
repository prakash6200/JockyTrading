package controllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	courseModels "fib/models/course"

	"github.com/gofiber/fiber/v2"
)

// AdminCreateCourse creates a new course
func AdminCreateCourse(c *fiber.Ctx) error {
	// Check admin role
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	// Get validated request data
	reqData, ok := c.Locals("validatedCourse").(*struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		Author       string `json:"author"`
		Duration     int64  `json:"duration"`
		ThumbnailURL string `json:"thumbnail_url"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	course := courseModels.Course{
		Title:        reqData.Title,
		Description:  reqData.Description,
		Author:       reqData.Author,
		Duration:     reqData.Duration,
		ThumbnailURL: reqData.ThumbnailURL,
		Status:       "DRAFT",
		IsPublished:  false,
	}

	if err := database.Database.Db.Create(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create course!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusCreated, true, "Course created successfully!", course)
}

// AdminUpdateCourse updates an existing course
func AdminUpdateCourse(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	courseID := c.Locals("courseID").(int)

	var course courseModels.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", courseID, false).First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found!", nil)
	}

	reqData, ok := c.Locals("validatedCourseUpdate").(*struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		Author       string `json:"author"`
		Duration     int64  `json:"duration"`
		ThumbnailURL string `json:"thumbnail_url"`
		Status       string `json:"status"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Update only provided fields
	if reqData.Title != "" {
		course.Title = reqData.Title
	}
	if reqData.Description != "" {
		course.Description = reqData.Description
	}
	if reqData.Author != "" {
		course.Author = reqData.Author
	}
	if reqData.Duration > 0 {
		course.Duration = reqData.Duration
	}
	if reqData.ThumbnailURL != "" {
		course.ThumbnailURL = reqData.ThumbnailURL
	}
	if reqData.Status != "" {
		course.Status = reqData.Status
	}

	if err := database.Database.Db.Save(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update course!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Course updated successfully!", course)
}

// AdminDeleteCourse soft deletes a course
func AdminDeleteCourse(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	courseID := c.Locals("courseID").(int)

	var course courseModels.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", courseID, false).First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found!", nil)
	}

	course.IsDeleted = true
	if err := database.Database.Db.Save(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to delete course!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Course deleted successfully!", nil)
}

// AdminGetAllCourses lists all courses for admin
func AdminGetAllCourses(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	reqData, ok := c.Locals("validatedAdminList").(*struct {
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

	var courses []courseModels.Course
	var total int64

	db := database.Database.Db.Model(&courseModels.Course{}).Where("is_deleted = ?", false)
	db.Count(&total)

	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&courses).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch courses!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Courses fetched successfully!", fiber.Map{
		"courses": courses,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// AdminGetCourseDetails gets a single course with modules
func AdminGetCourseDetails(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	courseID := c.Locals("courseID").(int)

	var course courseModels.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", courseID, false).First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found!", nil)
	}

	// Get modules
	var modules []courseModels.Module
	database.Database.Db.Where("course_id = ? AND is_deleted = ?", courseID, false).Order("order_index asc").Find(&modules)

	// Get content count
	var contentCount int64
	database.Database.Db.Model(&courseModels.CourseContent{}).Where("course_id = ? AND is_deleted = ?", courseID, false).Count(&contentCount)

	// Get enrollment count
	var enrollmentCount int64
	database.Database.Db.Model(&courseModels.Enrollment{}).Where("course_id = ? AND is_deleted = ?", courseID, false).Count(&enrollmentCount)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Course details fetched successfully!", fiber.Map{
		"course":           course,
		"modules":          modules,
		"content_count":    contentCount,
		"enrollment_count": enrollmentCount,
	})
}

// AdminPublishCourse publishes or unpublishes a course
func AdminPublishCourse(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	courseID := c.Locals("courseID").(int)
	publishStatus := c.Locals("publishStatus").(bool)

	var course courseModels.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", courseID, false).First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found!", nil)
	}

	course.IsPublished = publishStatus
	if publishStatus {
		course.Status = "ACTIVE"
	}

	if err := database.Database.Db.Save(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update course!", nil)
	}

	message := "Course unpublished successfully!"
	if publishStatus {
		message = "Course published successfully!"
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, message, course)
}
