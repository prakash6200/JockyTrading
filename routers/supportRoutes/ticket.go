package supportRoutes

import (
	controller "fib/controllers/support"
	"fib/middleware"
	validator "fib/validators/support"
	"github.com/gofiber/fiber/v2"
)

func SetupSupportRoutes(app *fiber.App) {
	support := app.Group("/support")

	support.Post("/create", validator.CreateSupportTicket(), middleware.JWTMiddleware, controller.CreateSupportTicket)
}
