package courseValidator

import (
	"fib/middleware"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func EnrollCourse() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("id"))
		if courseIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required!", nil)
		}

		// Validate CourseID is a valid integer
		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		c.Locals("courseID", courseID)
		return c.Next()
	}
}

func GetUserEnrollments() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page  *int `json:"page"`
			Limit *int `json:"limit"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid query parameters!", nil)
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

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedEnrollmentList", reqData)
		return c.Next()
	}
}
