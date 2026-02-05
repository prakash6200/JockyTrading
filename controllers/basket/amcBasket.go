package basketController

import (
	"encoding/json"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fib/models/basket"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CreateBasket creates a new basket for AMC
func CreateBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! AMC role required.", nil)
	}

	reqData, ok := c.Locals("validatedCreateBasket").(*struct {
		Name            string  `json:"name"`
		Description     string  `json:"description"`
		BasketType      string  `json:"basketType"`
		SubscriptionFee float64 `json:"subscriptionFee"`
		IsFeeBased      bool    `json:"isFeeBased"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Check if basket name already exists for this AMC
	var existingBasket basket.Basket
	if err := db.Where("name = ? AND amc_id = ? AND is_deleted = false", reqData.Name, userId).First(&existingBasket).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Basket with this name already exists!", nil)
	}

	// Create basket
	newBasket := basket.Basket{
		Name:            reqData.Name,
		Description:     reqData.Description,
		AMCID:           userId,
		BasketType:      reqData.BasketType,
		SubscriptionFee: reqData.SubscriptionFee,
		IsFeeBased:      reqData.IsFeeBased,
	}

	if err := db.Create(&newBasket).Error; err != nil {
		log.Printf("Error creating basket: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create basket!", nil)
	}

	// Create initial draft version
	version := basket.BasketVersion{
		BasketID:      newBasket.ID,
		VersionNumber: 1,
		Status:        basket.StatusDraft,
	}

	if err := db.Create(&version).Error; err != nil {
		log.Printf("Error creating basket version: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create basket version!", nil)
	}

	// Update basket with current version
	newBasket.CurrentVersionID = &version.ID
	db.Save(&newBasket)

	// Record history
	recordHistory(db, version.ID, basket.ActionCreated, userId, basket.ActorAMC, "Basket created", nil)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket created successfully!", fiber.Map{
		"basket":  newBasket,
		"version": version,
	})
}

// UpdateBasket updates basket details
func UpdateBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedUpdateBasket").(*struct {
		BasketID        uint     `json:"basketId"`
		Name            *string  `json:"name"`
		Description     *string  `json:"description"`
		SubscriptionFee *float64 `json:"subscriptionFee"`
		IsFeeBased      *bool    `json:"isFeeBased"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Find basket owned by this AMC
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND amc_id = ? AND is_deleted = false", reqData.BasketID, userId).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found or not yours!", nil)
	}

	// Update fields
	if reqData.Name != nil {
		existingBasket.Name = *reqData.Name
	}
	if reqData.Description != nil {
		existingBasket.Description = *reqData.Description
	}
	if reqData.SubscriptionFee != nil {
		existingBasket.SubscriptionFee = *reqData.SubscriptionFee
	}
	if reqData.IsFeeBased != nil {
		existingBasket.IsFeeBased = *reqData.IsFeeBased
	}

	if err := db.Save(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update basket!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket updated successfully!", existingBasket)
}

// AddStocksToBasket adds stocks to the current draft version
func AddStocksToBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedAddStocks").(*struct {
		BasketID      uint    `json:"basketId"`
		StockID       uint    `json:"stockId"`
		Quantity      int     `json:"quantity"`
		Weightage     float64 `json:"weightage"`
		OrderType     string  `json:"orderType"`
		TargetPrice   float64 `json:"targetPrice"`
		StopLossPrice float64 `json:"stopLossPrice"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Find basket
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND amc_id = ? AND is_deleted = false", reqData.BasketID, userId).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	// Verify stock exists
	var stock models.Stocks
	if err := db.Where("id = ? AND is_deleted = false", reqData.StockID).First(&stock).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Stock not found!", nil)
	}

	// Get current draft version
	var version basket.BasketVersion
	if err := db.Where("basket_id = ? AND status = ? AND is_deleted = false", reqData.BasketID, basket.StatusDraft).
		Order("version_number DESC").First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "No draft version available! Create a new version first.", nil)
	}

	// Check if stock already exists in this version
	var existingStock basket.BasketStock
	if err := db.Where("basket_version_id = ? AND stock_id = ? AND is_deleted = false", version.ID, reqData.StockID).First(&existingStock).Error; err == nil {
		// Update existing stock entry
		existingStock.Quantity = reqData.Quantity
		existingStock.Weightage = reqData.Weightage
		existingStock.OrderType = reqData.OrderType
		existingStock.TargetPrice = reqData.TargetPrice
		existingStock.StopLossPrice = reqData.StopLossPrice
		db.Save(&existingStock)

		return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock updated in basket!", existingStock)
	}

	// Add new stock
	orderType := reqData.OrderType
	if orderType == "" {
		orderType = "MARKET"
	}

	basketStock := basket.BasketStock{
		BasketVersionID: version.ID,
		StockID:         reqData.StockID,
		Quantity:        reqData.Quantity,
		Weightage:       reqData.Weightage,
		OrderType:       orderType,
		TargetPrice:     reqData.TargetPrice,
		StopLossPrice:   reqData.StopLossPrice,
	}

	if err := db.Create(&basketStock).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to add stock!", nil)
	}

	// Record history
	metadata, _ := json.Marshal(map[string]interface{}{
		"stockId":   reqData.StockID,
		"quantity":  reqData.Quantity,
		"weightage": reqData.Weightage,
	})
	recordHistory(db, version.ID, basket.ActionStockAdded, userId, basket.ActorAMC, "Stock added to basket", metadata)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock added to basket!", basketStock)
}

// RemoveStockFromBasket removes a stock from the draft version
func RemoveStockFromBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedRemoveStock").(*struct {
		BasketID uint `json:"basketId"`
		StockID  uint `json:"stockId"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Verify basket ownership
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND amc_id = ? AND is_deleted = false", reqData.BasketID, userId).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	// Get draft version
	var version basket.BasketVersion
	if err := db.Where("basket_id = ? AND status = ? AND is_deleted = false", reqData.BasketID, basket.StatusDraft).
		Order("version_number DESC").First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "No draft version available!", nil)
	}

	// Soft delete the stock
	result := db.Model(&basket.BasketStock{}).
		Where("basket_version_id = ? AND stock_id = ? AND is_deleted = false", version.ID, reqData.StockID).
		Update("is_deleted", true)

	if result.RowsAffected == 0 {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Stock not found in basket!", nil)
	}

	// Record history
	metadata, _ := json.Marshal(map[string]interface{}{"stockId": reqData.StockID})
	recordHistory(db, version.ID, basket.ActionStockRemoved, userId, basket.ActorAMC, "Stock removed from basket", metadata)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock removed from basket!", nil)
}

