package main

import (
	"fib/config"
	"fib/database"
	authRoutes "fib/routers/authRoutes"
	superAdminRoutes "fib/routers/superAdmin"
	userProfileRoutes "fib/routers/userRoutes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"log"
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

	log.Printf("Server is running on port %s", config.AppConfig.Port)
	log.Fatal(app.Listen(":" + config.AppConfig.Port))
}
