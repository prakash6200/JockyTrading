package controllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	courseModels "fib/models/course"

	"github.com/gofiber/fiber/v2"
)

// AdminCreateContent creates new content in a module
func AdminCreateContent(c *fiber.Ctx) error {
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
	moduleID := c.Locals("moduleID").(int)

	// Check if module exists
	var module courseModels.Module
	if err := database.Database.Db.Where("id = ? AND course_id = ? AND is_deleted = ?", moduleID, courseID, false).First(&module).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Module not found!", nil)
	}

	reqData, ok := c.Locals("validatedContent").(*struct {
		Day         int    `json:"day"`
		Title       string `json:"title"`
		Description string `json:"description"`
		ContentType string `json:"content_type"`
		TextContent string `json:"text_content"`
		VideoURL    string `json:"video_url"`
		ImageURL    string `json:"image_url"`
		OrderIndex  int    `json:"order_index"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Get the next order index if not provided
	orderIndex := reqData.OrderIndex
	if orderIndex == 0 {
		var maxOrder int
		database.Database.Db.Model(&courseModels.CourseContent{}).
			Where("module_id = ? AND day = ? AND is_deleted = ?", moduleID, reqData.Day, false).
			Select("COALESCE(MAX(order_index), 0)").Scan(&maxOrder)
		orderIndex = maxOrder + 1
	}

	content := courseModels.CourseContent{
		CourseID:    uint(courseID),
		ModuleID:    uint(moduleID),
		Day:         reqData.Day,
		Title:       reqData.Title,
		Description: reqData.Description,
		ContentType: reqData.ContentType,
		TextContent: reqData.TextContent,
		VideoURL:    reqData.VideoURL,
		ImageURL:    reqData.ImageURL,
		OrderIndex:  orderIndex,
		IsPublished: false,
	}

	if err := database.Database.Db.Create(&content).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create content!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusCreated, true, "Content created successfully!", content)
}

// AdminUpdateContent updates existing content
func AdminUpdateContent(c *fiber.Ctx) error {
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

	contentID := c.Locals("contentID").(int)

	var content courseModels.CourseContent
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", contentID, false).First(&content).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Content not found!", nil)
	}

	reqData, ok := c.Locals("validatedContentUpdate").(*struct {
		Day         int    `json:"day"`
		Title       string `json:"title"`
		Description string `json:"description"`
		ContentType string `json:"content_type"`
		TextContent string `json:"text_content"`
		VideoURL    string `json:"video_url"`
		ImageURL    string `json:"image_url"`
		OrderIndex  int    `json:"order_index"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	if reqData.Day > 0 {
		content.Day = reqData.Day
	}
	if reqData.Title != "" {
		content.Title = reqData.Title
	}
	if reqData.Description != "" {
		content.Description = reqData.Description
	}
	if reqData.ContentType != "" {
		content.ContentType = reqData.ContentType
	}
	if reqData.TextContent != "" {
		content.TextContent = reqData.TextContent
	}
	if reqData.VideoURL != "" {
		content.VideoURL = reqData.VideoURL
	}
	if reqData.ImageURL != "" {
		content.ImageURL = reqData.ImageURL
	}
	if reqData.OrderIndex > 0 {
		content.OrderIndex = reqData.OrderIndex
	}

	if err := database.Database.Db.Save(&content).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update content!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Content updated successfully!", content)
}

// AdminDeleteContent soft deletes content
func AdminDeleteContent(c *fiber.Ctx) error {
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

	contentID := c.Locals("contentID").(int)

	var content courseModels.CourseContent
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", contentID, false).First(&content).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Content not found!", nil)
	}

	tx := database.Database.Db.Begin()

	content.IsDeleted = true
	if err := tx.Save(&content).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to delete content!", nil)
	}

	// Delete MCQ options if content type is MCQ
	if content.ContentType == "MCQ" {
		if err := tx.Model(&courseModels.MCQOption{}).Where("content_id = ?", contentID).Update("is_deleted", true).Error; err != nil {
			tx.Rollback()
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to delete MCQ options!", nil)
		}
	}

	tx.Commit()

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Content deleted successfully!", nil)
}

// AdminPublishContent publishes or unpublishes content
func AdminPublishContent(c *fiber.Ctx) error {
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

	contentID := c.Locals("contentID").(int)
	publishStatus := c.Locals("publishStatus").(bool)

	var content courseModels.CourseContent
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", contentID, false).First(&content).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Content not found!", nil)
	}

	// If publishing MCQ, ensure it has options
	if publishStatus && content.ContentType == "MCQ" {
		var optionCount int64
		database.Database.Db.Model(&courseModels.MCQOption{}).Where("content_id = ? AND is_deleted = ?", contentID, false).Count(&optionCount)
		if optionCount < 2 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "MCQ must have at least 2 options before publishing!", nil)
		}

		// Check if at least one correct answer exists
		var correctCount int64
		database.Database.Db.Model(&courseModels.MCQOption{}).Where("content_id = ? AND is_correct = ? AND is_deleted = ?", contentID, true, false).Count(&correctCount)
		if correctCount == 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "MCQ must have at least one correct answer!", nil)
		}
	}

	content.IsPublished = publishStatus
	if err := database.Database.Db.Save(&content).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update content!", nil)
	}

	message := "Content unpublished successfully!"
	if publishStatus {
		message = "Content published successfully!"
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, message, content)
}

