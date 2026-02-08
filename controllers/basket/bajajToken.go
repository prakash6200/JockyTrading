package basketController

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fib/models/basket"
	"fib/utils"
	"log"
	"github.com/gofiber/fiber/v2"
)

// SetBajajAccessToken sets/updates the Bajaj broker access token (Admin only)
func SetBajajAccessToken(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN", "AMC"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin or AMC role required.", nil)
	}

	reqData, ok := c.Locals("validatedSetToken").(*struct {
		AccessToken string `json:"accessToken"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Create new token entry (keeping history)
	newToken := models.BajajAccessToken{
		Token: reqData.AccessToken,
	}

	if err := db.Create(&newToken).Error; err != nil {
		log.Printf("Error creating Bajaj token: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save access token!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Bajaj access token saved successfully!", fiber.Map{
		"id":        newToken.ID,
		"createdAt": newToken.CreatedAt,
	})
}

// GetLatestBajajToken retrieves the most recent Bajaj access token
func GetLatestBajajToken(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN", "AMC"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	db := database.Database.Db

	var latestToken models.BajajAccessToken
	if err := db.Where("is_deleted = false").Order("created_at DESC").First(&latestToken).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "No access token found!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Latest token retrieved!", fiber.Map{
		"id":        latestToken.ID,
		"createdAt": latestToken.CreatedAt,
		"hasToken":  latestToken.Token != "",
	})
}

// GetStockPrice fetches current stock price using Bajaj API
func GetStockPrice(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedGetPrice").(*struct {
		StockToken  *int    `json:"stockToken"`
		AccessToken *string `json:"accessToken"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Get access token
	accessToken := ""
	if reqData.AccessToken != nil && *reqData.AccessToken != "" {
		accessToken = *reqData.AccessToken
	} else {
		// Fetch from database
		var latestToken models.BajajAccessToken
		if err := database.Database.Db.Where("is_deleted = false").Order("created_at DESC").First(&latestToken).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "No access token found! Please provide accessToken or set it via admin API.", nil)
		}
		accessToken = latestToken.Token
	}

	// Fetch price from Bajaj API
	price, err := utils.GetBajajQuote(accessToken, *reqData.StockToken)
	if err != nil {
		log.Printf("Error fetching Bajaj quote: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch stock price: "+err.Error(), nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock price fetched!", fiber.Map{
		"stockToken": *reqData.StockToken,
		"lastPrice":  price,
	})
}

