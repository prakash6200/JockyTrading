package courseRoutes

import (
	controllers "fib/controllers/course"
	"fib/middleware"
	validators "fib/validators/course"

	"github.com/gofiber/fiber/v2"
)

// SetupAdminCourseRoutes sets up all admin course management routes
func SetupAdminCourseRoutes(app *fiber.App) {
	adminGroup := app.Group("/admin/course")

	// Course CRUD
	adminGroup.Post("/create", middleware.JWTMiddleware, validators.CreateCourseAdmin(), controllers.AdminCreateCourse)
	adminGroup.Put("/:id", middleware.JWTMiddleware, validators.UpdateCourseAdmin(), controllers.AdminUpdateCourse)
	adminGroup.Delete("/:id", middleware.JWTMiddleware, validators.DeleteCourse(), controllers.AdminDeleteCourse)
	adminGroup.Get("/list", middleware.JWTMiddleware, validators.AdminList(), controllers.AdminGetAllCourses)
	adminGroup.Get("/:id", middleware.JWTMiddleware, validators.DeleteCourse(), controllers.AdminGetCourseDetails)
	adminGroup.Post("/:id/publish", middleware.JWTMiddleware, validators.PublishCourse(), controllers.AdminPublishCourse)

	// Module Management
	adminGroup.Post("/:id/module", middleware.JWTMiddleware, validators.CreateModule(), controllers.AdminCreateModule)
	adminGroup.Put("/:course_id/module/:module_id", middleware.JWTMiddleware, validators.UpdateModule(), controllers.AdminUpdateModule)
	adminGroup.Delete("/:course_id/module/:module_id", middleware.JWTMiddleware, validators.DeleteModule(), controllers.AdminDeleteModule)
	adminGroup.Get("/:id/modules", middleware.JWTMiddleware, validators.ListModules(), controllers.AdminListModules)

	// Content Management
	adminGroup.Post("/:course_id/module/:module_id/content", middleware.JWTMiddleware, validators.CreateContentAdmin(), controllers.AdminCreateContent)
	adminGroup.Get("/:course_id/module/:module_id/content", middleware.JWTMiddleware, validators.DeleteModule(), controllers.AdminGetModuleContent)

	// Content endpoints (separate from course group for easier access)
	contentGroup := app.Group("/admin/content")
	contentGroup.Put("/:content_id", middleware.JWTMiddleware, validators.UpdateContentAdmin(), controllers.AdminUpdateContent)
	contentGroup.Delete("/:content_id", middleware.JWTMiddleware, validators.DeleteContentAdmin(), controllers.AdminDeleteContent)
	contentGroup.Post("/:content_id/publish", middleware.JWTMiddleware, validators.PublishContentAdmin(), controllers.AdminPublishContent)

	// MCQ Management
	contentGroup.Post("/:content_id/mcq", middleware.JWTMiddleware, validators.AddMCQOption(), controllers.AdminAddMCQOption)

	mcqGroup := app.Group("/admin/mcq")
	mcqGroup.Put("/:option_id", middleware.JWTMiddleware, validators.UpdateMCQOption(), controllers.AdminUpdateMCQOption)
	mcqGroup.Delete("/:option_id", middleware.JWTMiddleware, validators.DeleteMCQOption(), controllers.AdminDeleteMCQOption)

	// Enrollment & Progress Tracking
	adminGroup.Get("/:id/enrollments", middleware.JWTMiddleware, validators.GetCourseEnrollments(), controllers.AdminGetCourseEnrollments)
	adminGroup.Get("/:id/completed", middleware.JWTMiddleware, validators.GetCourseEnrollments(), controllers.AdminGetCompletedStudents)

	studentGroup := app.Group("/admin/student")
	studentGroup.Get("/:user_id/progress", middleware.JWTMiddleware, validators.GetStudentProgress(), controllers.AdminGetStudentProgress)

	// Certificate Management
	certGroup := app.Group("/admin/certificates")
	certGroup.Get("/pending", middleware.JWTMiddleware, validators.GetPendingCertificates(), controllers.AdminGetPendingCertificates)
	certGroup.Get("/issued", middleware.JWTMiddleware, validators.GetPendingCertificates(), controllers.AdminGetIssuedCertificates)

	certRequestGroup := app.Group("/admin/certificate")
	certRequestGroup.Post("/:request_id/approve", middleware.JWTMiddleware, validators.ApproveCertificate(), controllers.AdminApproveCertificate)
	certRequestGroup.Post("/:request_id/reject", middleware.JWTMiddleware, validators.RejectCertificate(), controllers.AdminRejectCertificate)

	// Dashboard
	dashGroup := app.Group("/admin/dashboard")
	dashGroup.Get("/stats", middleware.JWTMiddleware, controllers.AdminDashboardStats)
}
