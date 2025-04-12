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
