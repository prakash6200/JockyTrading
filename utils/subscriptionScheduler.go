package utils

import (
	"fib/database"
	"fib/models"
	"fib/models/basket"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// InitializeSubscriptionScheduler sets up the subscription expiry scheduler
func InitializeSubscriptionScheduler() {
	log.Println("[SUBSCRIPTION-SCHEDULER] Initializing subscription scheduler...")

	c := cron.New()

	// Run daily at 9 AM IST to check expiring subscriptions
	c.AddFunc("0 9 * * *", func() {
		log.Println("[SUBSCRIPTION-SCHEDULER] Running daily subscription check...")
		ProcessExpiringSubscriptions()
		ExpireSubscriptions()
	})

	c.Start()
	log.Println("[SUBSCRIPTION-SCHEDULER] Subscription scheduler started - runs daily at 9 AM IST")
}

// ProcessExpiringSubscriptions sends reminder emails for subscriptions expiring in 2 days
func ProcessExpiringSubscriptions() {
	db := database.Database.Db
	now := time.Now()
	twoDaysFromNow := now.AddDate(0, 0, 2)

	// Find subscriptions expiring in ~2 days that haven't received a reminder
	var expiringSubscriptions []basket.BasketSubscription
	if err := db.
		Where("status = ? AND reminder_sent = false AND expires_at IS NOT NULL", basket.SubscriptionActive).
		Where("expires_at BETWEEN ? AND ?", now, twoDaysFromNow).
		Preload("Basket").
		Find(&expiringSubscriptions).Error; err != nil {
		log.Printf("[SUBSCRIPTION-SCHEDULER] Error fetching expiring subscriptions: %v", err)
		return
	}

	log.Printf("[SUBSCRIPTION-SCHEDULER] Found %d subscriptions expiring soon", len(expiringSubscriptions))

	for _, sub := range expiringSubscriptions {
		// Get user details
		var user models.User
		if err := db.Where("id = ?", sub.UserID).First(&user).Error; err != nil {
			log.Printf("[SUBSCRIPTION-SCHEDULER] Error fetching user %d: %v", sub.UserID, err)
			continue
		}

		// Send reminder email
		SendSubscriptionExpiryReminder(user.Email, user.Name, sub.Basket.Name, sub.ExpiresAt)

		// Mark reminder as sent
		db.Model(&sub).Update("reminder_sent", true)
		log.Printf("[SUBSCRIPTION-SCHEDULER] Sent expiry reminder for subscription %d to %s", sub.ID, user.Email)
	}
}

// ExpireSubscriptions marks expired subscriptions as EXPIRED
func ExpireSubscriptions() {
	db := database.Database.Db
	now := time.Now()

	// Update expired subscriptions
	result := db.Model(&basket.BasketSubscription{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?", basket.SubscriptionActive, now).
		Updates(map[string]interface{}{"status": basket.SubscriptionExpired})

	if result.Error != nil {
		log.Printf("[SUBSCRIPTION-SCHEDULER] Error expiring subscriptions: %v", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("[SUBSCRIPTION-SCHEDULER] Expired %d subscriptions", result.RowsAffected)

		// Send expiry notification emails
		var expiredSubscriptions []basket.BasketSubscription
		db.Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?", basket.SubscriptionExpired, now).
			Where("updated_at > ?", now.Add(-time.Hour)). // Only recently expired
			Preload("Basket").
			Find(&expiredSubscriptions)

		for _, sub := range expiredSubscriptions {
			var user models.User
			if err := db.Where("id = ?", sub.UserID).First(&user).Error; err == nil {
				SendSubscriptionExpiredEmail(user.Email, user.Name, sub.Basket.Name)
			}
		}
	}
}

// SendSubscriptionExpiryReminder sends an email reminder before subscription expires
func SendSubscriptionExpiryReminder(email, name, basketName string, expiresAt *time.Time) {
	expiryStr := "soon"
	if expiresAt != nil {
		expiryStr = expiresAt.Format("January 2, 2006")
	}

	subject := "Your Classia Basket Subscription is Expiring Soon!"
	body := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Subscription Expiring Soon</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2563eb;">Subscription Expiring Soon</h2>
        <p>Dear ` + name + `,</p>
        <p>Your subscription to <strong>` + basketName + `</strong> is expiring on <strong>` + expiryStr + `</strong>.</p>
        <p>To continue receiving updates and access to this basket, please renew your subscription before it expires.</p>
        <div style="margin: 30px 0;">
            <a href="https://classiacapital.com" style="background-color: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px;">Renew Now</a>
        </div>
        <p>If you have any questions, please contact our support team.</p>
        <hr style="border: 1px solid #eee; margin: 20px 0;">
        <p style="font-size: 12px; color: #666;">This is an automated reminder from Classia Capital.</p>
    </div>
</body>
</html>`

	go SendEmail([]string{email}, subject, body)
}

// SendSubscriptionExpiredEmail sends an email when subscription has expired
func SendSubscriptionExpiredEmail(email, name, basketName string) {
	subject := "Your Classia Basket Subscription Has Expired"
	body := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Subscription Expired</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #dc2626;">Subscription Expired</h2>
        <p>Dear ` + name + `,</p>
        <p>Your subscription to <strong>` + basketName + `</strong> has expired.</p>
        <p>You will no longer receive updates or have access to this basket until you renew your subscription.</p>
        <div style="margin: 30px 0;">
            <a href="https://app.classiacapital.com" style="background-color: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px;">Renew Subscription</a>
        </div>
        <p>We hope to see you back soon!</p>
        <hr style="border: 1px solid #eee; margin: 20px 0;">
        <p style="font-size: 12px; color: #666;">This is an automated notification from Classia Capital.</p>
    </div>
</body>
</html>`

	go SendEmail([]string{email}, subject, body)
}
