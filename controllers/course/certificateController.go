package controllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	courseModels "fib/models/course"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RequestCertificate requests a certificate for a completed course
func RequestCertificate(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	courseID := c.Locals("courseID").(int)

	// Check enrollment and completion
	var enrollment courseModels.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&enrollment).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "User not enrolled in this course!", nil)
	}

	if enrollment.Status != "COMPLETED" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Please complete the course before requesting a certificate!", nil)
	}

	// Check if certificate already requested
	var existingRequest courseModels.CertificateRequest
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&existingRequest).Error; err == nil {
		if existingRequest.Status == "PENDING" {
			return middleware.JsonResponse(c, fiber.StatusConflict, false, "Certificate request already pending!", nil)
		}
		if existingRequest.Status == "APPROVED" {
			return middleware.JsonResponse(c, fiber.StatusConflict, false, "Certificate already issued!", nil)
		}
	}

	// Check if certificate already exists
	var existingCert courseModels.Certificate
	if err := database.Database.Db.Where("user_id = ? AND course_id = ? AND is_deleted = ?", userID, courseID, false).First(&existingCert).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Certificate already exists!", fiber.Map{
			"certificate": existingCert,
		})
	}

	request := courseModels.CertificateRequest{
		UserID:       userID,
		CourseID:     uint(courseID),
		EnrollmentID: enrollment.ID,
		Status:       "PENDING",
		RequestedAt:  time.Now(),
	}

	if err := database.Database.Db.Create(&request).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to submit certificate request!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusCreated, true, "Certificate request submitted successfully!", request)
}

// GetUserCertificates gets all certificates for the current user
func GetUserCertificates(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	type CertificateWithCourse struct {
		courseModels.Certificate
		CourseName string `json:"course_name"`
	}

	var certificates []courseModels.Certificate
	if err := database.Database.Db.Where("user_id = ? AND is_deleted = ?", userID, false).Order("issued_at desc").Find(&certificates).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch certificates!", nil)
	}

	result := make([]CertificateWithCourse, len(certificates))
	for i, cert := range certificates {
		var course courseModels.Course
		database.Database.Db.Where("id = ?", cert.CourseID).First(&course)
		result[i] = CertificateWithCourse{
			Certificate: cert,
			CourseName:  course.Title,
		}
	}

	// Also get pending requests
	var pendingRequests []courseModels.CertificateRequest
	database.Database.Db.Where("user_id = ? AND status = ? AND is_deleted = ?", userID, "PENDING", false).Find(&pendingRequests)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Certificates fetched successfully!", fiber.Map{
		"certificates":     result,
		"pending_requests": len(pendingRequests),
	})
}

// GetUserEnrollmentsList gets all enrollments for the current user
func GetUserEnrollmentsList(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userID, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	type EnrollmentWithCourse struct {
		courseModels.Enrollment
		CourseName        string `json:"course_name"`
		CourseDescription string `json:"course_description"`
		CourseAuthor      string `json:"course_author"`
		CourseDuration    int64  `json:"course_duration"`
	}

	var enrollments []courseModels.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND is_deleted = ?", userID, false).Order("created_at desc").Find(&enrollments).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch enrollments!", nil)
	}

	result := make([]EnrollmentWithCourse, len(enrollments))
	for i, e := range enrollments {
		var course courseModels.Course
		database.Database.Db.Where("id = ?", e.CourseID).First(&course)
		result[i] = EnrollmentWithCourse{
			Enrollment:        e,
			CourseName:        course.Title,
			CourseDescription: course.Description,
			CourseAuthor:      course.Author,
			CourseDuration:    course.Duration,
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Enrollments fetched successfully!", fiber.Map{
		"enrollments": result,
		"total":       len(result),
	})
}
