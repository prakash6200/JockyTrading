package controllers

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	courseModels "fib/models/course"
	"fib/utils"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// AdminGetCourseEnrollments gets all enrolled students for a course
func AdminGetCourseEnrollments(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	courseID := c.Locals("courseID").(int)

	reqData, _ := c.Locals("validatedEnrollmentQuery").(*struct {
		Page   *int   `json:"page"`
		Limit  *int   `json:"limit"`
		Status string `json:"status"`
	})

	page := 1
	limit := 10
	if reqData != nil && reqData.Page != nil && *reqData.Page > 0 {
		page = *reqData.Page
	}
	if reqData != nil && reqData.Limit != nil && *reqData.Limit > 0 {
		limit = *reqData.Limit
	}
	offset := (page - 1) * limit

	db := database.Database.Db.Model(&courseModels.Enrollment{}).Where("course_id = ? AND is_deleted = ?", courseID, false)

	if reqData != nil && reqData.Status != "" {
		db = db.Where("status = ?", reqData.Status)
	}

	var total int64
	db.Count(&total)

	type EnrollmentWithUser struct {
		courseModels.Enrollment
		UserName  string `json:"user_name"`
		UserEmail string `json:"user_email"`
	}

	var enrollments []courseModels.Enrollment
	if err := db.Offset(offset).Limit(limit).Order("created_at desc").Find(&enrollments).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch enrollments!", nil)
	}

	// Fetch user details for each enrollment
	result := make([]EnrollmentWithUser, len(enrollments))
	for i, e := range enrollments {
		var enrolledUser models.User
		database.Database.Db.Where("id = ?", e.UserID).First(&enrolledUser)
		result[i] = EnrollmentWithUser{
			Enrollment: e,
			UserName:   enrolledUser.Name,
			UserEmail:  enrolledUser.Email,
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Enrollments fetched successfully!", fiber.Map{
		"enrollments": result,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// AdminGetCompletedStudents gets students who completed a course
func AdminGetCompletedStudents(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	courseID := c.Locals("courseID").(int)

	type CompletedStudent struct {
		UserID      uint       `json:"user_id"`
		UserName    string     `json:"user_name"`
		UserEmail   string     `json:"user_email"`
		Progress    float64    `json:"progress"`
		CompletedAt *time.Time `json:"completed_at"`
	}

	var enrollments []courseModels.Enrollment
	if err := database.Database.Db.Where("course_id = ? AND status = ? AND is_deleted = ?", courseID, "COMPLETED", false).
		Order("completed_at desc").Find(&enrollments).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch completed students!", nil)
	}

	result := make([]CompletedStudent, len(enrollments))
	for i, e := range enrollments {
		var enrolledUser models.User
		database.Database.Db.Where("id = ?", e.UserID).First(&enrolledUser)
		result[i] = CompletedStudent{
			UserID:      e.UserID,
			UserName:    enrolledUser.Name,
			UserEmail:   enrolledUser.Email,
			Progress:    e.Progress,
			CompletedAt: e.CompletedAt,
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Completed students fetched successfully!", fiber.Map{
		"completed_students": result,
		"total":              len(result),
	})
}

// AdminGetStudentProgress gets detailed progress for a student
func AdminGetStudentProgress(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	targetUserID := c.Locals("targetUserID").(int)

	// Get target user
	var targetUser models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", targetUserID, false).First(&targetUser).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Student not found!", nil)
	}

	// Get all enrollments for the user
	var enrollments []courseModels.Enrollment
	if err := database.Database.Db.Where("user_id = ? AND is_deleted = ?", targetUserID, false).Find(&enrollments).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch enrollments!", nil)
	}

	type CourseProgress struct {
		CourseID          uint       `json:"course_id"`
		CourseName        string     `json:"course_name"`
		Status            string     `json:"status"`
		Progress          float64    `json:"progress"`
		CompletedContents int        `json:"completed_contents"`
		TotalContents     int        `json:"total_contents"`
		EnrolledAt        time.Time  `json:"enrolled_at"`
		CompletedAt       *time.Time `json:"completed_at"`
	}

	// Get MCQ attempts summary
	var mcqAttempts []courseModels.MCQAttempt
	database.Database.Db.Where("user_id = ? AND is_deleted = ?", targetUserID, false).Find(&mcqAttempts)

	totalMCQAttempts := len(mcqAttempts)
	correctMCQs := 0
	for _, attempt := range mcqAttempts {
		if attempt.IsCorrect {
			correctMCQs++
		}
	}

	courseProgress := make([]CourseProgress, len(enrollments))
	for i, e := range enrollments {
		var course courseModels.Course
		database.Database.Db.Where("id = ?", e.CourseID).First(&course)
		courseProgress[i] = CourseProgress{
			CourseID:          e.CourseID,
			CourseName:        course.Title,
			Status:            e.Status,
			Progress:          e.Progress,
			CompletedContents: e.CompletedContents,
			TotalContents:     e.TotalContents,
			EnrolledAt:        e.CreatedAt,
			CompletedAt:       e.CompletedAt,
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Student progress fetched successfully!", fiber.Map{
		"student": fiber.Map{
			"id":    targetUser.ID,
			"name":  targetUser.Name,
			"email": targetUser.Email,
		},
		"course_progress": courseProgress,
		"mcq_summary": fiber.Map{
			"total_attempts":  totalMCQAttempts,
			"correct_answers": correctMCQs,
			"accuracy_percent": func() float64 {
				if totalMCQAttempts == 0 {
					return 0
				}
				return float64(correctMCQs) / float64(totalMCQAttempts) * 100
			}(),
		},
	})
}

