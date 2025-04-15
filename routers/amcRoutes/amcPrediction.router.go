package amcRoutes

import (
	amcControllers "fib/controllers/amcControllers"
	"fib/middleware"
	amcValidators "fib/validators/amcValidator"

	"github.com/gofiber/fiber/v2"
)

func SetupAMCPredictionRoutes(app *fiber.App) {
	predictionGroup := app.Group("/amc/predictions")

	// Prediction management
	predictionGroup.Post("", amcValidators.AMCPredictionValidator(), middleware.JWTMiddleware, amcControllers.CreateOrUpdatePrediction)
	predictionGroup.Get("", amcValidators.AMCPredictionListValidator(), middleware.JWTMiddleware, amcControllers.GetPredictions)
	predictionGroup.Get("/:id", middleware.JWTMiddleware, amcControllers.GetPrediction)
	predictionGroup.Delete("/:id", middleware.JWTMiddleware, amcControllers.DeletePrediction)
	predictionGroup.Patch("/achieved", amcValidators.AMCAchievedValueValidator(), middleware.JWTMiddleware, amcControllers.UpdateAchievedValue)
}
