package basketController

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fib/models/basket"
	"fib/utils"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ListPublishedBaskets lists all published baskets for users
func ListPublishedBaskets(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedListPublished").(*struct {
		Page       *int    `json:"page"`
		Limit      *int    `json:"limit"`
		Search     *string `json:"search"`
		BasketType *string `json:"basketType"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db
	offset := (*reqData.Page - 1) * (*reqData.Limit)

	// Get baskets with PUBLISHED or SCHEDULED versions
	query := db.Model(&basket.Basket{}).
		Where("is_deleted = false").
		Where("current_version_id IN (?)",
			db.Model(&basket.BasketVersion{}).Select("id").
				Where("status IN ? AND is_deleted = false", []string{basket.StatusPublished, basket.StatusScheduled}))

	if reqData.Search != nil && *reqData.Search != "" {
		search := "%" + *reqData.Search + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", search, search)
	}
	if reqData.BasketType != nil && *reqData.BasketType != "" {
		query = query.Where("basket_type = ?", *reqData.BasketType)
	}

	var total int64
	query.Count(&total)

	var baskets []basket.Basket
	if err := query.
		Preload("Versions", "status IN ? AND is_deleted = false", []string{basket.StatusPublished, basket.StatusScheduled}).
		Preload("Versions.Stocks", "is_deleted = false").
		Preload("Versions.TimeSlot").
		Offset(offset).Limit(*reqData.Limit).
		Order("created_at DESC").
		Find(&baskets).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch baskets!", nil)
	}

	// Prepare response with pricing
	type BasketResponse struct {
		basket.Basket
		InitialPrice float64 `json:"initialPrice"`
		CurrentPrice float64 `json:"currentPrice"`
	}

	var response []BasketResponse

	// Get access token for live pricing
	var bajajToken models.BajajAccessToken
	database.Database.Db.Order("created_at DESC").First(&bajajToken)
	accessToken := bajajToken.Token

	for _, b := range baskets {
		var initialPrice float64 = 0
		var currentPrice float64 = 0

		// Use the relevant version (Published or Scheduled)
		// Since we preloaded Versions with matching status, usually there's only 1 active version or we pick the first one
		if len(b.Versions) > 0 {
			v := b.Versions[0]
			initialPrice = v.PriceAtApproval

			// Calculate fallback initial price if 0 (legacy data)
			if initialPrice == 0 {
				for _, stock := range v.Stocks {
					initialPrice += stock.PriceAtCreation * float64(stock.Quantity)
				}
			}

			// Calculate current price
			for _, stock := range v.Stocks {
				// Live price
				if accessToken != "" && stock.Token > 0 {
					if livePrice, err := utils.GetBajajQuote(accessToken, stock.Token); err == nil && livePrice > 0 {
						currentPrice += livePrice * float64(stock.Quantity)
						continue
					}
				}

				// Fallback to approval price or creation price
				if stock.PriceAtApproval > 0 {
					currentPrice += stock.PriceAtApproval * float64(stock.Quantity)
				} else {
					currentPrice += stock.PriceAtCreation * float64(stock.Quantity)
				}
			}

			// HIDE STOCKS from list view as requested
			b.Versions[0].Stocks = nil
		}

		response = append(response, BasketResponse{
			Basket:       b,
			InitialPrice: initialPrice,
			CurrentPrice: currentPrice,
		})
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Published baskets fetched!", fiber.Map{
		"baskets": response,
		"pagination": fiber.Map{
			"total": total,
			"page":  *reqData.Page,
			"limit": *reqData.Limit,
		},
	})
}

// GetLiveIntraHourBaskets returns currently LIVE INTRA_HOUR baskets
func GetLiveIntraHourBaskets(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	db := database.Database.Db
	now := time.Now()

	// Find INTRA_HOUR baskets that are currently PUBLISHED
	var versions []basket.BasketVersion
	if err := db.Model(&basket.BasketVersion{}).
		Where("basket_versions.status = ? AND basket_versions.is_deleted = false", basket.StatusPublished).
		Joins("JOIN baskets ON baskets.id = basket_versions.basket_id").
		Where("baskets.basket_type = ? AND baskets.is_deleted = false", basket.BasketTypeIntraHour).
		Preload("Basket").
		Preload("Stocks", "is_deleted = false").
		Preload("TimeSlot", "start_time <= ? AND end_time >= ?", now, now).
		Find(&versions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch live baskets!", nil)
	}

	// Filter versions that have valid time slots
	var liveVersions []basket.BasketVersion
	for _, v := range versions {
		if v.TimeSlot != nil {
			liveVersions = append(liveVersions, v)
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Live INTRA_HOUR baskets fetched!", liveVersions)
}

// GetUpcomingIntraHourBaskets returns SCHEDULED (upcoming) INTRA_HOUR baskets
func GetUpcomingIntraHourBaskets(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	db := database.Database.Db
	now := time.Now()

	// Find INTRA_HOUR baskets that are SCHEDULED with future start time
	var versions []basket.BasketVersion
	if err := db.Model(&basket.BasketVersion{}).
		Where("basket_versions.status = ? AND basket_versions.is_deleted = false", basket.StatusScheduled).
		Joins("JOIN baskets ON baskets.id = basket_versions.basket_id").
		Where("baskets.basket_type = ? AND baskets.is_deleted = false", basket.BasketTypeIntraHour).
		Preload("Basket").
		Preload("TimeSlot", "start_time > ?", now).
		Order("basket_versions.created_at ASC").
		Find(&versions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch upcoming baskets!", nil)
	}

	// Filter versions that have valid future time slots
	var upcomingVersions []basket.BasketVersion
	for _, v := range versions {
		if v.TimeSlot != nil {
			upcomingVersions = append(upcomingVersions, v)
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Upcoming INTRA_HOUR baskets fetched!", upcomingVersions)
}

// GetBasketDetails returns basket details with current stock list
func GetBasketDetails(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	basketId := c.Params("id")

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	db := database.Database.Db

	var existingBasket basket.Basket
	if err := db.Where("id = ? AND is_deleted = false", basketId).
		Preload("Versions", "status IN ? AND is_deleted = false", []string{basket.StatusPublished, basket.StatusScheduled}).
		Preload("Versions.Stocks", "is_deleted = false").
		Preload("Versions.TimeSlot").
		First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	// Check if user is subscribed
	var subscription basket.BasketSubscription
	isSubscribed := db.Where("user_id = ? AND basket_id = ? AND status = ?", userId, basketId, basket.SubscriptionActive).First(&subscription).Error == nil

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket details fetched!", fiber.Map{
		"basket":       existingBasket,
		"isSubscribed": isSubscribed,
		"subscription": subscription,
	})
}

// Subscribe subscribes a user to a basket
func Subscribe(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedSubscribe").(*struct {
		BasketID uint `json:"basketId"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Find basket
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND is_deleted = false", reqData.BasketID).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	// Get current published or scheduled version
	var version basket.BasketVersion
	if err := db.Where("basket_id = ? AND status IN ? AND is_deleted = false", reqData.BasketID, []string{basket.StatusPublished, basket.StatusScheduled}).
		Order("version_number DESC").First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "No published or scheduled version available for subscription!", nil)
	}

	// For INTRA_HOUR, check time slot
	if existingBasket.BasketType == basket.BasketTypeIntraHour {
		var timeSlot basket.BasketTimeSlot
		if err := db.Where("basket_version_id = ?", version.ID).First(&timeSlot).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "INTRA_HOUR basket has no scheduled time slot!", nil)
		}

		now := time.Now()
		// Allow subscription if:
		// 1. Basket is currently LIVE (status = PUBLISHED and within time window)
		// 2. Basket is SCHEDULED (pre-subscription allowed)
		if version.Status == basket.StatusPublished {
			// Must be within time window
			if now.Before(timeSlot.StartTime) || now.After(timeSlot.EndTime) {
				return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "INTRA_HOUR basket is not currently LIVE!", nil)
			}
		}
		// If SCHEDULED, allow pre-subscription
	}

	// Check if already subscribed
	var existingSubscription basket.BasketSubscription
	if err := db.Where("user_id = ? AND basket_id = ? AND status = ?", userId, reqData.BasketID, basket.SubscriptionActive).First(&existingSubscription).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Already subscribed to this basket!", nil)
	}

	// Check balance for fee-based baskets
	if existingBasket.IsFeeBased {
		if float64(user.MainBalance) < existingBasket.SubscriptionFee {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Insufficient balance for subscription!", nil)
		}

		// Record transaction and deduct fee
		balanceBefore := float64(user.MainBalance)
		user.MainBalance -= uint(existingBasket.SubscriptionFee)

		// Create wallet transaction record
		walletTxn := models.WalletTransaction{
			UserID:          userId,
			TransactionType: models.TransactionTypeSubscription,
			Amount:          existingBasket.SubscriptionFee,
			BalanceBefore:   balanceBefore,
			BalanceAfter:    float64(user.MainBalance),
			Status:          models.TransactionStatusCompleted,
			Description:     "Subscription: " + existingBasket.Name,
			ReferenceType:   "basket",
			ReferenceID:     existingBasket.ID,
			ReferenceName:   existingBasket.Name,
			TransactionDate: time.Now(),
		}

		if err := db.Create(&walletTxn).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to record transaction!", nil)
		}

		db.Save(&user)
	}

	// Create subscription
	subscription := basket.BasketSubscription{
		UserID:            userId,
		BasketID:          reqData.BasketID,
		BasketVersionID:   version.ID,
		SubscribedAt:      time.Now(),
		SubscriptionPrice: existingBasket.SubscriptionFee,
		BasketPrice:       version.PriceAtApproval,
		Status:            basket.SubscriptionActive,
	}

	// Set expiry based on basket type
	switch existingBasket.BasketType {
	case basket.BasketTypeIntraHour:
		// Expires when time slot ends
		var timeSlot basket.BasketTimeSlot
		if err := db.Where("basket_version_id = ?", version.ID).First(&timeSlot).Error; err == nil {
			subscription.ExpiresAt = &timeSlot.EndTime
		}
	case basket.BasketTypeIntraday:
		// Expires at market close
		marketClose := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 15, 30, 0, 0, time.FixedZone("IST", 5*60*60+30*60))
		subscription.ExpiresAt = &marketClose
	default:
		// DELIVERY: No expiry
		subscription.ExpiresAt = nil
	}

	if err := db.Create(&subscription).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to subscribe!", nil)
	}

	// Preload basket data for response
	db.Preload("Basket").Preload("BasketVersion").Preload("BasketVersion.Stocks", "is_deleted = false").First(&subscription)

	// Send Subscription Email
	utils.SendSubscriptionEmail(user.Email, user.Name, existingBasket.Name)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Subscribed successfully!", subscription)
}

