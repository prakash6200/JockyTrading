package amcController

import (
	"encoding/json"
	"fib/config"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/jinzhu/now"
	"golang.org/x/crypto/bcrypt"
)

const (
	SYMBOLS_URL  = "https://api.truedata.in/getAllSymbols"
	USER_ID      = "tdwsp703"
	PASSWORD     = "imran@703"
	AUTH_URL     = "https://auth.truedata.in/token"
	BARS_URL     = "https://history.truedata.in/getbars"
	USERNAME     = "tdwsp703"
	MARKET_OPEN  = "09:15:00"
	MARKET_CLOSE = "15:30:00"
)

func FetchAndStoreStocks() {

	url := fmt.Sprintf("%s?segment=eq&user=%s&password=%s&csv=true", SYMBOLS_URL, USER_ID, PASSWORD)
	log.Printf("Fetching stock list from: %s", url)

	// Make the HTTP request to TrueData API
	res, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch stock list: %v", err)
		return
	}
	defer res.Body.Close()
	// Check response status
	log.Printf("API response status: %s", res.Status)
	if res.StatusCode != http.StatusOK {
		log.Printf("API request failed with status: %s", res.Status)
		return
	}

	// Read the CSV response
	csvData, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Failed to read API response: %v", err)
		return
	}
	log.Printf("Received CSV data size: %d bytes", len(csvData))

	// Parse CSV
	lines := strings.Split(string(csvData), "\n")
	log.Printf("Parsed %d lines from CSV (including header)", len(lines))
	if len(lines) < 2 {
		log.Println("Invalid response: No symbols found")
		return
	}

	// Counter for processed and inserted stocks
	processed := 0
	inserted := 0
	updated := 0

	// Iterate over records and store each stock
	for i, line := range lines[1:] { // Skip header
		line = strings.TrimSpace(line)
		if line == "" {
			log.Printf("Skipping empty line at index %d", i+1)
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 5 {
			log.Printf("Invalid CSV line at index %d: %s (too few fields: %d)", i+1, line, len(fields))
			continue
		}

		// Map fields to Stocks model
		stock := models.Stocks{
			Symbol:    strings.TrimSpace(fields[1]),
			Name:      strings.TrimSpace(fields[len(fields)-1]),
			Exchange:  strings.TrimSpace(fields[4]),
			Isin:      strings.TrimSpace(fields[3]),
			Series:    strings.TrimSpace(fields[2]),
			IsDeleted: false,
		}

		// Log stock details before insertion
		log.Printf("Processing stock %d: Symbol=%s, Name=%s, Exchange=%s, Isin=%s, Series=%s",
			i+1, stock.Symbol, stock.Name, stock.Exchange, stock.Isin, stock.Series)
		processed++

		// Validate required fields
		if stock.Symbol == "" || stock.Name == "" || stock.Exchange == "" || stock.Isin == "" || stock.Series == "" {
			log.Printf("Skipping stock %d: Missing required fields (Symbol=%s, Name=%s, Exchange=%s, Isin=%s, Series=%s)",
				i+1, stock.Symbol, stock.Name, stock.Exchange, stock.Isin, stock.Series)
			continue
		}

		// Upsert stock in database
		result := database.Database.Db.Where("symbol = ? OR isin = ?", stock.Symbol, stock.Isin).
			Assign(stock).
			FirstOrCreate(&stock)
		if result.Error != nil {
			log.Printf("Error syncing stock %s (Isin=%s): %v", stock.Symbol, stock.Isin, result.Error)
			continue
		}

		if result.RowsAffected == 1 {
			log.Printf("Inserted new stock: Symbol=%s, Isin=%s", stock.Symbol, stock.Isin)
			inserted++
		} else {
			// Save updated fields for existing stock
			result = database.Database.Db.Save(&stock)
			if result.Error != nil {
				log.Printf("Error updating stock %s (Isin=%s): %v", stock.Symbol, stock.Isin, result.Error)
				continue
			}
			log.Printf("Updated existing stock: Symbol=%s, Isin=%s", stock.Symbol, stock.Isin)
			updated++
		}
	}

	log.Printf("Stock list sync completed: Processed=%d, Inserted=%d, Updated=%d", processed, inserted, updated)
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
		StockID    uint    `json:"stockId"`
		Action     string  `json:"action"` // "pick" or "unpick"
		HoldingPer float32 `josn:"holdingPer"`
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
		pick := models.AmcStocks{UserID: userId, StockId: reqData.StockID, HoldingPer: reqData.HoldingPer}
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

	var user models.User
	if err := database.Database.Db.
		Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").
		First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Define struct for the response
	type StockWithHolding struct {
		models.Stocks
		HoldingPer float32 `json:"holdingPer"`
	}

	var stocks []StockWithHolding
	if err := database.Database.Db.
		Table("amc_stocks").
		Select("stocks.*, amc_stocks.holding_per").
		Joins("JOIN stocks ON stocks.id = amc_stocks.stock_id").
		Where("amc_stocks.user_id = ? AND amc_stocks.is_deleted = false AND stocks.is_deleted = false", userId).
		Order("stocks.created_at DESC").
		Scan(&stocks).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch picked stocks", nil)
	}

	response := map[string]interface{}{
		"stocks": stocks,
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Picked stock list fetched successfully!", response)
}

