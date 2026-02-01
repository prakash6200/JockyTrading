package courseValidator

import (
	"fib/middleware"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func CreateCourseContent() fiber.Handler {
	return func(c *fiber.Ctx) error {

		courseID := strings.TrimSpace(c.Params("id"))
		if courseID == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required in the URL!", nil)
		}

		reqData := new(struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		// Normalize and sanitize inputs
		reqData.Title = strings.TrimSpace(reqData.Title)
		reqData.Description = strings.TrimSpace(reqData.Description)

		// Validate Title
		if reqData.Title == "" {
			errors["title"] = "Title is required!"
		} else {
			if len(reqData.Title) < 3 {
				errors["title"] = "Title must be at least 3 characters long!"
			}
			if len(reqData.Title) > 100 {
				errors["title"] = "Title must not exceed 100 characters!"
			}
			// Check for invalid characters (e.g., HTML tags)
			if matched, _ := regexp.MatchString(`[<>{}]`, reqData.Title); matched {
				errors["title"] = "Title contains invalid characters (e.g., <, >, {, })!"
			}
		}

		// Validate Description (optional field)
		if reqData.Description != "" {
			if len(reqData.Description) < 5 {
				errors["description"] = "Description must be at least 5 characters long if provided!"
			}
			if len(reqData.Description) > 1000 {
				errors["description"] = "Description must not exceed 1000 characters!"
			}
			// Check for invalid characters
			if matched, _ := regexp.MatchString(`[<>{}]`, reqData.Description); matched {
				errors["description"] = "Description contains invalid characters (e.g., <, >, {, })!"
			}
		}

		// Respond with validation errors if any exist
		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedCourseContent", reqData)
		c.Locals("courseID", courseID)

		return c.Next()
	}
}

func CourseContentList() fiber.Handler {
	return func(c *fiber.Ctx) error {

		courseID := strings.TrimSpace(c.Params("id"))
		if courseID == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required in the URL!", nil)
		}

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

		// Set defaults if not provided
		defaultPage := 1
		defaultLimit := 10
		if reqData.Page == nil || *reqData.Page < 1 {
			reqData.Page = &defaultPage
		}
		if reqData.Limit == nil || *reqData.Limit < 1 {
			reqData.Limit = &defaultLimit
		}

		c.Locals("validatedCourseContentList", reqData)
		c.Locals("courseID", courseID)
		return c.Next()
	}
}

func MarkContentComplete() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("course_id"))
		contentIDStr := strings.TrimSpace(c.Params("content_id"))

		// Validate CourseID
		if courseIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required!", nil)
		}
		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		// Validate CourseContentID
		if contentIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Content ID is required!", nil)
		}
		contentID, err := strconv.Atoi(contentIDStr)
		if err != nil || contentID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Content ID!", nil)
		}

		c.Locals("courseID", courseID)
		c.Locals("contentID", contentID)
		return c.Next()
	}
}

func GetContentCompletions() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("course_id"))

		// Validate CourseID
		if courseIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required!", nil)
		}
		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

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

		c.Locals("courseID", courseID)
		c.Locals("validatedCompletionList", reqData)
		return c.Next()
	}
}

// GetCourseDetail validates course detail request
func GetCourseDetail() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("id"))
		if courseIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		c.Locals("courseID", courseID)
		return c.Next()
	}
}

// GetDayContent validates day content request
func GetDayContent() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("course_id"))
		moduleIDStr := strings.TrimSpace(c.Params("module_id"))
		dayStr := strings.TrimSpace(c.Params("day"))

		if courseIDStr == "" || moduleIDStr == "" || dayStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID, Module ID, and Day are required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		moduleID, err := strconv.Atoi(moduleIDStr)
		if err != nil || moduleID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Module ID!", nil)
		}

		day, err := strconv.Atoi(dayStr)
		if err != nil || day < 1 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Day number!", nil)
		}

		c.Locals("courseID", courseID)
		c.Locals("moduleID", moduleID)
		c.Locals("day", day)
		return c.Next()
	}
}

// SubmitMCQ validates MCQ submission request
func SubmitMCQ() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("course_id"))
		contentIDStr := strings.TrimSpace(c.Params("content_id"))

		if courseIDStr == "" || contentIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID and Content ID are required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		contentID, err := strconv.Atoi(contentIDStr)
		if err != nil || contentID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Content ID!", nil)
		}

		c.Locals("courseID", courseID)
		c.Locals("contentID", contentID)
		return c.Next()
	}
}

// GetCourseProgress validates progress request
func GetCourseProgress() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("course_id"))
		if courseIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		c.Locals("courseID", courseID)
		return c.Next()
	}
}

// RequestCertificateValidator validates certificate request
func RequestCertificateValidator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("course_id"))
		if courseIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		c.Locals("courseID", courseID)
		return c.Next()
	}
}
