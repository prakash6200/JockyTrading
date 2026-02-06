package utils

import (
	"fib/database"
	"fib/models"
	"fib/models/basket"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// logScheduler logs scheduler events with timestamp
func logScheduler(message string) {
	log.Printf("[BASKET-SCHEDULER %s] %s", time.Now().Format(time.RFC3339), message)
}

// recordSystemHistory records history entry for system-triggered actions
func recordSystemHistory(versionID uint, action string, notes string) {
	history := basket.BasketHistory{
		BasketVersionID: versionID,
		Action:          action,
		ActorID:         0, // System
		ActorType:       basket.ActorSystem,
		Comments:        notes,
	}
	database.Database.Db.Create(&history)
}

// processIntraHourBaskets handles SCHEDULED → PUBLISHED and PUBLISHED → EXPIRED transitions
func processIntraHourBaskets() {
	db := database.Database.Db
	now := time.Now()

	// Auto-PUBLISH: SCHEDULED → PUBLISHED when start_time reached
	var scheduledSlots []basket.BasketTimeSlot
	if err := db.Where("start_time <= ? AND end_time > ?", now, now).
		Preload("BasketVersion").
		Find(&scheduledSlots).Error; err != nil {
		logScheduler("Error fetching scheduled time slots: " + err.Error())
		return
	}

	for _, slot := range scheduledSlots {
		if slot.BasketVersion.Status == basket.StatusScheduled {
			slot.BasketVersion.Status = basket.StatusPublished
			slot.ActualPublishTime = &now

			db.Save(&slot.BasketVersion)
			db.Save(&slot)

			recordSystemHistory(slot.BasketVersionID, basket.ActionWentLive, "Auto-published at time slot start")
			logScheduler("INTRA_HOUR Basket version " + string(rune(slot.BasketVersionID)) + " auto-PUBLISHED")

			// Notify Subscribers (Async)
			go func(v basket.BasketVersion, bName string) {
				var subs []basket.BasketSubscription
				if err := database.Database.Db.Where("basket_id = ? AND status = ? AND is_deleted = false", v.BasketID, basket.SubscriptionActive).Find(&subs).Error; err == nil {
					for _, sub := range subs {
						var u models.User
						if err := database.Database.Db.Select("name, email").First(&u, sub.UserID).Error; err == nil && u.Email != "" {
							SendNewVersionEmail(u.Email, u.Name, bName, v.VersionNumber)
						}
					}
				}
			}(slot.BasketVersion, slot.BasketVersion.Basket.Name)
		}
	}

	// Auto-EXPIRE: PUBLISHED → EXPIRED when end_time reached
	var expiredSlots []basket.BasketTimeSlot
	if err := db.Where("end_time <= ? AND actual_expire_time IS NULL", now).
		Preload("BasketVersion", "status = ?", basket.StatusPublished).
		Find(&expiredSlots).Error; err != nil {
		logScheduler("Error fetching expired time slots: " + err.Error())
		return
	}

	for _, slot := range expiredSlots {
		if slot.BasketVersion.ID > 0 && slot.BasketVersion.Status == basket.StatusPublished {
			slot.BasketVersion.Status = basket.StatusExpired
			slot.ActualExpireTime = &now

			db.Save(&slot.BasketVersion)
			db.Save(&slot)

			// Also expire subscriptions for this basket version
			db.Model(&basket.BasketSubscription{}).
				Where("basket_version_id = ? AND status = ?", slot.BasketVersionID, basket.SubscriptionActive).
				Update("status", basket.SubscriptionExpired)

			recordSystemHistory(slot.BasketVersionID, basket.ActionExpired, "Auto-expired at time slot end")
			logScheduler("INTRA_HOUR Basket version expired at end time")
		}
	}
}

// processIntradayBaskets handles INTRADAY basket expiry at market close
func processIntradayBaskets() {
	db := database.Database.Db
	today := time.Now().Format("2006-01-02")
	now := time.Now()

	// Find all INTRADAY baskets that are PUBLISHED for today
	var versions []basket.BasketVersion
	if err := db.Model(&basket.BasketVersion{}).
		Where("status = ? AND is_deleted = false", basket.StatusPublished).
		Joins("JOIN baskets ON baskets.id = basket_versions.basket_id").
		Where("baskets.basket_type = ? AND baskets.is_deleted = false", basket.BasketTypeIntraday).
		Where("DATE(basket_versions.trading_date) = ? OR basket_versions.trading_date IS NULL", today).
		Find(&versions).Error; err != nil {
		logScheduler("Error fetching INTRADAY baskets: " + err.Error())
		return
	}

	for _, version := range versions {
		version.Status = basket.StatusExpired
		db.Save(&version)

		// Expire subscriptions
		db.Model(&basket.BasketSubscription{}).
			Where("basket_version_id = ? AND status = ?", version.ID, basket.SubscriptionActive).
			Update("status", basket.SubscriptionExpired)

		recordSystemHistory(version.ID, basket.ActionExpired, "Auto-expired at market close")
		logScheduler("INTRADAY Basket version expired at market close")
	}

	logScheduler("Market close: processed INTRADAY basket expiry")
	_ = now // Avoid unused variable warning
}

// StartIntraHourScheduler runs every minute for INTRA_HOUR baskets
func StartIntraHourScheduler(c *cron.Cron) {
	c.AddFunc("* * * * *", func() {
		processIntraHourBaskets()
	})
	logScheduler("INTRA_HOUR scheduler started - runs every minute")
}

// StartIntradayScheduler runs at market close (3:30 PM IST) on weekdays
func StartIntradayScheduler(c *cron.Cron) {
	// 3:30 PM IST = 10:00 AM UTC (IST is UTC+5:30)
	// Run Monday-Friday
	c.AddFunc("30 15 * * 1-5", func() {
		processIntradayBaskets()
	})
	logScheduler("INTRADAY scheduler started - runs at 3:30 PM IST on weekdays")
}

// InitializeBasketSchedulers initializes all basket schedulers
func InitializeBasketSchedulers() *cron.Cron {
	logScheduler("Initializing basket schedulers...")

	// Create cron scheduler with IST timezone
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		loc = time.FixedZone("IST", 5*60*60+30*60)
	}

	c := cron.New(cron.WithLocation(loc))

	StartIntraHourScheduler(c)
	StartIntradayScheduler(c)

	c.Start()

	logScheduler("All basket schedulers initialized successfully")
	return c
}