func AmcPerformance(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.
		Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").
		First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Join stocks with amc_stocks to get holdingPer
	type StockWithHolding struct {
		models.Stocks
		HoldingPer float32 `json:"holdingPer"`
	}

	var stocks []StockWithHolding
	if err := database.Database.Db.
		Table("amc_stocks").
		Select("stocks.*, amc_stocks.holding_per").
		Joins("JOIN stocks ON stocks.id = amc_stocks.stock_id").
		Where("amc_stocks.user_id = ? AND amc_stocks.is_deleted = false AND stocks.is_deleted = false", userId).
		Order("stocks.created_at DESC").
		Scan(&stocks).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch picked stocks", nil)
	}

	client := resty.New()
	resp, err := client.R().
		SetFormData(map[string]string{
			"username":   USERNAME,
			"password":   PASSWORD,
			"grant_type": "password",
		}).
		Post(AUTH_URL)
	if err != nil || resp.StatusCode() != 200 {
		log.Printf("Auth error: %v %s", err, resp.String())
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "TrueData authentication failed", nil)
	}

	var authResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(resp.Body(), &authResp); err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Invalid auth response", nil)
	}
	token := authResp.AccessToken

	type Performance struct {
		Symbol        string  `json:"symbol"`
		OpenPrice     float64 `json:"openPrice"`
		CurrentPrice  float64 `json:"currentPrice"`
		Change        float64 `json:"change"`
		PercentChange float64 `json:"percentChange"`
		HoldingPer    float32 `json:"holdingPer"`
	}

	var performances []Performance
	today := now.BeginningOfDay()
	from := today.Format("060102") + "T" + MARKET_OPEN
	to := today.Format("060102") + "T" + MARKET_CLOSE

	for _, stock := range stocks {
		url := fmt.Sprintf("%s?symbol=%s&from=%s&to=%s&response=json&interval=1min", BARS_URL, stock.Symbol, from, to)

		resp, err := client.R().
			SetHeader("Authorization", "Bearer "+token).
			Get(url)
		if err != nil || resp.StatusCode() != 200 {
			log.Printf("Error fetching bars for %s: %v %s", stock.Symbol, err, resp.String())
			continue
		}

		var barData struct {
			Records [][]interface{} `json:"Records"`
		}
		if err := json.Unmarshal(resp.Body(), &barData); err != nil || len(barData.Records) == 0 {
			log.Printf("Invalid bar data for %s", stock.Symbol)
			continue
		}

		firstBar := barData.Records[0]
		lastBar := barData.Records[len(barData.Records)-1]
		if len(firstBar) < 5 || len(lastBar) < 5 {
			log.Printf("Invalid bar structure for %s", stock.Symbol)
			continue
		}

		openPrice, ok1 := firstBar[1].(float64)
		currentPrice, ok2 := lastBar[4].(float64)
		if !ok1 || !ok2 {
			log.Printf("Price conversion error for %s", stock.Symbol)
			continue
		}

		change := currentPrice - openPrice
		percentChange := (change / openPrice) * 100

		performances = append(performances, Performance{
			Symbol:        stock.Symbol,
			OpenPrice:     openPrice,
			CurrentPrice:  currentPrice,
			Change:        change,
			PercentChange: percentChange,
			HoldingPer:    stock.HoldingPer,
		})
	}

	// Weighted average based on HoldingPer
	var totalWeightedChange float64
	var totalHolding float64
	for _, p := range performances {
		totalWeightedChange += float64(p.HoldingPer) * p.PercentChange
		totalHolding += float64(p.HoldingPer)
	}

	var avgPercentChange float64
	if totalHolding > 0 {
		avgPercentChange = totalWeightedChange / totalHolding
	} else {
		avgPercentChange = 0
	}

	response := map[string]interface{}{
		"priceChanges":         performances,
		"averagePercentChange": fmt.Sprintf("%.2f%%", avgPercentChange),
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "AMC stock performance fetched successfully!", response)
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
		Where("is_deleted = ? AND role = ?", false, "AMC").
		Offset(offset).
		Limit(*reqData.Limit).
		Find(&users).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch user list!", nil)
	}

	// Count total records
	database.Database.Db.
		Model(&models.User{}).
		Where("is_deleted = ? AND role = ?", false, "AMC").
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

