package basketController

import (
	"fib/database"
	"fib/middleware"
	"fib/models"

	"github.com/gofiber/fiber/v2"
)

// GetStocksList returns paginated list of stocks for adding to baskets
func GetStocksList(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Parse query params
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("sizePerPage", 10)
	search := c.Query("search")
	id := c.QueryInt("id", 0)
	exchId := c.Query("exchId")                     // Filter by exchange: NSE, BSE, NSEFO, BSEFO
	instType := c.Query("instrumentType")           // Filter by instrument type: EQ, OPTSTK, FUTIDX, etc.
	equitiesOnly := c.Query("equitiesOnly", "true") // Default to equities only

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit
	db := database.Database.Db

	query := db.Model(&models.Stocks{}).Where("is_deleted = false")

	// By default, show only equities (NSE/BSE), not derivatives (NSEFO/BSEFO)
	if equitiesOnly == "true" && exchId == "" {
		query = query.Where("exch_id IN ?", []string{"NSE", "BSE"})
	}

	// Filter by exchange ID if provided
	if exchId != "" {
		query = query.Where("exch_id = ?", exchId)
	}

	// Filter by instrument type if provided
	if instType != "" {
		query = query.Where("inst_type = ?", instType)
	}

	// Search by symbol or full name
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("symbol ILIKE ? OR full_name ILIKE ? OR name ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	// Filter by ID
	if id > 0 {
		query = query.Where("id = ?", id)
	}

	// Count total
	var total int64
	query.Count(&total)

	// Get stocks - order by symbol for better UX
	var stocks []models.Stocks
	if err := query.
		Order("symbol ASC, exch_id ASC").
		Offset(offset).
		Limit(limit).
		Find(&stocks).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch stocks!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stocks list fetched!", fiber.Map{
		"totalRecords": total,
		"totalPages":   (total + int64(limit) - 1) / int64(limit),
		"currentPage":  page,
		"stocksList":   stocks,
	})
}

// GetStockByToken returns a stock by its exchange token
func GetStockByToken(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	token := c.QueryInt("token", 0)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	if token == 0 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Token is required!", nil)
	}

	var stock models.Stocks
	if err := database.Database.Db.Where("token = ? AND is_deleted = false", token).First(&stock).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Stock not found!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock found!", stock)
}

// GetStockBySymbol returns a stock by its symbol
func GetStockBySymbol(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	symbol := c.Query("symbol")

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	if symbol == "" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Symbol is required!", nil)
	}

	var stock models.Stocks
	if err := database.Database.Db.Where("symbol = ? AND is_deleted = false", symbol).First(&stock).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Stock not found!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock found!", stock)
}
