package amcValidator

import (
	"github.com/gofiber/fiber/v2"
)

func StockList() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page   *int    `json:"page"`
			Limit  *int    `json:"limit"`
			Search *string `json:"search"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request query!",
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

		// Return validation errors
		if len(errors) > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Validation failed!",
				"errors":  errors,
			})
		}

		// ✅ Set correct key to match the controller
		c.Locals("validatedStockList", reqData)
		return c.Next()
	}
}
