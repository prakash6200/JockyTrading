package basketController

import (
	"encoding/json"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fib/models/basket"
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

	// Update version
	now := time.Now()
	version.ApprovedAt = &now
	version.ApprovedBy = &userId

	if err := db.Save(&version).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to approve basket!", nil)
	}

	// Unpublish any previously published version of the same basket
	db.Model(&basket.BasketVersion{}).
		Where("basket_id = ? AND id != ? AND status = ?", version.BasketID, version.ID, basket.StatusPublished).
		Update("status", basket.StatusUnpublished)

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
