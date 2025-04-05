package userController

import (
	"encoding/json"
	"fib/config"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func AddBankAccount(c *fiber.Ctx) error {
	// Retrieve the userId from the JWT token (added by JWTMiddleware)
	userId := c.Locals("userId").(uint)

	// Parse the request body to get the bank details
	reqData := new(struct {
		BankName    string `json:"bankName"`
		AccountNo   string `json:"accountNo"`
		HolderName  string `json:"holderName"`
		IFSCCode    string `json:"ifscCode"`
		BranchName  string `json:"branchName"`
		AccountType string `json:"accountType"` // Optional, default to "savings"
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch user!", nil)
	}

	// Check if the user already has a bank account
	if user.BankDetails != 0 {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "You already have a bank account!", nil)
	}

	// Check if the bank account already exists
	var existingBankDetails models.BankDetails
	result := database.Database.Db.Where("account_no = ?", reqData.AccountNo).First(&existingBankDetails)

	if result.RowsAffected > 0 {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Bank account already exists!", nil)
	}

	// Create a new BankDetails object
	newBankDetails := models.BankDetails{
		BankName:    reqData.BankName,
		AccountNo:   reqData.AccountNo,
		HolderName:  reqData.HolderName,
		IFSCCode:    reqData.IFSCCode,
		BranchName:  reqData.BranchName,
		AccountType: reqData.AccountType,
		UserID:      userId,
	}

	// Save the new bank account to the database
	if err := database.Database.Db.Create(&newBankDetails).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to add bank account!", nil)
	}

	// If user exists, update their bank details field with the new bank account ID
	user.BankDetails = newBankDetails.ID

	// Save the updated user with the new bank details
	if err := database.Database.Db.Save(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update user with bank details!", nil)
	}

	// Respond with success message
	return middleware.JsonResponse(c, fiber.StatusOK, true, "Bank account added successfully.", newBankDetails)
}

func sandboxJwt() (string, error) {
	config := config.AppConfig
	url := config.SandboxApiURL + "authenticate"

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Add("x-api-key", config.SandboxApiKey)
	req.Header.Add("x-api-secret", config.SandboxSecretKey)
	req.Header.Add("x-api-version", config.SandboxApiVersion)
	req.Header.Add("accept", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("authentication failed with status %d: %s", res.StatusCode, string(body))
	}

	var response struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %v", err)
	}

	if response.AccessToken == "" {
		return "", fmt.Errorf("access token is missing in the response")
	}

	return response.AccessToken, nil
}

func SendAdharOtp(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user ID!", nil)
	}

	reqData := new(struct {
		AadharNumber string `json:"aadharNumber"`
	})
	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "User not found!", nil)
	}

	var existingKYC models.UserKYC
	if err := database.Database.Db.Where("user_id = ?", userId).First(&existingKYC).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "KYC record already exists!", nil)
	}

	var existingAadhar models.AadharDetails
	if err := database.Database.Db.Where("aadhar_number = ?", reqData.AadharNumber).First(&existingAadhar).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Aadhaar number already exists!", nil)
	}

	url := config.AppConfig.SandboxApiURL + "kyc/aadhaar/okyc/otp"

	payload := fmt.Sprintf(`{
		"@entity":"in.co.sandbox.kyc.aadhaar.okyc.otp.request",
		"consent":"y",
		"aadhaar_number":"%s",
		"reason":"Verification"
	}`, reqData.AadharNumber)

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create OTP request!", nil)
	}

	authToken, err := sandboxJwt()
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to get authentication token!", nil)
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", authToken)
	req.Header.Add("x-api-key", config.AppConfig.SandboxApiKey)
	req.Header.Add("x-api-version", "2.0")
	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send Aadhaar OTP!", nil)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to read response body!", nil)
	}

	if res.StatusCode != http.StatusOK {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send Aadhaar OTP: "+string(body), nil)
	}

	var response struct {
		TransactionID string `json:"transaction_id"`
		Data          struct {
			ReferenceID int    `json:"reference_id"`
			Message     string `json:"message"`
		} `json:"data"`
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to parse response JSON!", nil)
	}

	// Return success with extracted details
	return middleware.JsonResponse(c, fiber.StatusOK, true, "Aadhaar OTP sent successfully.", map[string]interface{}{
		"transaction_id": response.TransactionID,
		"reference_id":   response.Data.ReferenceID,
	})
}

