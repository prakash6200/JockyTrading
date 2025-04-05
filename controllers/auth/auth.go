package authController

import (
	"fib/config"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fib/utils"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Signup(c *fiber.Ctx) error {
	user := new(models.User)
	if err := c.BodyParser(user); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	// Check if email already exists
	existingUser := models.User{}
	result := database.Database.Db.Where("email = ?", user.Email).First(&existingUser)
	if result.RowsAffected > 0 {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Email is already registered!", nil)
	}

	// Check if mobile already exists
	existingUserByMobile := models.User{}
	result = database.Database.Db.Where("mobile = ?", user.Mobile).First(&existingUserByMobile)
	if result.RowsAffected > 0 {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Mobile number is already registered!", nil)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), config.AppConfig.SaltRound)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to process your request!", nil)
	}
	user.Password = string(hashedPassword)

	if err := database.Database.Db.Create(user).Error; err != nil {
		log.Printf("Error saving user to database: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to Signup user!", nil)
	}

	user.Password = ""

	return middleware.JsonResponse(c, fiber.StatusCreated, true, "User registered successfully.", user)
}

func Login(c *fiber.Ctx) error {
	reqData := new(struct {
		Mobile   string `json:"mobile"`
		Email    string `json:"email"`
		Password string `json:"password"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	var user models.User
	var result *gorm.DB

	// Retrieve user by email or mobile
	if reqData.Email != "" {
		result = database.Database.Db.Where("email = ? AND is_deleted = ?", reqData.Email, false).First(&user)
	} else {
		result = database.Database.Db.Where("mobile = ? AND is_deleted = ?", reqData.Mobile, false).First(&user)
	}

	if result.Error != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid credentials!", nil)
	}

	if !user.IsEmailVerified {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Email not verified!", nil)
	}

	if !user.IsMobileVerified {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Mobile not verified!", nil)
	}

	// Check if the user is blocked
	if user.IsBlocked && user.BlockedUntil != nil && user.BlockedUntil.After(time.Now()) {

		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Your account is temporarily blocked. Try again later.", nil)
	}

	if user.LastFailedLogin != nil && time.Since(*user.LastFailedLogin) > 15*time.Minute {

		user.FailedLoginAttempts = 0
		user.LastFailedLogin = nil
		database.Database.Db.Save(&user)
	}

	// Validate password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(reqData.Password)); err != nil {

		user.FailedLoginAttempts++
		now := time.Now()
		user.LastFailedLogin = &now

		// Block user after 3 failed attempts
		if user.FailedLoginAttempts >= 3 {
			user.IsBlocked = true

			unblockTime := now.Add(1 * time.Minute)
			user.BlockedUntil = &unblockTime

			if err := database.Database.Db.Save(&user).Error; err != nil {
				log.Printf("Error blocking user: %v", err)
			}
		}

		// Save the updated user details
		database.Database.Db.Save(&user)

		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Wrong Password", nil)
	}

	// Update last login time
	user.LastLogin = time.Now()
	user.FailedLoginAttempts = 0 // Reset failed login attempts after successful login
	user.IsBlocked = false
	if err := database.Database.Db.Save(&user).Error; err != nil {
		log.Printf("Error saving last login time: %v", err)
	}

	ip := c.IP()
	if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	userAgent := c.Get("User-Agent")

	log.Printf("Login attempt: User-Agent: %s, IP Address: %s", userAgent, ip)

	// Capture login tracking details
	loginTracking := models.LoginTracking{
		UserID:    user.ID,
		IPAddress: ip,
		Device:    userAgent,
		Timestamp: time.Now(),
	}

	// Log the user login tracking
	log.Printf("User %d logged in from IP: %s", user.ID, loginTracking.IPAddress)

	if err := database.Database.Db.Create(&loginTracking).Error; err != nil {
		log.Printf("Error saving login tracking details: %v", err)
	}

	// Sanitize user data (remove sensitive fields)
	user.Password = ""
	user.ProfileImage = ""

	// Generate JWT token
	token, err := middleware.GenerateJWT(user.ID, user.Name, user.Role)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Error generating JWT token!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Login successful.", fiber.Map{
		"user":  user,
		"token": token,
	})
}

func LoginHistoryList(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Retrieve validated request data
	reqData, ok := c.Locals("validatedLoginHistory").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	offset := (*reqData.Page - 1) * (*reqData.Limit)

	var loginTraking []models.LoginTracking
	var total int64

	// Fetch loginTraking with pagination
	if err := database.Database.Db.Where("user_id = ? AND is_deleted = ?", userId, false).
		Offset(offset).
		Limit(*reqData.Limit).
		Find(&loginTraking).
		Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch robots!", nil)
	}

	// Count total records
	database.Database.Db.Model(&models.LoginTracking{}).Where("user_id = ? AND is_deleted = ?", userId, false).Count(&total)

	// Response structure
	response := map[string]interface{}{
		"loginTraking": loginTraking,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  *reqData.Page,
			"limit": *reqData.Limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Login History List.", response)
}

func SendOTP(c *fiber.Ctx) error {
	reqData := new(struct {
		Mobile string `json:"mobile"`
		Email  string `json:"email"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	// Check if email or mobile is already verified
	var user models.User
	var result *gorm.DB

	if reqData.Email != "" {
		result = database.Database.Db.Where("email = ? AND is_deleted = ?", reqData.Email, false).First(&user)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid email!", nil)
		}
		if user.IsEmailVerified {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Email already verified!", nil)
		}
	} else {
		result = database.Database.Db.Where("mobile = ? AND is_deleted = ?", reqData.Mobile, false).First(&user)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid mobile!", nil)
		}
		if user.IsMobileVerified {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Mobile already verified!", nil)
		}
	}

	// Generate OTP
	otp := utils.GenerateOTP()

	// Set OTP expiration time (e.g., 5 minutes from now)
	expiresAt := time.Now().Add(5 * time.Minute)

	// Create OTP record
	otpRecord := models.OTP{
		UserID:      user.ID,
		Email:       reqData.Email,
		Mobile:      reqData.Mobile,
		Code:        otp,
		ExpiresAt:   expiresAt,
		Description: "Email/Mobile Verification OTP",
	}

	// Send OTP via SMS if mobile is provided
	if reqData.Mobile != "" {
		if err := utils.SendOTPToMobile(reqData.Mobile, otp); err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send OTP to mobile!", nil)
		}
	}

	// Send OTP via email if email is provided
	if reqData.Email != "" {
		if err := utils.SendOTPEmail(otp, reqData.Email); err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send OTP to email!", nil)
		}
	}

	// Save OTP record to the database
	if err := database.Database.Db.Create(&otpRecord).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to Create OTP!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "OTP sent successfully.", nil)
}

