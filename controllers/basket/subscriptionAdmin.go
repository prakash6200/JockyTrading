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

// GetAllActiveSubscriptions returns all active subscriptions for admin
func GetAllActiveSubscriptions(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	// Verify admin role
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Admin access required!", nil)
	}

	// Parse query params
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	status := c.Query("status", basket.SubscriptionActive)
	offset := (page - 1) * limit

	db := database.Database.Db

	// Auto-expire any expired subscriptions first
	now := time.Now()
	db.Model(&basket.BasketSubscription{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?", basket.SubscriptionActive, now).
		Updates(map[string]interface{}{"status": basket.SubscriptionExpired})

	// Query subscriptions
	query := db.Model(&basket.BasketSubscription{}).Where("is_deleted = false")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var subscriptions []basket.BasketSubscription
	if err := query.
		Preload("Basket").
		Preload("BasketVersion").
		Offset(offset).Limit(limit).
		Order("expires_at ASC").
		Find(&subscriptions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch subscriptions!", nil)
	}

	// Add user details to response
	type SubscriptionWithUser struct {
		basket.BasketSubscription
		UserName  string `json:"userName"`
		UserEmail string `json:"userEmail"`
	}

	var response []SubscriptionWithUser
	for _, sub := range subscriptions {
		var subUser models.User
		db.Select("name, email").Where("id = ?", sub.UserID).First(&subUser)

		response = append(response, SubscriptionWithUser{
			BasketSubscription: sub,
			UserName:           subUser.Name,
			UserEmail:          subUser.Email,
		})
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Subscriptions fetched!", fiber.Map{
		"subscriptions": response,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetExpiringSubscriptions returns subscriptions expiring soon (for admin dashboard)
func GetExpiringSubscriptions(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	// Verify admin role
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Admin access required!", nil)
	}

	db := database.Database.Db
	now := time.Now()
	expiryWindow := now.AddDate(0, 0, 7) // Next 7 days

	var subscriptions []basket.BasketSubscription
	if err := db.
		Where("status = ? AND expires_at IS NOT NULL AND expires_at BETWEEN ? AND ?", basket.SubscriptionActive, now, expiryWindow).
		Preload("Basket").
		Order("expires_at ASC").
		Find(&subscriptions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch expiring subscriptions!", nil)
	}

	// Add user details
	type ExpiringSubscription struct {
		basket.BasketSubscription
		UserName     string `json:"userName"`
		UserEmail    string `json:"userEmail"`
		DaysToExpiry int    `json:"daysToExpiry"`
	}

	var response []ExpiringSubscription
	for _, sub := range subscriptions {
		var subUser models.User
		db.Select("name, email").Where("id = ?", sub.UserID).First(&subUser)

		daysToExpiry := 0
		if sub.ExpiresAt != nil {
			daysToExpiry = int(sub.ExpiresAt.Sub(now).Hours() / 24)
		}

		response = append(response, ExpiringSubscription{
			BasketSubscription: sub,
			UserName:           subUser.Name,
			UserEmail:          subUser.Email,
			DaysToExpiry:       daysToExpiry,
		})
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Expiring subscriptions fetched!", response)
}

// SendExpiryReminder sends a manual expiry reminder to a user
func SendExpiryReminder(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	// Verify admin role
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Admin access required!", nil)
	}

	// Parse request body
	reqData := new(struct {
		SubscriptionID uint `json:"subscriptionId"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	if reqData.SubscriptionID == 0 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Subscription ID is required!", nil)
	}

	db := database.Database.Db

	// Find subscription
	var subscription basket.BasketSubscription
	if err := db.Preload("Basket").Where("id = ?", reqData.SubscriptionID).First(&subscription).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Subscription not found!", nil)
	}

	// Get user details
	var subUser models.User
	if err := db.Where("id = ?", subscription.UserID).First(&subUser).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "User not found!", nil)
	}

	// Send reminder email
	utils.SendSubscriptionExpiryReminder(subUser.Email, subUser.Name, subscription.Basket.Name, subscription.ExpiresAt)

	// Mark reminder as sent
	db.Model(&subscription).Update("reminder_sent", true)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Reminder sent successfully!", nil)
}
