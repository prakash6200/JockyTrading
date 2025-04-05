package authRoutes

import (
	authControllers "fib/controllers/auth"
	"fib/middleware"
	authValidators "fib/validators/auth"

	"github.com/gofiber/fiber/v2"
)

func SetupAuthRoutes(app *fiber.App) {
	authGroup := app.Group("/auth")

	authGroup.Post("/signup", authValidators.Signup(), authControllers.Signup)
	authGroup.Post("/login", authValidators.Login(), authControllers.Login)
	authGroup.Get("/login/history", authValidators.LoginHistoryList(), middleware.JWTMiddleware, authControllers.LoginHistoryList)
	authGroup.Post("/send/otp", authValidators.SendOTP(), authControllers.SendOTP)
	authGroup.Patch("/verify/otp", authValidators.VerifyOTP(), authControllers.VerifyOTP)
	authGroup.Post("/forgot/password/send/otp", authValidators.SendOTP(), authControllers.ForgotPasswordSendOTP)
	authGroup.Patch("/forgot/password/verify/otp", authValidators.VerifyOTP(), authControllers.ForgotPasswordVerifyOTP)
	authGroup.Patch("/reset/password", authValidators.ResetPassword(), middleware.JWTMiddleware, authControllers.ResetPassword)
	authGroup.Put("/change/login/password", authValidators.ChangeLoginPassword(), middleware.JWTMiddleware, authControllers.ChangeLoginPassword)
}
