package amcController

import (
	"encoding/csv"
	"encoding/json"
	"fib/config"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fmt"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func SyncStockHandler(c *fiber.Ctx) error {
	go FetchAndStoreStocks()
	return c.JSON(fiber.Map{"message": "Stock sync started"})
}

// fetch stocks from market
func FetchAndStoreStocks() {
	url := "https://www.alphavantage.co/query?function=LISTING_STATUS&apikey=" + config.AppConfig.AlphaVantageApiKey

	res, err := http.Get(url)
	if err != nil {
		log.Println("Failed to fetch stock list:", err)
		return
	}
	defer res.Body.Close()

	reader := csv.NewReader(res.Body)
	records, err := reader.ReadAll()
	if err != nil {
		log.Println("Failed to parse stock CSV:", err)
		return
	}

	for _, row := range records[1:] {
		symbol := row[0]
		name := row[1]
		length := len(row)
		status := row[length-1]

		if status == "Active" {
			stock := models.Stocks{Symbol: symbol, Name: name}
			database.Database.Db.FirstOrCreate(&stock, models.Stocks{Symbol: symbol})
		}
	}

	log.Println("Stock list sync completed")
}

func StockList(c *fiber.Ctx) error {
	// Retrieve the userId from the JWT token (added by JWTMiddleware)
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Access Denied!", nil)
	}

	// Retrieve validated request data
	reqData, ok := c.Locals("validatedStockList").(*struct {
		Page   *int    `json:"page"`
		Limit  *int    `json:"limit"`
		Search *string `json:"search"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Default pagination
	page := 1
	limit := 10
	if reqData.Page != nil {
		page = *reqData.Page
	}
	if reqData.Limit != nil {
		limit = *reqData.Limit
	}
	offset := (page - 1) * limit

	// Prepare query
	db := database.Database.Db.Model(&models.Stocks{}).Where("is_deleted = ?", false)

	// Optional: Apply search filter
	if reqData.Search != nil && *reqData.Search != "" {
		search := "%" + *reqData.Search + "%"
		db = db.Where("symbol ILIKE ? OR name ILIKE ? OR sector ILIKE ?", search, search, search)
	}

	// Get total count
	var total int64
	db.Count(&total)

	// Fetch paginated data
	var stocks []models.Stocks
	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&stocks).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch stocks!", nil)
	}

	// Response
	response := map[string]interface{}{
		"stocks": stocks,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock list fetched successfully!", response)
}

func AmcPickUnpickStock(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Get validated request
	reqData, ok := c.Locals("validatedAmcPickUnpickStock").(*struct {
		StockID uint   `json:"stockId"`
		Action  string `json:"action"` // "pick" or "unpick"
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	switch reqData.Action {
	case "pick":
		// Limit to 10 active picks
		var count int64
		db.Model(&models.AmcStocks{}).Where("user_id = ? AND is_deleted = false", userId).Count(&count)
		if count >= 10 {
			return middleware.JsonResponse(c, fiber.StatusForbidden, false, "You can pick up to 10 stocks only", nil)
		}

		// Check if already picked
		var existing models.AmcStocks
		err := db.Where("user_id = ? AND stock_id = ? AND is_deleted = false", userId, reqData.StockID).
			First(&existing).Error
		if err == nil {
			return middleware.JsonResponse(c, fiber.StatusConflict, false, "Stock already picked", nil)
		}

		// Save new pick
		pick := models.AmcStocks{UserID: userId, StockId: reqData.StockID}
		if err := db.Create(&pick).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to pick stock", nil)
		}

		return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock picked successfully", nil)

	case "unpick":
		// Soft delete the picked stock
		if err := db.Model(&models.AmcStocks{}).
			Where("user_id = ? AND stock_id = ? AND is_deleted = false", userId, reqData.StockID).
			Update("is_deleted", true).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to unpick stock", nil)
		}

		return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock unpicked successfully", nil)

	default:
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid action. Use 'pick' or 'unpick'.", nil)
	}
}

func StockPickedByAMCList(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	// Validate AMC user
	var user models.User
	if err := database.Database.Db.
		Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").
		First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Get validated pagination request
	reqData, ok := c.Locals("validatedStockList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Set default pagination
	page := 1
	limit := 10
	if reqData.Page != nil {
		page = *reqData.Page
	}
	if reqData.Limit != nil {
		limit = *reqData.Limit
	}
	offset := (page - 1) * limit

	// Subquery to get picked stock IDs
	var pickedStockIDs []uint
	if err := database.Database.Db.
		Model(&models.AmcStocks{}).
		Where("user_id = ? AND is_deleted = false", userId).
		Pluck("stock_id", &pickedStockIDs).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch picked stock IDs", nil)
	}

	// Get total count of picked stocks
	var total int64
	db := database.Database.Db.Model(&models.Stocks{}).
		Where("id IN ? AND is_deleted = false", pickedStockIDs)
	db.Count(&total)

	// Fetch paginated picked stocks
	var stocks []models.Stocks
	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&stocks).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch picked stocks", nil)
	}

	// Prepare response
	response := map[string]interface{}{
		"stocks": stocks,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Picked stock list fetched successfully!", response)
}

func AmcPerformance(c *fiber.Ctx) error {
	// Retrieve userId from JWT token
	userId := c.Locals("userId").(uint)

	// Validate AMC user
	var user models.User
	if err := database.Database.Db.
		Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").
		First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Get validated pagination request
	reqData, ok := c.Locals("validatedStockList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Set default pagination
	page := 1
	limit := 10
	if reqData.Page != nil {
		page = *reqData.Page
	}
	if reqData.Limit != nil {
		limit = *reqData.Limit
	}
	offset := (page - 1) * limit

	// Fetch picked stock IDs
	var pickedStockIDs []uint
	if err := database.Database.Db.
		Model(&models.AmcStocks{}).
		Where("user_id = ? AND is_deleted = false", userId).
		Pluck("stock_id", &pickedStockIDs).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch picked stock IDs", nil)
	}

	// Get total count of picked stocks
	var total int64
	db := database.Database.Db.Model(&models.Stocks{}).
		Where("id IN ? AND is_deleted = false", pickedStockIDs)
	db.Count(&total)

	// Fetch paginated picked stocks
	var stocks []models.Stocks
	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&stocks).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch picked stocks", nil)
	}

	// Fetch performance data for each stock
	type Performance struct {
		StockID       uint    `json:"stockId"`
		Symbol        string  `json:"symbol"`
		Name          string  `json:"name"`
		ClosingPrice  float64 `json:"closingPrice"`
		PercentChange float64 `json:"percentChange"`
	}

	var performances []Performance
	for _, stock := range stocks {
		// Fetch daily time series from Alpha Vantage
		url := fmt.Sprintf(
			"https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&apikey=%s",
			stock.Symbol, config.AppConfig.AlphaVantageApiKey,
		)

		res, err := http.Get(url)
		if err != nil {
			log.Printf("Failed to fetch performance for %s: %v", stock.Symbol, err)
			continue
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			log.Printf("Non-200 status for %s: %d", stock.Symbol, res.StatusCode)
			continue
		}

		// Parse API response
		var data map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
			log.Printf("Failed to parse JSON for %s: %v", stock.Symbol, err)
			continue
		}

		// Check for API errors (e.g., rate limit)
		if _, ok := data["Error Message"]; ok {
			log.Printf("API error for %s: %v", stock.Symbol, data["Error Message"])
			continue
		}

		// Extract daily time series
		timeSeries, ok := data["Time Series (Daily)"].(map[string]interface{})
		if !ok {
			log.Printf("Invalid time series data for %s", stock.Symbol)
			continue
		}

		// Get the latest and previous day's data
		var latestDate, prevDate string
		for date := range timeSeries {
			if latestDate == "" || date > latestDate {
				prevDate = latestDate
				latestDate = date
			} else if prevDate == "" || date > prevDate {
				prevDate = date
			}
		}

		if latestDate == "" || prevDate == "" {
			log.Printf("Insufficient data for %s", stock.Symbol)
			continue
		}

		latestData, ok := timeSeries[latestDate].(map[string]interface{})
		if !ok {
			log.Printf("Invalid latest data for %s", stock.Symbol)
			continue
		}
		prevData, ok := timeSeries[prevDate].(map[string]interface{})
		if !ok {
			log.Printf("Invalid previous data for %s", stock.Symbol)
			continue
		}

		// Extract closing prices
		latestClose, ok := latestData["4. close"].(string)
		if !ok {
			log.Printf("Invalid closing price for %s", stock.Symbol)
			continue
		}
		previousClose, ok := prevData["4. close"].(string)
		if !ok {
			log.Printf("Invalid previous closing price for %s", stock.Symbol)
			continue
		}

		// Convert to float
		latestCloseVal, err := parseFloat(latestClose)
		if err != nil {
			log.Printf("Failed to parse latest close for %s: %v", stock.Symbol, err)
			continue
		}
		previousCloseVal, err := parseFloat(previousClose)
		if err != nil {
			log.Printf("Failed to parse previous close for %s: %v", stock.Symbol, err)
			continue
		}

		// Calculate percentage change
		percentChange := ((latestCloseVal - previousCloseVal) / previousCloseVal) * 100

		// Append performance data
		performances = append(performances, Performance{
			StockID:       stock.ID,
			Symbol:        stock.Symbol,
			Name:          stock.Name,
			ClosingPrice:  latestCloseVal,
			PercentChange: percentChange,
		})
	}

	// Prepare response
	response := map[string]interface{}{
		"performances": performances,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "AMC stock performance fetched successfully!", response)
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func AMCList(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.First(&user, userId).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	// Retrieve validated request data
	reqData, ok := c.Locals("validateUserList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	offset := (*reqData.Page - 1) * (*reqData.Limit)

	var users []models.User
	var total int64

	// Fetch user list excluding SUPER-ADMIN
	if err := database.Database.Db.
		Where("is_deleted = ? AND role NOT IN ?", false, []string{"SUPER-ADMIN", "USER"}).
		Offset(offset).
		Limit(*reqData.Limit).
		Find(&users).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch user list!", nil)
	}

	// Count total records
	database.Database.Db.
		Model(&models.User{}).
		Where("is_deleted = ? AND role NOT IN ?", false, []string{"SUPER-ADMIN", "USER"}).
		Count(&total)

	// Response structure
	response := map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  *reqData.Page,
			"limit": *reqData.Limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "AMC List.", response)
}
