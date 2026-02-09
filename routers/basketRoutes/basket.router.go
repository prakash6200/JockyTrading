package basketRoutes

import (
	basketController "fib/controllers/basket"
	"fib/middleware"
	basketValidator "fib/validators/basket"

	"github.com/gofiber/fiber/v2"
)

// SetupAMCBasketRoutes sets up AMC basket management routes
func SetupAMCBasketRoutes(app *fiber.App) {
	amcGroup := app.Group("/amc/basket")

	// Basket CRUD
	amcGroup.Post("/create", basketValidator.CreateBasket(), middleware.JWTMiddleware, basketController.CreateBasket)
	amcGroup.Put("/update", basketValidator.UpdateBasket(), middleware.JWTMiddleware, basketController.UpdateBasket)
	amcGroup.Get("/list", basketValidator.ListBaskets(), middleware.JWTMiddleware, basketController.GetMyBaskets)

	// Stocks list for adding to basket
	amcGroup.Get("/stocks-list", middleware.JWTMiddleware, basketController.GetStocksList)
	amcGroup.Get("/stock-by-token", middleware.JWTMiddleware, basketController.GetStockByToken)
	amcGroup.Get("/stock-by-symbol", middleware.JWTMiddleware, basketController.GetStockBySymbol)

	// Stock management
	amcGroup.Post("/stocks/add", basketValidator.AddStocks(), middleware.JWTMiddleware, basketController.AddStocksToBasket)
	amcGroup.Post("/stocks/remove", basketValidator.RemoveStock(), middleware.JWTMiddleware, basketController.RemoveStockFromBasket)
	amcGroup.Post("/stocks/add-with-pricing", basketValidator.AddStocksWithToken(), middleware.JWTMiddleware, basketController.AddStocksWithPricing)

	// Review Management (AMC)
	amcGroup.Get("/reviews/all", middleware.JWTMiddleware, basketController.GetAMCReviews)
	amcGroup.Post("/review/moderate", middleware.JWTMiddleware, basketController.ModerateReview)

	// Bajaj token management (AMC can also set token)
	amcGroup.Post("/set-access-token", basketValidator.SetBajajAccessToken(), middleware.JWTMiddleware, basketController.SetBajajAccessToken)

	// Approval workflow
	amcGroup.Post("/submit", basketValidator.SubmitForApproval(), middleware.JWTMiddleware, basketController.SubmitForApproval)

	// Subscribers list
	amcGroup.Get("/:id/subscribers", middleware.JWTMiddleware, basketController.GetBasketSubscribers)

	// Messaging (AMC Broadcast)
	amcGroup.Post("/message", middleware.JWTMiddleware, basketController.AMCSendMessage)
	amcGroup.Get("/messages/all", middleware.JWTMiddleware, basketController.GetAllMessages) // Global Inbox

	// Detailed basket view (MUST come before /:id)
	amcGroup.Get("/details/:id", middleware.JWTMiddleware, basketController.GetAMCBasketDetails)

	// Get basket by ID (MUST be last - catches all /:id patterns)
	amcGroup.Get("/:id", middleware.JWTMiddleware, basketController.GetBasketHistory)
}

