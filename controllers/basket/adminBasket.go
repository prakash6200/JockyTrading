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

// ListPendingApprovals lists all baskets pending approval for admin
func ListPendingApprovals(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin role required.", nil)
	}

	reqData, ok := c.Locals("validatedListPending").(*struct {
		Page       *int    `json:"page"`
		Limit      *int    `json:"limit"`
		BasketType *string `json:"basketType"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db
	offset := (*reqData.Page - 1) * (*reqData.Limit)

	query := db.Model(&basket.BasketVersion{}).
		Where("status = ? AND is_deleted = false", basket.StatusPendingApproval)

	// Join with basket to filter by type
	if reqData.BasketType != nil && *reqData.BasketType != "" {
		query = query.Joins("JOIN baskets ON baskets.id = basket_versions.basket_id").
			Where("baskets.basket_type = ?", *reqData.BasketType)
	}

	var total int64
	query.Count(&total)

	var versions []basket.BasketVersion
	if err := db.Model(&basket.BasketVersion{}).
		Where("status = ? AND is_deleted = false", basket.StatusPendingApproval).
		Preload("Basket").
		Preload("Stocks", "is_deleted = false").
		Offset(offset).Limit(*reqData.Limit).
		Order("submitted_at ASC").
		Find(&versions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch pending approvals!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Pending approvals fetched!", fiber.Map{
		"versions": versions,
		"pagination": fiber.Map{
			"total": total,
			"page":  *reqData.Page,
			"limit": *reqData.Limit,
		},
	})
}

// ApproveBasket approves a basket version (with time slot for INTRA_HOUR)
func ApproveBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedApproveBasket").(*struct {
		BasketVersionID uint       `json:"basketVersionId"`
		StartTime       *time.Time `json:"startTime"`
		EndTime         *time.Time `json:"endTime"`
		ScheduledDate   *time.Time `json:"scheduledDate"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Find the version
	var version basket.BasketVersion
	if err := db.Preload("Basket").Where("id = ? AND status = ? AND is_deleted = false", reqData.BasketVersionID, basket.StatusPendingApproval).First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket version not found or not pending approval!", nil)
	}

	// For INTRA_HOUR baskets, time slot is required
	if version.Basket.BasketType == basket.BasketTypeIntraHour {
		if reqData.StartTime == nil || reqData.EndTime == nil || reqData.ScheduledDate == nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Time slot (startTime, endTime, scheduledDate) is required for INTRA_HOUR baskets!", nil)
		}

		// Create time slot
		duration := int(reqData.EndTime.Sub(*reqData.StartTime).Minutes())
		timeSlot := basket.BasketTimeSlot{
			BasketVersionID: version.ID,
			ScheduledDate:   *reqData.ScheduledDate,
			StartTime:       *reqData.StartTime,
			EndTime:         *reqData.EndTime,
			DurationMinutes: duration,
			Timezone:        "Asia/Kolkata",
			SetByAdminID:    userId,
		}

		if err := db.Create(&timeSlot).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create time slot!", nil)
		}

		// Set status to SCHEDULED (will auto-publish at start time)
		version.Status = basket.StatusScheduled

		// Record time slot history
		metadata, _ := json.Marshal(map[string]interface{}{
			"startTime":     reqData.StartTime,
			"endTime":       reqData.EndTime,
			"scheduledDate": reqData.ScheduledDate,
		})
		recordAdminHistory(version.ID, basket.ActionTimeSlotSet, userId, "Time slot set by admin", metadata)
	} else if version.Basket.BasketType == basket.BasketTypeIntraday {
		// INTRADAY: Set to PUBLISHED immediately, will auto-expire at market close
		version.Status = basket.StatusPublished

		// Set trading date to today if not specified
		today := time.Now()
		version.TradingDate = &today
	} else {
		// DELIVERY: Set to PUBLISHED immediately, no auto-expiry
		version.Status = basket.StatusPublished
	}

	// Calculate Initial Pricing at Approval Time
	var bajajToken models.BajajAccessToken
	db.Order("created_at DESC").First(&bajajToken)
	accessToken := bajajToken.Token

	var stocks []basket.BasketStock
	db.Where("basket_version_id = ? AND is_deleted = false", version.ID).Find(&stocks)

	var totalInitialValuation float64 = 0

	for _, stock := range stocks {
		price := stock.PriceAtCreation // Fallback

		// Auto-heal: If Token is missing, fetch from master
		if stock.Token == 0 {
			var masterStock models.Stocks
			if err := db.Where("id = ?", stock.StockID).First(&masterStock).Error; err == nil {
				stock.Token = masterStock.Token
				stock.Symbol = masterStock.Symbol
				// Save healed data
				db.Model(&stock).Select("Token", "Symbol").Updates(basket.BasketStock{Token: masterStock.Token, Symbol: masterStock.Symbol})
			}
		}

		// Try to fetch live price if token is available
		if accessToken != "" && stock.Token > 0 {
			livePrice, err := utils.GetBajajQuote(accessToken, stock.Token)
			if err == nil && livePrice > 0 {
				price = livePrice
			} else {
				// Log error but continue with creation price
				// log.Printf("Failed to fetch price for token %d: %v", stock.Token, err)
			}
		}

		// Update stock with approval price
		db.Model(&stock).Update("price_at_approval", price)

		totalInitialValuation += price * float64(stock.Quantity)
	}

	// Update version
	now := time.Now()
	version.ApprovedAt = &now
	version.ApprovedBy = &userId
	version.PriceAtApproval = totalInitialValuation

	if err := db.Save(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to approve basket!", nil)
	}

	// Unpublish/expire any previously published or scheduled version of the same basket
	db.Model(&basket.BasketVersion{}).
		Where("basket_id = ? AND id != ? AND status IN ?", version.BasketID, version.ID, []string{basket.StatusPublished, basket.StatusScheduled}).
		Update("status", basket.StatusExpired)

	// Update basket's current version
	db.Model(&basket.Basket{}).Where("id = ?", version.BasketID).Update("current_version_id", version.ID)

	// Record history
	recordAdminHistory(version.ID, basket.ActionApproved, userId, "Basket approved by admin", nil)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket approved successfully!", version)
}

