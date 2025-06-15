package courseValidator

import (
	"fib/middleware"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func CreateCourse() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Author      string `json:"author"`
			Duration    int64  `json:"duration"` // expected to be a valid ISO 8601 date-time string
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Trim all fields
		reqData.Title = strings.TrimSpace(reqData.Title)
		reqData.Description = strings.TrimSpace(reqData.Description)
		reqData.Author = strings.TrimSpace(reqData.Author)

		// Title Validation
		if reqData.Title == "" {
			errors["title"] = "Title is required!"
		} else if len(reqData.Title) < 3 {
			errors["title"] = "Title must be at least 3 characters long!"
		}

		// Description Validation
		if reqData.Description == "" {
			errors["description"] = "Description is required!"
		} else if len(reqData.Description) < 5 {
			errors["description"] = "Description must be at least 5 characters long!"
		}

		// Author Validation
		if reqData.Author == "" {
			errors["author"] = "Author is required!"
		} else if len(reqData.Author) < 3 {
			errors["author"] = "Author must be at least 3 characters long!"
		} else if matched, _ := regexp.MatchString(`[<>{}]`, reqData.Author); matched {
			errors["author"] = "Author name contains invalid characters!"
		}

		// Duration Validation (must be future date)
		if reqData.Duration <= 0 {
			errors["duration"] = "Duration must be a positive number (in hours)!"
		}

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
