package basketValidator

import (
	"fib/middleware"

	"github.com/gofiber/fiber/v2"
)

// Subscribe validates user subscription request
func Subscribe() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			BasketID uint   `json:"basketId"`
			Period   string `json:"period"` // MONTHLY or YEARLY (optional, default: MONTHLY)
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		if reqData.BasketID == 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Basket ID is required!", nil)
		}

		// Validate period - default to MONTHLY if not provided
		if reqData.Period == "" {
			reqData.Period = "MONTHLY"
		} else if reqData.Period != "MONTHLY" && reqData.Period != "YEARLY" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Period must be MONTHLY or YEARLY!", nil)
		}

		c.Locals("validatedSubscribe", reqData)
		return c.Next()
	}
}

// ListPublishedBaskets validates user list request
func ListPublishedBaskets() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page       *int    `json:"page"`
			Limit      *int    `json:"limit"`
			Search     *string `json:"search"`
			BasketType *string `json:"basketType"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request query!", nil)
		}

		errors := make(map[string]string)

		if reqData.Page == nil || *reqData.Page < 1 {
			errors["page"] = "Page must be greater than 0!"
		}
		if reqData.Limit == nil || *reqData.Limit < 1 {
			errors["limit"] = "Limit must be greater than 0!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedListPublished", reqData)
		return c.Next()
	}
}

// GetMySubscriptions validates user subscriptions list request
func GetMySubscriptions() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page   *int    `json:"page"`
			Limit  *int    `json:"limit"`
			Status *string `json:"status"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request query!", nil)
		}

		errors := make(map[string]string)

		if reqData.Page == nil {
			defaultPage := 1
			reqData.Page = &defaultPage
		} else if *reqData.Page < 1 {
			errors["page"] = "Page must be greater than 0!"
		}

		if reqData.Limit == nil {
			defaultLimit := 10
			reqData.Limit = &defaultLimit
		} else if *reqData.Limit < 1 {
			errors["limit"] = "Limit must be greater than 0!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedMySubscriptions", reqData)
		return c.Next()
	}
}
