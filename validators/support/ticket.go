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
			Title       string `json:"title"`
			Description string `json:"description"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Normalize and sanitize inputs
		reqData.Title = strings.TrimSpace(reqData.Title)
		reqData.Description = strings.TrimSpace(reqData.Description)

		// Validate Title
		if reqData.Title == "" {
			errors["title"] = "Title is required!"
		} else {
			if len(reqData.Title) < 3 {
				errors["title"] = "Title must be at least 3 characters long!"
			}
			if len(reqData.Title) > 100 {
				errors["title"] = "Title must not exceed 100 characters!"
			}
			// Check for invalid characters (e.g., HTML tags)
			if matched, _ := regexp.MatchString(`[<>{}]`, reqData.Title); matched {
				errors["title"] = "Title contains invalid characters (e.g., <, >, {, })!"
			}
		}

		// Validate Description
		if reqData.Description == "" {
			errors["description"] = "Description is required!"
		} else {
			if len(reqData.Description) < 10 {
				errors["description"] = "Description must be at least 10 characters long!"
			}
			if len(reqData.Description) > 1000 {
				errors["description"] = "Description must not exceed 1000 characters!"
			}
			// Check for invalid characters
			if matched, _ := regexp.MatchString(`[<>{}]`, reqData.Description); matched {
				errors["description"] = "Description contains invalid characters (e.g., <, >, {, })!"
			}
		}

		// Respond with validation errors if any exist
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
			Page  *int `json:"page"`
			Limit *int `json:"limit"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body!",
				"errors":  nil,
			})
		}

		errors := make(map[string]string)

		// Validate Page
		if reqData.Page == nil || *reqData.Page < 1 {
			errors["page"] = "Page must be greater than 0!"
		}

		// Validate Limit
		if reqData.Limit == nil || *reqData.Limit < 1 {
			errors["limit"] = "Limit must be greater than 0!"
		}

		// Respond with validation errors if any exist
		if len(errors) > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Validation failed!",
				"errors":  errors,
			})
		}

		c.Locals("validatedList", reqData)
		return c.Next()
	}
}
