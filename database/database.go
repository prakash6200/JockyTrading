package database

import (
	"fib/models"
	"fib/models/basket"
	course "fib/models/course"
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

	// Pre-migration: Fix existing NULL values in transactions table
	db.Exec("ALTER TABLE transactions ADD COLUMN IF NOT EXISTS transaction_type TEXT DEFAULT 'DEPOSIT'")
	db.Exec("ALTER TABLE transactions ADD COLUMN IF NOT EXISTS amount BIGINT DEFAULT 0")
	db.Exec("ALTER TABLE transactions ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'pending'")

	db.Exec("UPDATE transactions SET transaction_type = 'DEPOSIT' WHERE transaction_type IS NULL OR transaction_type = ''")
	db.Exec("UPDATE transactions SET amount = 0 WHERE amount IS NULL")
	db.Exec("UPDATE transactions SET status = 'pending' WHERE status IS NULL")

	// Drop foreign key constraint on baskets.current_version_id if it exists (to avoid circular dependency)
	db.Exec("ALTER TABLE baskets DROP CONSTRAINT IF EXISTS fk_baskets_current_version")

	// First, create the baskets table without the CurrentVersion relation
	db.Exec(`CREATE TABLE IF NOT EXISTS baskets (
		id bigserial PRIMARY KEY,
		created_at timestamptz,
		updated_at timestamptz,
		deleted_at timestamptz,
		name text NOT NULL,
		description text,
		amc_id bigint NOT NULL,
		basket_type varchar(20) NOT NULL,
		current_version_id bigint,
		subscription_fee decimal DEFAULT 0,
		is_fee_based boolean DEFAULT false,
		is_deleted boolean DEFAULT false
	)`)

	// Now migrate all tables - baskets already exists, others will be created
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
		&models.Transactions{},
		&models.Folio{},
		&models.StockPrices{},
		&models.Permission{},
		&course.Course{},
		&course.Module{},
		&course.CourseContent{},
		&course.MCQOption{},
		&course.MCQAttempt{},
		&models.SupportTicket{},
		&course.Enrollment{},
		&course.ContentCompletion{},
		&course.CertificateRequest{},
		&course.Certificate{},
		&models.Maintenance{},
		&models.Review{},
		&models.BajajAccessToken{},
		&models.WalletTransaction{},
		&basket.Basket{},
		&basket.BasketVersion{},
		&basket.BasketTimeSlot{},
		&basket.BasketStock{},
		&basket.BasketSubscription{},
		&basket.BasketHistory{},
		&basket.BasketReview{},
		&basket.BasketMessage{},
	)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migrations completed successfully.")
}