// GetStockPriceDetails fetches detailed stock quote using Bajaj API
func GetStockPriceDetails(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedGetPrice").(*struct {
		StockToken  *int    `json:"stockToken"`
		AccessToken *string `json:"accessToken"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Get access token
	accessToken := ""
	if reqData.AccessToken != nil && *reqData.AccessToken != "" {
		accessToken = *reqData.AccessToken
	} else {
		var latestToken models.BajajAccessToken
		if err := database.Database.Db.Where("is_deleted = false").Order("created_at DESC").First(&latestToken).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "No access token found!", nil)
		}
		accessToken = latestToken.Token
	}

	// Fetch detailed quote from Bajaj API
	quoteDetails, err := utils.GetBajajQuoteDetails(accessToken, *reqData.StockToken)
	if err != nil {
		log.Printf("Error fetching Bajaj quote details: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch stock details: "+err.Error(), nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock details fetched!", fiber.Map{
		"stockToken": *reqData.StockToken,
		"quote":      quoteDetails.Data,
	})
}

// AddStocksWithPricing adds stocks to basket and captures initial price
func AddStocksWithPricing(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! AMC role required.", nil)
	}

	reqData, ok := c.Locals("validatedAddStocksWithToken").(*struct {
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
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Verify basket ownership
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND amc_id = ? AND is_deleted = false", reqData.BasketID, userId).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found or not yours!", nil)
	}

	// Verify stock exists
	var stock models.Stocks
	if err := db.Where("id = ? AND is_deleted = false", reqData.StockID).First(&stock).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Stock not found!", nil)
	}

	// Use token from stock database if not provided or if token mismatch
	stockToken := reqData.Token
	if stockToken == 0 && stock.Token > 0 {
		stockToken = stock.Token
	}

	// Use symbol from stock database if not provided
	symbol := reqData.Symbol
	if symbol == "" {
		symbol = stock.Symbol
	}

	// Get draft version or create new one if basket is already published
	var version basket.BasketVersion
	if err := db.Where("basket_id = ? AND status = ? AND is_deleted = false", reqData.BasketID, basket.StatusDraft).
		Order("version_number DESC").First(&version).Error; err != nil {
		// No draft version - check if we should create a new one
		var latestVersion basket.BasketVersion
		if err := db.Where("basket_id = ? AND is_deleted = false", reqData.BasketID).
			Order("version_number DESC").First(&latestVersion).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "No version found for this basket!", nil)
		}

		// If latest version is PUBLISHED, SCHEDULED, or EXPIRED, create a new draft version
		if latestVersion.Status == basket.StatusPublished || latestVersion.Status == basket.StatusScheduled || latestVersion.Status == basket.StatusExpired {
			version = basket.BasketVersion{
				BasketID:      reqData.BasketID,
				VersionNumber: latestVersion.VersionNumber + 1,
				Status:        basket.StatusDraft,
			}
			if err := db.Create(&version).Error; err != nil {
				return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create new draft version!", nil)
			}
		} else if latestVersion.Status == basket.StatusRejected {
			// REJECTED -> Revert to DRAFT so it can be edited
			latestVersion.Status = basket.StatusDraft
			if err := db.Save(&latestVersion).Error; err != nil {
				return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to revert basket to draft!", nil)
			}
			version = latestVersion
		} else {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Cannot add stock - basket is in "+string(latestVersion.Status)+" status!", nil)
		}
	}

	// Try to get access token for pricing (optional)
	accessToken := ""
	if reqData.AccessToken != nil && *reqData.AccessToken != "" {
		accessToken = *reqData.AccessToken
	} else {
		var latestToken models.BajajAccessToken
		if err := db.Where("is_deleted = false").Order("created_at DESC").First(&latestToken).Error; err == nil {
			accessToken = latestToken.Token
		}
	}

	// Fetch current price from Bajaj API (optional - don't fail if it doesn't work)
	var currentPrice float64 = 0
	var priceWarning string = ""
	if accessToken != "" && stockToken > 0 {
		price, err := utils.GetBajajQuote(accessToken, stockToken)
		if err != nil {
			log.Printf("Warning: Could not fetch price for token %d: %v", stockToken, err)
			priceWarning = "Price could not be fetched from Bajaj API. Stock added with price=0."
		} else {
			currentPrice = price
		}
	} else {
		priceWarning = "No access token or stock token available. Price set to 0."
	}

	orderType := reqData.OrderType
	if orderType == "" {
		orderType = "MARKET"
	}

	// Check if stock already exists in this version
	var existingStock basket.BasketStock
	if err := db.Where("basket_version_id = ? AND stock_id = ? AND is_deleted = false", version.ID, reqData.StockID).First(&existingStock).Error; err == nil {
		// Update existing stock entry - INCREMENT quantity instead of replacing
		existingStock.Weightage = reqData.HoldingPercentage
		existingStock.OrderType = orderType
		existingStock.TargetPrice = reqData.TgtPrice
		existingStock.StopLossPrice = reqData.SLPrice
		existingStock.Symbol = symbol
		existingStock.Token = stockToken
		// Increment quantity and units
		existingStock.Quantity += reqData.Units
		existingStock.Units += reqData.Units
		// Update price only if we got a valid one
		if currentPrice > 0 {
			existingStock.PriceAtCreation = currentPrice
		}
		db.Save(&existingStock)

		response := fiber.Map{
			"stock":        existingStock,
			"currentPrice": currentPrice,
		}
		if priceWarning != "" {
			response["warning"] = priceWarning
		}
		return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock updated in basket!", response)
	}

	// Add new stock with price at creation
	basketStock := basket.BasketStock{
		BasketVersionID: version.ID,
		StockID:         reqData.StockID,
		Quantity:        reqData.Units,
		Weightage:       reqData.HoldingPercentage,
		PriceAtCreation: currentPrice,
		OrderType:       orderType,
		TargetPrice:     reqData.TgtPrice,
		StopLossPrice:   reqData.SLPrice,
		Token:           stockToken,
		Symbol:          symbol,
		Units:           reqData.Units,
	}

	if err := db.Create(&basketStock).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to add stock!", nil)
	}

	// Send Email (Async)
	go func() {
		if user.Email != "" {
			utils.SendStockAddedEmail(user.Email, user.Name, existingBasket.Name, symbol, orderType)
		}
	}()

	response := fiber.Map{
		"stock":        basketStock,
		"currentPrice": currentPrice,
	}
	if priceWarning != "" {
		response["warning"] = priceWarning
	}
	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock added with pricing!", response)
}

// GetBasketWithPricing returns basket details with initial and current pricing
func GetBasketWithPricing(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	basketId := c.Params("id")
	accessTokenQuery := c.Query("accessToken")

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	db := database.Database.Db

	// Get basket with versions and stocks
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND is_deleted = false", basketId).
		Preload("Versions", "status IN ? AND is_deleted = false", []string{basket.StatusDraft, basket.StatusPublished, basket.StatusScheduled}).
		Preload("Versions.Stocks", "is_deleted = false").
		First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	// Get access token
	accessToken := accessTokenQuery
	if accessToken == "" {
		var latestToken models.BajajAccessToken
		if err := db.Where("is_deleted = false").Order("created_at DESC").First(&latestToken).Error; err == nil {
			accessToken = latestToken.Token
		}
	}

	// Calculate initial and current prices
	var basketInitialPrice float64 = 0
	var basketCurrentPrice float64 = 0

	// Get the current/latest version
	if len(existingBasket.Versions) > 0 {
		currentVersion := existingBasket.Versions[0]

		// Initial price from version (snapshot at approval)
		basketInitialPrice = currentVersion.PriceAtApproval

		for _, stock := range currentVersion.Stocks {
			// If PriceAtApproval is 0 (legacy data), fallback to calculated creation price
			if basketInitialPrice == 0 {
				basketInitialPrice += stock.PriceAtCreation * float64(stock.Quantity)
			}

			// Current price (live from API if token available)
			if accessToken != "" && stock.Token > 0 {
				livePrice, err := utils.GetBajajQuote(accessToken, stock.Token)
				if err == nil && livePrice > 0 {
					basketCurrentPrice += livePrice * float64(stock.Quantity)
				} else {
					// Fallback to price at creation if API fails
					// If PriceAtApproval is available, maybe use that? Better use PriceAtCreation as fallback for now
					// Or PriceAtApproval if available to match "current" state as close as possible
					if stock.PriceAtApproval > 0 {
						basketCurrentPrice += stock.PriceAtApproval * float64(stock.Quantity)
					} else {
						basketCurrentPrice += stock.PriceAtCreation * float64(stock.Quantity)
					}
				}
			} else {
				// No token, use stored price
				if stock.PriceAtApproval > 0 {
					basketCurrentPrice += stock.PriceAtApproval * float64(stock.Quantity)
				} else {
					basketCurrentPrice += stock.PriceAtCreation * float64(stock.Quantity)
				}
			}
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket with pricing fetched!", fiber.Map{
		"basket":             existingBasket,
		"basketInitialPrice": basketInitialPrice,
		"basketCurrentPrice": basketCurrentPrice,
		"priceChange":        basketCurrentPrice - basketInitialPrice,
		"priceChangePercent": func() float64 {
			if basketInitialPrice == 0 {
				return 0
			}
			return ((basketCurrentPrice - basketInitialPrice) / basketInitialPrice) * 100
		}(),
	})
}
