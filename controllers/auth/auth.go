package authController

import (
	"fib/config"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fib/utils"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Signup(c *fiber.Ctx) error {
	var reqData models.User

	// Parse Request Body
	if err := c.BodyParser(&reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	db := database.Database.Db

	// Check if email already exists
	if err := db.Where("email = ?", reqData.Email).First(&models.User{}).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Email is already registered!", nil)
	}

	// Check if mobile already exists
	if err := db.Where("mobile = ?", reqData.Mobile).First(&models.User{}).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Mobile number is already registered!", nil)
	}

	// Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(reqData.Password), config.AppConfig.SaltRound)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to process your request!", nil)
	}

	// Prepare User Struct for DB Entry
	newUser := models.User{
		Name:     reqData.Name,
		Email:    reqData.Email,
		Mobile:   reqData.Mobile,
		Password: string(hashedPassword),
	}

	// Create User
	if err := db.Create(&newUser).Error; err != nil {
		log.Printf("Error saving user to database: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to Signup user!", nil)
	}

	if err := SeedPermissions(db, newUser.Role, newUser.ID); err != nil {
		log.Printf("Error seeding permissions: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to assign permissions!", nil)
	}

	go func(user models.User, password string) {
		formData := url.Values{}
		formData.Set("name", user.Name)
		formData.Set("email", user.Email)
		formData.Set("mobile", user.Mobile)
		formData.Set("password", password) // send original password, not hashed

		req, err := http.NewRequest("POST", "http://localhost:8000/auth/register", strings.NewReader(formData.Encode()))
		if err != nil {
			log.Printf("Error creating request to external API: %v", err)
			return
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error calling external API: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("External API registration failed: %s", string(body))
		} else {
			log.Printf("User synced successfully to external server: %s", user.Email)
		}
	}(newUser, reqData.Password)
	// Clean Response
	newUser.Password = ""

	return middleware.JsonResponse(c, fiber.StatusCreated, true, "User registered successfully.", newUser)
}

// SeedPermissions seeds default permissions for a given role and user ID
func SeedPermissions(db *gorm.DB, role string, userID uint) error {
	permissions := getDefaultPermissions()

	var permissionRecords []models.Permission
	for _, p := range permissions {
		permissionRecords = append(permissionRecords, models.Permission{
			UserID:     userID,
			Role:       role,
			Permission: p,
		})
	}

	if err := db.Create(&permissionRecords).Error; err != nil {
		return err
	}

	return nil
}

