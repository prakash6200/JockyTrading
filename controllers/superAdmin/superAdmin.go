package superAdminController

import (
	"fib/config"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"log"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

func UserList(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.First(&user, userId).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	// Retrieve validated request data
	reqData, ok := c.Locals("validateUserList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	offset := (*reqData.Page - 1) * (*reqData.Limit)

	var users []models.User
	var total int64

	// Fetch user list excluding SUPER-ADMIN
	if err := database.Database.Db.
		Where("is_deleted = ? AND role != ?", false, "SUPER-ADMIN").
		Offset(offset).
		Limit(*reqData.Limit).
		Find(&users).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch user list!", nil)
	}

	// Count total records
	database.Database.Db.
		Model(&models.User{}).
		Where("is_deleted = ? AND role != ?", false, "SUPER-ADMIN").
		Count(&total)

	// Response structure
	response := map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  *reqData.Page,
			"limit": *reqData.Limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "User List.", response)
}

func RegisterAMC(c *fiber.Ctx) error {
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
		Name:             reqData.Name,
		Email:            reqData.Email,
		Mobile:           reqData.Mobile,
		Password:         string(hashedPassword),
		Role:             string("AMC"),
		IsMobileVerified: true,
		IsEmailVerified:  true,
	}

	// Create User
	if err := db.Create(&newUser).Error; err != nil {
		log.Printf("Error saving AMC to database: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to Register AMC!", nil)
	}

	// Clean Response
	newUser.Password = ""

	return middleware.JsonResponse(c, fiber.StatusCreated, true, "AMC registered successfully.", newUser)
}
