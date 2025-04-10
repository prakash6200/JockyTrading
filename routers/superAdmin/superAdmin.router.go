package superAdminRoutes

import (
	superAdminController "fib/controllers/superAdmin"
	"fib/middleware"
	superAdminValidator "fib/validators/superAdmin"
	"github.com/gofiber/fiber/v2"
)

func SetupSuperAdminRoutes(app *fiber.App) {
	userGroup := app.Group("/super-admin")

	userGroup.Get("/user/list", superAdminValidator.UserList(), middleware.JWTMiddleware, superAdminController.UserList)
}
