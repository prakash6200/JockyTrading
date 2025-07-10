package supportValidators

import (
	"fib/middleware"
	"github.com/gofiber/fiber/v2"
	"regexp"
	"strings"
)

func CreateSupportTicket() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Title    string  `json:"title"`
			Subject  *string `json:"subject"`
			Message  string  `json:"message"`
			Priority *string `json:"priority"`
			Category *string `json:"category"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Title validation
		reqData.Title = strings.TrimSpace(reqData.Title)
		if reqData.Title == "" {
			errors["title"] = "Title is required!"
		} else {
			if len(reqData.Title) < 3 {
				errors["title"] = "Title must be at least 3 characters long!"
			}
			if len(reqData.Title) > 100 {
				errors["title"] = "Title must not exceed 100 characters!"
			}
			if matched, _ := regexp.MatchString(`[<>{}]`, reqData.Title); matched {
				errors["title"] = "Title contains invalid characters (e.g., <, >, {, })!"
			}
		}

		// Subject validation (optional)
		if reqData.Subject != nil {
			*reqData.Subject = strings.TrimSpace(*reqData.Subject)
			if len(*reqData.Subject) > 200 {
				errors["subject"] = "Subject must not exceed 200 characters!"
			}
		}

		// Message (must be valid JSON)
		reqData.Message = strings.TrimSpace(reqData.Message)
		if reqData.Message == "" {
			errors["message"] = "message is required!"
		}

		validPriority := map[string]bool{"LOW": true, "MEDIUM": true, "HIGH": true}
		validCategory := map[string]bool{"GENERAL": true, "TECHNICAL": true, "BILLING": true}

		if reqData.Priority != nil && !validPriority[strings.ToUpper(*reqData.Priority)] {
			errors["priority"] = "Invalid priority! Allowed: LOW, MEDIUM, HIGH"
		}
		if reqData.Category != nil && !validCategory[strings.ToUpper(*reqData.Category)] {
			errors["category"] = "Invalid category! Allowed: GENERAL, TECHNICAL, BILLING"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedSupportTicket", reqData)
		return c.Next()
	}
}

func TicketList() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page     *int    `query:"page"`
			Limit    *int    `query:"limit"`
			Status   *string `query:"status"`
			Priority *string `query:"priority"`
			Category *string `query:"category"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid query parameters!", nil)
		}

		errors := make(map[string]string)

		// Basic pagination validation
		if reqData.Page != nil && *reqData.Page < 1 {
			errors["page"] = "Page must be greater than 0!"
		}
		if reqData.Limit != nil && *reqData.Limit < 1 {
			errors["limit"] = "Limit must be greater than 0!"
		}

		// Enum validations
		if reqData.Status != nil {
			valid := map[string]bool{"OPEN": true, "CLOSED": true, "PENDING": true}
			if !valid[strings.ToUpper(*reqData.Status)] {
				errors["status"] = "Invalid status! Must be one of: OPEN, CLOSED, PENDING."
			}
		}
		if reqData.Priority != nil {
			valid := map[string]bool{"LOW": true, "MEDIUM": true, "HIGH": true}
			if !valid[strings.ToUpper(*reqData.Priority)] {
				errors["priority"] = "Invalid priority! Must be one of: LOW, MEDIUM, HIGH."
			}
		}
		if reqData.Category != nil {
			valid := map[string]bool{"GENERAL": true, "TECHNICAL": true, "BILLING": true}
			if !valid[strings.ToUpper(*reqData.Category)] {
				errors["category"] = "Invalid category! Must be one of: GENERAL, TECHNICAL, BILLING."
			}
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedList", reqData)
		return c.Next()
	}
}

func AdminTicketList() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page     *int    `query:"page"`
			Limit    *int    `query:"limit"`
			Status   *string `query:"status"`
			Priority *string `query:"priority"`
			Category *string `query:"category"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid query parameters!",
				"errors":  nil,
			})
		}

		errors := make(map[string]string)

		if reqData.Page != nil && *reqData.Page < 1 {
			errors["page"] = "Page must be greater than 0!"
		}
		if reqData.Limit != nil && *reqData.Limit < 1 {
			errors["limit"] = "Limit must be greater than 0!"
		}

		// Validate optional enums
		validStatus := map[string]bool{"OPEN": true, "CLOSED": true, "PENDING": true}
		validPriority := map[string]bool{"LOW": true, "MEDIUM": true, "HIGH": true}
		validCategory := map[string]bool{"GENERAL": true, "TECHNICAL": true, "BILLING": true}

		if reqData.Status != nil && !validStatus[strings.ToUpper(*reqData.Status)] {
			errors["status"] = "Invalid status! Must be one of: OPEN, CLOSED, PENDING."
		}
		if reqData.Priority != nil && !validPriority[strings.ToUpper(*reqData.Priority)] {
			errors["priority"] = "Invalid priority! Must be one of: LOW, MEDIUM, HIGH."
		}
		if reqData.Category != nil && !validCategory[strings.ToUpper(*reqData.Category)] {
			errors["category"] = "Invalid category! Must be one of: GENERAL, TECHNICAL, BILLING."
		}

		if len(errors) > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Validation failed!",
				"errors":  errors,
			})
		}

		c.Locals("validatedAdminList", reqData)
		return c.Next()
	}
}

func AdminReplyTicket() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			TicketID uint   `json:"ticketId"`
			Message  string `json:"message"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.TicketID == 0 {
			errors["ticketId"] = "Ticket ID is required!"
		}

		reqData.Message = strings.TrimSpace(reqData.Message)
		if reqData.Message == "" {
			errors["message"] = "Reply message is required!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedAdminReply", reqData)
		return c.Next()
	}
}

func CloseTicket() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			TicketID uint `json:"ticketId"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.TicketID == 0 {
			errors["ticketId"] = "Ticket ID is required and must be greater than 0!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedCloseTicket", reqData)
		return c.Next()
	}
}
