package database

import (
	"fib/models"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	// "gorm.io/gorm/logger"
)

// DbInstance struct holds the database connection instance
type DbInstance struct {
	Db *gorm.DB
}

// Database is the global database instance
var Database DbInstance

// ConnectDb establishes a connection to PostgreSQL
// ConnectDb establishes a connection to PostgreSQL
func ConnectDb() {
	// Build the PostgreSQL connection string
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
		os.Exit(2)
	}

	// log.Println("Connected Successfully to PostgreSQL")

	// Set GORM logger to Info mode
	// db.Logger = logger.Default.LogMode(logger.Info)

	// Set up connection pooling
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	sqlDB.SetMaxOpenConns(10)   // Maximum open connections
	sqlDB.SetMaxIdleConns(5)    // Maximum idle connections
	sqlDB.SetConnMaxLifetime(0) // No timeout

	// Run database migrations
	runMigrations(db)

	// Save database instance globally
	Database = DbInstance{Db: db}

}

// runMigrations performs database migrations
func runMigrations(db *gorm.DB) {
	log.Println("Running Migrations...")

	err := db.AutoMigrate(
		&models.User{},
		&models.OTP{},
		&models.LoginTracking{},
		&models.BankDetails{},
		&models.UserKYC{},
		&models.Stocks{},
		&models.AmcStocks{},
		&models.AMCProfile{},
		&models.AMCPredictionValue{},
	)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migrations completed successfully.")
}
