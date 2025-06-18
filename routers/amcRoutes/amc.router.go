package amcRoutes

import (
	amcControllers "fib/controllers/amc"
	"fib/middleware"
	amcValidators "fib/validators/amc"

	"github.com/gofiber/fiber/v2"
)

func SetupAMCRoutes(app *fiber.App) {
	userGroup := app.Group("/amc")

	userGroup.Get("/stock/list", amcValidators.StockList(), middleware.JWTMiddleware, amcControllers.StockList)
	// userGroup.Get("/stock/list", amcValidators.StockList(), middleware.JWTMiddleware,
	// 	middleware.CheckPermissionMiddleware("view-profile"), amcControllers.StockList)
	userGroup.Get("/picked/stock/list", middleware.JWTMiddleware, amcControllers.StockPickedByAMCList)
	userGroup.Post("/select/stock", amcValidators.AmcPickUnpickStockValidator(), middleware.JWTMiddleware, amcControllers.AmcPickUnpickStock)
	userGroup.Get("/performance", amcValidators.AmcPerformance(), middleware.JWTMiddleware, amcControllers.AmcPerformance)
	userGroup.Get("/list", amcValidators.AMCList(), middleware.JWTMiddleware, amcControllers.AMCList)
	// userGroup.Get("/list", amcValidators.AMCList(), middleware.JWTMiddleware, amcControllers.AMCList)
}