// SubmitForApproval submits the draft version for admin approval
func SubmitForApproval(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedSubmitForApproval").(*struct {
		BasketID uint `json:"basketId"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Verify basket ownership
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND amc_id = ? AND is_deleted = false", reqData.BasketID, userId).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	// Get draft version
	var version basket.BasketVersion
	if err := db.Where("basket_id = ? AND status = ? AND is_deleted = false", reqData.BasketID, basket.StatusDraft).
		Order("version_number DESC").First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "No draft version available for submission!", nil)
	}

	// Check if there are stocks in the version
	var stockCount int64
	db.Model(&basket.BasketStock{}).Where("basket_version_id = ? AND is_deleted = false", version.ID).Count(&stockCount)
	if stockCount == 0 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Cannot submit basket with no stocks!", nil)
	}

	// Update version status
	now := time.Now()
	version.Status = basket.StatusPendingApproval
	version.SubmittedAt = &now
	db.Save(&version)

	// Record history
	recordHistory(db, version.ID, basket.ActionSubmitted, userId, basket.ActorAMC, "Basket submitted for approval", nil)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket submitted for approval!", version)
}

// GetMyBaskets lists all baskets for the AMC
func GetMyBaskets(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedListBaskets").(*struct {
		Page       *int    `json:"page"`
		Limit      *int    `json:"limit"`
		Search     *string `json:"search"`
		BasketType *string `json:"basketType"`
		Status     *string `json:"status"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db
	offset := (*reqData.Page - 1) * (*reqData.Limit)

	query := db.Model(&basket.Basket{}).Where("amc_id = ? AND is_deleted = false", userId)

	if reqData.Search != nil && *reqData.Search != "" {
		search := "%" + *reqData.Search + "%"
		query = query.Where("name ILIKE ?", search)
	}
	if reqData.BasketType != nil && *reqData.BasketType != "" {
		query = query.Where("basket_type = ?", *reqData.BasketType)
	}

	var total int64
	query.Count(&total)

	var baskets []basket.Basket
	if err := query.Preload("Versions", "is_deleted = false").
		Offset(offset).Limit(*reqData.Limit).
		Order("created_at DESC").
		Find(&baskets).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch baskets!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Baskets fetched successfully!", fiber.Map{
		"baskets": baskets,
		"pagination": fiber.Map{
			"total": total,
			"page":  *reqData.Page,
			"limit": *reqData.Limit,
		},
	})
}

// GetBasketHistory gets all versions and history of a basket
func GetBasketHistory(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	basketId := c.Params("id")

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "AMC").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	db := database.Database.Db

	// Verify basket ownership
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND amc_id = ? AND is_deleted = false", basketId, userId).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	// Get all versions with stocks and history
	var versions []basket.BasketVersion
	if err := db.Where("basket_id = ? AND is_deleted = false", basketId).
		Preload("Stocks", "is_deleted = false").
		Preload("TimeSlot").
		Preload("History").
		Order("version_number DESC").
		Find(&versions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch history!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket history fetched!", fiber.Map{
		"basket":   existingBasket,
		"versions": versions,
	})
}

// Helper function to record history
func recordHistory(db interface{}, versionId uint, action string, actorId uint, actorType string, comments string, metadata []byte) {
	history := basket.BasketHistory{
		BasketVersionID: versionId,
		Action:          action,
		ActorID:         actorId,
		ActorType:       actorType,
		Comments:        comments,
		Metadata:        string(metadata),
	}

	database.Database.Db.Create(&history)
}
