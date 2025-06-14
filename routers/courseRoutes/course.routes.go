package courseRoutes

import (
	controllers "fib/controllers/course"
	"fib/middleware"
	validators "fib/validators/course"

	"github.com/gofiber/fiber/v2"
)

func SetupCourseRoutes(app *fiber.App) {
	userGroup := app.Group("/course")

	userGroup.Post("/create", validators.CreateCourse(), middleware.JWTMiddleware, controllers.CreateCourse)
	userGroup.Get("/list", validators.CourseList(), middleware.JWTMiddleware, controllers.GetAllCourses)
	userGroup.Post("/:id/content", validators.CreateCourseContent(), middleware.JWTMiddleware, controllers.CreateCourseContent)
	userGroup.Get("/:id/content", validators.CourseContentList(), middleware.JWTMiddleware, controllers.GetCourseContent)
}

// func SetupAMCRoutes(app *fiber.App) {
// 	userGroup := app.Group("/amc")

// 	userGroup.Get("/stock/list", amcValidators.StockList(), middleware.JWTMiddleware,
// 		middleware.CheckPermissionMiddleware("view-profile"), amcControllers.StockList)
// 	userGroup.Get("/picked/stock/list", amcValidators.StockPickedByAMCList(), middleware.JWTMiddleware, amcControllers.StockPickedByAMCList)
// 	userGroup.Post("/select/stock", amcValidators.AmcPickUnpickStockValidator(), middleware.JWTMiddleware, amcControllers.AmcPickUnpickStock)
// 	userGroup.Get("/performance", amcValidators.AmcPerformance(), middleware.JWTMiddleware, amcControllers.AmcPerformance)
// 	userGroup.Get("/list", amcValidators.AMCList(), middleware.JWTMiddleware, amcControllers.AMCList)
// 	// userGroup.Get("/list", amcValidators.AMCList(), middleware.JWTMiddleware, amcControllers.AMCList)
// }
