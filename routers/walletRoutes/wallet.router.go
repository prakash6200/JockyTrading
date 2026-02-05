package walletRoutes

import (
	walletController "fib/controllers/wallet"
	"fib/middleware"
	walletValidator "fib/validators/wallet"

	"github.com/gofiber/fiber/v2"
)

func SetupWalletRoutes(app *fiber.App) {
	walletGroup := app.Group("/wallet")

	// User routes
	walletGroup.Get("/balance", middleware.JWTMiddleware, walletController.GetWalletBalance)
	walletGroup.Post("/deposit", walletValidator.Deposit(), middleware.JWTMiddleware, walletController.DepositToWallet)
	walletGroup.Get("/history", middleware.JWTMiddleware, walletController.GetWalletHistory)

	// Admin routes
	adminGroup := walletGroup.Group("/admin")
	adminGroup.Post("/add-balance", walletValidator.AddBalance(), middleware.JWTMiddleware, walletController.AddBalance)
	adminGroup.Post("/deduct-balance", walletValidator.DeductBalance(), middleware.JWTMiddleware, walletController.DeductBalance)
	adminGroup.Get("/user-balance", middleware.JWTMiddleware, walletController.GetUserBalance)
	adminGroup.Get("/user-history", middleware.JWTMiddleware, walletController.GetUserWalletHistory)
}
