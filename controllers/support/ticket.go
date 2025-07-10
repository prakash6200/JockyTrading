package supportControllers

import (
	"encoding/json"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func CreateSupportTicket(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	// Get validated data
	reqData, ok := c.Locals("validatedSupportTicket").(*struct {
		Title    string  `json:"title"`
		Subject  *string `json:"subject"`
		Message  string  `json:"message"`
		Priority *string `json:"priority"`
		Category *string `json:"category"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Build structured JSON message
	msgStruct := map[string]interface{}{
		"sender": "user",
		"text":   reqData.Message,
		"time":   time.Now().UTC().Format(time.RFC3339),
	}
	msgJSON, err := json.Marshal(msgStruct)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to format message!", nil)
	}

	// Prepare ticket model
	ticket := models.SupportTicket{
		UserID:   userId,
		Title:    reqData.Title,
		Message:  msgJSON,
		Status:   "OPEN",
		Priority: "MEDIUM",
		Category: "GENERAL",
	}

	if reqData.Subject != nil {
		ticket.Subject = *reqData.Subject
	}
	if reqData.Priority != nil {
		ticket.Priority = *reqData.Priority
	}
	if reqData.Category != nil {
		ticket.Category = *reqData.Category
	}

	// Save ticket
	if err := database.Database.Db.Create(&ticket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create support ticket!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Support ticket created successfully!", ticket)
}

func TicketList(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedList").(*struct {
		Page     *int    `query:"page"`
		Limit    *int    `query:"limit"`
		Status   *string `query:"status"`
		Priority *string `query:"priority"`
		Category *string `query:"category"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request!", nil)
	}

	// Pagination setup
	page := 1
	limit := 10
	if reqData.Page != nil {
		page = *reqData.Page
	}
	if reqData.Limit != nil {
		limit = *reqData.Limit
	}
	offset := (page - 1) * limit

	// Build query with filters
	db := database.Database.Db.Model(&models.SupportTicket{}).Where("user_id = ? AND is_deleted = false", userId)

	// Count total results
	var total int64
	db.Count(&total)

	var tickets []models.SupportTicket
	if err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&tickets).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch tickets!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Tickets fetched successfully!", fiber.Map{
		"tickets": tickets,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

func AdminTicketList(c *fiber.Ctx) error {
	// Check if user is admin
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "ADMIN").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access denied!", nil)
	}

	// Get validated query data
	reqData, ok := c.Locals("validatedAdminList").(*struct {
		Page     *int    `query:"page"`
		Limit    *int    `query:"limit"`
		Status   *string `query:"status"`
		Priority *string `query:"priority"`
		Category *string `query:"category"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Defaults
	page := 1
	limit := 10
	if reqData.Page != nil {
		page = *reqData.Page
	}
	if reqData.Limit != nil {
		limit = *reqData.Limit
	}
	offset := (page - 1) * limit

	// Base query
	db := database.Database.Db.Model(&models.SupportTicket{}).Where("is_deleted = false")

	// Apply filters
	if reqData.Status != nil {
		db = db.Where("UPPER(status) = ?", strings.ToUpper(*reqData.Status))
	}
	if reqData.Priority != nil {
		db = db.Where("UPPER(priority) = ?", strings.ToUpper(*reqData.Priority))
	}
	if reqData.Category != nil {
		db = db.Where("UPPER(category) = ?", strings.ToUpper(*reqData.Category))
	}

	// Count total
	var total int64
	db.Count(&total)

	// Fetch paginated tickets
	var tickets []models.SupportTicket
	if err := db.Offset(offset).Limit(limit).Order("created_at DESC").Find(&tickets).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch tickets!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Tickets fetched successfully!", fiber.Map{
		"tickets": tickets,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

func AdminReplyTicket(c *fiber.Ctx) error {
	adminID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if admin is valid
	var admin models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", adminID, "ADMIN").First(&admin).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access denied!", nil)
	}

	// Extract validated payload
	reqData := c.Locals("validatedAdminReply").(*struct {
		TicketID uint   `json:"ticketId"`
		Message  string `json:"message"`
	})

	// Fetch the ticket
	var ticket models.SupportTicket
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", reqData.TicketID).First(&ticket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Support ticket not found!", nil)
	}

	// Load and decode existing message thread
	var messages []map[string]interface{}
	if len(ticket.Message) > 0 {
		if err := json.Unmarshal(ticket.Message, &messages); err != nil {
			messages = []map[string]interface{}{} // fallback to empty
		}
	}

	// Append admin reply to the thread
	adminMsg := map[string]interface{}{
		"sender": "admin",
		"text":   reqData.Message,
		"time":   time.Now().UTC().Format(time.RFC3339),
	}
	messages = append(messages, adminMsg)

	// Encode updated messages
	updatedJSON, err := json.Marshal(messages)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update message thread!", nil)
	}

	// Update ticket message and optionally status
	ticket.Message = updatedJSON

	if ticket.Status == "OPEN" {
		ticket.Status = "PENDING"
	}

	if err := database.Database.Db.Save(&ticket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save reply!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Reply added successfully!", ticket)
}

func UserReplyTicket(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Validate user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userID).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Extract validated request
	reqData := c.Locals("validatedAdminReply").(*struct {
		TicketID uint   `json:"ticketId"`
		Message  string `json:"message"`
	})

	// Fetch ticket
	var ticket models.SupportTicket
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND user_id = ?", reqData.TicketID, userID).First(&ticket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Ticket not found or access denied!", nil)
	}

	// Parse existing messages
	var messages []map[string]interface{}
	if len(ticket.Message) > 0 {
		_ = json.Unmarshal(ticket.Message, &messages)
	}

	// Append user reply
	reply := map[string]interface{}{
		"sender": "user",
		"text":   reqData.Message,
		"time":   time.Now().UTC().Format(time.RFC3339),
	}
	messages = append(messages, reply)

	// ✅ Assign updated message JSON back to ticket.Message
	updatedJSON, err := json.Marshal(messages)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to encode reply!", nil)
	}
	ticket.Message = updatedJSON

	// Save changes
	if err := database.Database.Db.Save(&ticket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save user reply!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Reply added successfully!", ticket)
}

func closeTicket(c *fiber.Ctx, userId uint, isAdmin bool) error {
	reqData := new(struct {
		TicketID uint `json:"ticketId"`
	})
	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}
	if reqData.TicketID == 0 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Ticket ID is required!", nil)
	}

	// Fetch ticket with conditions
	query := database.Database.Db.Model(&models.SupportTicket{}).Where("id = ? AND is_deleted = false", reqData.TicketID)
	if !isAdmin {
		query = query.Where("user_id = ?", userId)
	}

	var ticket models.SupportTicket
	if err := query.First(&ticket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Ticket not found or access denied!", nil)
	}

	// Check if already closed
	if ticket.Status == "CLOSED" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Ticket is already closed!", nil)
	}

	// Update status to closed
	ticket.Status = "CLOSED"
	if err := database.Database.Db.Save(&ticket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to close ticket!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Ticket closed successfully.", ticket)
}

func AdminCloseTicket(c *fiber.Ctx) error {
	adminId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Verify admin
	var admin models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", adminId, "ADMIN").First(&admin).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied!", nil)
	}

	return closeTicket(c, adminId, true)
}

func UserCloseTicket(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	return closeTicket(c, userId, false)
}
