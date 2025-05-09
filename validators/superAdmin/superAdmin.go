package superAdminValidator

import (
	"fib/middleware"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Helper to validate email format
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// Helper to validate mobile number format
func isValidMobile(mobile string) bool {
	re := regexp.MustCompile(`^\d{10}$`)
	return re.MatchString(mobile)
}

func UserList() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page  *int `json:"page"`
			Limit *int `json:"limit"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body!",
				"errors":  nil,
			})
		}

		errors := make(map[string]string)

		// Validate Page
		if reqData.Page == nil || *reqData.Page < 1 {
			errors["page"] = "Page must be greater than 0!"
		}

		// Validate Limit
		if reqData.Limit == nil || *reqData.Limit < 1 {
			errors["limit"] = "Limit must be greater than 0!"
		}

		// Respond with validation errors if any exist
		if len(errors) > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Validation failed!",
				"errors":  errors,
			})
		}

		c.Locals("validateUserList", reqData)
		return c.Next()
	}
}

func RegisterAMC() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// user := new(models.User)
		reqData := new(struct {
			Mobile   string `json:"mobile"`
			Email    string `json:"email"`
			Password string `json:"password"`
			Name     string `json:"name"`
		})
		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Validate Name
		if len(strings.TrimSpace(reqData.Name)) < 5 {
			errors["name"] = "Name must be at least 5 characters long!"
		}

		// Validate Email
		if reqData.Email == "" || !isValidEmail(reqData.Email) {
			errors["email"] = "Invalid email!"
		}

		// Validate Mobile
		if reqData.Mobile == "" || !isValidMobile(reqData.Mobile) {
			errors["mobile"] = "Invalid mobile number!"
		}

		// Validate Password
		if len(strings.TrimSpace(reqData.Password)) < 8 {
			errors["password"] = "Password must be at least 8 characters long!"
		}

		// Respond with errors if any exist
		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		// Pass AMC user to the next middleware
		c.Locals("validatedAMC", reqData)
		return c.Next()
	}
}
