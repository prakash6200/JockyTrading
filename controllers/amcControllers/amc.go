package amcController

import (
	"encoding/csv"
	"fib/config"
	"fib/database"
	"fib/middleware"
	"fib/models"
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
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
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
