package walletValidator

import (
	"fib/middleware"

	"github.com/gofiber/fiber/v2"
)

// Deposit validates user deposit request
func Deposit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Amount           float64 `json:"amount"`
			PaymentGateway   string  `json:"paymentGateway"`
			PaymentOrderID   string  `json:"paymentOrderId"`
			PaymentID        string  `json:"paymentId"`
			PaymentSignature string  `json:"paymentSignature"`
			PaymentMethod    string  `json:"paymentMethod"`
			PaymentStatus    string  `json:"paymentStatus"`
			PaymentResponse  any     `json:"paymentResponse"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.Amount <= 0 {
			errors["amount"] = "Amount must be greater than 0!"
		}
		if reqData.PaymentGateway == "" {
			errors["paymentGateway"] = "Payment gateway is required!"
		}
		if reqData.PaymentID == "" {
			errors["paymentId"] = "Payment ID is required!"
		}
		if reqData.PaymentStatus == "" {
			errors["paymentStatus"] = "Payment status is required!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedDeposit", reqData)
		return c.Next()
	}
}

// AddBalance validates add balance request
func AddBalance() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			UserID uint    `json:"userId"`
			Amount float64 `json:"amount"`
			Reason string  `json:"reason"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.UserID == 0 {
			errors["userId"] = "User ID is required!"
		}
		if reqData.Amount <= 0 {
			errors["amount"] = "Amount must be greater than 0!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedAddBalance", reqData)
		return c.Next()
	}
}

// DeductBalance validates deduct balance request
func DeductBalance() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			UserID uint    `json:"userId"`
			Amount float64 `json:"amount"`
			Reason string  `json:"reason"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.UserID == 0 {
			errors["userId"] = "User ID is required!"
		}
		if reqData.Amount <= 0 {
			errors["amount"] = "Amount must be greater than 0!"
		}
		if reqData.Reason == "" {
			errors["reason"] = "Reason is required for deduction!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedDeductBalance", reqData)
		return c.Next()
	}
}