// AdminAddMCQOption adds an option to MCQ content
func AdminAddMCQOption(c *fiber.Ctx) error {
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

	contentID := c.Locals("contentID").(int)

	// Verify content exists and is MCQ type
	var content courseModels.CourseContent
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", contentID, false).First(&content).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Content not found!", nil)
	}

	if content.ContentType != "MCQ" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Content is not an MCQ type!", nil)
	}

	reqData, ok := c.Locals("validatedMCQOption").(*struct {
		OptionText string `json:"option_text"`
		IsCorrect  bool   `json:"is_correct"`
		OrderIndex int    `json:"order_index"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Get the next order index if not provided
	orderIndex := reqData.OrderIndex
	if orderIndex == 0 {
		var maxOrder int
		database.Database.Db.Model(&courseModels.MCQOption{}).
			Where("content_id = ? AND is_deleted = ?", contentID, false).
			Select("COALESCE(MAX(order_index), 0)").Scan(&maxOrder)
		orderIndex = maxOrder + 1
	}

	option := courseModels.MCQOption{
		ContentID:  uint(contentID),
		OptionText: reqData.OptionText,
		IsCorrect:  reqData.IsCorrect,
		OrderIndex: orderIndex,
	}

	if err := database.Database.Db.Create(&option).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to add MCQ option!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusCreated, true, "MCQ option added successfully!", option)
}

// AdminUpdateMCQOption updates an MCQ option
func AdminUpdateMCQOption(c *fiber.Ctx) error {
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

	optionID := c.Locals("optionID").(int)

	var option courseModels.MCQOption
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", optionID, false).First(&option).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "MCQ option not found!", nil)
	}

	reqData, ok := c.Locals("validatedMCQOptionUpdate").(*struct {
		OptionText string `json:"option_text"`
		IsCorrect  bool   `json:"is_correct"`
		OrderIndex int    `json:"order_index"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	if reqData.OptionText != "" {
		option.OptionText = reqData.OptionText
	}
	option.IsCorrect = reqData.IsCorrect
	if reqData.OrderIndex > 0 {
		option.OrderIndex = reqData.OrderIndex
	}

	if err := database.Database.Db.Save(&option).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update MCQ option!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "MCQ option updated successfully!", option)
}

// AdminDeleteMCQOption soft deletes an MCQ option
func AdminDeleteMCQOption(c *fiber.Ctx) error {
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

	optionID := c.Locals("optionID").(int)

	var option courseModels.MCQOption
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", optionID, false).First(&option).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "MCQ option not found!", nil)
	}

	option.IsDeleted = true
	if err := database.Database.Db.Save(&option).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to delete MCQ option!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "MCQ option deleted successfully!", nil)
}

// AdminContentWithMCQ represents admin content with MCQ options
type AdminContentWithMCQ struct {
	courseModels.CourseContent
	MCQOptions []courseModels.MCQOption `json:"mcq_options,omitempty"`
}

// AdminGetModuleContent gets all content for a module organized by day with MCQ options
func AdminGetModuleContent(c *fiber.Ctx) error {
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
	moduleID := c.Locals("moduleID").(int)

	var module courseModels.Module
	if err := database.Database.Db.Where("id = ? AND course_id = ? AND is_deleted = ?", moduleID, courseID, false).First(&module).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Module not found!", nil)
	}

	var contents []courseModels.CourseContent
	if err := database.Database.Db.Where("module_id = ? AND is_deleted = ?", moduleID, false).
		Order("day asc, order_index asc").Find(&contents).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch content!", nil)
	}

	// Enrich contents with MCQ options
	enrichedContents := make([]AdminContentWithMCQ, len(contents))
	for i, content := range contents {
		enrichedContents[i] = AdminContentWithMCQ{
			CourseContent: content,
		}

		// Get MCQ options if content is MCQ type
		if content.ContentType == "MCQ" {
			var options []courseModels.MCQOption
			database.Database.Db.Where("content_id = ? AND is_deleted = ?", content.ID, false).Order("order_index asc").Find(&options)
			enrichedContents[i].MCQOptions = options
		}
	}

	// Group by day
	contentByDay := make(map[int][]AdminContentWithMCQ)
	for _, content := range enrichedContents {
		contentByDay[content.Day] = append(contentByDay[content.Day], content)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Content fetched successfully!", fiber.Map{
		"module":         module,
		"content_by_day": contentByDay,
		"total_content":  len(contents),
	})
}
