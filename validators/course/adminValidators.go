package courseValidator

import (
	"fib/middleware"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ============ Course Validators ============

// CreateCourseAdmin validates admin course creation request
func CreateCourseAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Title        string `json:"title"`
			Description  string `json:"description"`
			Author       string `json:"author"`
			Duration     int64  `json:"duration"`
			ThumbnailURL string `json:"thumbnail_url"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		reqData.Title = strings.TrimSpace(reqData.Title)
		reqData.Description = strings.TrimSpace(reqData.Description)
		reqData.Author = strings.TrimSpace(reqData.Author)

		if reqData.Title == "" {
			errors["title"] = "Title is required!"
		} else if len(reqData.Title) < 3 {
			errors["title"] = "Title must be at least 3 characters long!"
		}

		if reqData.Description == "" {
			errors["description"] = "Description is required!"
		} else if len(reqData.Description) < 5 {
			errors["description"] = "Description must be at least 5 characters long!"
		}

		if reqData.Author == "" {
			errors["author"] = "Author is required!"
		} else if len(reqData.Author) < 3 {
			errors["author"] = "Author must be at least 3 characters long!"
		} else if matched, _ := regexp.MatchString(`[<>{}]`, reqData.Author); matched {
			errors["author"] = "Author name contains invalid characters!"
		}

		if reqData.Duration <= 0 {
			errors["duration"] = "Duration must be a positive number!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("validatedCourse", reqData)
		return c.Next()
	}
}

// UpdateCourseAdmin validates admin course update request
func UpdateCourseAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("id"))
		if courseIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		reqData := new(struct {
			Title        string `json:"title"`
			Description  string `json:"description"`
			Author       string `json:"author"`
			Duration     int64  `json:"duration"`
			ThumbnailURL string `json:"thumbnail_url"`
			Status       string `json:"status"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		reqData.Title = strings.TrimSpace(reqData.Title)
		reqData.Description = strings.TrimSpace(reqData.Description)
		reqData.Author = strings.TrimSpace(reqData.Author)
		reqData.Status = strings.TrimSpace(reqData.Status)

		if reqData.Title != "" && len(reqData.Title) < 3 {
			errors["title"] = "Title must be at least 3 characters long!"
		}

		if reqData.Description != "" && len(reqData.Description) < 5 {
			errors["description"] = "Description must be at least 5 characters long!"
		}

		if reqData.Author != "" {
			if len(reqData.Author) < 3 {
				errors["author"] = "Author must be at least 3 characters long!"
			} else if matched, _ := regexp.MatchString(`[<>{}]`, reqData.Author); matched {
				errors["author"] = "Author name contains invalid characters!"
			}
		}

		if reqData.Status != "" {
			validStatuses := map[string]bool{"DRAFT": true, "ACTIVE": true, "INACTIVE": true}
			if !validStatuses[reqData.Status] {
				errors["status"] = "Status must be DRAFT, ACTIVE, or INACTIVE!"
			}
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("courseID", courseID)
		c.Locals("validatedCourseUpdate", reqData)
		return c.Next()
	}
}

