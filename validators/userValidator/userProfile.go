package userValidator

import (
	"fib/middleware"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func isValidNumeric(input string) bool {
	_, err := strconv.Atoi(input)
	return err == nil
}

func isValidIFSC(ifsc string) bool {
	re := regexp.MustCompile(`^[A-Za-z]{4}0[A-Za-z0-9]{6}$`)
	return re.MatchString(ifsc)
}

func AddBankAccount() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse request body
		reqData := new(struct {
			BankName    string `json:"bankName"`
			AccountNo   string `json:"accountNo"`
			HolderName  string `json:"holderName"`
			IFSCCode    string `json:"ifscCode"`
			BranchName  string `json:"branchName"`
			AccountType string `json:"accountType"` // Optional
		})
		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Validate Bank Name
		if len(strings.TrimSpace(reqData.BankName)) < 3 {
			errors["bankName"] = "Bank name must be at least 3 characters long!"
		}

		// Validate Account Number
		if len(strings.TrimSpace(reqData.AccountNo)) < 10 || len(reqData.AccountNo) > 18 {
			errors["accountNo"] = "Account number must be between 10 and 18 digits!"
		} else if !isValidNumeric(reqData.AccountNo) {
			errors["accountNo"] = "Account number must contain only numeric characters!"
		}

		// Validate Holder Name
		if len(strings.TrimSpace(reqData.HolderName)) < 3 {
			errors["holderName"] = "Holder name must be at least 3 characters long!"
		}

		// Validate IFSC Code
		if len(strings.TrimSpace(reqData.IFSCCode)) != 11 || !isValidIFSC(reqData.IFSCCode) {
			errors["ifscCode"] = "Invalid IFSC code! It must be 11 characters long and alphanumeric."
		}

		// Validate Branch Name (Optional but must not be empty if provided)
		if reqData.BranchName != "" && len(strings.TrimSpace(reqData.BranchName)) < 3 {
			errors["branchName"] = "Branch name must be at least 3 characters long if provided!"
		}

		// Validate Account Type (Optional but must match valid values)
		validAccountTypes := map[string]bool{"savings": true, "current": true}
		if reqData.AccountType != "" && !validAccountTypes[strings.ToLower(reqData.AccountType)] {
			errors["accountType"] = "Account type must be 'savings' or 'current'!"
		}

		// Respond with errors if any exist
		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		// Pass validated bank details to the next middleware
		c.Locals("validatedBankDetails", reqData)
		return c.Next()
	}
}

func SendAdharOtp() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse request body
		reqData := new(struct {
			AadharNumber string `json:"aadharNumber"`
		})
		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Validate Aadhar number
		if len(strings.TrimSpace(reqData.AadharNumber)) != 12 {
			errors["adharNumber"] = "Invalid Aadhar number! It must be 12 digits long!"
		}

		// Respond with errors if any exist
		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		// Pass Validated Send adhar otp details to the next middleware
		c.Locals("validatedAdhar", reqData)
		return c.Next()
	}
}

func VerifyAdharOtp() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse request body into struct
		reqData := new(struct {
			ReferenceID string `json:"referenceId"` // Fixed naming convention
			Otp         string `json:"otp"`
		})
		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		// Initialize error map for validation errors
		errors := make(map[string]string)

		// Validate ReferenceID
		if strings.TrimSpace(reqData.ReferenceID) == "" {
			errors["referenceId"] = "Reference ID is required!"
		}

		// Validate OTP
		if strings.TrimSpace(reqData.Otp) == "" {
			errors["otp"] = "OTP is required!"
		}

		// If there are validation errors, respond with error details
		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		// Store validated data in context for further use
		c.Locals("verifyAdharOtp", reqData)
		return c.Next()
	}
}

func AddFolioNumber() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse request body
		reqData := new(struct {
			FolioNumber string `json:"folioNumber"`
		})
		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Validate Folio number
		if reqData.FolioNumber == "" {
			errors["folioNumber"] = "Not found Folio Number"
		}

		// Respond with errors if any exist
		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		// Pass Validated Folio Number to the next middleware
		c.Locals("validatedFolioNumber", reqData)
		return c.Next()
	}
}

func Deposit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse request body
		reqData := new(struct {
			Amount uint `json:"amount"`
		})
		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Validate Amount
		if reqData.Amount <= 0 {
			errors["amount"] = "Amount can't be zero !"
		}

		// Respond with errors if any exist
		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		// Pass validated Amount to the next middleware
		c.Locals("validatedAmount", reqData)
		return c.Next()
	}
}

func Withdraw() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse request body
		reqData := new(struct {
			Amount uint `json:"amount"`
		})
		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Validate Amount
		if reqData.Amount <= 0 {
			errors["amount"] = "Amount can't be zero !"
		}

		// Respond with errors if any exist
		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		// Pass validated Amount to the next middleware
		c.Locals("validatedAmount", reqData)
		return c.Next()
	}
}

func TransactionList() fiber.Handler {
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
		c.Locals("validatedTransactionList", reqData)
		return c.Next()
	}
}
