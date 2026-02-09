package basketController

import (
	"encoding/json"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fib/models/basket"
	"fib/utils"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CreateBasket creates a new basket for AMC
func CreateBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! AMC or Admin role required.", nil)
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

	// Send Email (Async)
	go func() {
		if user.Email != "" {
			utils.SendBasketCreatedEmail(user.Email, user.Name, newBasket.Name)
		}
	}()

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket created successfully!", fiber.Map{
		"basket":  newBasket,
		"version": version,
	})
}

// UpdateBasket updates basket details
func UpdateBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
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

	// Send Email (Async)
	go func() {
		if user.Email != "" {
			utils.SendBasketUpdatedEmail(user.Email, user.Name, existingBasket.Name)
		}
	}()

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket updated successfully!", existingBasket)
}

// AddStocksToBasket adds stocks to the current draft version
func AddStocksToBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
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

	// Fetch live price for PriceAtCreation
	var priceAtCreation float64 = 0
	var bajajToken models.BajajAccessToken
	db.Order("created_at DESC").First(&bajajToken)

	if bajajToken.Token != "" && stock.Token > 0 {
		if p, err := utils.GetBajajQuote(bajajToken.Token, stock.Token); err == nil {
			priceAtCreation = p
		}
	}

	// Get draft or rejected version
	var version basket.BasketVersion
	if err := db.Where("basket_id = ? AND status IN ? AND is_deleted = false", reqData.BasketID, []string{basket.StatusDraft, basket.StatusRejected}).
		Order("version_number DESC").First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "No draft or rejected version available! Create a new version first.", nil)
	}

	// If rejected, revert to draft
	if version.Status == basket.StatusRejected {
		version.Status = basket.StatusDraft
		db.Save(&version)
	}

	// Check if stock already exists in this version
	var existingStock basket.BasketStock
	var isUpdate bool
	if err := db.Where("basket_version_id = ? AND stock_id = ? AND is_deleted = false", version.ID, reqData.StockID).First(&existingStock).Error; err == nil {
		isUpdate = true
	}

	// Auto-calculate weightage if not provided (0)
	if reqData.Weightage == 0 {
		var currentCount int64
		db.Model(&basket.BasketStock{}).Where("basket_version_id = ? AND is_deleted = false", version.ID).Count(&currentCount)

		totalStocks := currentCount
		if !isUpdate {
			totalStocks++
		}

		if totalStocks > 0 {
			newWeight := 100.0 / float64(totalStocks)

			// Update ALL existing stocks in this version
			// Note: This updates existingStock too if it exists, which is fine as we overwrite it later if needed,
			// but better to update DB first then local variable.
			if err := db.Model(&basket.BasketStock{}).Where("basket_version_id = ? AND is_deleted = false", version.ID).Update("weightage", newWeight).Error; err != nil {
				log.Printf("Error auto-balancing weights: %v", err)
			}
			reqData.Weightage = newWeight
		}
	}

	if isUpdate {
		// Update existing stock entry
		existingStock.Quantity = reqData.Quantity
		existingStock.Weightage = reqData.Weightage
		existingStock.OrderType = reqData.OrderType
		existingStock.TargetPrice = reqData.TargetPrice
		existingStock.StopLossPrice = reqData.StopLossPrice

		// Update meta fields if missing or if newer price available
		if existingStock.Token == 0 {
			existingStock.Token = stock.Token
		}
		if existingStock.Symbol == "" {
			existingStock.Symbol = stock.Symbol
		}
		if priceAtCreation > 0 {
			existingStock.PriceAtCreation = priceAtCreation
		}

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
		Token:           stock.Token,
		Symbol:          stock.Symbol,
		PriceAtCreation: priceAtCreation,
	}

	if err := db.Create(&basketStock).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to add stock!", nil)
	}

	// Record history
	metadata, _ := json.Marshal(map[string]interface{}{
		"stockId":   reqData.StockID,
		"quantity":  reqData.Quantity,
		"weightage": reqData.Weightage,
		"price":     priceAtCreation,
	})
	recordHistory(db, version.ID, basket.ActionStockAdded, userId, basket.ActorAMC, "Stock added to basket", metadata)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock added to basket!", basketStock)
}

