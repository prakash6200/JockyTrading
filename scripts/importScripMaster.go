package main

import (
	"encoding/csv"
	"fib/config"
	"fib/database"
	"fib/models"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	// Load config and connect to database
	config.LoadConfig()
	database.ConnectDb()

	// Open CSV file
	file, err := os.Open("ScripMaster.csv")
	if err != nil {
		log.Fatalf("Failed to open CSV file: %v", err)
	}
	defer file.Close()

	// Create CSV reader
	reader := csv.NewReader(file)

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) < 2 {
		log.Fatal("CSV file is empty or has only headers")
	}

	// Skip header row
	header := records[0]
	log.Printf("CSV Headers: %v", header)
	log.Printf("Total rows to import: %d", len(records)-1)

	// Map header indices
	headerIndex := make(map[string]int)
	for i, h := range header {
		headerIndex[strings.TrimSpace(h)] = i
	}

	inserted := 0
	updated := 0
	skipped := 0

	for i, row := range records[1:] {
		if i%1000 == 0 {
			log.Printf("Processing row %d...", i+1)
		}

		// Parse fields from CSV
		stock := models.Stocks{
			ExchID:         getField(row, headerIndex, "exchId"),
			Token:          parseInt(getField(row, headerIndex, "token")),
			Symbol:         getField(row, headerIndex, "symbol"),
			Series:         getField(row, headerIndex, "series"),
			FullName:       getField(row, headerIndex, "fullName"),
			Expiry:         getField(row, headerIndex, "expiry"),
			StrikePrice:    parseFloat(getField(row, headerIndex, "strikeprice")),
			MarketLot:      parseInt(getField(row, headerIndex, "mktLot")),
			InstrumentType: getField(row, headerIndex, "instType"),
			ISIN:           getField(row, headerIndex, "isin"),
			FaceValue:      parseFloat(getField(row, headerIndex, "faceValue")),
			TickSize:       parseFloat(getField(row, headerIndex, "tick")),
			Sector:         getField(row, headerIndex, "Sector"),
			Industry:       getField(row, headerIndex, "Industry"),
			MarketCap:      parseFloat(getField(row, headerIndex, "MktCap")),
			MarketCapType:  getField(row, headerIndex, "MktCapType"),
			IndexSymbol:    getField(row, headerIndex, "indexsymbol"),
			Name:           getField(row, headerIndex, "fullName"), // Use fullName as Name
			Exchange:       getField(row, headerIndex, "exchId"),   // Use exchId as Exchange
			IsDeleted:      false,
		}

		// Skip if no symbol or token
		if stock.Symbol == "" || stock.Token == 0 {
			skipped++
			continue
		}

		// Check if stock exists by token
		var existing models.Stocks
		result := database.Database.Db.Where("token = ?", stock.Token).First(&existing)

		if result.Error != nil {
			// Insert new stock
			if err := database.Database.Db.Create(&stock).Error; err != nil {
				log.Printf("Error inserting stock %s (token=%d): %v", stock.Symbol, stock.Token, err)
				continue
			}
			inserted++
		} else {
			// Update existing stock
			existing.ExchID = stock.ExchID
			existing.Symbol = stock.Symbol
			existing.Series = stock.Series
			existing.FullName = stock.FullName
			existing.Expiry = stock.Expiry
			existing.StrikePrice = stock.StrikePrice
			existing.MarketLot = stock.MarketLot
			existing.InstrumentType = stock.InstrumentType
			existing.ISIN = stock.ISIN
			existing.FaceValue = stock.FaceValue
			existing.TickSize = stock.TickSize
			existing.Sector = stock.Sector
			existing.Industry = stock.Industry
			existing.MarketCap = stock.MarketCap
			existing.MarketCapType = stock.MarketCapType
			existing.IndexSymbol = stock.IndexSymbol
			existing.Name = stock.Name
			existing.Exchange = stock.Exchange

			if err := database.Database.Db.Save(&existing).Error; err != nil {
				log.Printf("Error updating stock %s (token=%d): %v", stock.Symbol, stock.Token, err)
				continue
			}
			updated++
		}
	}

	log.Printf("=== Import Complete ===")
	log.Printf("Inserted: %d", inserted)
	log.Printf("Updated: %d", updated)
	log.Printf("Skipped: %d", skipped)
	log.Printf("Total processed: %d", inserted+updated+skipped)
}

// getField safely gets a field from the row by header name
func getField(row []string, headerIndex map[string]int, field string) string {
	if idx, ok := headerIndex[field]; ok && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

// parseInt converts string to int
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

// parseFloat converts string to float64
func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}