// update amc details
func Update(c *fiber.Ctx) error {

	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	reqData, ok := c.Locals("validatedAMCUpdate").(*struct {
		Name                  *string  `json:"name"`
		Email                 *string  `json:"email"`
		Mobile                *string  `json:"mobile"`
		Password              *string  `json:"password"`
		PanNumber             *string  `json:"panNumber"`
		Address               *string  `json:"address"`
		City                  *string  `json:"city"`
		State                 *string  `json:"state"`
		PinCode               *string  `json:"pinCode"`
		ContactPersonName     *string  `json:"contactPersonName"`
		ContactPerDesignation *string  `json:"contactPerDesignation"`
		FundName              *string  `json:"fundName"`
		EquityPer             *float32 `json:"equityPer"`
		DebtPer               *float32 `json:"debtPer"`
		CashSplit             *float32 `json:"cashSplit"`
		IsDeleted             *bool    `json:"isDeleted"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Fetch AMC
	var amc models.User
	if err := db.Where("id = ? AND role = ?", userId, "AMC").First(&amc).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Access Denied!", nil)
	}

	// Update only provided fields
	if reqData.Name != nil {
		amc.Name = *reqData.Name
	}
	if reqData.Email != nil {
		amc.Email = *reqData.Email
	}
	if reqData.Mobile != nil {
		amc.Mobile = *reqData.Mobile
	}
	if reqData.Password != nil && *reqData.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*reqData.Password), config.AppConfig.SaltRound)
		if err != nil {
			log.Printf("Password hash error: %v", err)
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to hash password", nil)
		}
		amc.Password = string(hashedPassword)
	}
	if reqData.PanNumber != nil {
		amc.PanNumber = *reqData.PanNumber
	}
	if reqData.Address != nil {
		amc.Address = *reqData.Address
	}
	if reqData.City != nil {
		amc.City = *reqData.City
	}
	if reqData.State != nil {
		amc.State = *reqData.State
	}
	if reqData.PinCode != nil {
		amc.PinCode = *reqData.PinCode
	}
	if reqData.ContactPersonName != nil {
		amc.ContactPersonName = *reqData.ContactPersonName
	}
	if reqData.ContactPerDesignation != nil {
		amc.ContactPerDesignation = *reqData.ContactPerDesignation
	}
	if reqData.FundName != nil {
		amc.FundName = *reqData.FundName
	}
	if reqData.EquityPer != nil {
		amc.EquityPer = *reqData.EquityPer
	}
	if reqData.DebtPer != nil {
		amc.DebtPer = *reqData.DebtPer
	}
	if reqData.CashSplit != nil {
		amc.CashSplit = *reqData.CashSplit
	}
	if reqData.IsDeleted != nil {
		amc.IsDeleted = *reqData.IsDeleted
	}

	// Save changes
	if err := db.Save(&amc).Error; err != nil {
		log.Printf("Update error: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update", nil)
	}

	amc.Password = ""
	return middleware.JsonResponse(c, fiber.StatusOK, true, "Updated successfully", amc)
}
