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

func TicketList(c *fiber.Ctx) error {
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

	// Retrieve validated pagination request
	reqData, ok := c.Locals("validatedList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		// If no pagination validator is set, proceed without pagination
		var tickets []models.SupportTicket
		if err := database.Database.Db.Where("is_deleted = ? and user_id = ?", false, userId).Find(&tickets).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch support tickets!", nil)
		}
		return middleware.JsonResponse(c, fiber.StatusOK, true, "Tickets fetched successfully!", fiber.Map{
			"tickets": tickets,
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

	// Fetch tickets with pagination
	var tickets []models.SupportTicket
	db := database.Database.Db.Model(&models.SupportTicket{}).Where("is_deleted = ? AND user_id = ?", false, userId)

	// Get total count
	var total int64
	db.Count(&total)

	// Fetch paginated data
	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&tickets).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch tickets!", nil)
	}

	// Prepare response
	response := map[string]interface{}{
		"tickets": tickets,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Ticket fetched successfully!", response)
}