// DeleteCourse validates course deletion request
func DeleteCourse() fiber.Handler {
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

// PublishCourse validates course publish/unpublish request
func PublishCourse() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("id"))
		if courseIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		reqData := new(struct {
			IsPublished bool `json:"is_published"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		c.Locals("courseID", courseID)
		c.Locals("publishStatus", reqData.IsPublished)
		return c.Next()
	}
}

// ============ Module Validators ============

// CreateModule validates module creation request
func CreateModule() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("id"))
		if courseIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		reqData := new(struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			OrderIndex  int    `json:"order_index"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		reqData.Title = strings.TrimSpace(reqData.Title)
		reqData.Description = strings.TrimSpace(reqData.Description)

		if reqData.Title == "" {
			errors["title"] = "Module title is required!"
		} else if len(reqData.Title) < 3 {
			errors["title"] = "Module title must be at least 3 characters long!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("courseID", courseID)
		c.Locals("validatedModule", reqData)
		return c.Next()
	}
}

// UpdateModule validates module update request
func UpdateModule() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("course_id"))
		moduleIDStr := strings.TrimSpace(c.Params("module_id"))

		if courseIDStr == "" || moduleIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID and Module ID are required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		moduleID, err := strconv.Atoi(moduleIDStr)
		if err != nil || moduleID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Module ID!", nil)
		}

		reqData := new(struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			OrderIndex  int    `json:"order_index"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)
		reqData.Title = strings.TrimSpace(reqData.Title)

		if reqData.Title != "" && len(reqData.Title) < 3 {
			errors["title"] = "Module title must be at least 3 characters long!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("courseID", courseID)
		c.Locals("moduleID", moduleID)
		c.Locals("validatedModuleUpdate", reqData)
		return c.Next()
	}
}

// DeleteModule validates module deletion request
func DeleteModule() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("course_id"))
		moduleIDStr := strings.TrimSpace(c.Params("module_id"))

		if courseIDStr == "" || moduleIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID and Module ID are required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		moduleID, err := strconv.Atoi(moduleIDStr)
		if err != nil || moduleID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Module ID!", nil)
		}

		c.Locals("courseID", courseID)
		c.Locals("moduleID", moduleID)
		return c.Next()
	}
}

// ListModules validates module listing request
func ListModules() fiber.Handler {
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

// ============ Content Validators ============

// CreateContentAdmin validates content creation request
func CreateContentAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("course_id"))
		moduleIDStr := strings.TrimSpace(c.Params("module_id"))

		if courseIDStr == "" || moduleIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID and Module ID are required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		moduleID, err := strconv.Atoi(moduleIDStr)
		if err != nil || moduleID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Module ID!", nil)
		}

		reqData := new(struct {
			Day         int    `json:"day"`
			Title       string `json:"title"`
			Description string `json:"description"`
			ContentType string `json:"content_type"`
			TextContent string `json:"text_content"`
			VideoURL    string `json:"video_url"`
			ImageURL    string `json:"image_url"`
			OrderIndex  int    `json:"order_index"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)

		reqData.Title = strings.TrimSpace(reqData.Title)
		reqData.Description = strings.TrimSpace(reqData.Description)
		reqData.ContentType = strings.ToUpper(strings.TrimSpace(reqData.ContentType))

		if reqData.Title == "" {
			errors["title"] = "Content title is required!"
		} else if len(reqData.Title) < 3 {
			errors["title"] = "Content title must be at least 3 characters long!"
		}

		if reqData.Day < 1 {
			errors["day"] = "Day must be at least 1!"
		}

		validContentTypes := map[string]bool{"TEXT": true, "MCQ": true, "VIDEO": true, "IMAGE": true}
		if reqData.ContentType == "" {
			errors["content_type"] = "Content type is required!"
		} else if !validContentTypes[reqData.ContentType] {
			errors["content_type"] = "Content type must be TEXT, MCQ, VIDEO, or IMAGE!"
		}

		// Validate based on content type
		switch reqData.ContentType {
		case "TEXT":
			if strings.TrimSpace(reqData.TextContent) == "" {
				errors["text_content"] = "Text content is required for TEXT type!"
			}
		case "VIDEO":
			if strings.TrimSpace(reqData.VideoURL) == "" {
				errors["video_url"] = "Video URL is required for VIDEO type!"
			}
		case "IMAGE":
			if strings.TrimSpace(reqData.ImageURL) == "" {
				errors["image_url"] = "Image URL is required for IMAGE type!"
			}
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("courseID", courseID)
		c.Locals("moduleID", moduleID)
		c.Locals("validatedContent", reqData)
		return c.Next()
	}
}

// UpdateContentAdmin validates content update request
func UpdateContentAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		contentIDStr := strings.TrimSpace(c.Params("content_id"))
		if contentIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Content ID is required!", nil)
		}

		contentID, err := strconv.Atoi(contentIDStr)
		if err != nil || contentID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Content ID!", nil)
		}

		reqData := new(struct {
			Day         int    `json:"day"`
			Title       string `json:"title"`
			Description string `json:"description"`
			ContentType string `json:"content_type"`
			TextContent string `json:"text_content"`
			VideoURL    string `json:"video_url"`
			ImageURL    string `json:"image_url"`
			OrderIndex  int    `json:"order_index"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)
		reqData.Title = strings.TrimSpace(reqData.Title)
		reqData.ContentType = strings.ToUpper(strings.TrimSpace(reqData.ContentType))

		if reqData.Title != "" && len(reqData.Title) < 3 {
			errors["title"] = "Content title must be at least 3 characters long!"
		}

		if reqData.ContentType != "" {
			validContentTypes := map[string]bool{"TEXT": true, "MCQ": true, "VIDEO": true, "IMAGE": true}
			if !validContentTypes[reqData.ContentType] {
				errors["content_type"] = "Content type must be TEXT, MCQ, VIDEO, or IMAGE!"
			}
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("contentID", contentID)
		c.Locals("validatedContentUpdate", reqData)
		return c.Next()
	}
}

// DeleteContentAdmin validates content deletion request
func DeleteContentAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		contentIDStr := strings.TrimSpace(c.Params("content_id"))
		if contentIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Content ID is required!", nil)
		}

		contentID, err := strconv.Atoi(contentIDStr)
		if err != nil || contentID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Content ID!", nil)
		}

		c.Locals("contentID", contentID)
		return c.Next()
	}
}

// PublishContentAdmin validates content publish/unpublish request
func PublishContentAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		contentIDStr := strings.TrimSpace(c.Params("content_id"))
		if contentIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Content ID is required!", nil)
		}

		contentID, err := strconv.Atoi(contentIDStr)
		if err != nil || contentID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Content ID!", nil)
		}

		reqData := new(struct {
			IsPublished bool `json:"is_published"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		c.Locals("contentID", contentID)
		c.Locals("publishStatus", reqData.IsPublished)
		return c.Next()
	}
}

