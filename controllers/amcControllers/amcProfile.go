package amcController

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fib/utils"
	"github.com/gofiber/fiber/v2"
)

func CreateOrUpdateAMCProfile(c *fiber.Ctx) error {
	// Get userID from context
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user ID", nil)
	}

	// Fetch the user from database
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "User not found", nil)
	}

	// Check if user has AMC role
	if user.Role != "AMC" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Unauthorized - AMC role required", nil)
	}

	// Parse form data (for file uploads)
	form, err := c.MultipartForm()
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid form data", nil)
	}

	// Parse JSON data from form
	var body models.AMCProfile
	if err := c.BodyParser(&body); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data", nil)
	}

	// Set user ID
	body.UserID = user.ID

	// Handle file uploads
	if form.File != nil {
		// Process Required Document
		if files, ok := form.File["requiredDocument"]; ok && len(files) > 0 {
			filePath, err := utils.SaveUploadedFile(files[0], "amc/documents")
			if err != nil {
				return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save required document", nil)
			}
			body.RequiredDocument = filePath
		}

		// Process Optional Document 1
		if files, ok := form.File["optionalDocument1"]; ok && len(files) > 0 {
			filePath, err := utils.SaveUploadedFile(files[0], "amc/documents")
			if err != nil {
				return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save optional document 1", nil)
			}
			body.OptionalDocument1 = filePath
		}

		// Process Optional Document 2
		if files, ok := form.File["optionalDocument2"]; ok && len(files) > 0 {
			filePath, err := utils.SaveUploadedFile(files[0], "amc/documents")
			if err != nil {
				return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save optional document 2", nil)
			}
			body.OptionalDocument2 = filePath
		}

		// Process AMC Logo
		if files, ok := form.File["amcLogo"]; ok && len(files) > 0 {
			filePath, err := utils.SaveUploadedFile(files[0], "amc/logos")
			if err != nil {
				return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save AMC logo", nil)
			}
			body.AmcLogo = filePath
		}

		// Process Manager Image
		if files, ok := form.File["managerImage"]; ok && len(files) > 0 {
			filePath, err := utils.SaveUploadedFile(files[0], "amc/images")
			if err != nil {
				return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save manager image", nil)
			}
			body.ManagerImage = filePath
		}
	}

	// Create or update AMC profile
	if err := database.Database.Db.
		Where(models.AMCProfile{UserID: user.ID}).
		Assign(body).
		FirstOrCreate(&body).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save AMC profile", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "AMC profile updated successfully", body)
}

func GetAMCProfile(c *fiber.Ctx) error {
	// Get userID from context
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user ID", nil)
	}

	// Fetch the user from database
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "User not found", nil)
	}

	// Check if user has AMC role
	if user.Role != "AMC" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Unauthorized - AMC role required", nil)
	}

	// Get AMC profile
	var profile models.AMCProfile
	if err := database.Database.Db.Where("user_id = ?", user.ID).First(&profile).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "AMC profile not found", nil)
	}

	// Convert file paths to URLs if needed
	profile.RequiredDocument = utils.GetFileURL(profile.RequiredDocument)
	profile.OptionalDocument1 = utils.GetFileURL(profile.OptionalDocument1)
	profile.OptionalDocument2 = utils.GetFileURL(profile.OptionalDocument2)
	profile.AmcLogo = utils.GetFileURL(profile.AmcLogo)
	profile.ManagerImage = utils.GetFileURL(profile.ManagerImage)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "AMC profile retrieved successfully", profile)
}
