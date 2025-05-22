package superAdminRoutes

import (
	superAdminController "fib/controllers/superAdmin"
	"fib/middleware"
	superAdminValidator "fib/validators/superAdmin"
	"github.com/gofiber/fiber/v2"
)

func SetupSuperAdminRoutes(app *fiber.App) {
	adminGroup := app.Group("/admin")

	adminGroup.Get("/user/list", superAdminValidator.List(), middleware.JWTMiddleware, superAdminController.UserList)
	adminGroup.Post("/register-amc", superAdminValidator.RegisterAMC(), middleware.JWTMiddleware, superAdminController.RegisterAMC)
	adminGroup.Get("/transaction/list", superAdminValidator.List(), middleware.JWTMiddleware, superAdminController.TransactionList)
}
