package middleware

import (
	"fib/database"
	"fib/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// CheckPermissionMiddleware returns a middleware that checks if the user has the required permission
func CheckPermissionMiddleware(requiredPermission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user ID from context (set by your auth middleware)
		userID, ok := c.Locals("userId").(uint)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  false,
				"message": "Unauthorized: User ID not found",
				"data":    nil,
			})
		}

		var permission models.Permission
		err := database.Database.Db.Where("user_id = ? AND permission = ? AND is_deleted = false",
			userID, requiredPermission).First(&permission).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"status":  false,
					"message": "You do not have permission to access this resource!",
					"data":    nil,
				})
			}
			// Other DB error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  false,
				"message": "Server error while checking permissions!",
				"data":    nil,
			})
		}

		// Permission found, proceed
		return c.Next()
	}
}
