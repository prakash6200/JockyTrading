package superAdminController

import (
	"fib/config"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func UserList(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "ADMIN").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Retrieve validated request data
	reqData, ok := c.Locals("list").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	offset := (*reqData.Page - 1) * (*reqData.Limit)

	var users []models.User
	var total int64

	if err := database.Database.Db.
		Where("is_deleted = ? AND role = ?", false, "USER").
		Offset(offset).
		Limit(*reqData.Limit).
		Find(&users).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch user list!", nil)
	}

	// Count total records
	database.Database.Db.
		Model(&models.User{}).
		Where("is_deleted = ? AND role != ?", false, "ADMIN").
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

func DistributorList(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "ADMIN").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Retrieve validated request data
	reqData, ok := c.Locals("list").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	offset := (*reqData.Page - 1) * (*reqData.Limit)

	var users []models.User
	var total int64

	if err := database.Database.Db.
		Where("is_deleted = ? AND role = ?", false, "DISTRIBUTOR").
		Offset(offset).
		Limit(*reqData.Limit).
		Find(&users).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch user list!", nil)
	}

	// Count total records
	database.Database.Db.
		Model(&models.User{}).
		Where("is_deleted = ? AND role != ?", false, "ADMIN").
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

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Distributor List.", response)
}

func TransactionList(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "ADMIN").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Retrieve validated request data
	reqData, ok := c.Locals("list").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	offset := (*reqData.Page - 1) * (*reqData.Limit)

	var transactions []models.Transactions
	var total int64

	// Fetch user list excluding SUPER-ADMIN
	if err := database.Database.Db.
		Where("is_deleted = ?", false).
		Offset(offset).
		Limit(*reqData.Limit).
		Find(&transactions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch user list!", nil)
	}

	// Count total records
	database.Database.Db.
		Model(&models.Transactions{}).
		Where("is_deleted = ?", false).
		Count(&total)

	// Response structure
	response := map[string]interface{}{
		"Transactions": transactions,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  *reqData.Page,
			"limit": *reqData.Limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Transaction List.", response)
}

func RegisterAMC(c *fiber.Ctx) error {
	var reqData models.User

	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if ADMIN exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "ADMIN").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

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
		Name:                  reqData.Name,
		Email:                 reqData.Email,
		Mobile:                reqData.Mobile,
		Password:              string(hashedPassword),
		Role:                  "AMC",
		IsMobileVerified:      true,
		IsEmailVerified:       true,
		PanNumber:             reqData.PanNumber,
		Address:               reqData.Address,
		City:                  reqData.City,
		State:                 reqData.State,
		PinCode:               reqData.PinCode,
		ContactPersonName:     reqData.ContactPersonName,
		ContactPerDesignation: reqData.ContactPerDesignation,
		FundName:              reqData.FundName,
		EquityPer:             reqData.DebtPer,
		DebtPer:               reqData.DebtPer,
		CashSplit:             reqData.CashSplit,
	}

	// Create User
	if err := db.Create(&newUser).Error; err != nil {
		log.Printf("Error saving AMC to database: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to Register AMC!", nil)
	}

	// Seed permissions for new AMC user
	if err := SeedPermissions(db, newUser.Role, newUser.ID); err != nil {
		log.Printf("Error seeding permissions: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to assign permissions!", nil)
	}

	// Clean Response
	newUser.Password = ""

	return middleware.JsonResponse(c, fiber.StatusCreated, true, "AMC registered successfully.", newUser)
}