func VerifyAdharOtp(c *fiber.Ctx) error {
	// Parse request body
	reqData := new(struct {
		ReferenceID string `json:"referenceId"`
		Otp         string `json:"otp"`
	})
	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	// Construct API URL
	url := config.AppConfig.SandboxApiURL + "kyc/aadhaar/okyc/otp/verify"

	// Prepare payload
	payload := fmt.Sprintf(`{
		"@entity": "in.co.sandbox.kyc.aadhaar.okyc.request",
		"reference_id": "%s",
		"otp": "%s"
	}`, reqData.ReferenceID, reqData.Otp)

	// Send request
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create request!", nil)
	}

	// Add headers
	authToken, err := sandboxJwt()
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to generate auth token!", nil)
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", authToken)
	req.Header.Add("x-api-key", config.AppConfig.SandboxApiKey)
	req.Header.Add("x-api-version", "2.0")
	req.Header.Add("content-type", "application/json")

	// Send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send request!", nil)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to read response!", nil)
	}

	// Check for error status
	if res.StatusCode != http.StatusOK {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "OTP verification failed: "+string(body), nil)
	}

	// Parse JSON response
	var response struct {
		TransactionID string `json:"transaction_id"`
		Data          struct {
			ReferenceID int    `json:"reference_id"`
			Message     string `json:"message"`
			Name        string `json:"name"`
			DateOfBirth string `json:"date_of_birth"`
			Gender      string `json:"gender"`
			Address     struct {
				Country     string `json:"country"`
				State       string `json:"state"`
				District    string `json:"district"`
				Pincode     int    `json:"pincode"`
				Landmark    string `json:"landmark"`
				PostOffice  string `json:"post_office"`
				Subdistrict string `json:"subdistrict"`
			} `json:"address"`
			Photo string `json:"photo"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to parse response!", nil)
	}

	// Print the parsed response for debugging
	fmt.Println("Parsed Response: ", response)

	// Return success response
	return middleware.JsonResponse(c, fiber.StatusOK, true, "Aadhaar OTP verified and details saved successfully.", map[string]interface{}{
		"transaction_id": response.TransactionID,
		"reference_id":   response.Data.ReferenceID,
		"message":        response.Data.Message,
	})
}

// PAN-Aadhaar Link Status
func PanLinkStatus(c *fiber.Ctx) error {
	reqData := new(struct {
		AdharNumber string `json:"adharNumber"`
		PanNumber   string `json:"panNumber"`
	})

	// Body parsing
	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	// Get authentication token
	authToken, err := sandboxJwt()
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Internal Server Error!", nil)
	}

	payload := fmt.Sprintf(`{
		"@entity": "in.co.sandbox.kyc.pan_aadhaar.status",
		"pan": "%s",
		"aadhaar_number": "%s",
		"consent": "Y",
		"reason": "Verification"
	}`, reqData.PanNumber, reqData.AdharNumber)

	url := config.AppConfig.SandboxApiURL + "kyc/pan-aadhaar/status"
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create request!", nil)
	}

	// Set headers
	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", authToken)
	req.Header.Add("x-api-key", config.AppConfig.SandboxApiKey)
	req.Header.Add("x-api-version", "2.0")
	req.Header.Add("content-type", "application/json")

	// Make the HTTP request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to execute request!", nil)
	}
	defer res.Body.Close()

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to read response body!", nil)
	}

	// Log the response for debugging purposes
	fmt.Println(string(body))

	// Return a successful response
	return middleware.JsonResponse(c, fiber.StatusOK, true, "Pan Aadhaar Link Status Verified.", nil)
}
