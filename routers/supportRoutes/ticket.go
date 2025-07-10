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
	support.Get("/list", validator.TicketList(), middleware.JWTMiddleware, controller.TicketList)
	support.Get("/admin-list", validator.AdminTicketList(), middleware.JWTMiddleware, controller.AdminTicketList)
	support.Post("/admin-replay", validator.AdminReplyTicket(), middleware.JWTMiddleware, controller.AdminReplyTicket)
	support.Post("/user-replay", validator.AdminReplyTicket(), middleware.JWTMiddleware, controller.UserReplyTicket)
}
