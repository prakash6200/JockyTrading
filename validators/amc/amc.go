package amcValidator

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

func isValidPAN(pan string) bool {
	match, _ := regexp.MatchString(`^[A-Z]{5}[0-9]{4}[A-Z]$`, pan)
	return match
}

func StockList() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page   *int    `json:"page"`
			Limit  *int    `json:"limit"`
			Search *string `json:"search"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request query!",
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

		// Return validation errors
		if len(errors) > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Validation failed!",
				"errors":  errors,
			})
		}

		// ✅ Set correct key to match the controller
		c.Locals("validatedStockList", reqData)
		return c.Next()
	}
}

func AmcPickUnpickStockValidator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			StockID    uint    `json:"stockId"`
			Action     string  `json:"action"`
			HoldingPer float32 `josn:"holdingPer"`
		})

		// Parse JSON body
		if err := c.BodyParser(reqData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body!",
				"errors":  nil,
			})
		}

		errors := make(map[string]string)

		// Validate StockID
		if reqData.StockID == 0 {
			errors["stockId"] = "Stock ID must be a positive number!"
		}

		// Validate Action
		validActions := map[string]bool{"pick": true, "unpick": true}
		if _, ok := validActions[reqData.Action]; !ok {
			errors["action"] = "Action must be either 'pick' or 'unpick'!"
		}

		if reqData.HoldingPer < 0 || reqData.HoldingPer > 100 {
			errors["holdingPer"] = "Holding percentage must be between 0 and 100!"
		}

		// Return errors if any
		if len(errors) > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Validation failed!",
				"errors":  errors,
			})
		}

		// Set the validated request in context
		c.Locals("validatedAmcPickUnpickStock", reqData)
		return c.Next()
	}
}

func AmcPerformance() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page  *int `json:"page"`
			Limit *int `json:"limit"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request query!",
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

		// Return validation errors
		if len(errors) > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Validation failed!",
				"errors":  errors,
			})
		}

		// ✅ Set correct key to match the controller
		c.Locals("validatedStockList", reqData)
		return c.Next()
	}
}

func AMCList() fiber.Handler {
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

// update amc
func Update() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Name                  *string  `json:"name"`
			Email                 *string  `json:"email"`
			Mobile                *string  `json:"mobile"`
			Password              *string  `json:"password"`
			PanNumber             *string  `json:"panNumber"`
			Address               *string  `json:"address"`
			City                  *string  `json:"city"`
			State                 *string  `json:"state"`
			PinCode               *string  `json:"pinCode"`
			ContactPersonName     *string  `json:"contactPersonName"`
			ContactPerDesignation *string  `json:"contactPerDesignation"`
			FundName              *string  `json:"fundName"`
			EquityPer             *float32 `json:"equityPer"`
			DebtPer               *float32 `json:"debtPer"`
			CashSplit             *float32 `json:"cashSplit"`
			IsDeleted             *bool    `json:"isDeleted"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Validate email if present
		if reqData.Email != nil && !isValidEmail(*reqData.Email) {
			errors["email"] = "Invalid email!"
		}

		// Validate mobile if present
		if reqData.Mobile != nil && !isValidMobile(*reqData.Mobile) {
			errors["mobile"] = "Invalid mobile number!"
		}

		// Validate password if present
		if reqData.Password != nil && len(strings.TrimSpace(*reqData.Password)) < 8 {
			errors["password"] = "Password must be at least 8 characters long!"
		}

		// Validate PAN if present
		if reqData.PanNumber != nil && !isValidPAN(*reqData.PanNumber) {
			errors["panNumber"] = "Invalid PAN number format!"
		}

		// Validate equity/debt/cash if all 3 are present
		if reqData.EquityPer != nil && reqData.DebtPer != nil && reqData.CashSplit != nil {
			total := *reqData.EquityPer + *reqData.DebtPer + *reqData.CashSplit
			if *reqData.EquityPer < 0 || *reqData.EquityPer > 100 {
				errors["equityPer"] = "Equity must be between 0 and 100!"
			}
			if *reqData.DebtPer < 0 || *reqData.DebtPer > 100 {
				errors["debtPer"] = "Debt must be between 0 and 100!"
			}
			if *reqData.CashSplit < 0 || *reqData.CashSplit > 100 {
				errors["cashSplit"] = "Cash must be between 0 and 100!"
			}
			if int(total) != 100 {
				errors["totalSplit"] = "Sum of Equity, Debt, and Cash must be 100!"
			}
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedAMCUpdate", reqData)
		return c.Next()
	}
}
