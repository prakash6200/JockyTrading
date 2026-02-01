package controllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	courseModels "fib/models/course"

	"github.com/gofiber/fiber/v2"
)

// AdminCreateModule creates a new module in a course
func AdminCreateModule(c *fiber.Ctx) error {
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

	// Check if course exists
	var course courseModels.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", courseID, false).First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found!", nil)
	}

	reqData, ok := c.Locals("validatedModule").(*struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		OrderIndex  int    `json:"order_index"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Get the next order index if not provided
	orderIndex := reqData.OrderIndex
	if orderIndex == 0 {
		var maxOrder int
		database.Database.Db.Model(&courseModels.Module{}).Where("course_id = ? AND is_deleted = ?", courseID, false).Select("COALESCE(MAX(order_index), 0)").Scan(&maxOrder)
		orderIndex = maxOrder + 1
	}

	module := courseModels.Module{
		CourseID:    uint(courseID),
		Title:       reqData.Title,
		Description: reqData.Description,
		OrderIndex:  orderIndex,
	}

	if err := database.Database.Db.Create(&module).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create module!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusCreated, true, "Module created successfully!", module)
}

// AdminUpdateModule updates an existing module
func AdminUpdateModule(c *fiber.Ctx) error {
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

	reqData, ok := c.Locals("validatedModuleUpdate").(*struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		OrderIndex  int    `json:"order_index"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	if reqData.Title != "" {
		module.Title = reqData.Title
	}
	if reqData.Description != "" {
		module.Description = reqData.Description
	}
	if reqData.OrderIndex > 0 {
		module.OrderIndex = reqData.OrderIndex
	}

	if err := database.Database.Db.Save(&module).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update module!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Module updated successfully!", module)
}

// AdminDeleteModule soft deletes a module
func AdminDeleteModule(c *fiber.Ctx) error {
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

	// Soft delete module and its contents
	tx := database.Database.Db.Begin()

	module.IsDeleted = true
	if err := tx.Save(&module).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to delete module!", nil)
	}

	// Soft delete all contents in this module
	if err := tx.Model(&courseModels.CourseContent{}).Where("module_id = ?", moduleID).Update("is_deleted", true).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to delete module contents!", nil)
	}

	tx.Commit()

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Module deleted successfully!", nil)
}

// AdminListModules lists all modules in a course
func AdminListModules(c *fiber.Ctx) error {
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

	// Check if course exists
	var course courseModels.Course
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", courseID, false).First(&course).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Course not found!", nil)
	}

	var modules []courseModels.Module
	if err := database.Database.Db.Where("course_id = ? AND is_deleted = ?", courseID, false).Order("order_index asc").Find(&modules).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch modules!", nil)
	}

	// Get content counts for each module
	type ModuleWithCount struct {
		courseModels.Module
		ContentCount int64 `json:"content_count"`
	}

	modulesWithCount := make([]ModuleWithCount, len(modules))
	for i, mod := range modules {
		var count int64
		database.Database.Db.Model(&courseModels.CourseContent{}).Where("module_id = ? AND is_deleted = ?", mod.ID, false).Count(&count)
		modulesWithCount[i] = ModuleWithCount{
			Module:       mod,
			ContentCount: count,
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Modules fetched successfully!", fiber.Map{
		"modules": modulesWithCount,
	})
}
