package utils

import (
	"fib/config"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"time"
)

// GenerateOTP generates a 6-digit OTP
func GenerateOTP() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) // Create a new random number generator
	otp := ""
	for i := 0; i < 6; i++ {
		otp += fmt.Sprintf("%d", rng.Intn(10)) // Generate a random digit (0-9) and append to OTP string
	}
	return otp
}

func SendOTPToMobile(mobile, otp string) error {
	// Replace with your own values
	apiKey := "0Opfk6PHdUBWhVy18bqGFSecsturJMjCInm5TiD24wZKAv9QlRKl1iZgMwt8peuNC47P5nFsABj3D29E"
	senderID := "CLASIA"
	messageID := "197302" // DLT Template ID
	flash := "0"

	// Variables (OTP and validity time in minutes)
	variables := fmt.Sprintf("%s|10", otp)

	// Build request URL
	url := fmt.Sprintf(
		"https://www.fast2sms.com/dev/bulkV2?authorization=%s&route=dlt&sender_id=%s&message=%s&variables_values=%s&flash=%s&numbers=%s",
		apiKey, senderID, messageID, variables, flash, mobile,
	)

	// Make GET request
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error while sending OTP: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Check if response is OK
	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to send OTP, response code: %d", resp.StatusCode)
		return fmt.Errorf("failed to send OTP, code: %d", resp.StatusCode)
	}

	log.Println("OTP sent successfully to", mobile)
	return nil
}

type EmailContent struct {
	Subject string
	HTML    string
}

func SendOTPEmail(otp, email string) error {
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	from := config.AppConfig.EmailSender
	password := config.AppConfig.Password // App password

	// Receiver email
	to := []string{
		email, // dynamic receiver
	}

	// Email content
	subject := "Subject: OTP Verification Code for Jockey Trading\nMIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	body := fmt.Sprintf(`
		<html>
			<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px;">
				<div style="max-width: 500px; margin: auto; background-color: #ffffff; border-radius: 8px; padding: 30px; box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);">
					<h2 style="color: #333333; text-align: center;">Jockey Trading OTP Verification</h2>
					<p style="font-size: 16px; color: #555555; text-align: center;">Your One Time Password (OTP) is:</p>
					<h1 style="text-align: center; color: #4CAF50; font-size: 40px; margin: 20px 0;">%s</h1>
					<p style="font-size: 14px; color: #999999; text-align: center;">Do not share this OTP with anyone.</p>
					<p style="text-align: center; font-size: 12px; color: #bbbbbb; margin-top: 30px;">Thank you for using our service.</p>
				</div>
			</body>
		</html>
	`, otp)

	message := []byte(subject + "\n" + body)

	// Auth setup
	auth := smtp.PlainAuth("", from, password, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		fmt.Println("Error sending email:", err)
		return err
	}

	fmt.Println("Email sent successfully to", email)
	return nil
}
