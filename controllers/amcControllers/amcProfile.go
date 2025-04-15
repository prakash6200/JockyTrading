package amcController

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
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

	// Parse request body
	var body models.AMCProfile
	if err := c.BodyParser(&body); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data", nil)
	}

	// Set user ID
	body.UserID = user.ID

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

	return middleware.JsonResponse(c, fiber.StatusOK, true, "AMC profile retrieved successfully", profile)
}
