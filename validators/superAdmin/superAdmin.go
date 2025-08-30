package superAdminValidator

import (
	"fib/middleware"
	"regexp"
	"strconv"
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

// Simple semantic version check (e.g., 1.2.3)
func isValidSemVer(version string) bool {
	re := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	return re.MatchString(version)
}

func List() fiber.Handler {
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

		c.Locals("list", reqData)
		return c.Next()
	}
}

func RegisterAMC() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// user := new(models.User)
		reqData := new(struct {
			Mobile                string  `json:"mobile"`
			Email                 string  `json:"email"`
			Password              string  `json:"password"`
			Name                  string  `json:"name"`
			PanNumber             string  `json:"panNumber"`
			Address               string  `json:"address"`
			City                  string  `json:"city"`
			State                 string  `json:"state"`
			PinCode               string  `json:"pinCode"`
			ContactPersonName     string  `json:"contactPersonName"`
			ContactPerDesignation string  `json:"contactPerDesignation"`
			FundName              string  `json:"fundName"`
			EquityPer             float32 `json:"equityPer"`
			DebtPer               float32 `json:"debtPer"`
			CashSplit             float32 `json:"cashSplit"`
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

		// PAN Number
		if reqData.PanNumber == "" || !isValidPAN(reqData.PanNumber) {
			errors["panNumber"] = "Invalid PAN number format!"
		}

		// Address
		if len(strings.TrimSpace(reqData.Address)) < 5 {
			errors["address"] = "Address must be at least 5 characters long!"
		}

		// City
		if len(strings.TrimSpace(reqData.City)) < 2 {
			errors["city"] = "City must be at least 2 characters long!"
		}

		// State
		if len(strings.TrimSpace(reqData.State)) < 2 {
			errors["state"] = "State must be at least 2 characters long!"
		}

		// PinCode
		if reqData.PinCode == "" {
			errors["pinCode"] = "Invalid pin code!"
		}

		// Contact Person Name
		if len(strings.TrimSpace(reqData.ContactPersonName)) < 3 {
			errors["contactPersonName"] = "Contact person name must be at least 3 characters long!"
		}

		// Contact Person Designation
		if len(strings.TrimSpace(reqData.ContactPerDesignation)) < 2 {
			errors["contactPerDesignation"] = "Designation must be at least 2 characters long!"
		}

		// Fund Name
		if len(strings.TrimSpace(reqData.FundName)) < 2 {
			errors["fundName"] = "Fund Name must be at least 2 characters long!"
		}

		// ✅ Validate EquityPer, DebtPer, CashSplit
		total := reqData.EquityPer + reqData.DebtPer + reqData.CashSplit

		if reqData.EquityPer < 0 || reqData.EquityPer > 100 {
			errors["equityPer"] = "Equity percentage must be between 0 and 100!"
		}
		if reqData.DebtPer < 0 || reqData.DebtPer > 100 {
			errors["debtPer"] = "Debt percentage must be between 0 and 100!"
		}
		if reqData.CashSplit < 0 || reqData.CashSplit > 100 {
			errors["cashSplit"] = "Cash percentage must be between 0 and 100!"
		}
		if int(total) != 100 {
			errors["totalSplit"] = "Sum of Equity, Debt, and Cash must be exactly 100!"
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

func UpdateAMCValidator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			ID                    uint     `json:"id"`
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

		if reqData.ID == 0 {
			errors["id"] = "AMC ID is required!"
		}

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

func PermissionByUserID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIDParam := c.Query("userId")

		errors := make(map[string]string)

		// Check if userId is provided
		if userIDParam == "" {
			errors["userId"] = "userId is required!"
		} else {
			// Check if userId is a valid positive integer
			if id, err := strconv.Atoi(userIDParam); err != nil || id < 1 {
				errors["userId"] = "userId must be a valid positive number!"
			}
		}

		// Respond with validation errors if any exist
		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedUserId", userIDParam)
		return c.Next()
	}
}

func ValidateMaintenance() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			AppMaintenance       bool   `json:"app_maintenance"`
			ForceUpdate          bool   `json:"force_update"`
			IosLatestVersion     string `json:"ios_latest_version"`
			AndroidLatestVersion string `json:"android_latest_version"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Validate iOS version
		if strings.TrimSpace(reqData.IosLatestVersion) == "" {
			errors["ios_latest_version"] = "iOS version is required!"
		} else if !isValidSemVer(reqData.IosLatestVersion) {
			errors["ios_latest_version"] = "Invalid iOS version format! (expected x.y.z)"
		}

		// Validate Android version
		if strings.TrimSpace(reqData.AndroidLatestVersion) == "" {
			errors["android_latest_version"] = "Android version is required!"
		} else if !isValidSemVer(reqData.AndroidLatestVersion) {
			errors["android_latest_version"] = "Invalid Android version format! (expected x.y.z)"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		// Pass validated struct forward
		c.Locals("validatedMaintenance", reqData)
		return c.Next()
	}
}
