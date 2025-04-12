package amcController

import (
	"encoding/csv"
	"fib/config"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func SyncStockHandler(c *fiber.Ctx) error {
	go FetchAndStoreStocks()
	return c.JSON(fiber.Map{"message": "Stock sync started"})
}

func FetchAndStoreStocks() {
	url := "https://www.alphavantage.co/query?function=LISTING_STATUS&apikey=" + config.AppConfig.AlphaVantageApiKey

	res, err := http.Get(url)
	if err != nil {
		log.Println("Failed to fetch stock list:", err)
		return
	}
	defer res.Body.Close()

	reader := csv.NewReader(res.Body)
	records, err := reader.ReadAll()
	if err != nil {
		log.Println("Failed to parse stock CSV:", err)
		return
	}

	for _, row := range records[1:] {
		symbol := row[0]
		name := row[1]
		length := len(row)
		status := row[length-1]

		if status == "Active" {
			stock := models.Stocks{Symbol: symbol, Name: name}
			database.Database.Db.FirstOrCreate(&stock, models.Stocks{Symbol: symbol})
		}
	}

	log.Println("Stock list sync completed")
}