// GetMySubscriptions returns user's subscriptions with performance
func GetMySubscriptions(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedMySubscriptions").(*struct {
		Page   *int    `json:"page"`
		Limit  *int    `json:"limit"`
		Status *string `json:"status"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db
	offset := (*reqData.Page - 1) * (*reqData.Limit)

	query := db.Model(&basket.BasketSubscription{}).Where("user_id = ? AND is_deleted = false", userId)

	if reqData.Status != nil && *reqData.Status != "" {
		query = query.Where("status = ?", *reqData.Status)
	}

	var total int64
	query.Count(&total)

	var subscriptions []basket.BasketSubscription
	if err := query.
		Preload("Basket").
		Preload("Basket.CurrentVersion.Stocks", "is_deleted = false").
		Preload("BasketVersion.Stocks", "is_deleted = false"). // Load basketVersion stocks as well
		Offset(offset).Limit(*reqData.Limit).
		Order("subscribed_at DESC").
		Find(&subscriptions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch subscriptions!", nil)
	}

	// Populate stock names from stocks table
	for i := range subscriptions {
		// Populate stock names for basket.currentVersion.stocks
		if subscriptions[i].Basket.CurrentVersion != nil {
			for j := range subscriptions[i].Basket.CurrentVersion.Stocks {
				stock := &subscriptions[i].Basket.CurrentVersion.Stocks[j]
				if stock.StockID > 0 {
					var stockData models.Stocks
					if err := db.Select("name, full_name").Where("id = ?", stock.StockID).First(&stockData).Error; err == nil {
						// Prefer full_name, fallback to name
						if stockData.FullName != "" {
							stock.StockName = stockData.FullName
						} else {
							stock.StockName = stockData.Name
						}
					}
				}
			}
		}

		// Populate stock names for basketVersion.stocks
		for j := range subscriptions[i].BasketVersion.Stocks {
			stock := &subscriptions[i].BasketVersion.Stocks[j]
			if stock.StockID > 0 {
				var stockData models.Stocks
				if err := db.Select("name, full_name").Where("id = ?", stock.StockID).First(&stockData).Error; err == nil {
					// Prefer full_name, fallback to name
					if stockData.FullName != "" {
						stock.StockName = stockData.FullName
					} else {
						stock.StockName = stockData.Name
					}
				}
			}
		}
	}

	// Prepare response with pricing
	type SubscriptionResponse struct {
		basket.BasketSubscription
		InitialPrice float64 `json:"initialPrice"`
		CurrentPrice float64 `json:"currentPrice"`
	}

	var response []SubscriptionResponse

	// Get access token for live pricing
	var bajajToken models.BajajAccessToken
	database.Database.Db.Order("created_at DESC").First(&bajajToken)
	accessToken := bajajToken.Token

	for _, sub := range subscriptions {
		var initialPrice float64 = 0
		var currentPrice float64 = 0

		// Use Current Active Version if available, else fallback to Subscribed Version
		var targetVersion basket.BasketVersion
		if sub.Basket.CurrentVersion != nil {
			targetVersion = *sub.Basket.CurrentVersion
		} else {
			targetVersion = sub.BasketVersion
		}

		// Calculation logic using targetVersion
		if targetVersion.ID != 0 {
			initialPrice = targetVersion.PriceAtApproval

			// Fallback initial price
			if initialPrice == 0 {
				for _, stock := range targetVersion.Stocks {
					initialPrice += stock.PriceAtCreation * float64(stock.Quantity)
				}
			}

			// Current price
			for _, stock := range targetVersion.Stocks {
				// Live price
				if accessToken != "" && stock.Token > 0 {
					if livePrice, err := utils.GetBajajQuote(accessToken, stock.Token); err == nil && livePrice > 0 {
						currentPrice += livePrice * float64(stock.Quantity)
						continue
					}
				}

				// Fallback
				if stock.PriceAtApproval > 0 {
					currentPrice += stock.PriceAtApproval * float64(stock.Quantity)
				} else {
					currentPrice += stock.PriceAtCreation * float64(stock.Quantity)
				}
			}
		}

		// Override the version in the response object to show the latest one
		subResponse := sub
		subResponse.BasketVersion = targetVersion

		response = append(response, SubscriptionResponse{
			BasketSubscription: subResponse,
			InitialPrice:       initialPrice,
			CurrentPrice:       currentPrice,
		})
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Subscriptions fetched!", fiber.Map{
		"subscriptions": response,
		"pagination": fiber.Map{
			"total": total,
			"page":  *reqData.Page,
			"limit": *reqData.Limit,
		},
	})
}

// GetPublishedHistory returns published version history for users (limited view)
func GetPublishedHistory(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	basketId := c.Params("id")

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	db := database.Database.Db

	// Get only published/expired versions (user view)
	var versions []basket.BasketVersion
	if err := db.Where("basket_id = ? AND status IN ? AND is_deleted = false", basketId, []string{basket.StatusPublished, basket.StatusExpired, basket.StatusUnpublished}).
		Preload("Stocks", "is_deleted = false").
		Order("version_number DESC").
		Find(&versions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch history!", nil)
	}

	// Calculate pricing for each version
	type HistoryResponse struct {
		basket.BasketVersion
		InitialPrice  float64 `json:"initialPrice"`
		AchievedPrice float64 `json:"achievedPrice"`
	}

	var response []HistoryResponse

	// Get Bajaj Token for live price
	var bajajToken models.BajajAccessToken
	db.Order("created_at DESC").First(&bajajToken)
	accessToken := bajajToken.Token

	for _, v := range versions {
		var initialPrice float64 = v.PriceAtApproval
		// Fallback Initial Price
		if initialPrice == 0 {
			for _, s := range v.Stocks {
				initialPrice += s.PriceAtCreation * float64(s.Quantity)
			}
		}

		var achievedPrice float64 = 0

		// If EXPIRED, use PriceAtExpiry (or fallback to live/creation)
		if v.Status == basket.StatusExpired {
			if v.PriceAtExpiry > 0 {
				achievedPrice = v.PriceAtExpiry
			} else {
				// Legacy data: fallback to current Live Price (Best Effort)
				for _, stock := range v.Stocks {
					if accessToken != "" && stock.Token > 0 {
						if livePrice, err := utils.GetBajajQuote(accessToken, stock.Token); err == nil && livePrice > 0 {
							achievedPrice += livePrice * float64(stock.Quantity)
							continue
						}
					}
					// Fallback to creation/approval if live fails
					if stock.PriceAtApproval > 0 {
						achievedPrice += stock.PriceAtApproval * float64(stock.Quantity)
					} else {
						achievedPrice += stock.PriceAtCreation * float64(stock.Quantity)
					}
				}
			}
		} else {
			// PUBLISHED / SCHEDULED (Current): Use Live Price
			for _, stock := range v.Stocks {
				if accessToken != "" && stock.Token > 0 {
					if livePrice, err := utils.GetBajajQuote(accessToken, stock.Token); err == nil && livePrice > 0 {
						achievedPrice += livePrice * float64(stock.Quantity)
						continue
					}
				}
				// Fallback
				if stock.PriceAtApproval > 0 {
					achievedPrice += stock.PriceAtApproval * float64(stock.Quantity)
				} else {
					achievedPrice += stock.PriceAtCreation * float64(stock.Quantity)
				}
			}
		}

		response = append(response, HistoryResponse{
			BasketVersion: v,
			InitialPrice:  initialPrice,
			AchievedPrice: achievedPrice,
		})
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Published history fetched!", response)
}
