package walletController

import (
	"encoding/json"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GetWalletBalance returns user's current wallet balance
func GetWalletBalance(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Wallet balance fetched!", fiber.Map{
		"balance":  user.MainBalance,
		"currency": "INR",
	})
}

// DepositToWallet allows user to deposit money with payment gateway details
func DepositToWallet(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedDeposit").(*struct {
		Amount           float64 `json:"amount"`
		PaymentGateway   string  `json:"paymentGateway"`
		PaymentOrderID   string  `json:"paymentOrderId"`
		PaymentID        string  `json:"paymentId"`
		PaymentSignature string  `json:"paymentSignature"`
		PaymentMethod    string  `json:"paymentMethod"`
		PaymentStatus    string  `json:"paymentStatus"`
		PaymentResponse  any     `json:"paymentResponse"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Check if payment ID already exists (duplicate transaction)
	var existingTxn models.WalletTransaction
	if err := db.Where("payment_id = ? AND is_deleted = false", reqData.PaymentID).First(&existingTxn).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Transaction already processed!", nil)
	}

	// Convert payment response to JSON string
	paymentResponseJSON := ""
	if reqData.PaymentResponse != nil {
		if jsonBytes, err := json.Marshal(reqData.PaymentResponse); err == nil {
			paymentResponseJSON = string(jsonBytes)
		}
	}

	// Calculate balances
	balanceBefore := float64(user.MainBalance)
	balanceAfter := balanceBefore + reqData.Amount

	// Create transaction record
	transaction := models.WalletTransaction{
		UserID:             userId,
		TransactionType:    models.TransactionTypeDeposit,
		Amount:             reqData.Amount,
		BalanceBefore:      balanceBefore,
		BalanceAfter:       balanceAfter,
		Status:             models.TransactionStatusCompleted,
		Description:        "Wallet deposit via " + reqData.PaymentGateway,
		PaymentGateway:     reqData.PaymentGateway,
		PaymentOrderID:     reqData.PaymentOrderID,
		PaymentID:          reqData.PaymentID,
		PaymentSignature:   reqData.PaymentSignature,
		PaymentMethod:      reqData.PaymentMethod,
		PaymentStatus:      reqData.PaymentStatus,
		PaymentResponseRaw: paymentResponseJSON,
		TransactionDate:    time.Now(),
	}

	// Start transaction
	tx := db.Begin()

	// Create transaction record
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create transaction!", nil)
	}

	// Update user balance
	user.MainBalance = uint(balanceAfter)
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update balance!", nil)
	}

	tx.Commit()

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Deposit successful!", fiber.Map{
		"transactionId": transaction.ID,
		"amount":        reqData.Amount,
		"balanceBefore": balanceBefore,
		"balanceAfter":  balanceAfter,
		"paymentId":     reqData.PaymentID,
		"status":        transaction.Status,
	})
}

// GetWalletHistory returns user's wallet transaction history
func GetWalletHistory(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false", userId).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	// Parse query params
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	txnType := c.Query("type") // DEPOSIT, SUBSCRIPTION, etc.

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit
	db := database.Database.Db

	query := db.Model(&models.WalletTransaction{}).Where("user_id = ? AND is_deleted = false", userId)

	if txnType != "" {
		query = query.Where("transaction_type = ?", txnType)
	}

	var total int64
	query.Count(&total)

	var transactions []models.WalletTransaction
	if err := query.
		Order("transaction_date DESC").
		Offset(offset).
		Limit(limit).
		Find(&transactions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch history!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Wallet history fetched!", fiber.Map{
		"transactions":   transactions,
		"currentBalance": user.MainBalance,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// AddBalance adds balance to user's wallet (Admin only)
func AddBalance(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var admin models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&admin).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin role required.", nil)
	}

	reqData, ok := c.Locals("validatedAddBalance").(*struct {
		UserID uint    `json:"userId"`
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Find target user
	var targetUser models.User
	if err := db.Where("id = ? AND is_deleted = false", reqData.UserID).First(&targetUser).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "User not found!", nil)
	}

	// Calculate balances
	balanceBefore := float64(targetUser.MainBalance)
	balanceAfter := balanceBefore + reqData.Amount

	// Start transaction
	tx := db.Begin()

	// Create transaction record
	transaction := models.WalletTransaction{
		UserID:          reqData.UserID,
		TransactionType: models.TransactionTypeAdminCredit,
		Amount:          reqData.Amount,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceAfter,
		Status:          models.TransactionStatusCompleted,
		Description:     "Admin credit: " + reqData.Reason,
		AdminID:         userId,
		Reason:          reqData.Reason,
		TransactionDate: time.Now(),
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create transaction!", nil)
	}

	// Update user balance
	targetUser.MainBalance = uint(balanceAfter)
	if err := tx.Save(&targetUser).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to add balance!", nil)
	}

	tx.Commit()

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Balance added successfully!", fiber.Map{
		"transactionId":   transaction.ID,
		"userId":          reqData.UserID,
		"previousBalance": balanceBefore,
		"amountAdded":     reqData.Amount,
		"newBalance":      targetUser.MainBalance,
		"reason":          reqData.Reason,
		"addedBy":         admin.Name,
	})
}

// DeductBalance deducts balance from user's wallet (Admin only)
func DeductBalance(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var admin models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&admin).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin role required.", nil)
	}

	reqData, ok := c.Locals("validatedDeductBalance").(*struct {
		UserID uint    `json:"userId"`
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	db := database.Database.Db

	// Find target user
	var targetUser models.User
	if err := db.Where("id = ? AND is_deleted = false", reqData.UserID).First(&targetUser).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "User not found!", nil)
	}

	// Check if user has enough balance
	if float64(targetUser.MainBalance) < reqData.Amount {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Insufficient balance to deduct!", nil)
	}

	// Calculate balances
	balanceBefore := float64(targetUser.MainBalance)
	balanceAfter := balanceBefore - reqData.Amount

	// Start transaction
	tx := db.Begin()

	// Create transaction record
	transaction := models.WalletTransaction{
		UserID:          reqData.UserID,
		TransactionType: models.TransactionTypeAdminDebit,
		Amount:          reqData.Amount,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceAfter,
		Status:          models.TransactionStatusCompleted,
		Description:     "Admin debit: " + reqData.Reason,
		AdminID:         userId,
		Reason:          reqData.Reason,
		TransactionDate: time.Now(),
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create transaction!", nil)
	}

	// Deduct balance
	targetUser.MainBalance = uint(balanceAfter)
	if err := tx.Save(&targetUser).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to deduct balance!", nil)
	}

	tx.Commit()

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Balance deducted successfully!", fiber.Map{
		"transactionId":   transaction.ID,
		"userId":          reqData.UserID,
		"previousBalance": balanceBefore,
		"amountDeducted":  reqData.Amount,
		"newBalance":      targetUser.MainBalance,
		"reason":          reqData.Reason,
		"deductedBy":      admin.Name,
	})
}

// GetUserBalance returns a specific user's balance (Admin only)
func GetUserBalance(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	targetUserId := c.QueryInt("userId", 0)

	var admin models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&admin).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin role required.", nil)
	}

	if targetUserId == 0 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "userId is required!", nil)
	}

	db := database.Database.Db

	var targetUser models.User
	if err := db.Where("id = ? AND is_deleted = false", targetUserId).First(&targetUser).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "User not found!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "User balance fetched!", fiber.Map{
		"userId":   targetUser.ID,
		"name":     targetUser.Name,
		"email":    targetUser.Email,
		"balance":  targetUser.MainBalance,
		"currency": "INR",
	})
}

// GetUserWalletHistory returns a specific user's wallet history (Admin only)
func GetUserWalletHistory(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	targetUserId := c.QueryInt("userId", 0)

	var admin models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role IN ?", userId, []string{"ADMIN", "SUPER-ADMIN"}).First(&admin).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! Admin role required.", nil)
	}

	if targetUserId == 0 {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "userId is required!", nil)
	}

	// Parse query params
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	txnType := c.Query("type")

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit
	db := database.Database.Db

	// Check user exists
	var targetUser models.User
	if err := db.Where("id = ? AND is_deleted = false", targetUserId).First(&targetUser).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "User not found!", nil)
	}

	query := db.Model(&models.WalletTransaction{}).Where("user_id = ? AND is_deleted = false", targetUserId)

	if txnType != "" {
		query = query.Where("transaction_type = ?", txnType)
	}

	var total int64
	query.Count(&total)

	var transactions []models.WalletTransaction
	if err := query.
		Order("transaction_date DESC").
		Offset(offset).
		Limit(limit).
		Find(&transactions).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch history!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "User wallet history fetched!", fiber.Map{
		"user": fiber.Map{
			"id":      targetUser.ID,
			"name":    targetUser.Name,
			"email":   targetUser.Email,
			"balance": targetUser.MainBalance,
		},
		"transactions": transactions,
		"pagination": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// RecordSubscriptionTransaction records a subscription deduction (called from subscribe function)
func RecordSubscriptionTransaction(db any, userId uint, amount float64, balanceBefore float64, basketId uint, basketName string) error {
	gormDb := db.(*database.DbInstance).Db

	transaction := models.WalletTransaction{
		UserID:          userId,
		TransactionType: models.TransactionTypeSubscription,
		Amount:          amount,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceBefore - amount,
		Status:          models.TransactionStatusCompleted,
		Description:     "Subscription: " + basketName,
		ReferenceType:   "basket",
		ReferenceID:     basketId,
		ReferenceName:   basketName,
		TransactionDate: time.Now(),
	}

	return gormDb.Create(&transaction).Error
}
