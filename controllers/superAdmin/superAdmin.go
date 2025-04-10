package superAdminController

import (
	"fib/database"
	"fib/middleware"
	"fib/models"

	"github.com/gofiber/fiber/v2"
)

func UserList(c *fiber.Ctx) error {
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
		Where("is_deleted = ? AND role != ?", false, "SUPER-ADMIN").
		Offset(offset).
		Limit(*reqData.Limit).
		Find(&users).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch user list!", nil)
	}

	// Count total records
	database.Database.Db.
		Model(&models.User{}).
		Where("is_deleted = ? AND role != ?", false, "SUPER-ADMIN").
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

	return middleware.JsonResponse(c, fiber.StatusOK, true, "User List.", response)
}
