package basketValidator

import (
	"fib/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetBajajAccessToken validates admin token setting request
func SetBajajAccessToken() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			AccessToken string `json:"accessToken"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		if reqData.AccessToken == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Access token is required!", nil)
		}

		c.Locals("validatedSetToken", reqData)
		return c.Next()
	}
}

// GetStockPrice validates stock price request
func GetStockPrice() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			StockToken  *int    `json:"stockToken"`
			AccessToken *string `json:"accessToken"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request query!", nil)
		}

		if reqData.StockToken == nil || *reqData.StockToken == 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Stock token is required!", nil)
		}

		c.Locals("validatedGetPrice", reqData)
		return c.Next()
	}
}

// AddStocksWithToken validates adding stocks with Bajaj token
func AddStocksWithToken() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			BasketID          uint    `json:"basketId"`
			StockID           uint    `json:"stockId"`
			HoldingPercentage float64 `json:"holdinPercentage"`
			SLPrice           float64 `json:"slPrice"`
			TgtPrice          float64 `json:"tgtPrice"`
			OrderType         string  `json:"orderType"`
			Units             int     `json:"units"`
			Symbol            string  `json:"symbol"`
			Token             int     `json:"token"`
			AccessToken       *string `json:"accessToken"`
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
		if reqData.Units <= 0 {
			errors["units"] = "Units must be greater than 0!"
		}
		if reqData.Token == 0 {
			errors["token"] = "Stock token is required for price fetching!"
		}
		if reqData.Symbol == "" {
			errors["symbol"] = "Stock symbol is required!"
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

		c.Locals("validatedAddStocksWithToken", reqData)
		return c.Next()
	}
}
