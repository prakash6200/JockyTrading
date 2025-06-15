package supportControllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"

	"github.com/gofiber/fiber/v2"
)

func CreateSupportTicket(c *fiber.Ctx) error {
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

	// Retrieve validated ticket data
	reqData, ok := c.Locals("validatedSupportTicket").(*struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Create new support ticket
	ticket := models.SupportTicket{
		UserID:      userId,
		Title:       reqData.Title,
		Description: reqData.Description,
		Status:      "OPEN",
	}

	// Save to database
	if err := database.Database.Db.Create(&ticket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create support ticket!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Support ticket created successfully!", ticket)
}
