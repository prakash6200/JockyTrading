package courseRoutes

import (
	controllers "fib/controllers/course"
	"fib/middleware"
	validators "fib/validators/course"

	"github.com/gofiber/fiber/v2"
)

// SetupCourseRoutes sets up all user-facing course routes
func SetupCourseRoutes(app *fiber.App) {
	userGroup := app.Group("/course")

	// Course listing and details (public published courses)
	userGroup.Get("/list", middleware.JWTMiddleware, validators.CourseList(), controllers.GetAllCourses)
	userGroup.Get("/:id", middleware.JWTMiddleware, validators.GetCourseDetail(), controllers.GetCourseDetails)

	// Enrollment
	userGroup.Post("/:id/enroll", middleware.JWTMiddleware, validators.EnrollCourse(), controllers.EnrollInCourse)

	// Content viewing (for enrolled users)
	userGroup.Get("/:id/content", middleware.JWTMiddleware, validators.CourseContentList(), controllers.GetCourseContent)
	userGroup.Get("/:course_id/module/:module_id/day/:day", middleware.JWTMiddleware, validators.GetDayContent(), controllers.GetDayContent)

	// Content completion
	userGroup.Post("/:course_id/content/:content_id/complete", middleware.JWTMiddleware, validators.MarkContentComplete(), controllers.MarkContentComplete)

	// MCQ submission
	userGroup.Post("/:course_id/content/:content_id/mcq/submit", middleware.JWTMiddleware, validators.SubmitMCQ(), controllers.SubmitMCQAnswer)

	// Progress tracking
	userGroup.Get("/:course_id/progress", middleware.JWTMiddleware, validators.GetCourseProgress(), controllers.GetUserProgress)

	// User enrollments and certificates
	userEnrollGroup := app.Group("/user")
	userEnrollGroup.Get("/enrollments", middleware.JWTMiddleware, controllers.GetUserEnrollmentsList)
	userEnrollGroup.Get("/certificates", middleware.JWTMiddleware, controllers.GetUserCertificates)

	// Certificate request
	userGroup.Post("/:course_id/certificate/request", middleware.JWTMiddleware, validators.RequestCertificateValidator(), controllers.RequestCertificate)
}
