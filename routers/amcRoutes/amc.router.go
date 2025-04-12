package amcRoutes

import (
	amcControllers "fib/controllers/amcControllers"
	"fib/middleware"
	amcValidators "fib/validators/amcValidator"

	"github.com/gofiber/fiber/v2"
)

func SetupAMCRoutes(app *fiber.App) {
	userGroup := app.Group("/amc")

	userGroup.Get("/stock/list", amcValidators.StockList(), middleware.JWTMiddleware, amcControllers.StockList)
}
