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
	userGroup.Post("/select/stock", amcValidators.AmcPickUnpickStockValidator(), middleware.JWTMiddleware, amcControllers.AmcPickUnpickStock)
	userGroup.Get("/picked/stock/list", amcValidators.StockPickedByAMCList(), middleware.JWTMiddleware, amcControllers.StockPickedByAMCList)
	userGroup.Get("/performance", amcValidators.AmcPerformance(), middleware.JWTMiddleware, amcControllers.AmcPerformance)
}