func VerifyOTP(c *fiber.Ctx) error {
	reqData := new(struct {
		Mobile string `json:"mobile"`
		Email  string `json:"email"`
		Code   string `json:"code"`
	})

	// Parse the request body
	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	var user models.User
	var otpRecord models.OTP
	var result *gorm.DB

	// Retrieve user and OTP record based on email or mobile
	if reqData.Email != "" {
		// Find user by email
		result = database.Database.Db.Where("email = ? AND is_deleted = ?", reqData.Email, false).First(&user)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
		}

		// Find the OTP record for the email
		result = database.Database.Db.Where("email = ? AND code = ? AND is_used = ? AND is_deleted = ?", reqData.Email, reqData.Code, false, false).First(&otpRecord)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid OTP or OTP expired!", nil)
		}
	} else {
		// Find user by mobile
		result = database.Database.Db.Where("mobile = ? AND is_deleted = ?", reqData.Mobile, false).First(&user)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
		}

		// Find the OTP record for the mobile
		result = database.Database.Db.Where("mobile = ? AND code = ? AND is_used = ? AND is_deleted = ?", reqData.Mobile, reqData.Code, false, false).First(&otpRecord)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid OTP or OTP expired!", nil)
		}
	}

	// Check if OTP has expired
	if otpRecord.ExpiresAt.Before(time.Now()) {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "OTP has expired!", nil)
	}

	// Mark OTP as used
	otpRecord.IsUsed = true
	if err := database.Database.Db.Save(&otpRecord).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update OTP status!", nil)
	}

	// Update user's verification status based on email or mobile
	if reqData.Email != "" {
		user.IsEmailVerified = true
	} else {
		user.IsMobileVerified = true
	}

	// Save updated user verification status
	if err := database.Database.Db.Save(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update user verification status!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "OTP verified successfully!", nil)
}

