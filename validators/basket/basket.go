package basketValidator

import (
	"fib/middleware"
	"fib/models/basket"

	"github.com/gofiber/fiber/v2"
)

// CreateBasket validates basket creation request
func CreateBasket() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Name            string  `json:"name"`
			Description     string  `json:"description"`
			BasketType      string  `json:"basketType"`
			SubscriptionFee float64 `json:"subscriptionFee"`
			IsFeeBased      bool    `json:"isFeeBased"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.Name == "" {
			errors["name"] = "Basket name is required!"
		}

		validTypes := map[string]bool{
			basket.BasketTypeIntraHour: true,
			basket.BasketTypeIntraday:  true,
			basket.BasketTypeDelivery:  true,
		}
		if _, ok := validTypes[reqData.BasketType]; !ok {
			errors["basketType"] = "Basket type must be INTRA_HOUR, INTRADAY, or DELIVERY!"
		}

		if reqData.IsFeeBased && reqData.SubscriptionFee <= 0 {
			errors["subscriptionFee"] = "Fee-based baskets must have a subscription fee greater than 0!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedCreateBasket", reqData)
		return c.Next()
	}
}

// UpdateBasket validates basket update request
func UpdateBasket() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			BasketID        uint     `json:"basketId"`
			Name            *string  `json:"name"`
			Description     *string  `json:"description"`
			SubscriptionFee *float64 `json:"subscriptionFee"`
			IsFeeBased      *bool    `json:"isFeeBased"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.BasketID == 0 {
			errors["basketId"] = "Basket ID is required!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedUpdateBasket", reqData)
		return c.Next()
	}
}

// AddStocks validates adding stocks to basket
func AddStocks() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			BasketID      uint    `json:"basketId"`
			StockID       uint    `json:"stockId"`
			Quantity      int     `json:"quantity"`
			Weightage     float64 `json:"weightage"`
			OrderType     string  `json:"orderType"`
			TargetPrice   float64 `json:"targetPrice"`
			StopLossPrice float64 `json:"stopLossPrice"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.BasketID == 0 {
			errors["basketId"] = "Basket ID is required!"
		}
		if reqData.StockID == 0 {
			errors["stockId"] = "Stock ID is required!"
		}
		if reqData.Quantity <= 0 {
			errors["quantity"] = "Quantity must be greater than 0!"
		}
		if reqData.Weightage < 0 || reqData.Weightage > 100 {
			errors["weightage"] = "Weightage must be between 0 and 100!"
		}

		validOrderTypes := map[string]bool{"MARKET": true, "LIMIT": true}
		if reqData.OrderType != "" {
			if _, ok := validOrderTypes[reqData.OrderType]; !ok {
				errors["orderType"] = "Order type must be MARKET or LIMIT!"
			}
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedAddStocks", reqData)
		return c.Next()
	}
}

// RemoveStock validates removing stock from basket
func RemoveStock() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			BasketID uint `json:"basketId"`
			StockID  uint `json:"stockId"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.BasketID == 0 {
			errors["basketId"] = "Basket ID is required!"
		}
		if reqData.StockID == 0 {
			errors["stockId"] = "Stock ID is required!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedRemoveStock", reqData)
		return c.Next()
	}
}

// SubmitForApproval validates submission request
func SubmitForApproval() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			BasketID uint `json:"basketId"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		if reqData.BasketID == 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Basket ID is required!", nil)
		}

		c.Locals("validatedSubmitForApproval", reqData)
		return c.Next()
	}
}

// ListBaskets validates list request with pagination
func ListBaskets() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page       *int    `json:"page"`
			Limit      *int    `json:"limit"`
			Search     *string `json:"search"`
			BasketType *string `json:"basketType"`
			Status     *string `json:"status"`
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

		c.Locals("validatedListBaskets", reqData)
		return c.Next()
	}
}

// GetBasketByID validates get basket request
func GetBasketByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		basketId := c.Params("id")
		if basketId == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Basket ID is required!", nil)
		}

		c.Locals("validatedBasketID", basketId)
		return c.Next()
	}
}

// EditStock validates editing stock details
func EditStock() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			BasketID      uint     `json:"basketId"`
			StockID       uint     `json:"stockId"`
			Quantity      *int     `json:"quantity"`
			Weightage     *float64 `json:"holdingPercentage"`
			OrderType     *string  `json:"orderType"`
			TargetPrice   *float64 `json:"tgtPrice"`
			StopLossPrice *float64 `json:"slPrice"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.BasketID == 0 {
			errors["basketId"] = "Basket ID is required!"
		}
		if reqData.StockID == 0 {
			errors["stockId"] = "Stock ID is required!"
		}

		if reqData.Weightage != nil && (*reqData.Weightage < 0 || *reqData.Weightage > 100) {
			errors["holdingPercentage"] = "Holding percentage must be between 0 and 100!"
		}

		validOrderTypes := map[string]bool{"MARKET": true, "LIMIT": true}
		if reqData.OrderType != nil && *reqData.OrderType != "" {
			if _, ok := validOrderTypes[*reqData.OrderType]; !ok {
				errors["orderType"] = "Order type must be MARKET or LIMIT!"
			}
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedEditStock", reqData)
		return c.Next()
	}
}
