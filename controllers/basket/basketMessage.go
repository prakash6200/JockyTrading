package basketController

import (
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fib/models/basket"
	"sort"

	"github.com/gofiber/fiber/v2"
)

// AMC Send Message (Broadcast or Direct)
// POST /amc/basket/message
func AMCSendMessage(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	// Validate AMC
	var amc models.User
	if err := database.Database.Db.Where("id = ? AND role = 'AMC' AND is_deleted = false", userId).First(&amc).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied! AMC role required.", nil)
	}

	reqData := new(struct {
		BasketID     uint   `json:"basketId"`
		TargetUserID uint   `json:"targetUserId"` // Optional: If set, sends DM to specific user
		Action       string `json:"action"`       // BUY, SELL, HOLD, GENERAL
		Message      string `json:"message"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	if reqData.BasketID == 0 || reqData.Message == "" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "BasketID and Message are required!", nil)
	}
	if reqData.Action == "" {
		reqData.Action = basket.ActionGeneral
	}

	db := database.Database.Db

	// Verify AMC owns the basket
	var existingBasket basket.Basket
	if err := db.Where("id = ? AND amc_id = ? AND is_deleted = false", reqData.BasketID, userId).First(&existingBasket).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "Basket not found or not owned by you!", nil)
	}

	isBroadcast := true
	if reqData.TargetUserID > 0 {
		isBroadcast = false
		// Optional: Verify target user is a subscriber (active or expired)
		var sub basket.BasketSubscription
		if err := db.Where("user_id = ? AND basket_id = ?", reqData.TargetUserID, reqData.BasketID).First(&sub).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Target user is not a subscriber of this basket!", nil)
		}
	}

	// Create Message
	msg := basket.BasketMessage{
		BasketID:     reqData.BasketID,
		SenderID:     userId,
		SenderType:   basket.SenderAMC,
		TargetUserID: reqData.TargetUserID,
		Action:       reqData.Action,
		Message:      reqData.Message,
		IsBroadcast:  isBroadcast,
	}

	if err := db.Create(&msg).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send message!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Message sent successfully!", msg)
}

// User Send Message (Direct to AMC)
// POST /basket/message
func UserSendMessage(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	reqData := new(struct {
		BasketID uint   `json:"basketId"`
		Message  string `json:"message"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	if reqData.BasketID == 0 || reqData.Message == "" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "BasketID and Message are required!", nil)
	}

	db := database.Database.Db

	// Verify User has ACTIVE subscription
	var sub basket.BasketSubscription
	if err := db.Where("user_id = ? AND basket_id = ? AND status = ? AND is_deleted = false", userId, reqData.BasketID, basket.SubscriptionActive).First(&sub).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "You must have an active subscription to message the AMC!", nil)
	}

	// Create Message
	msg := basket.BasketMessage{
		BasketID:     reqData.BasketID,
		SenderID:     userId,
		SenderType:   basket.SenderUser,
		Action:       basket.ActionGeneral,
		Message:      reqData.Message,
		IsBroadcast:  false,
		TargetUserID: 0, // Implicitly targets AMC/Basket Owner
	}

	if err := db.Create(&msg).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send message!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Message sent to AMC!", msg)
}

