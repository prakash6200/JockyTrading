package utils

import (
	"fib/config"
	"fmt"
	"net/smtp"
	"strings"
)

// Generic Send Email
func SendEmail(to []string, subject string, htmlBody string) error {
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	from := config.AppConfig.EmailSender
	password := config.AppConfig.Password

	// MIME basics
	msg := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n"
	msg += fmt.Sprintf("From: Classia Capital <%s>\r\n", from)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(to, ","))
	msg += fmt.Sprintf("Subject: %s\r\n\r\n", subject)
	msg += htmlBody

	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Debug Logs
	fmt.Printf("--- Sending Email ---\nTo: %v\nSubject: %s\nFrom: %s\n", to, subject, from)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, []byte(msg))
	if err != nil {
		fmt.Println("Error sending email:", err)
		return err
	}
	fmt.Println("--- Email Sent Successfully ---")
	return nil
}

// HTML Wrapper for "Resin Design" (Professional Look)
func getEmailTemplate(title string, bodyContent string) string {
	return fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<style>
			body { font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; background-color: #F6F6F6; margin: 0; padding: 0; }
			.container { max-width: 600px; margin: 40px auto; background: #FFFFFF; border-radius: 8px; overflow: hidden; box-shadow: 0 4px 15px rgba(0,0,0,0.05); }
			.header { background-color: #00004D; padding: 30px; text-align: center; }
			.header h1 { color: #FFFFFF; margin: 0; font-size: 24px; letter-spacing: 1px; }
			.content { padding: 40px 30px; color: #00004D; line-height: 1.6; }
			.content h2 { color: #00004D; margin-top: 0; }
			.footer { background-color: #F6F6F6; padding: 20px; text-align: center; font-size: 12px; color: #666666; border-top: 1px solid #E0E0E0; }
			.btn { display: inline-block; padding: 12px 24px; background-color: #d7b56d; color: #FFFFFF; text-decoration: none; border-radius: 4px; font-weight: bold; margin-top: 20px; }
			.info-box { background: #E8F0FE; padding: 15px; border-radius: 4px; border-left: 4px solid #d7b56d; margin: 20px 0; }
			.action-badge { display: inline-block; padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; color: white; }
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>CLASSIA CAPITAL</h1>
			</div>
			<div class="content">
				<h2>%s</h2>
				%s
			</div>
			<div class="footer">
				&copy; 2026 Classia Capital. All rights reserved.<br>
				Trading involves risk. Please read all documents carefully.
			</div>
		</div>
	</body>
	</html>
	`, title, bodyContent)
}

// --- Triggers ---

// 1. Welcome / Signup
func SendWelcomeEmail(email, name string) {
	subject := "Welcome to Classia Capital"
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>Welcome to <strong>Classia Capital</strong>! We are thrilled to have you onboard.</p>
		<p>Your account has been successfully created. You can now explore our curated baskets and start your investment journey.</p>
		<p>If you have any questions, feel free to reach out to our support team.</p>
	`, name)

	go SendEmail([]string{email}, subject, getEmailTemplate("Welcome Onboard!", body))
}

// 2. Subscription Confirmation
func SendSubscriptionEmail(email, name, basketName string) {
	subject := "Subscription Confirmed: " + basketName
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>You have successfully subscribed to <strong>%s</strong>.</p>
		<p>You will now receive real-time updates for rebalancing and trade signals for this basket.</p>
		<div class="info-box">
			<strong>Next Steps:</strong> Check your dashboard for the latest stock composition.
		</div>
	`, name, basketName)

	fmt.Println("Triggering Subscription Email for:", email)
	go SendEmail([]string{email}, subject, getEmailTemplate("Subscription Successful", body))
}

// 3. Wallet Deposit
func SendWalletDepositEmail(email, name string, amount float64) {
	subject := "Funds Added to Wallet"
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>We have received your deposit of <strong>₹%.2f</strong>.</p>
		<p>Your wallet balance has been updated successfully.</p>
	`, name, amount)

	go SendEmail([]string{email}, subject, getEmailTemplate("Deposit Confirmed", body))
}

// 4. New Message (User Receives from AMC)
func SendNewMessageEmail(email, name, basketName, action, message string) {
	subject := fmt.Sprintf("Update on %s: %s", basketName, action)

	actionColor := "#666666" // secondaryText
	if action == "BUY" {
		actionColor = "#28A745" // success
	}
	if action == "SELL" {
		actionColor = "#DC3545" // error
	}
	if action == "HOLD" {
		actionColor = "#FFC107" // warning
	}

	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>The AMC has posted an update for <strong>%s</strong>.</p>
		<div style="margin: 20px 0; padding: 15px; border: 1px solid #E0E0E0; border-radius: 5px;">
			<div style="margin-bottom: 10px;">
				<span class="action-badge" style="background-color: %s;">%s</span>
			</div>
			<p style="font-size: 16px; font-weight: 500;">"%s"</p>
		</div>
		<p>Login to your dashboard to view full details.</p>
	`, name, basketName, actionColor, action, message)

	go SendEmail([]string{email}, subject, getEmailTemplate("New Basket Update", body))
}

// 6. New Version Available
func SendNewVersionEmail(email, name, basketName string, version int) {
	subject := fmt.Sprintf("Rebalance Alert: %s", basketName)
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>A new version (v%d) of <strong>%s</strong> is now available.</p>
		<div class="info-box">
			Please review the changes and rebalance your portfolio to stay aligned with the strategy.
		</div>
		<a href="#" class="btn">Review Update</a>
	`, name, version, basketName)

	go SendEmail([]string{email}, subject, getEmailTemplate("Basket Rebalanced", body))
}

// 7. AMC Receives User Message
func SendAMCMessageReceivedEmail(amcEmail, amcName, userName, basketName, message string) {
	subject := fmt.Sprintf("New User Query: %s", basketName)
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>User <strong>%s</strong> has sent a query regarding <strong>%s</strong>.</p>
		<div style="margin: 20px 0; padding: 15px; background: #E8F0FE; border-radius: 4px;">
			<em>"%s"</em>
		</div>
		<p>Please reply via your AMC dashboard.</p>
	`, amcName, userName, basketName, message)

	go SendEmail([]string{amcEmail}, subject, getEmailTemplate("New User Message", body))
}

// 8. Basket Approved (To AMC)
func SendBasketApprovedEmail(amcEmail, amcName, basketName string) {
	subject := "Basket Approved: " + basketName
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>Great news! Your basket <strong>%s</strong> has been APPROVED by the admin.</p>
		<p>It is now live/scheduled for users to subscribe.</p>
	`, amcName, basketName)

	go SendEmail([]string{amcEmail}, subject, getEmailTemplate("Basket Approved", body))
}

// 9. Basket Rejected (To AMC)
func SendBasketRejectedEmail(amcEmail, amcName, basketName, reason string) {
	subject := "Basket Rejected: " + basketName
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>Unfortunately, your basket <strong>%s</strong> was rejected.</p>
		<div style="color: #dc3545; font-weight: bold;">Reason: %s</div>
		<p>Please make necessary changes and submit again.</p>
	`, amcName, basketName, reason)

	go SendEmail([]string{amcEmail}, subject, getEmailTemplate("Basket Rejected", body))
}

// 10. Login Notification
func SendLoginNotificationEmail(email, name, ip, device, timeStr string) {
	subject := "New Login Alert"
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>We noticed a new login to your account.</p>
		<div class="info-box" style="background: #FFFFFF; border: 1px solid #E0E0E0; border-left: 4px solid #d7b56d;">
			<ul style="list-style: none; padding: 0; margin: 0;">
				<li style="margin-bottom: 8px;"><strong>Time:</strong> %s</li>
				<li style="margin-bottom: 8px;"><strong>IP Address:</strong> %s</li>
				<li><strong>Device:</strong> %s</li>
			</ul>
		</div>
		<p>If this was you, you can safely ignore this email.</p>
		<p style="color: #DC3545; font-weight: bold;">If you did not authorize this login, please contact support immediately.</p>
	`, name, timeStr, ip, device)

	go SendEmail([]string{email}, subject, getEmailTemplate("New Login Detected", body))
}

// 11. Basket Created (To AMC)
func SendBasketCreatedEmail(email, name, basketName string) {
	subject := "Basket Created: " + basketName
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>You have successfully created a new basket: <strong>%s</strong>.</p>
		<p>It is currently in <strong>DRAFT</strong> status. Add stocks and submit it for admin approval to go live.</p>
		<a href="#" class="btn">Manage Basket</a>
	`, name, basketName)

	go SendEmail([]string{email}, subject, getEmailTemplate("Basket Created", body))
}

// 12. Basket Updated (To AMC)
func SendBasketUpdatedEmail(email, name, basketName string) {
	subject := "Basket Updated: " + basketName
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>Your basket <strong>%s</strong> has been updated successfully.</p>
		<p>Changes have been saved to the current draft/version.</p>
	`, name, basketName)

	go SendEmail([]string{email}, subject, getEmailTemplate("Basket Updated", body))
}

// 13. Basket Submitted (To AMC)
func SendBasketSubmittedEmail(email, name, basketName string) {
	subject := "Basket Submitted: " + basketName
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>Your basket <strong>%s</strong> has been submitted for admin approval.</p>
		<p>Status: <strong style="color: #FFC107;">PENDING APPROVAL</strong></p>
		<p>You will receive an email once it is approved or rejected.</p>
	`, name, basketName)

	go SendEmail([]string{email}, subject, getEmailTemplate("Basket Submitted", body))
}

// 14. Stock Added (To AMC)
func SendStockAddedEmail(email, name, basketName, symbol, action string) {
	subject := "Stock Added: " + basketName
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>You have successfully added <strong>%s</strong> (%s) to your basket <strong>%s</strong>.</p>
		<p>This change is saved to the current draft/version.</p>
	`, name, symbol, action, basketName)

	go SendEmail([]string{email}, subject, getEmailTemplate("Stock Added", body))
}
