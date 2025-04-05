package utils

import (
	"fib/config"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
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
	// Construct the SMS message
	smsMsg := fmt.Sprintf("OTP for Credbull App Registration is %s. Do not share it with anyone.", otp)

	data := url.Values{}
	data.Set("apikey", config.AppConfig.LocalTextApi) // Replace with your actual API key
	data.Set("numbers", mobile)
	data.Set("sender", "CRDBUL")
	data.Set("message", smsMsg)

	// Make the API request
	resp, err := http.PostForm(config.AppConfig.LocalTextApiUrl, data)
	if err != nil {
		log.Printf("Error while sending OTP: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Check if the response status code is not OK
	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to send OTP, response code: %d", resp.StatusCode)
		return fmt.Errorf("failed to send OTP")
	}

	log.Println("OTP sent successfully to", mobile)
	return nil
}

type EmailContent struct {
	Subject string
	HTML    string
}

func SendOTPEmail(otp, email string) error {
	// Create email content
	subject := "Your OTP Code"
	htmlContent := "<p>Your OTP code is: <strong>" + otp + "</strong></p>"
	content := EmailContent{
		Subject: subject,
		HTML:    htmlContent,
	}

	// Create the sender and recipient email objects
	from := mail.NewEmail(config.AppConfig.SandgridSenderName, config.AppConfig.SendgridSenderMail)
	toEmail := mail.NewEmail("", email)

	// Construct the email message
	message := mail.NewSingleEmail(from, content.Subject, toEmail, "", content.HTML)

	// Initialize the SendGrid client
	client := sendgrid.NewSendClient(config.AppConfig.SendgridApiKey)

	// Send the email
	response, err := client.Send(message)
	if err != nil {
		log.Println("Error sending email:", err)
		return err
	}

	// Log response details
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		log.Println("Email sent successfully:", response.StatusCode)
	} else {
		log.Println("Failed to send email. Status code:", response.StatusCode)
		log.Println("Response body:", response.Body)
	}
	return nil
}