func ForgotPasswordSendOTP(c *fiber.Ctx) error {
	reqData := new(struct {
		Mobile string `json:"mobile"`
		Email  string `json:"email"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	// Check if email or mobile is already verified
	var user models.User
	var result *gorm.DB

	if reqData.Email != "" {
		result = database.Database.Db.Where("email = ? AND is_deleted = ?", reqData.Email, false).First(&user)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid email credentials!", nil)
		}
	} else {
		result = database.Database.Db.Where("mobile = ? AND is_deleted = ?", reqData.Mobile, false).First(&user)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid mobile credentials!", nil)
		}
	}

	// Generate OTP
	otp := utils.GenerateOTP()

	// Set OTP expiration time (e.g., 5 minutes from now)
	expiresAt := time.Now().Add(5 * time.Minute)

	// Create OTP record
	otpRecord := models.OTP{
		UserID:      user.ID,
		Email:       reqData.Email,
		Mobile:      reqData.Mobile,
		Code:        otp,
		ExpiresAt:   expiresAt,
		Description: "Forgot Password OTP",
	}

	// Send OTP via SMS if mobile is provided
	if reqData.Mobile != "" {
		if err := utils.SendOTPToMobile(reqData.Mobile, otp); err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send OTP to mobile!", nil)
		}
	}

	// Send OTP via email if email is provided
	if reqData.Email != "" {
		if err := utils.SendOTPEmail(otp, reqData.Email); err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send OTP to email!", nil)
		}
	}

	// Save OTP record to the database
	if err := database.Database.Db.Create(&otpRecord).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to Create OTP!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "OTP sent successfully.", nil)
}

func ForgotPasswordVerifyOTP(c *fiber.Ctx) error {
	reqData := new(struct {
		Mobile string `json:"mobile"`
		Email  string `json:"email"`
		Code   string `json:"code"`
	})

	// Parse the request body
	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	var user models.User
	var otpRecord models.OTP
	var result *gorm.DB

	// Retrieve user and OTP record based on email or mobile
	if reqData.Email != "" {
		// Find user by email
		result = database.Database.Db.Where("email = ? AND is_deleted = ?", reqData.Email, false).First(&user)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
		}

		// Find the OTP record for the email
		result = database.Database.Db.Where("email = ? AND code = ? AND is_used = ? AND is_deleted = ?", reqData.Email, reqData.Code, false, false).First(&otpRecord)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid OTP or OTP expired!", nil)
		}
	} else {
		// Find user by mobile
		result = database.Database.Db.Where("mobile = ? AND is_deleted = ?", reqData.Mobile, false).First(&user)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
		}

		// Find the OTP record for the mobile
		result = database.Database.Db.Where("mobile = ? AND code = ? AND is_used = ? AND is_deleted = ?", reqData.Mobile, reqData.Code, false, false).First(&otpRecord)
		if result.Error != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid OTP or OTP expired!", nil)
		}
	}

	// Check if OTP has expired
	if otpRecord.ExpiresAt.Before(time.Now()) {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "OTP has expired!", nil)
	}

	// Mark OTP as used
	otpRecord.IsUsed = true
	if err := database.Database.Db.Save(&otpRecord).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update OTP status!", nil)
	}

	// Generate JWT token
	token, err := middleware.GenerateJWT(user.ID, user.Name, user.Role)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Error generating JWT token!", nil)
	}

	// Return success response along with the JWT token
	return middleware.JsonResponse(c, fiber.StatusOK, true, "Now You can reset your password.", fiber.Map{
		"token": token,
	})
}

func ResetPassword(c *fiber.Ctx) error {
	// Retrieve the userId from the JWT token (added by JWTMiddleware)
	userId := c.Locals("userId").(uint)

	fmt.Println(userId)

	// Parse the request body to get the new password
	reqData := new(struct {
		Password string `json:"password"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	// Retrieve the user from the database using userId from JWT token
	var user models.User

	result := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user)

	if result.Error != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found or invalid credentials!", nil)
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(reqData.Password), config.AppConfig.SaltRound)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to hash password!", nil)
	}

	// Update the user's password in the database
	user.Password = string(hashedPassword)
	if err := database.Database.Db.Save(&user).Error; err != nil {
		log.Printf("Error updating user password: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update password!", nil)
	}

	// Respond with success message and the new JWT token
	return middleware.JsonResponse(c, fiber.StatusOK, true, "Password reset successfully.", nil)
}

func ChangeLoginPassword(c *fiber.Ctx) error {
	// Retrieve user ID from JWT token
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user session!", nil)
	}

	// Parse request body
	reqData := new(struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
		CnfPassword     string `json:"cnfPassword"`
	})
	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	// Validate password fields
	if reqData.NewPassword != reqData.CnfPassword {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "New password and confirm password do not match!", nil)
	}
	if len(strings.TrimSpace(reqData.NewPassword)) < 8 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Password must be at least 8 characters long!", nil)
	}

	// Retrieve user from the database
	var user models.User
	result := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user)
	if result.Error != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	// Validate current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(reqData.CurrentPassword)); err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Current password is incorrect!", nil)
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(reqData.NewPassword), config.AppConfig.SaltRound)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to hash password!", nil)
	}

	// Update password using transaction
	err = database.Database.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Printf("Error updating user password: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update password!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Password changed successfully.", nil)
}
