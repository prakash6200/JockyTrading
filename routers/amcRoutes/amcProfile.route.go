package amcRoutes

import (
	amcControllers "fib/controllers/amcControllers"
	"fib/middleware"
	"github.com/gofiber/fiber/v2"
)

func AMCProfileRoutes(app fiber.Router) {
	amcGroup := app.Group("/amc")

	amcGroup.Post("/profle-update", middleware.JWTMiddleware, amcControllers.CreateOrUpdateAMCProfile)
	amcGroup.Get("/get-profle", middleware.JWTMiddleware, amcControllers.GetAMCProfile)
}
