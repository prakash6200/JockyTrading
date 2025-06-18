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

func AmcPickUnpickStockValidator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			StockID uint   `json:"stockId"`
			Action  string `json:"action"`
		})

		// Parse JSON body
		if err := c.BodyParser(reqData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body!",
				"errors":  nil,
			})
		}

		errors := make(map[string]string)

		// Validate StockID
		if reqData.StockID == 0 {
			errors["stockId"] = "Stock ID must be a positive number!"
		}

		// Validate Action
		validActions := map[string]bool{"pick": true, "unpick": true}
		if _, ok := validActions[reqData.Action]; !ok {
			errors["action"] = "Action must be either 'pick' or 'unpick'!"
		}

		// Return errors if any
		if len(errors) > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Validation failed!",
				"errors":  errors,
			})
		}

		// Set the validated request in context
		c.Locals("validatedAmcPickUnpickStock", reqData)
		return c.Next()
	}
}

func AmcPerformance() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page  *int `json:"page"`
			Limit *int `json:"limit"`
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

func AMCList() fiber.Handler {
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

		c.Locals("validateUserList", reqData)
		return c.Next()
	}
}
