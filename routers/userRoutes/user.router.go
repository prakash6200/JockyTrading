package userProfileRoutes

import (
	userProfileController "fib/controllers/userControllers"
	"fib/middleware"
	userPorfileValidator "fib/validators/userValidator"

	"github.com/gofiber/fiber/v2"
)

func SetupUserRoutes(app *fiber.App) {
	userGroup := app.Group("/user")

	userGroup.Post("/add/bank/account", userPorfileValidator.AddBankAccount(), middleware.JWTMiddleware, userProfileController.AddBankAccount)
	userGroup.Post("/send/adhar/otp", userPorfileValidator.SendAdharOtp(), middleware.JWTMiddleware, userProfileController.SendAdharOtp)
	userGroup.Post("/verify/adhar/otp", userPorfileValidator.VerifyAdharOtp(), middleware.JWTMiddleware, userProfileController.VerifyAdharOtp)
	userGroup.Post("/pan/adhar/link/status", userProfileController.PanLinkStatus)
	userGroup.Post("/add/folio/number", userPorfileValidator.AddFolioNumber(), middleware.JWTMiddleware, userProfileController.AddFolioNumber)
	userGroup.Post("/deposit/amount", userPorfileValidator.Deposit(), middleware.JWTMiddleware, userProfileController.Deposit)
	userGroup.Post("/Withdraw/amount", userPorfileValidator.Withdraw(), middleware.JWTMiddleware, userProfileController.Withdraw)
	userGroup.Get("/transaction/list", userPorfileValidator.TransactionList(), middleware.JWTMiddleware, userProfileController.TransactionList)
}