// SetupAdminBasketRoutes sets up admin basket management routes
func SetupAdminBasketRoutes(app *fiber.App) {
	adminGroup := app.Group("/admin/basket")

	// Dashboard and stats
	adminGroup.Get("/stats", middleware.JWTMiddleware, basketController.GetDashboardStats)
	adminGroup.Get("/list", middleware.JWTMiddleware, basketController.ListAllBaskets)
	adminGroup.Get("/details/:id", middleware.JWTMiddleware, basketController.GetAdminBasketDetails)
	adminGroup.Get("/subscribers", middleware.JWTMiddleware, basketController.GetAllSubscribers)

	// Subscription management (Admin)
	adminGroup.Get("/subscriptions", middleware.JWTMiddleware, basketController.GetAllActiveSubscriptions)
	adminGroup.Get("/subscriptions/expiring", middleware.JWTMiddleware, basketController.GetExpiringSubscriptions)
	adminGroup.Post("/subscription/send-reminder", middleware.JWTMiddleware, basketController.SendExpiryReminder)

	// Approval management
	adminGroup.Get("/pending", basketValidator.ListPendingApprovals(), middleware.JWTMiddleware, basketController.ListPendingApprovals)
	adminGroup.Post("/approve", basketValidator.ApproveBasket(), middleware.JWTMiddleware, basketController.ApproveBasket)
	adminGroup.Post("/reject", basketValidator.RejectBasket(), middleware.JWTMiddleware, basketController.RejectBasket)

	// Basket management
	adminGroup.Post("/unpublish", middleware.JWTMiddleware, basketController.UnpublishBasket)
	adminGroup.Delete("/delete", middleware.JWTMiddleware, basketController.AdminDeleteBasket)

	// Time slot management (INTRA_HOUR)
	adminGroup.Post("/time-slot", basketValidator.SetTimeSlot(), middleware.JWTMiddleware, basketController.SetTimeSlot)

	// Calendar and audit
	adminGroup.Get("/calendar", basketValidator.GetCalendarView(), middleware.JWTMiddleware, basketController.GetCalendarView)
	adminGroup.Get("/audit/:id", middleware.JWTMiddleware, basketController.GetAuditLog)

	// Basket subscribers (admin)
	adminGroup.Get("/:id/subscribers", middleware.JWTMiddleware, basketController.GetBasketSubscribersAdmin)

	// Bajaj token management (Admin)
	adminGroup.Post("/set-access-token", basketValidator.SetBajajAccessToken(), middleware.JWTMiddleware, basketController.SetBajajAccessToken)
	adminGroup.Get("/access-token", middleware.JWTMiddleware, basketController.GetLatestBajajToken)
}

// SetupUserBasketRoutes sets up user-facing basket routes
func SetupUserBasketRoutes(app *fiber.App) {
	userGroup := app.Group("/basket")

	// User - Subscribe
	userGroup.Post("/subscribe", middleware.JWTMiddleware, basketValidator.Subscribe(), basketController.Subscribe)

	// Reviews (User)
	userGroup.Post("/:id/review", middleware.JWTMiddleware, basketController.SubmitReview)
	userGroup.Get("/:id/reviews", basketController.GetPublicReviews)

	// Browse baskets (specific routes MUST come before :id routes)
	userGroup.Get("/list", basketValidator.ListPublishedBaskets(), middleware.JWTMiddleware, basketController.ListPublishedBaskets)
	userGroup.Get("/intra-hour/live", middleware.JWTMiddleware, basketController.GetLiveIntraHourBaskets)
	userGroup.Get("/intra-hour/upcoming", middleware.JWTMiddleware, basketController.GetUpcomingIntraHourBaskets)

	// My Basket & Subscriptions
	userGroup.Get("/my-basket", basketValidator.GetMySubscriptions(), middleware.JWTMiddleware, basketController.GetMySubscriptions)
	userGroup.Get("/my-subscriptions", middleware.JWTMiddleware, basketController.GetMyBasket)

	// Messaging (User)
	userGroup.Post("/message", middleware.JWTMiddleware, basketController.UserSendMessage)
	userGroup.Get("/messages/all", middleware.JWTMiddleware, basketController.GetAllMessages) // Global Inbox
	userGroup.Get("/:id/messages", middleware.JWTMiddleware, basketController.GetBasketMessages)

	// Stock price lookup
	userGroup.Get("/stock-price", basketValidator.GetStockPrice(), middleware.JWTMiddleware, basketController.GetStockPrice)
	userGroup.Get("/stock-price/details", basketValidator.GetStockPrice(), middleware.JWTMiddleware, basketController.GetStockPriceDetails)

	// Stocks list for adding to basket
	userGroup.Get("/stocks-list", middleware.JWTMiddleware, basketController.GetStocksList)
	userGroup.Get("/stock-by-token", middleware.JWTMiddleware, basketController.GetStockByToken)
	userGroup.Get("/stock-by-symbol", middleware.JWTMiddleware, basketController.GetStockBySymbol)

	// Dynamic ID routes (MUST come AFTER specific routes)
	userGroup.Get("/:id", middleware.JWTMiddleware, basketController.GetBasketDetails)
	userGroup.Get("/:id/history", middleware.JWTMiddleware, basketController.GetPublishedHistory)
	userGroup.Get("/:id/pricing", middleware.JWTMiddleware, basketController.GetBasketWithPricing)
}
