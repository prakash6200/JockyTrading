package basketValidator

import (
	"fib/middleware"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ApproveBasket validates admin approval request
func ApproveBasket() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			BasketVersionID uint       `json:"basketVersionId"`
			StartTime       *time.Time `json:"startTime"`
			EndTime         *time.Time `json:"endTime"`
			ScheduledDate   *time.Time `json:"scheduledDate"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.BasketVersionID == 0 {
			errors["basketVersionId"] = "Basket version ID is required!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedApproveBasket", reqData)
		return c.Next()
	}
}

// RejectBasket validates admin rejection request
func RejectBasket() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			BasketVersionID uint   `json:"basketVersionId"`
			Reason          string `json:"reason"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.BasketVersionID == 0 {
			errors["basketVersionId"] = "Basket version ID is required!"
		}
		if reqData.Reason == "" {
			errors["reason"] = "Rejection reason is required!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedRejectBasket", reqData)
		return c.Next()
	}
}

// SetTimeSlot validates time slot setting for INTRA_HOUR baskets
func SetTimeSlot() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			BasketVersionID uint      `json:"basketVersionId"`
			ScheduledDate   time.Time `json:"scheduledDate"`
			StartTime       time.Time `json:"startTime"`
			EndTime         time.Time `json:"endTime"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		if reqData.BasketVersionID == 0 {
			errors["basketVersionId"] = "Basket version ID is required!"
		}
		if reqData.ScheduledDate.IsZero() {
			errors["scheduledDate"] = "Scheduled date is required!"
		}
		if reqData.StartTime.IsZero() {
			errors["startTime"] = "Start time is required!"
		}
		if reqData.EndTime.IsZero() {
			errors["endTime"] = "End time is required!"
		}
		if !reqData.StartTime.IsZero() && !reqData.EndTime.IsZero() {
			if reqData.EndTime.Before(reqData.StartTime) || reqData.EndTime.Equal(reqData.StartTime) {
				errors["endTime"] = "End time must be after start time!"
			}
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedSetTimeSlot", reqData)
		return c.Next()
	}
}

// ListPendingApprovals validates admin list pending request
func ListPendingApprovals() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page       *int    `json:"page"`
			Limit      *int    `json:"limit"`
			BasketType *string `json:"basketType"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request query!", nil)
		}

		errors := make(map[string]string)

		if reqData.Page == nil || *reqData.Page < 1 {
			errors["page"] = "Page must be greater than 0!"
		}
		if reqData.Limit == nil || *reqData.Limit < 1 {
			errors["limit"] = "Limit must be greater than 0!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedListPending", reqData)
		return c.Next()
	}
}

// GetCalendarView validates calendar view request
func GetCalendarView() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			StartDate *time.Time `json:"startDate"`
			EndDate   *time.Time `json:"endDate"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request query!", nil)
		}

		c.Locals("validatedCalendarView", reqData)
		return c.Next()
	}
}