// AdminGetPendingCertificates gets pending certificate requests
func AdminGetPendingCertificates(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	reqData, _ := c.Locals("validatedCertificateQuery").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})

	page := 1
	limit := 10
	if reqData != nil && reqData.Page != nil && *reqData.Page > 0 {
		page = *reqData.Page
	}
	if reqData != nil && reqData.Limit != nil && *reqData.Limit > 0 {
		limit = *reqData.Limit
	}
	offset := (page - 1) * limit

	var total int64
	database.Database.Db.Model(&courseModels.CertificateRequest{}).Where("status = ? AND is_deleted = ?", "PENDING", false).Count(&total)

	type RequestWithDetails struct {
		courseModels.CertificateRequest
		UserName   string `json:"user_name"`
		UserEmail  string `json:"user_email"`
		CourseName string `json:"course_name"`
	}

	var requests []courseModels.CertificateRequest
	if err := database.Database.Db.Where("status = ? AND is_deleted = ?", "PENDING", false).
		Offset(offset).Limit(limit).Order("requested_at asc").Find(&requests).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch requests!", nil)
	}

	result := make([]RequestWithDetails, len(requests))
	for i, r := range requests {
		var reqUser models.User
		var course courseModels.Course
		database.Database.Db.Where("id = ?", r.UserID).First(&reqUser)
		database.Database.Db.Where("id = ?", r.CourseID).First(&course)
		result[i] = RequestWithDetails{
			CertificateRequest: r,
			UserName:           reqUser.Name,
			UserEmail:          reqUser.Email,
			CourseName:         course.Title,
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Pending certificate requests fetched successfully!", fiber.Map{
		"requests": result,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// AdminGetIssuedCertificates gets all issued certificates
func AdminGetIssuedCertificates(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	reqData, _ := c.Locals("validatedCertificateQuery").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})

	page := 1
	limit := 10
	if reqData != nil && reqData.Page != nil && *reqData.Page > 0 {
		page = *reqData.Page
	}
	if reqData != nil && reqData.Limit != nil && *reqData.Limit > 0 {
		limit = *reqData.Limit
	}
	offset := (page - 1) * limit

	var total int64
	database.Database.Db.Model(&courseModels.Certificate{}).Where("is_deleted = ?", false).Count(&total)

	type CertificateWithDetails struct {
		courseModels.Certificate
		UserName   string `json:"user_name"`
		UserEmail  string `json:"user_email"`
		CourseName string `json:"course_name"`
	}

	var certificates []courseModels.Certificate
	if err := database.Database.Db.Where("is_deleted = ?", false).
		Offset(offset).Limit(limit).Order("issued_at desc").Find(&certificates).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch certificates!", nil)
	}

	result := make([]CertificateWithDetails, len(certificates))
	for i, cert := range certificates {
		var certUser models.User
		var course courseModels.Course
		database.Database.Db.Select("name, email").Where("id = ?", cert.UserID).First(&certUser)
		database.Database.Db.Select("title").Where("id = ?", cert.CourseID).First(&course)
		result[i] = CertificateWithDetails{
			Certificate: cert,
			UserName:    certUser.Name,
			UserEmail:   certUser.Email,
			CourseName:  course.Title,
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Issued certificates fetched successfully!", fiber.Map{
		"certificates": result,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// AdminApproveCertificate approves a certificate request
func AdminApproveCertificate(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	requestID := c.Locals("requestID").(int)

	var request courseModels.CertificateRequest
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", requestID, false).First(&request).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Certificate request not found!", nil)
	}

	if request.Status != "PENDING" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Request is not pending!", nil)
	}

	tx := database.Database.Db.Begin()

	// Update request status
	now := time.Now()
	request.Status = "APPROVED"
	request.ApprovedAt = &now
	request.ApprovedBy = &userId

	if err := tx.Save(&request).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to approve request!", nil)
	}

	// Generate certificate number
	certNumber := fmt.Sprintf("CERT-%d-%d-%d", request.CourseID, request.UserID, time.Now().Unix())

	// Create certificate
	certificate := courseModels.Certificate{
		UserID:            request.UserID,
		CourseID:          request.CourseID,
		CertificateNumber: certNumber,
		IssuedAt:          now,
	}

	if err := tx.Create(&certificate).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create certificate!", nil)
	}

	tx.Commit()

	// Send certificate email asynchronously
	var reqUser models.User
	var course courseModels.Course
	database.Database.Db.Where("id = ?", request.UserID).First(&reqUser)
	database.Database.Db.Where("id = ?", request.CourseID).First(&course)

	go utils.SendCertificateEmail(reqUser.Email, reqUser.Name, course.Title, certNumber)

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Certificate approved and generated successfully!", certificate)
}