// RejectBasket rejects a basket version
func RejectBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedRejectBasket").(*struct {
		BasketVersionID uint   `json:"basketVersionId"`
		Reason          string `json:"reason"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Find the version
	var version basket.BasketVersion
	if err := db.Where("id = ? AND status = ? AND is_deleted = false", reqData.BasketVersionID, basket.StatusPendingApproval).First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket version not found or not pending approval!", nil)
	}

	// Update version
	version.Status = basket.StatusRejected
	version.RejectionReason = reqData.Reason

	if err := db.Save(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to reject basket!", nil)
	}

	// Record history
	metadata, _ := json.Marshal(map[string]interface{}{"reason": reqData.Reason})
	recordAdminHistory(version.ID, basket.ActionRejected, userId, reqData.Reason, metadata)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket rejected!", version)
}

// SetTimeSlot sets or updates time slot for INTRA_HOUR basket
func SetTimeSlot(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedSetTimeSlot").(*struct {
		BasketVersionID uint      `json:"basketVersionId"`
		ScheduledDate   time.Time `json:"scheduledDate"`
		StartTime       time.Time `json:"startTime"`
		EndTime         time.Time `json:"endTime"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Find version and verify it's INTRA_HOUR
	var version basket.BasketVersion
	if err := db.Preload("Basket").Where("id = ? AND is_deleted = false", reqData.BasketVersionID).First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket version not found!", nil)
	}

	if version.Basket.BasketType != basket.BasketTypeIntraHour {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Time slots are only for INTRA_HOUR baskets!", nil)
	}

	if version.Status != basket.StatusApproved && version.Status != basket.StatusScheduled {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Can only set time slot for APPROVED or SCHEDULED baskets!", nil)
	}

	// Check for existing time slot
	var existingSlot basket.BasketTimeSlot
	if err := db.Where("basket_version_id = ?", version.ID).First(&existingSlot).Error; err == nil {
		// Update existing slot
		existingSlot.ScheduledDate = reqData.ScheduledDate
		existingSlot.StartTime = reqData.StartTime
		existingSlot.EndTime = reqData.EndTime
		existingSlot.DurationMinutes = int(reqData.EndTime.Sub(reqData.StartTime).Minutes())
		db.Save(&existingSlot)

		return middleware.JsonResponse(c, fiber.StatusOK, true, "Time slot updated!", existingSlot)
	}

	// Create new time slot
	duration := int(reqData.EndTime.Sub(reqData.StartTime).Minutes())
	timeSlot := basket.BasketTimeSlot{
		BasketVersionID: version.ID,
		ScheduledDate:   reqData.ScheduledDate,
		StartTime:       reqData.StartTime,
		EndTime:         reqData.EndTime,
		DurationMinutes: duration,
		Timezone:        "Asia/Kolkata",
		SetByAdminID:    userId,
	}

	if err := db.Create(&timeSlot).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create time slot!", nil)
	}

	// Update version status to SCHEDULED
	version.Status = basket.StatusScheduled
	db.Save(&version)

	// Record history
	metadata, _ := json.Marshal(map[string]interface{}{
		"startTime":     reqData.StartTime,
		"endTime":       reqData.EndTime,
		"scheduledDate": reqData.ScheduledDate,
	})
	recordAdminHistory(version.ID, basket.ActionTimeSlotSet, userId, "Time slot set by admin", metadata)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Time slot set successfully!", timeSlot)
}