// Get All Messages (Global Inbox for User or AMC)
// GET /basket/messages/all
func GetAllMessages(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	db := database.Database.Db

	// Determine Role
	var user models.User
	if err := db.First(&user, userId).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	var messages []basket.BasketMessage

	if user.Role == "AMC" {
		// AMC: Get messages for all owned baskets
		var ownedBasketIDs []uint
		db.Model(&basket.Basket{}).Where("amc_id = ?", userId).Pluck("id", &ownedBasketIDs)

		if len(ownedBasketIDs) > 0 {
			db.Where("basket_id IN ?", ownedBasketIDs).
				Order("created_at DESC").
				Find(&messages)
		}
	} else {
		// User: Get messages for subscribed baskets (Broadcasts) OR Direct User Messages
		// 1. Get List of Subscribed Baskets (Active or Expired)
		var subBasketIDs []uint
		db.Model(&basket.BasketSubscription{}).Where("user_id = ?", userId).Pluck("basket_id", &subBasketIDs)

		// 2. Fetch Messages
		query := db.Model(&basket.BasketMessage{})

		if len(subBasketIDs) > 0 {
			query = query.Where(
				db.Where("basket_id IN ? AND is_broadcast = true", subBasketIDs).
					Or("sender_id = ?", userId).
					Or("target_user_id = ?", userId),
			)
		} else {
			// No subscriptions, only see own messages (historical?) or direct replies
			query = query.Where("sender_id = ? OR target_user_id = ?", userId, userId)
		}

		query.Order("created_at DESC").Find(&messages)
	}

	// Enrich
	type MessageResponse struct {
		basket.BasketMessage
		SenderName string `json:"senderName"`
		BasketName string `json:"basketName"`
	}

	var response []MessageResponse

	// Optimize: Cache Basket Names and User Names
	basketNames := make(map[uint]string)
	userNames := make(map[uint]string) // UserID -> Name
	amcNames := make(map[uint]string)  // UserID (AMC) -> Name

	for _, m := range messages {
		// Basket Name
		bName, ok := basketNames[m.BasketID]
		if !ok {
			var b basket.Basket
			if err := db.Select("name").First(&b, m.BasketID).Error; err == nil {
				basketNames[m.BasketID] = b.Name
				bName = b.Name
			}
		}

		// Sender Name
		sName := "User"
		if m.SenderType == basket.SenderAMC {
			name, ok := amcNames[m.SenderID]
			if !ok {
				var amcProfile models.AMCProfile
				if err := db.Where("user_id = ?", m.SenderID).First(&amcProfile).Error; err == nil {
					amcNames[m.SenderID] = amcProfile.AmcName
					name = amcProfile.AmcName
				} else {
					amcNames[m.SenderID] = "AMC"
					name = "AMC"
				}
			}
			sName = name
		} else {
			name, ok := userNames[m.SenderID]
			if !ok {
				var u models.User
				if err := db.Select("name").First(&u, m.SenderID).Error; err == nil {
					userNames[m.SenderID] = u.Name
					name = u.Name
				} else {
					name = "User"
				}
			}
			sName = name
		}

		response = append(response, MessageResponse{
			BasketMessage: m,
			SenderName:    sName,
			BasketName:    bName,
		})
	}

	// Sort again just in case (though DB ordered)
	sort.Slice(response, func(i, j int) bool {
		return response[i].CreatedAt.After(response[j].CreatedAt)
	})

	return middleware.JsonResponse(c, fiber.StatusOK, true, "All messages fetched!", response)
}

// Get Basket Messages (Filtered Feed)
// GET /basket/:id/messages
func GetBasketMessages(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)
	basketId := c.Params("id")

	db := database.Database.Db

	// Check permissions
	var sub basket.BasketSubscription
	isSubscriber := false
	if err := db.Where("user_id = ? AND basket_id = ? AND status = ? AND is_deleted = false", userId, basketId, basket.SubscriptionActive).First(&sub).Error; err == nil {
		isSubscriber = true
	}

	var b basket.Basket
	isOwner := false
	if err := db.Where("id = ? AND amc_id = ?", basketId, userId).First(&b).Error; err == nil {
		isOwner = true
	}

	var user models.User
	isAdmin := false
	if err := db.First(&user, userId).Error; err == nil && (user.Role == "ADMIN" || user.Role == "SUPER-ADMIN") {
		isAdmin = true
	}

	if !isSubscriber && !isOwner && !isAdmin {
		return middleware.JsonResponse(c, fiber.StatusForbidden, false, "Access Denied!", nil)
	}

	var messages []basket.BasketMessage

	if isOwner || isAdmin {
		// AMC/Admin: See everything
		db.Where("basket_id = ?", basketId).
			Order("created_at DESC").
			Find(&messages)
	} else {
		// Subscriber: See Broadcasts, Own Messages, and Messages Directed to Me
		db.Where("basket_id = ?", basketId).
			Where("is_broadcast = true OR sender_id = ? OR target_user_id = ?", userId, userId).
			Order("created_at DESC").
			Find(&messages)
	}

	// Enrich
	type MessageResponse struct {
		basket.BasketMessage
		SenderName string `json:"senderName"`
	}

	var response []MessageResponse
	for _, m := range messages {
		var senderName string
		if m.SenderType == basket.SenderAMC {
			var amcProfile models.AMCProfile
			if err := db.Where("user_id = ?", m.SenderID).First(&amcProfile).Error; err == nil {
				senderName = amcProfile.AmcName
			} else {
				senderName = "AMC"
			}
		} else {
			if isOwner || isAdmin || m.SenderID == userId {
				var u models.User
				if err := db.Where("id = ?", m.SenderID).First(&u).Error; err == nil {
					senderName = u.Name
				}
			} else {
				senderName = "User"
			}
		}

		response = append(response, MessageResponse{
			BasketMessage: m,
			SenderName:    senderName,
		})
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Messages fetched!", response)
}
