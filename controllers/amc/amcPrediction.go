package amcController

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	"github.com/gofiber/fiber/v2"
)

// CreateOrUpdatePrediction creates or updates a prediction value
func CreateOrUpdatePrediction(c *fiber.Ctx) error {
	// Get userID from context
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user ID", nil)
	}

	// Validate AMC user
	var user models.User
	if err := database.Database.Db.
		Where("id = ? AND is_deleted = ? AND role = ?", userID, false, "AMC").
		First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Unauthorized - AMC role required", nil)
	}

	// Parse request body
	var body struct {
		ID          uint   `json:"id"`
		Title       string `json:"title"`
		Prediction  int    `json:"prediction"`
		Achieved    *int   `json:"achieved"`
		Description string `json:"description"`
	}

	if err := c.BodyParser(&body); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data", nil)
	}

	// Create or update prediction
	prediction := models.AMCPredictionValue{
		UserID:      userID,
		Title:       body.Title,
		Prediction:  body.Prediction,
		Achieved:    body.Achieved,
		Description: body.Description,
	}

	// If ID is provided, update existing prediction
	if body.ID != 0 {
		prediction.ID = body.ID
		if err := database.Database.Db.Model(&prediction).Updates(prediction).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update prediction", nil)
		}
	} else {
		// Create new prediction
		if err := database.Database.Db.Create(&prediction).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create prediction", nil)
		}
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Prediction saved successfully", prediction)
}

// GetPredictions retrieves all predictions for an AMC with pagination
func GetPredictions(c *fiber.Ctx) error {
	// Get userID from context
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user ID", nil)
	}

	// Validate AMC user
	var user models.User
	if err := database.Database.Db.
		Where("id = ? AND is_deleted = ? AND role = ?", userID, false, "AMC").
		First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Unauthorized - AMC role required", nil)
	}

	// Get pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	// Query predictions
	var predictions []models.AMCPredictionValue
	var total int64

	db := database.Database.Db.
		Where("user_id = ? AND is_deleted = ?", userID, false)

	// Count total records
	if err := db.Model(&models.AMCPredictionValue{}).Count(&total).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to count predictions", nil)
	}

	// Get paginated results
	if err := db.Offset(offset).Limit(limit).Find(&predictions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch predictions", nil)
	}

	// Prepare response
	response := map[string]interface{}{
		"predictions": predictions,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Predictions retrieved successfully", response)
}

// GetPrediction retrieves a specific prediction
func GetPrediction(c *fiber.Ctx) error {
	// Get userID from context
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user ID", nil)
	}

	// Validate AMC user
	var user models.User
	if err := database.Database.Db.
		Where("id = ? AND is_deleted = ? AND role = ?", userID, false, "AMC").
		First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Unauthorized - AMC role required", nil)
	}

	// Get prediction ID from params
	predictionID, err := c.ParamsInt("id")
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid prediction ID", nil)
	}

	// Get prediction
	var prediction models.AMCPredictionValue
	if err := database.Database.Db.
		Where("id = ? AND user_id = ? AND is_deleted = ?", predictionID, userID, false).
		First(&prediction).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Prediction not found", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Prediction retrieved successfully", prediction)
}

// DeletePrediction soft deletes a prediction
func DeletePrediction(c *fiber.Ctx) error {
	// Get userID from context
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user ID", nil)
	}

	// Validate AMC user
	var user models.User
	if err := database.Database.Db.
		Where("id = ? AND is_deleted = ? AND role = ?", userID, false, "AMC").
		First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Unauthorized - AMC role required", nil)
	}

	// Get prediction ID from params
	predictionID, err := c.ParamsInt("id")
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid prediction ID", nil)
	}

	// Soft delete prediction
	result := database.Database.Db.Model(&models.AMCPredictionValue{}).
		Where("id = ? AND user_id = ?", predictionID, userID).
		Update("is_deleted", true)

	if result.Error != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to delete prediction", nil)
	}

	if result.RowsAffected == 0 {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Prediction not found", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Prediction deleted successfully", nil)
}

// UpdateAchievedValue updates the achieved value for a prediction
func UpdateAchievedValue(c *fiber.Ctx) error {
	// Get userID from context
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user ID", nil)
	}

	// Validate AMC user
	var user models.User
	if err := database.Database.Db.
		Where("id = ? AND is_deleted = ? AND role = ?", userID, false, "AMC").
		First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Unauthorized - AMC role required", nil)
	}

	// Parse request body
	var body struct {
		ID       uint `json:"id"`
		Achieved int  `json:"achieved"`
	}

	if err := c.BodyParser(&body); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data", nil)
	}

	// Update achieved value
	result := database.Database.Db.Model(&models.AMCPredictionValue{}).
		Where("id = ? AND user_id = ? AND is_deleted = ?", body.ID, userID, false).
		Update("achieved", body.Achieved)

	if result.Error != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update achieved value", nil)
	}

	if result.RowsAffected == 0 {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Prediction not found", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Achieved value updated successfully", nil)
}
