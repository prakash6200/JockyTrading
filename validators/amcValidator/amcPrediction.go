package amcValidator

import (
	"fib/middleware"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// AMCPredictionValidator validates prediction creation/update
func AMCPredictionValidator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			ID          uint   `json:"id"`
			Title       string `json:"title"`
			Prediction  int    `json:"prediction"`
			Description string `json:"description"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data", nil)
		}

		errors := make(map[string]string)

		// Validate Title
		if reqData.Title == "" {
			errors["title"] = "Title is required"
		} else if len(reqData.Title) > 100 {
			errors["title"] = "Title must be less than 100 characters"
		}

		// Validate Prediction
		if reqData.Prediction == 0 {
			errors["prediction"] = "Prediction value is required"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedPrediction", reqData)
		return c.Next()
	}
}

// AMCPredictionListValidator validates prediction list request
func AMCPredictionListValidator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, _ := strconv.Atoi(c.Query("page", "1"))
		limit, _ := strconv.Atoi(c.Query("limit", "10"))

		errors := make(map[string]string)

		if page < 1 {
			errors["page"] = "Page must be greater than 0"
		}

		if limit < 1 {
			errors["limit"] = "Limit must be greater than 0"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedPredictionList", struct {
			Page  int
			Limit int
		}{
			Page:  page,
			Limit: limit,
		})
		return c.Next()
	}
}

// AMCAchievedValueValidator validates achieved value update
func AMCAchievedValueValidator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			ID       uint `json:"id"`
			Achieved int  `json:"achieved"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data", nil)
		}

		errors := make(map[string]string)

		if reqData.ID == 0 {
			errors["id"] = "Prediction ID is required"
		}

		if reqData.Achieved == 0 {
			errors["achieved"] = "Achieved value is required"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedAchievedValue", reqData)
		return c.Next()
	}
}