// getDefaultPermissions returns a list of default permission strings
func getDefaultPermissions() []string {
	return []string{
		"login",
		"deposit",
		"withdraw",
		"invest",
		"view-profile",
		"transaction-list",
		"create-folio",
	}
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
	token, err := middleware.GenerateJWT(user.ID, user.Name, user.Role, user.Email, user.Mobile)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to generate token", nil)
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
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Access Denied!", nil)
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
		user.IsMobileVerified = true
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
	token, err := middleware.GenerateJWT(user.ID, user.Name, user.Role, user.Email, user.Mobile)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to generate token", nil)
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

// Login Send otp
func LoginSendOTP(c *fiber.Ctx) error {
	reqData := new(struct {
		Mobile string `json:"mobile"`
		Email  string `json:"email"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	// Validate input: only one of email or mobile should be provided
	if (reqData.Email == "" && reqData.Mobile == "") || (reqData.Email != "" && reqData.Mobile != "") {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Provide either email or mobile (only one).", nil)
	}

	var user models.User
	var result *gorm.DB

	// Try finding user by email or mobile
	if reqData.Email != "" {
		result = database.Database.Db.Where("email = ? AND is_deleted = false", reqData.Email).First(&user)
	} else {
		result = database.Database.Db.Where("mobile = ? AND is_deleted = false", reqData.Mobile).First(&user)
	}

	// If not found, create a new user
	if result.Error != nil {
		newUser := models.User{
			Email:  reqData.Email,
			Mobile: reqData.Mobile,
			Role:   "USER",
		}
		if err := database.Database.Db.Create(&newUser).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create user!", nil)
		}
		user = newUser
	}

	// Generate OTP and expiry
	otp := utils.GenerateOTP()
	expiresAt := time.Now().Add(5 * time.Minute)

	// Create OTP record
	otpRecord := models.OTP{
		UserID:      user.ID,
		Email:       reqData.Email,
		Mobile:      reqData.Mobile,
		Code:        otp,
		ExpiresAt:   expiresAt,
		Description: "Login OTP",
	}

	// Send OTP
	if reqData.Mobile != "" {
		if err := utils.SendOTPToMobile(reqData.Mobile, otp); err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send OTP to mobile!", nil)
		}
	} else {
		if err := utils.SendOTPEmail(otp, reqData.Email); err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send OTP to email!", nil)
		}
	}

	// Save OTP to DB
	if err := database.Database.Db.Create(&otpRecord).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save OTP!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "OTP sent successfully.", nil)
}

// Login Verify otp
// func LoginVerifyOTP(c *fiber.Ctx) error {
// 	reqData := new(struct {
// 		Mobile string `json:"mobile"`
// 		Email  string `json:"email"`
// 		Code   string `json:"code"`
// 	})

// 	if err := c.BodyParser(reqData); err != nil {
// 		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
// 	}

// 	// Validate that exactly one of email or mobile is provided
// 	if (reqData.Email == "" && reqData.Mobile == "") || (reqData.Email != "" && reqData.Mobile != "") {
// 		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Provide either email or mobile (only one).", nil)
// 	}

// 	var user models.User
// 	var otpRecord models.OTP

// 	// Case: Email-based OTP
// 	if reqData.Email != "" {
// 		// Find user
// 		if err := database.Database.Db.Where("email = ? AND is_deleted = false", reqData.Email).First(&user).Error; err != nil {
// 			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
// 		}

// 		// Find OTP
// 		if err := database.Database.Db.Where("email = ? AND code = ? AND is_used = false AND is_deleted = false", reqData.Email, reqData.Code).First(&otpRecord).Error; err != nil {
// 			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid or expired OTP!", nil)
// 		}
// 	}

// 	// Case: Mobile-based OTP
// 	if reqData.Mobile != "" {
// 		if err := database.Database.Db.Where("mobile = ? AND is_deleted = false", reqData.Mobile).First(&user).Error; err != nil {
// 			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
// 		}

// 		if err := database.Database.Db.Where("mobile = ? AND code = ? AND is_used = false AND is_deleted = false", reqData.Mobile, reqData.Code).First(&otpRecord).Error; err != nil {
// 			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid or expired OTP!", nil)
// 		}
// 	}

// 	// Check OTP expiration
// 	if otpRecord.ExpiresAt.Before(time.Now()) {
// 		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "OTP has expired!", nil)
// 	}

// 	// Mark OTP as used
// 	otpRecord.IsUsed = true
// 	if err := database.Database.Db.Save(&otpRecord).Error; err != nil {
// 		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update OTP status!", nil)
// 	}

// 	// Generate JWT
// 	token, err := middleware.GenerateJWT(user.ID, user.Name, user.Role)
// 	if err != nil {
// 		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to generate token", nil)
// 	}

// 	return middleware.JsonResponse(c, fiber.StatusOK, true, "OTP verified successfully.", fiber.Map{
// 		"user":  user,
// 		"token": token,
// 	})
// }

func LoginVerifyOTP(c *fiber.Ctx) error {
	reqData := new(struct {
		Mobile string `json:"mobile"`
		Email  string `json:"email"`
		Code   string `json:"code"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	// Validate that exactly one of email or mobile is provided
	if (reqData.Email == "" && reqData.Mobile == "") || (reqData.Email != "" && reqData.Mobile != "") {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Provide either email or mobile (only one).", nil)
	}

	var user models.User

	// Case: Email-based OTP
	if reqData.Email != "" {
		// Find user
		if err := database.Database.Db.Where("email = ? AND is_deleted = false", reqData.Email).First(&user).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
		}
	}

	// Case: Mobile-based OTP
	if reqData.Mobile != "" {
		if err := database.Database.Db.Where("mobile = ? AND is_deleted = false", reqData.Mobile).First(&user).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
		}
	}

	// ✅ Hardcoded OTP check
	if reqData.Code != "1234" {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid OTP!", nil)
	}

	// Generate JWT
	// token, err := middleware.GenerateJWT(user.ID, user.Name, user.Role)
	token, err := middleware.GenerateJWT(user.ID, user.Name, user.Role, user.Email, user.Mobile)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to generate token", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "OTP verified successfully.", fiber.Map{
		"user":  user,
		"token": token,
	})
}
