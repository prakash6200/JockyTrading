package main

import (
	"fib/config"
	stockCronController "fib/controllers/amc"
	"fib/database"
	amcRoutes "fib/routers/amcRoutes"
	authRoutes "fib/routers/authRoutes"
	courseRoutes "fib/routers/courseRoutes"
	superAdminRoutes "fib/routers/superAdmin"
	supportRoutes "fib/routers/supportRoutes"
	userProfileRoutes "fib/routers/userRoutes"

	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/robfig/cron/v3"
)

func main() {
	config.LoadConfig()
	database.ConnectDb()

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE",        // Allowed HTTP methods
		AllowHeaders: "Content-Type,Authorization", // Allowed headers
	}))

	// Enable the built-in logger middleware to log all requests
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip} ${method} ${path} ${status} ${latency}\n",
	}))

	// Serve static files from the public folder
	app.Static("/", "./public")

	authRoutes.SetupAuthRoutes(app)
	userProfileRoutes.SetupUserRoutes(app)
	superAdminRoutes.SetupSuperAdminRoutes(app)
	amcRoutes.SetupAMCRoutes(app)
	amcRoutes.AMCProfileRoutes(app)
	amcRoutes.SetupAMCPredictionRoutes(app)
	courseRoutes.SetupCourseRoutes(app)
	courseRoutes.SetupAdminCourseRoutes(app)
	supportRoutes.SetupSupportRoutes(app)

	// startCron()

	log.Printf("Server is running on port %s", config.AppConfig.Port)
	log.Fatal(app.Listen(":" + config.AppConfig.Port))
}

func startCron() {
	stockCronController.FetchAndStoreStocks()
	// stockCronController.SyncStockPrices()
	c := cron.New()
	c.AddFunc("0 6 * * *", func() {
		log.Println("Running daily stock sync cron job...")
		stockCronController.FetchAndStoreStocks()
	})
	c.Start()
}