// EditBasketStock updates detailed stock holdings
// POST /amc/basket/stocks/edit
func EditBasketStock(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedEditStock").(*struct {
		BasketID      uint     `json:"basketId"`
		StockID       uint     `json:"stockId"`
		Quantity      *int     `json:"quantity"`
		Weightage     *float64 `json:"holdingPercentage"` // Users holdingPercentage maps to Weightage
		OrderType     *string  `json:"orderType"`
		TargetPrice   *float64 `json:"tgtPrice"` // Users tgtPrice maps to TargetPrice
		StopLossPrice *float64 `json:"slPrice"`  // Users slPrice maps to StopLossPrice
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

	// Get draft or rejected version
	var version basket.BasketVersion
	if err := db.Where("basket_id = ? AND status IN ? AND is_deleted = false", reqData.BasketID, []string{basket.StatusDraft, basket.StatusRejected}).
		Order("version_number DESC").First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "No editable version available!", nil)
	}

	// If rejected, revert to draft to edit
	if version.Status == basket.StatusRejected {
		version.Status = basket.StatusDraft
		db.Save(&version)
	}

	// Find the stock
	var stock basket.BasketStock
	if err := db.Where("basket_version_id = ? AND stock_id = ? AND is_deleted = false", version.ID, reqData.StockID).First(&stock).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Stock not found in this basket version!", nil)
	}

	// Update fields
	if reqData.Quantity != nil {
		stock.Quantity = *reqData.Quantity
	}
	if reqData.Weightage != nil {
		stock.Weightage = *reqData.Weightage
	}
	if reqData.OrderType != nil {
		stock.OrderType = *reqData.OrderType
	}
	if reqData.TargetPrice != nil {
		stock.TargetPrice = *reqData.TargetPrice
	}
	if reqData.StopLossPrice != nil {
		stock.StopLossPrice = *reqData.StopLossPrice
	}

	if err := db.Save(&stock).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update stock!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock holdings updated!", stock)
}

// RemoveStockFromBasket removes a stock from the draft version
func RemoveStockFromBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
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

	// Get draft or rejected version
	var version basket.BasketVersion
	if err := db.Where("basket_id = ? AND status IN ? AND is_deleted = false", reqData.BasketID, []string{basket.StatusDraft, basket.StatusRejected}).
		Order("version_number DESC").First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "No draft or rejected version available!", nil)
	}

	// If rejected, revert to draft
	if version.Status == basket.StatusRejected {
		version.Status = basket.StatusDraft
		db.Save(&version)
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

	// Auto-rebalance weights after removal (Optional, but consistent with 'always 100%')
	// Count remaining stocks
	var remainingCount int64
	db.Model(&basket.BasketStock{}).Where("basket_version_id = ? AND is_deleted = false", version.ID).Count(&remainingCount)
	if remainingCount > 0 {
		newWeight := 100.0 / float64(remainingCount)
		db.Model(&basket.BasketStock{}).Where("basket_version_id = ? AND is_deleted = false", version.ID).Update("weightage", newWeight)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Stock removed from basket!", nil)
}

// SubmitForApproval submits the draft version for admin approval
func SubmitForApproval(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
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

	// Get draft or rejected version
	var version basket.BasketVersion
	if err := db.Where("basket_id = ? AND status IN ? AND is_deleted = false", reqData.BasketID, []string{basket.StatusDraft, basket.StatusRejected}).
		Order("version_number DESC").First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "No draft or rejected version available for submission!", nil)
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

	// Send Email (Async)
	go func() {
		if user.Email != "" {
			utils.SendBasketSubmittedEmail(user.Email, user.Name, existingBasket.Name)
		}
	}()

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket submitted for approval!", version)
}

