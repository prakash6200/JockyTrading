package basketController

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fib/models/basket"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SubmitReview allows a user to submit a review for a basket
func SubmitReview(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	basketId := c.Params("id")

	reqData := new(struct {
		Rating int    `json:"rating"`
		Review string `json:"review"`
	})
	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	if reqData.Rating < 1 || reqData.Rating > 5 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Rating must be between 1 and 5!", nil)
	}

	db := database.Database.Db

	// Check if basket exists
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND is_deleted = false", basketId).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
	}

	// Check if user has already reviewed this basket
	var existingReview basket.BasketReview
	if err := db.Where("basket_id = ? AND user_id = ? AND deleted_at IS NULL", basketId, userId).First(&existingReview).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "You have already reviewed this basket!", nil)
	}

	// Optional: Check if user has subscribed to the basket (uncomment if required)
	/*
		var subscription basket.BasketSubscription
		if err := db.Where("basket_id = ? AND user_id = ? AND status = ?", basketId, userId, basket.SubscriptionActive).First(&subscription).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusForbidden, false, "You must be subscribed to review this basket!", nil)
		}
	*/

	review := basket.BasketReview{
		BasketID: existingBasket.ID,
		UserID:   userId,
		Rating:   reqData.Rating,
		Review:   reqData.Review,
		Status:   basket.ReviewStatusPending, // Default to pending
	}

	if err := db.Create(&review).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to submit review!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Review submitted successfully! Pending approval.", review)
}

// GetPublicReviews returns approved reviews for a basket (Visible to all)
func GetPublicReviews(c *fiber.Ctx) error {
	basketId := c.Params("id")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	db := database.Database.Db

	var total int64
	db.Model(&basket.BasketReview{}).
		Where("basket_id = ? AND status = ? AND deleted_at IS NULL", basketId, basket.ReviewStatusApproved).
		Count(&total)

	var reviews []basket.BasketReview
	if err := db.Where("basket_id = ? AND status = ? AND deleted_at IS NULL", basketId, basket.ReviewStatusApproved).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name") // Only fetch name
		}).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch reviews!", nil)
	}

	// Mask user data manually if needed, distinct from Preload select
	type ReviewResponse struct {
		basket.BasketReview
		UserName string `json:"userName"`
	}

	var response []ReviewResponse
	for _, r := range reviews {
		response = append(response, ReviewResponse{
			BasketReview: r,
			UserName:     r.User.Name,
		})
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Reviews fetched!", fiber.Map{
		"reviews": response,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetAMCReviews returns all reviews for an AMC's baskets (AMC/Admin only)
func GetAMCReviews(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	// Check role
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	status := c.Query("status") // Optional filter
	basketId := c.QueryInt("basketId", 0)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	db := database.Database.Db

	query := db.Model(&basket.BasketReview{}).Preload("User").Where("deleted_at IS NULL")

	// If AMC, filter by their baskets
	if user.Role == "AMC" {
		// Get basket IDs owned by AMC
		var basketIds []uint
		db.Model(&basket.Basket{}).Where("amc_id = ?", userId).Pluck("id", &basketIds)
		query = query.Where("basket_id IN ?", basketIds)
	}

	if basketId > 0 {
		query = query.Where("basket_id = ?", basketId)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var reviews []basket.BasketReview
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch reviews!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Reviews fetched!", fiber.Map{
		"reviews": reviews,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// ModerateReview allows AMC/Admin to approve/reject a review
func ModerateReview(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	reqData := new(struct {
		ReviewID uint   `json:"reviewId"`
		Action   string `json:"action"` // APPROVE, REJECT
		Reply    string `json:"reply"`  // Optional reply
	})
	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	// Check role
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"AMC", "ADMIN", "SUPER-ADMIN"}).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	db := database.Database.Db

	var review basket.BasketReview
	if err := db.Where("id = ?", reqData.ReviewID).First(&review).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Review not found!", nil)
	}

	// If AMC, ensure they own the basket
	if user.Role == "AMC" {
		var b basket.Basket
		if err := db.Where("id = ?", review.BasketID).First(&b).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found!", nil)
		}
		if b.AMCID != userId {
			return middleware.JsonResponse(c, fiber.StatusForbidden, false, "You can only moderate reviews for your baskets!", nil)
		}
	}

	if reqData.Action == "APPROVE" {
		review.Status = basket.ReviewStatusApproved
	} else if reqData.Action == "REJECT" {
		review.Status = basket.ReviewStatusRejected
	} else {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid action! Use APPROVE or REJECT.", nil)
	}

	if reqData.Reply != "" {
		review.Reply = reqData.Reply
		now := time.Now()
		review.RepliedAt = &now
	}

	db.Save(&review)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Review updated successfully!", review)
}