// AdminRejectCertificate rejects a certificate request
func AdminRejectCertificate(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	requestID := c.Locals("requestID").(int)
	reason := c.Locals("rejectionReason").(string)

	var request courseModels.CertificateRequest
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", requestID, false).First(&request).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Certificate request not found!", nil)
	}

	if request.Status != "PENDING" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Request is not pending!", nil)
	}

	request.Status = "REJECTED"
	request.RejectionReason = reason

	if err := database.Database.Db.Save(&request).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to reject request!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Certificate request rejected!", request)
}

// AdminDashboardStats gets dashboard statistics
func AdminDashboardStats(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	if user.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied! Admin only.", nil)
	}

	var totalCourses, publishedCourses, totalEnrollments, completedEnrollments, pendingCertificates int64

	database.Database.Db.Model(&courseModels.Course{}).Where("is_deleted = ?", false).Count(&totalCourses)
	database.Database.Db.Model(&courseModels.Course{}).Where("is_deleted = ? AND is_published = ?", false, true).Count(&publishedCourses)
	database.Database.Db.Model(&courseModels.Enrollment{}).Where("is_deleted = ?", false).Count(&totalEnrollments)
	database.Database.Db.Model(&courseModels.Enrollment{}).Where("is_deleted = ? AND status = ?", false, "COMPLETED").Count(&completedEnrollments)
	database.Database.Db.Model(&courseModels.CertificateRequest{}).Where("is_deleted = ? AND status = ?", false, "PENDING").Count(&pendingCertificates)

	// Get recent enrollments
	type RecentEnrollment struct {
		UserName   string    `json:"user_name"`
		CourseName string    `json:"course_name"`
		EnrolledAt time.Time `json:"enrolled_at"`
	}

	var recentEnrollments []courseModels.Enrollment
	database.Database.Db.Where("is_deleted = ?", false).Order("created_at desc").Limit(5).Find(&recentEnrollments)

	recent := make([]RecentEnrollment, len(recentEnrollments))
	for i, e := range recentEnrollments {
		var enrolledUser models.User
		var course courseModels.Course
		database.Database.Db.Where("id = ?", e.UserID).First(&enrolledUser)
		database.Database.Db.Where("id = ?", e.CourseID).First(&course)
		recent[i] = RecentEnrollment{
			UserName:   enrolledUser.Name,
			CourseName: course.Title,
			EnrolledAt: e.CreatedAt,
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Dashboard stats fetched successfully!", fiber.Map{
		"stats": fiber.Map{
			"total_courses":         totalCourses,
			"published_courses":     publishedCourses,
			"total_enrollments":     totalEnrollments,
			"completed_enrollments": completedEnrollments,
			"pending_certificates":  pendingCertificates,
		},
		"recent_enrollments": recent,
	})
}