// GetMyBaskets lists all baskets for the AMC
func GetMyBaskets(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
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
		Preload("Versions.Stocks", "is_deleted = false"). // Preload Stocks
		Offset(offset).Limit(*reqData.Limit).
		Order("created_at DESC").
		Find(&baskets).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch baskets!", nil)
	}

	// Populate stock names manually (as Preload doesn't support joins for non-relation fields easily without custom struct)
	var stockIDs []uint
	for i := range baskets {
		for j := range baskets[i].Versions {
			for k := range baskets[i].Versions[j].Stocks {
				stockIDs = append(stockIDs, baskets[i].Versions[j].Stocks[k].StockID)
			}
		}
	}

	if len(stockIDs) > 0 {
		var stocks []models.Stocks
		// Fetch only ID and Name to optimize
		db.Table("stocks").Select("id, name").Where("id IN ?", stockIDs).Find(&stocks)

		stockNameMap := make(map[uint]string)
		for _, s := range stocks {
			stockNameMap[s.ID] = s.Name
		}

		// Assign names back to basket stocks
		for i := range baskets {
			for j := range baskets[i].Versions {
				for k := range baskets[i].Versions[j].Stocks {
					if name, ok := stockNameMap[baskets[i].Versions[j].Stocks[k].StockID]; ok {
						baskets[i].Versions[j].Stocks[k].StockName = name
					}
				}
			}
		}
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
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
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

// GetBasketSubscribers returns all users subscribed to a basket (for AMC/Admin)
func GetBasketSubscribers(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	basketId := c.Params("id")

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! AMC or Admin role required.", nil)
	}

	db := database.Database.Db

	// Check basket exists
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND is_deleted = false", basketId).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	// For AMC, ensure they own the basket
	if user.Role == "AMC" && existingBasket.AMCID != userId {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "You don't have access to this basket!", nil)
	}

	// Parse pagination
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	status := c.Query("status") // ACTIVE, EXPIRED, CANCELLED

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := db.Model(&basket.BasketSubscription{}).
		Where("basket_id = ? AND is_deleted = false", basketId)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	// Get subscriptions with user details
	var subscriptions []basket.BasketSubscription
	if err := query.
		Preload("BasketVersion").
		Order("subscribed_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&subscriptions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch subscribers!", nil)
	}

	// Get user details separately to include name and email
	type SubscriberInfo struct {
		basket.BasketSubscription
		UserName  string `json:"userName"`
		UserEmail string `json:"userEmail"`
		UserPhone string `json:"userPhone"`
	}

	var subscribers []SubscriberInfo
	for _, sub := range subscriptions {
		var subUser models.User
		db.Where("id = ?", sub.UserID).First(&subUser)

		subscribers = append(subscribers, SubscriberInfo{
			BasketSubscription: sub,
			UserName:           subUser.Name,
			UserEmail:          subUser.Email,
			UserPhone:          subUser.Mobile,
		})
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Subscribers fetched!", fiber.Map{
		"basket":      existingBasket,
		"subscribers": subscribers,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
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

// GetAMCBasketDetails returns comprehensive basket details for AMC
// Includes all versions, stocks, history, reviews, and subscribers
func GetAMCBasketDetails(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	basketId := c.Params("id")

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	db := database.Database.Db

	// Verify basket ownership (AMC can only see their own baskets)
	var existingBasket basket.Basket
	query := db.Where("id = ? AND is_deleted = false", basketId)
	if user.Role == "AMC" {
		query = query.Where("amc_id = ?", userId)
	}
	if err := query.First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	// Fetch all versions with stocks, history, and time slots
	var versions []basket.BasketVersion
	if err := db.Where("basket_id = ? AND is_deleted = false", basketId).
		Preload("Stocks", "is_deleted = false").
		Preload("TimeSlot").
		Preload("History").
		Order("version_number DESC").
		Find(&versions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch versions!", nil)
	}

	// Fetch reviews for this basket
	var reviews []basket.BasketReview
	db.Where("basket_id = ? AND is_deleted = false", basketId).
		Preload("User").
		Order("created_at DESC").
		Find(&reviews)

	// Build review response with user info
	type ReviewDetail struct {
		basket.BasketReview
		UserName string `json:"userName"`
	}
	var reviewDetails []ReviewDetail
	for _, r := range reviews {
		var reviewUser models.User
		db.Select("name").Where("id = ?", r.UserID).First(&reviewUser)
		reviewDetails = append(reviewDetails, ReviewDetail{
			BasketReview: r,
			UserName:     reviewUser.Name,
		})
	}

	// Fetch subscribers count and list (limited)
	var totalSubscribers int64
	var activeSubscribers int64
	db.Model(&basket.BasketSubscription{}).Where("basket_id = ? AND is_deleted = false", basketId).Count(&totalSubscribers)
	db.Model(&basket.BasketSubscription{}).Where("basket_id = ? AND status = ? AND is_deleted = false", basketId, basket.SubscriptionActive).Count(&activeSubscribers)

	// Get recent 10 subscribers with details
	var subscriptions []basket.BasketSubscription
	db.Where("basket_id = ? AND is_deleted = false", basketId).
		Order("subscribed_at DESC").
		Limit(10).
		Find(&subscriptions)

	type SubscriberInfo struct {
		basket.BasketSubscription
		UserName  string `json:"userName"`
		UserEmail string `json:"userEmail"`
	}
	var subscriberDetails []SubscriberInfo
	for _, sub := range subscriptions {
		var subUser models.User
		db.Select("name, email").Where("id = ?", sub.UserID).First(&subUser)
		subscriberDetails = append(subscriberDetails, SubscriberInfo{
			BasketSubscription: sub,
			UserName:           subUser.Name,
			UserEmail:          subUser.Email,
		})
	}

	// Get Bajaj Token for live pricing
	var bajajToken models.BajajAccessToken
	db.Order("created_at DESC").First(&bajajToken)
	accessToken := bajajToken.Token

	// Calculate version details with pricing
	type VersionDetail struct {
		basket.BasketVersion
		InitialPrice float64 `json:"initialPrice"`
		CurrentPrice float64 `json:"currentPrice"`
		StockCount   int     `json:"stockCount"`
	}
	var versionDetails []VersionDetail

	for _, v := range versions {
		// Calculate initial price
		var initialPrice float64 = 0
		if v.PriceAtApproval > 0 {
			initialPrice = v.PriceAtApproval
		} else {
			for _, s := range v.Stocks {
				initialPrice += s.PriceAtCreation * float64(s.Quantity)
			}
		}

		// Calculate current/achieved price
		var currentPrice float64 = 0
		if v.Status == basket.StatusExpired && v.PriceAtExpiry > 0 {
			currentPrice = v.PriceAtExpiry
		} else {
			for _, s := range v.Stocks {
				// Try live price first
				if accessToken != "" && s.Token > 0 {
					if livePrice, err := utils.GetBajajQuote(accessToken, s.Token); err == nil && livePrice > 0 {
						currentPrice += livePrice * float64(s.Quantity)
						continue
					}
				}
				// Fallback to stored prices
				if s.PriceAtApproval > 0 {
					currentPrice += s.PriceAtApproval * float64(s.Quantity)
				} else {
					currentPrice += s.PriceAtCreation * float64(s.Quantity)
				}
			}
		}

		versionDetails = append(versionDetails, VersionDetail{
			BasketVersion: v,
			InitialPrice:  initialPrice,
			CurrentPrice:  currentPrice,
			StockCount:    len(v.Stocks),
		})
	}

	// Calculate average rating
	var avgRating float64 = 0
	var totalRatings int64
	db.Model(&basket.BasketReview{}).Where("basket_id = ? AND is_deleted = false AND status = ?", basketId, "APPROVED").Count(&totalRatings)
	if totalRatings > 0 {
		db.Model(&basket.BasketReview{}).Where("basket_id = ? AND is_deleted = false AND status = ?", basketId, "APPROVED").Select("COALESCE(AVG(rating), 0)").Scan(&avgRating)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket details fetched successfully!", fiber.Map{
		"basket":   existingBasket,
		"versions": versionDetails,
		"reviews": fiber.Map{
			"items":        reviewDetails,
			"totalCount":   len(reviews),
			"avgRating":    avgRating,
			"totalRatings": totalRatings,
		},
		"subscribers": fiber.Map{
			"total":      totalSubscribers,
			"active":     activeSubscribers,
			"recentList": subscriberDetails,
		},
		"stats": fiber.Map{
			"totalVersions":    len(versions),
			"currentVersionId": existingBasket.CurrentVersionID,
		},
	})
}