func UpdateAMC(c *fiber.Ctx) error {
	reqData, ok := c.Locals("validatedAMCUpdate").(*struct {
		ID                    uint     `json:"id"`
		Name                  *string  `json:"name"`
		Email                 *string  `json:"email"`
		Mobile                *string  `json:"mobile"`
		Password              *string  `json:"password"`
		PanNumber             *string  `json:"panNumber"`
		Address               *string  `json:"address"`
		City                  *string  `json:"city"`
		State                 *string  `json:"state"`
		PinCode               *string  `json:"pinCode"`
		ContactPersonName     *string  `json:"contactPersonName"`
		ContactPerDesignation *string  `json:"contactPerDesignation"`
		FundName              *string  `json:"fundName"`
		EquityPer             *float32 `json:"equityPer"`
		DebtPer               *float32 `json:"debtPer"`
		CashSplit             *float32 `json:"cashSplit"`
		IsDeleted             *bool    `json:"isDeleted"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Fetch AMC
	var amc models.User
	if err := db.Where("id = ? AND role = ?", reqData.ID, "AMC").First(&amc).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "AMC not found!", nil)
	}

	// Update only provided fields
	if reqData.Name != nil {
		amc.Name = *reqData.Name
	}
	if reqData.Email != nil {
		amc.Email = *reqData.Email
	}
	if reqData.Mobile != nil {
		amc.Mobile = *reqData.Mobile
	}
	if reqData.Password != nil && *reqData.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*reqData.Password), config.AppConfig.SaltRound)
		if err != nil {
			log.Printf("Password hash error: %v", err)
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to hash password", nil)
		}
		amc.Password = string(hashedPassword)
	}
	if reqData.PanNumber != nil {
		amc.PanNumber = *reqData.PanNumber
	}
	if reqData.Address != nil {
		amc.Address = *reqData.Address
	}
	if reqData.City != nil {
		amc.City = *reqData.City
	}
	if reqData.State != nil {
		amc.State = *reqData.State
	}
	if reqData.PinCode != nil {
		amc.PinCode = *reqData.PinCode
	}
	if reqData.ContactPersonName != nil {
		amc.ContactPersonName = *reqData.ContactPersonName
	}
	if reqData.ContactPerDesignation != nil {
		amc.ContactPerDesignation = *reqData.ContactPerDesignation
	}
	if reqData.FundName != nil {
		amc.FundName = *reqData.FundName
	}
	if reqData.EquityPer != nil {
		amc.EquityPer = *reqData.EquityPer
	}
	if reqData.DebtPer != nil {
		amc.DebtPer = *reqData.DebtPer
	}
	if reqData.CashSplit != nil {
		amc.CashSplit = *reqData.CashSplit
	}
	if reqData.IsDeleted != nil {
		amc.IsDeleted = *reqData.IsDeleted
	}

	// Save changes
	if err := db.Save(&amc).Error; err != nil {
		log.Printf("Update error: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update AMC", nil)
	}

	amc.Password = ""
	return middleware.JsonResponse(c, fiber.StatusOK, true, "AMC updated successfully", amc)
}

func RegisterDistributor(c *fiber.Ctx) error {
	var reqData models.User

	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Check if ADMIN exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "ADMIN").First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

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
		Name:                  reqData.Name,
		Email:                 reqData.Email,
		Mobile:                reqData.Mobile,
		Password:              string(hashedPassword),
		Role:                  "DISTRIBUTOR",
		IsMobileVerified:      true,
		IsEmailVerified:       true,
		PanNumber:             reqData.PanNumber,
		Address:               reqData.Address,
		City:                  reqData.City,
		State:                 reqData.State,
		PinCode:               reqData.PinCode,
		ContactPersonName:     reqData.ContactPersonName,
		ContactPerDesignation: reqData.ContactPerDesignation,
	}

	// Create User
	if err := db.Create(&newUser).Error; err != nil {
		log.Printf("Error saving Distributor to database: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to Register AMC!", nil)
	}

	// Seed permissions for new AMC user
	if err := SeedPermissions(db, newUser.Role, newUser.ID); err != nil {
		log.Printf("Error seeding permissions: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to assign permissions!", nil)
	}

	// Clean Response
	newUser.Password = ""

	return middleware.JsonResponse(c, fiber.StatusCreated, true, "Disributor registered successfully.", newUser)
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

func PermissionsByUserID(c *fiber.Ctx) error {
	db := database.Database.Db

	// Get the logged-in user's ID and role from context
	requesterID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	var requester models.User
	if err := db.First(&requester, requesterID).Error; err != nil || requester.Role != "ADMIN" {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access denied!", nil)
	}

	// Parse target userId from query param (e.g. /permissions?userId=2)
	userIDParam := c.Query("userId")
	if userIDParam == "" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "userId is required", nil)
	}

	var targetUserID uint
	if _, err := fmt.Sscanf(userIDParam, "%d", &targetUserID); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid userId", nil)
	}

	// Check if target user exists
	var targetUser models.User
	if err := db.First(&targetUser, targetUserID).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "User not found", nil)
	}

	// Fetch permissions
	var permissions []models.Permission
	if err := db.Where("user_id = ?", targetUserID).Find(&permissions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch permissions", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Permissions fetched successfully", permissions)
}

// func UpdatePermissions(c *fiber.Ctx) error {
// 	// ✅ Only ADMIN can call this
// 	userId, ok := c.Locals("userId").(uint)
// 	if !ok {
// 		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
// 	}

// 	var user models.User
// 	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", userId, "ADMIN").First(&user).Error; err != nil {
// 		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
// 	}

// 	// ✅ Parse request body
// 	var req map[string]bool
// 	if err := c.BodyParser(&req); err != nil {
// 		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request!", nil)
// 	}

// 	// ✅ Get target user ID
// 	targetUserIdRaw, exists := req["userId"]
// 	if !exists {
// 		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "userId is required!", nil)
// 	}
// 	targetUserId := uint(targetUserIdRaw)

// 	// ✅ Delete all previous permissions (soft delete)
// 	if err := database.Database.Db.Model(&models.Permission{}).
// 		Where("user_id = ?", targetUserId).
// 		Update("is_deleted", true).Error; err != nil {
// 		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update old permissions!", nil)
// 	}

// 	// ✅ Create new permissions
// 	var newPermissions []models.Permission
// 	for key, _ := range req {
// 		if key == "userId" {
// 			continue
// 		}
// 		newPermissions = append(newPermissions, models.Permission{
// 			UserID:    targetUserId,
// 			IsDeleted: false,
// 		})
// 	}

// 	if len(newPermissions) > 0 {
// 		if err := database.Database.Db.Create(&newPermissions).Error; err != nil {
// 			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save new permissions!", nil)
// 		}
// 	}

// 	return middleware.JsonResponse(c, fiber.StatusOK, true, "Permissions updated successfully!", nil)
// }