// ============ MCQ Validators ============

// AddMCQOption validates MCQ option creation
func AddMCQOption() fiber.Handler {
	return func(c *fiber.Ctx) error {
		contentIDStr := strings.TrimSpace(c.Params("content_id"))
		if contentIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Content ID is required!", nil)
		}

		contentID, err := strconv.Atoi(contentIDStr)
		if err != nil || contentID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Content ID!", nil)
		}

		reqData := new(struct {
			OptionText string `json:"option_text"`
			IsCorrect  bool   `json:"is_correct"`
			OrderIndex int    `json:"order_index"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		errors := make(map[string]string)
		reqData.OptionText = strings.TrimSpace(reqData.OptionText)

		if reqData.OptionText == "" {
			errors["option_text"] = "Option text is required!"
		}

		if len(errors) > 0 {
			return middleware.ValidationErrorResponse(c, errors)
		}

		c.Locals("contentID", contentID)
		c.Locals("validatedMCQOption", reqData)
		return c.Next()
	}
}

// UpdateMCQOption validates MCQ option update
func UpdateMCQOption() fiber.Handler {
	return func(c *fiber.Ctx) error {
		optionIDStr := strings.TrimSpace(c.Params("option_id"))
		if optionIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Option ID is required!", nil)
		}

		optionID, err := strconv.Atoi(optionIDStr)
		if err != nil || optionID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Option ID!", nil)
		}

		reqData := new(struct {
			OptionText string `json:"option_text"`
			IsCorrect  bool   `json:"is_correct"`
			OrderIndex int    `json:"order_index"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		c.Locals("optionID", optionID)
		c.Locals("validatedMCQOptionUpdate", reqData)
		return c.Next()
	}
}

// DeleteMCQOption validates MCQ option deletion
func DeleteMCQOption() fiber.Handler {
	return func(c *fiber.Ctx) error {
		optionIDStr := strings.TrimSpace(c.Params("option_id"))
		if optionIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Option ID is required!", nil)
		}

		optionID, err := strconv.Atoi(optionIDStr)
		if err != nil || optionID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Option ID!", nil)
		}

		c.Locals("optionID", optionID)
		return c.Next()
	}
}

// ============ Enrollment & Progress Validators ============

// GetCourseEnrollments validates course enrollments list request
func GetCourseEnrollments() fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseIDStr := strings.TrimSpace(c.Params("id"))
		if courseIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Course ID is required!", nil)
		}

		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil || courseID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Course ID!", nil)
		}

		reqData := new(struct {
			Page   *int   `json:"page"`
			Limit  *int   `json:"limit"`
			Status string `json:"status"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid query parameters!", nil)
		}

		c.Locals("courseID", courseID)
		c.Locals("validatedEnrollmentQuery", reqData)
		return c.Next()
	}
}

// GetStudentProgress validates student progress request
func GetStudentProgress() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIDStr := strings.TrimSpace(c.Params("user_id"))
		if userIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "User ID is required!", nil)
		}

		userID, err := strconv.Atoi(userIDStr)
		if err != nil || userID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid User ID!", nil)
		}

		c.Locals("targetUserID", userID)
		return c.Next()
	}
}

// ============ Certificate Validators ============

// GetPendingCertificates validates pending certificates list request
func GetPendingCertificates() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page  *int `json:"page"`
			Limit *int `json:"limit"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid query parameters!", nil)
		}

		c.Locals("validatedCertificateQuery", reqData)
		return c.Next()
	}
}

// ApproveCertificate validates certificate approval request
func ApproveCertificate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestIDStr := strings.TrimSpace(c.Params("request_id"))
		if requestIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Request ID is required!", nil)
		}

		requestID, err := strconv.Atoi(requestIDStr)
		if err != nil || requestID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Request ID!", nil)
		}

		c.Locals("requestID", requestID)
		return c.Next()
	}
}

// RejectCertificate validates certificate rejection request
func RejectCertificate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestIDStr := strings.TrimSpace(c.Params("request_id"))
		if requestIDStr == "" {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Request ID is required!", nil)
		}

		requestID, err := strconv.Atoi(requestIDStr)
		if err != nil || requestID <= 0 {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid Request ID!", nil)
		}

		reqData := new(struct {
			Reason string `json:"reason"`
		})

		if err := c.BodyParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
		}

		c.Locals("requestID", requestID)
		c.Locals("rejectionReason", strings.TrimSpace(reqData.Reason))
		return c.Next()
	}
}

// AdminList validates admin list request with pagination
func AdminList() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqData := new(struct {
			Page  *int `json:"page"`
			Limit *int `json:"limit"`
		})

		if err := c.QueryParser(reqData); err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid query parameters!", nil)
		}

		c.Locals("validatedAdminList", reqData)
		return c.Next()
	}
}