// GetCalendarView returns scheduled INTRA_HOUR baskets in calendar format
func GetCalendarView(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedCalendarView").(*struct {
		StartDate *time.Time `json:"startDate"`
		EndDate   *time.Time `json:"endDate"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	query := db.Model(&basket.BasketTimeSlot{})

	if reqData.StartDate != nil {
		query = query.Where("scheduled_date >= ?", *reqData.StartDate)
	}
	if reqData.EndDate != nil {
		query = query.Where("scheduled_date <= ?", *reqData.EndDate)
	}

	var timeSlots []basket.BasketTimeSlot
	if err := query.Preload("BasketVersion.Basket").
		Order("scheduled_date ASC, start_time ASC").
		Find(&timeSlots).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch calendar!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Calendar data fetched!", timeSlots)
}

// GetAuditLog returns full audit trail for a basket
func GetAuditLog(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	basketId := c.Params("id")

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	db := database.Database.Db

	// Get all versions for this basket
	var versions []basket.BasketVersion
	if err := db.Where("basket_id = ?", basketId).Find(&versions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	var versionIds []uint
	for _, v := range versions {
		versionIds = append(versionIds, v.ID)
	}

	// Get all history entries
	var history []basket.BasketHistory
	if err := db.Where("basket_version_id IN ?", versionIds).
		Order("created_at DESC").
		Find(&history).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch audit log!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Audit log fetched!", history)
}

// ListAllBaskets lists all baskets for admin with filtering
func ListAllBaskets(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	log.Println("ListAllBaskets called by admin:", userId)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin role required.", nil)
	}

	// Parse query params
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	status := c.Query("status")         // DRAFT, PENDING_APPROVAL, PUBLISHED, SCHEDULED, EXPIRED, REJECTED
	basketType := c.Query("basketType") // INTRA_HOUR, INTRADAY, DELIVERY
	amcId := c.QueryInt("amcId", 0)
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	db := database.Database.Db

	query := db.Model(&basket.Basket{}).Where("is_deleted = false")

	// Apply filters
	if basketType != "" {
		query = query.Where("basket_type = ?", basketType)
	}
	if amcId > 0 {
		query = query.Where("amc_id = ?", amcId)
	}
	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var baskets []basket.Basket
	// Explicitly using simplified Preload to avoid any function closures issues
	if err := query.
		Preload("Versions").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&baskets).Error; err != nil {
		log.Printf("Error fetching baskets: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch baskets!", nil)
	}

	// If status filter, filter by version status
	if status != "" {
		var filteredBaskets []basket.Basket
		for _, b := range baskets {
			for _, v := range b.Versions {
				if v.Status == status {
					filteredBaskets = append(filteredBaskets, b)
					break
				}
			}
		}
		baskets = filteredBaskets
	}

	// Get AMC details for each basket
	type BasketWithAMC struct {
		basket.Basket
		AMCName  string `json:"amcName"`
		AMCEmail string `json:"amcEmail"`
	}

	var result []BasketWithAMC
	for _, b := range baskets {
		var amc models.User
		db.Where("id = ?", b.AMCID).First(&amc)
		result = append(result, BasketWithAMC{
			Basket:   b,
			AMCName:  amc.Name,
			AMCEmail: amc.Email,
		})
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Baskets fetched!", fiber.Map{
		"baskets": result,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetDashboardStats returns basket statistics for admin dashboard
func GetDashboardStats(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin role required.", nil)
	}

	db := database.Database.Db

	// Basket counts by status
	var totalBaskets int64
	var draftCount, pendingCount, publishedCount, scheduledCount, expiredCount, rejectedCount int64

	db.Model(&basket.Basket{}).Where("is_deleted = false").Count(&totalBaskets)

	db.Model(&basket.BasketVersion{}).Where("status = ? AND is_deleted = false", basket.StatusDraft).Count(&draftCount)
	db.Model(&basket.BasketVersion{}).Where("status = ? AND is_deleted = false", basket.StatusPendingApproval).Count(&pendingCount)
	db.Model(&basket.BasketVersion{}).Where("status = ? AND is_deleted = false", basket.StatusPublished).Count(&publishedCount)
	db.Model(&basket.BasketVersion{}).Where("status = ? AND is_deleted = false", basket.StatusScheduled).Count(&scheduledCount)
	db.Model(&basket.BasketVersion{}).Where("status = ? AND is_deleted = false", basket.StatusExpired).Count(&expiredCount)
	db.Model(&basket.BasketVersion{}).Where("status = ? AND is_deleted = false", basket.StatusRejected).Count(&rejectedCount)

	// Basket counts by type
	var intraHourCount, intradayCount, deliveryCount int64
	db.Model(&basket.Basket{}).Where("basket_type = ? AND is_deleted = false", basket.BasketTypeIntraHour).Count(&intraHourCount)
	db.Model(&basket.Basket{}).Where("basket_type = ? AND is_deleted = false", basket.BasketTypeIntraday).Count(&intradayCount)
	db.Model(&basket.Basket{}).Where("basket_type = ? AND is_deleted = false", basket.BasketTypeDelivery).Count(&deliveryCount)

	// Subscription stats
	var totalSubscriptions, activeSubscriptions int64
	db.Model(&basket.BasketSubscription{}).Where("is_deleted = false").Count(&totalSubscriptions)
	db.Model(&basket.BasketSubscription{}).Where("status = ? AND is_deleted = false", basket.SubscriptionActive).Count(&activeSubscriptions)

	// Today's stats
	today := time.Now().Truncate(24 * time.Hour)
	var todaySubscriptions int64
	db.Model(&basket.BasketSubscription{}).Where("subscribed_at >= ? AND is_deleted = false", today).Count(&todaySubscriptions)

	// Total revenue
	var totalRevenue float64
	db.Model(&basket.BasketSubscription{}).Where("is_deleted = false").Select("COALESCE(SUM(subscription_price), 0)").Scan(&totalRevenue)

	// AMC count
	var amcCount int64
	db.Model(&models.User{}).Where("role = ? AND is_deleted = false", "AMC").Count(&amcCount)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Dashboard stats fetched!", fiber.Map{
		"baskets": fiber.Map{
			"total": totalBaskets,
			"byStatus": fiber.Map{
				"draft":           draftCount,
				"pendingApproval": pendingCount,
				"published":       publishedCount,
				"scheduled":       scheduledCount,
				"expired":         expiredCount,
				"rejected":        rejectedCount,
			},
			"byType": fiber.Map{
				"intraHour": intraHourCount,
				"intraday":  intradayCount,
				"delivery":  deliveryCount,
			},
		},
		"subscriptions": fiber.Map{
			"total":        totalSubscriptions,
			"active":       activeSubscriptions,
			"today":        todaySubscriptions,
			"totalRevenue": totalRevenue,
		},
		"amcCount": amcCount,
	})
}

// GetAllSubscribers returns all subscribers across all baskets
func GetAllSubscribers(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin role required.", nil)
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	status := c.Query("status")
	basketId := c.QueryInt("basketId", 0)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	db := database.Database.Db

	query := db.Model(&basket.BasketSubscription{}).Where("is_deleted = false")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if basketId > 0 {
		query = query.Where("basket_id = ?", basketId)
	}

	var total int64
	query.Count(&total)

	var subscriptions []basket.BasketSubscription
	if err := query.
		Preload("Basket").
		Preload("BasketVersion").
		Order("subscribed_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&subscriptions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch subscribers!", nil)
	}

	// Add user details
	type SubscriberDetail struct {
		basket.BasketSubscription
		UserName   string `json:"userName"`
		UserEmail  string `json:"userEmail"`
		UserPhone  string `json:"userPhone"`
		BasketName string `json:"basketName"`
	}

	var result []SubscriberDetail
	for _, sub := range subscriptions {
		var subUser models.User
		db.Where("id = ?", sub.UserID).First(&subUser)

		result = append(result, SubscriberDetail{
			BasketSubscription: sub,
			UserName:           subUser.Name,
			UserEmail:          subUser.Email,
			UserPhone:          subUser.Mobile,
			BasketName:         sub.Basket.Name,
		})
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Subscribers fetched!", fiber.Map{
		"subscribers": result,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// UnpublishBasket unpublishes a basket version (admin only)
func UnpublishBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	versionId := c.QueryInt("versionId", 0)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin role required.", nil)
	}

	if versionId == 0 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "versionId is required!", nil)
	}

	db := database.Database.Db

	var version basket.BasketVersion
	if err := db.Where("id = ? AND is_deleted = false", versionId).First(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Version not found!", nil)
	}

	if version.Status != basket.StatusPublished && version.Status != basket.StatusScheduled {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Can only unpublish PUBLISHED or SCHEDULED versions!", nil)
	}

	version.Status = basket.StatusExpired
	db.Save(&version)

	recordAdminHistory(version.ID, "UNPUBLISHED", userId, "Basket unpublished by admin", nil)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket unpublished successfully!", version)
}

// AdminDeleteBasket deletes a basket (soft delete)
func AdminDeleteBasket(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	basketId := c.QueryInt("basketId", 0)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin role required.", nil)
	}

	if basketId == 0 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "basketId is required!", nil)
	}

	db := database.Database.Db

	var existingBasket basket.Basket
	if err := db.Where("id = ? AND is_deleted = false", basketId).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	// Check for active subscriptions
	var activeSubCount int64
	db.Model(&basket.BasketSubscription{}).Where("basket_id = ? AND status = ? AND is_deleted = false", basketId, basket.SubscriptionActive).Count(&activeSubCount)

	if activeSubCount > 0 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Cannot delete basket with active subscriptions!", nil)
	}

	// Soft delete basket and all versions
	db.Model(&basket.Basket{}).Where("id = ?", basketId).Update("is_deleted", true)
	db.Model(&basket.BasketVersion{}).Where("basket_id = ?", basketId).Update("is_deleted", true)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Basket deleted successfully!", nil)
}

// GetBasketSubscribersAdmin returns all subscribers for a specific basket (admin endpoint)
func GetBasketSubscribersAdmin(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	basketId := c.Params("id")

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin role required.", nil)
	}

	db := database.Database.Db

	// Check basket exists
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND is_deleted = false", basketId).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	status := c.Query("status")

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

	var subscriptions []basket.BasketSubscription
	if err := query.
		Preload("BasketVersion").
		Order("subscribed_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&subscriptions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch subscribers!", nil)
	}

	// Add user details
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

// Helper function to record admin history
func recordAdminHistory(versionId uint, action string, actorId uint, comments string, metadata []byte) {
	history := basket.BasketHistory{
		BasketVersionID: versionId,
		Action:          action,
		ActorID:         actorId,
		ActorType:       basket.ActorAdmin,
		Comments:        comments,
		Metadata:        string(metadata),
	}

	database.Database.Db.Create(&history)
}
