package courseValidator

import (
	"fib/middleware"
	"github.com/gofiber/fiber/v2"
	"strings"
)

func CreateCourse() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Validate Title
		if strings.TrimSpace(reqData.Title) == "" {
			errors["title"] = "Title is required!"
		} else if len(strings.TrimSpace(reqData.Title)) < 3 {
			errors["title"] = "Title must be at least 3 characters long!"
		}

		// Validate Description
		if strings.TrimSpace(reqData.Description) == "" {
			errors["description"] = "Description is required!"
		} else if len(strings.TrimSpace(reqData.Description)) < 5 {
			errors["description"] = "Description must be at least 5 characters long!"
		}

		// Respond with validation errors if any exist
		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedCourse", reqData)
		return c.Next()
	}
}

func CourseList() fiber.Handler {
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
