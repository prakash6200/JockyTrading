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
	userGroup.Post("/:id/enrollment", validators.EnrollCourse(), middleware.JWTMiddleware, controllers.EnrollInCourse)
	userGroup.Get("/:id/enrollment", validators.GetUserEnrollments(), middleware.JWTMiddleware, controllers.GetEnrollments)
	userGroup.Post("/:course_id/content/:content_id/complete", validators.MarkContentComplete(), middleware.JWTMiddleware, controllers.MarkContentComplete)
	userGroup.Get("/:course_id/completions", validators.GetContentCompletions(), middleware.JWTMiddleware, controllers.GetContentCompletions)

}
