package userController

import (
	"encoding/json"
	"errors"
	"fib/config"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/jinzhu/now"
	"gorm.io/gorm"
)

type BankVerifyResponse struct {
	Code          int    `json:"code"`
	Timestamp     int64  `json:"timestamp"`
	TransactionID string `json:"transaction_id"`
	Data          struct {
		Entity          string `json:"@entity"`
		Message         string `json:"message"`
		AccountExists   bool   `json:"account_exists"`
		NameAtBank      string `json:"name_at_bank"`
		Utr             string `json:"utr"`
		AmountDeposited string `json:"amount_deposited"`
	} `json:"data"`
}

func VerifyBankDetails(accountNo, ifscCode, holderName, mobile string) (bool, error) {
	// 1. Get token
	authToken, err := sandboxJwt()
	if err != nil {
		return false, fmt.Errorf("failed to get auth token: %v", err)
	}

	// 2. Prepare URL
	endpoint := fmt.Sprintf(
		"%sbank/%s/accounts/%s/verify?name=%s&mobile=%s",
		config.AppConfig.SandboxApiURL,
		ifscCode,
		accountNo,
		url.QueryEscape(holderName),
		url.QueryEscape(mobile),
	)

	// 3. Make HTTP request
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", authToken)
	req.Header.Set("x-api-key", config.AppConfig.SandboxApiKey)
	req.Header.Set("x-api-version", "2.0")
	req.Header.Set("x-accept-cache", "true")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 4. Read Response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %v", err)
	}

	// 5. Parse JSON
	var verifyResp BankVerifyResponse
	err = json.Unmarshal(body, &verifyResp)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// 6. Check if verification was successful
	if verifyResp.Code == 200 && verifyResp.Data.AccountExists {
		return true, nil
	}

	return false, fmt.Errorf("bank verification failed: %s", verifyResp.Data.Message)
}

func AddBankAccount(c *fiber.Ctx) error {
	// Retrieve userId from JWT token
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user ID!", nil)
	}

	// Parse the request body
	reqData := new(struct {
		BankName    string `json:"bankName"`
		AccountNo   string `json:"accountNo"`
		HolderName  string `json:"holderName"`
		IFSCCode    string `json:"ifscCode"`
		BranchName  string `json:"branchName"`
		AccountType string `json:"accountType"` // optional
		Mobile      string `json:"mobile"`
	})
	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Access Denied!", nil)
	}

	// Check if user already has bank details linked
	if user.BankDetails != 0 {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "You already have a bank account!", nil)
	}

	// Check if the bank account already exists in DB
	var existingBankDetails models.BankDetails
	if err := database.Database.Db.Where("account_no = ?", reqData.AccountNo).First(&existingBankDetails).Error; err == nil {
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Bank account already exists!", nil)
	}

	// Sandbox API call - Verify bank details (new way)
	isVerified, err := VerifyBankDetails(reqData.AccountNo, reqData.IFSCCode, reqData.HolderName, reqData.Mobile)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Bank verification failed: "+err.Error(), nil)
	}

	// Prepare BankDetails record
	now := time.Now()
	newBankDetails := models.BankDetails{
		BankName:    reqData.BankName,
		AccountNo:   reqData.AccountNo,
		HolderName:  reqData.HolderName,
		IFSCCode:    reqData.IFSCCode,
		BranchName:  reqData.BranchName,
		AccountType: reqData.AccountType,
		UserID:      userId,
		IsVerified:  isVerified,
	}
	if isVerified {
		newBankDetails.VerifiedAt = &now
	}

	// Save new bank details
	if err := database.Database.Db.Create(&newBankDetails).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to add bank account!", nil)
	}

	// Update user table with BankDetails ID
	user.BankDetails = newBankDetails.ID
	if err := database.Database.Db.Save(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update user with bank details!", nil)
	}

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

type AadhaarOKYCResponse struct {
	Code          int    `json:"code"`
	Timestamp     int64  `json:"timestamp"`
	TransactionID string `json:"transaction_id"`
	Data          struct {
		Entity      string      `json:"@entity"`
		ReferenceID interface{} `json:"reference_id"`
		Status      string      `json:"status"`
		Message     string      `json:"message"`
		CareOf      string      `json:"care_of"`
		FullAddress string      `json:"full_address"`
		DateOfBirth string      `json:"date_of_birth"`
		EmailHash   string      `json:"email_hash"`
		Gender      string      `json:"gender"`
		Name        string      `json:"name"`
		Address     struct {
			Entity      string      `json:"@entity"`
			Country     string      `json:"country"`
			District    string      `json:"district"`
			House       string      `json:"house"`
			Landmark    string      `json:"landmark"`
			Pincode     interface{} `json:"pincode"`
			PostOffice  string      `json:"post_office"`
			State       string      `json:"state"`
			Street      string      `json:"street"`
			Subdistrict string      `json:"subdistrict"`
			Vtc         string      `json:"vtc"`
		} `json:"address"`
		YearOfBirth interface{} `json:"year_of_birth"`
		MobileHash  string      `json:"mobile_hash"`
		Photo       string      `json:"photo"`
		ShareCode   string      `json:"share_code"`
	} `json:"data"`
}

func VerifyAdharOtp(c *fiber.Ctx) error {
	// Extract user ID from context
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user ID!", nil)
	}

	// Parse request body
	reqData := new(struct {
		AadharNumber string `json:"aadharNumber"`
		ReferenceID  string `json:"referenceId"`
		Otp          string `json:"otp"`
	})
	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	// Validate request fields
	if reqData.AadharNumber == "" || reqData.ReferenceID == "" || reqData.Otp == "" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Missing required fields!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "User not found!", nil)
	}

	// Prepare API request
	url := config.AppConfig.SandboxApiURL + "kyc/aadhaar/okyc/otp/verify"
	payload := fmt.Sprintf(`{
		"@entity": "in.co.sandbox.kyc.aadhaar.okyc.request",
		"reference_id": "%s",
		"otp": "%s"
	}`, reqData.ReferenceID, reqData.Otp)

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create request!", nil)
	}

	// Set headers as per Sandbox documentation
	authToken, err := sandboxJwt()
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to generate auth token!", nil)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("authorization", authToken)
	req.Header.Set("x-api-key", config.AppConfig.SandboxApiKey)
	req.Header.Set("x-api-version", "2.0")
	req.Header.Set("content-type", "application/json")

	// Send API request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send request!", nil)
	}
	defer res.Body.Close()

	// Read response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to read response!", nil)
	}

	// Check for non-200 status
	if res.StatusCode != http.StatusOK {
		var errorResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Message != "" {
			return middleware.JsonResponse(c, res.StatusCode, false, fmt.Sprintf("OTP verification failed: %s", errorResp.Message), nil)
		}
		return middleware.JsonResponse(c, res.StatusCode, false, fmt.Sprintf("OTP verification failed: %s", string(body)), nil)
	}

	// Parse API response
	var response AadhaarOKYCResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("Failed to parse API response: %v, Body: %s", err, string(body))
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to parse API response!", nil)
	}

	// Validate critical fields with detailed logging
	var validationErrors []string
	if response.Data.Name == "" {
		validationErrors = append(validationErrors, "Name is empty")
	}
	if response.Data.DateOfBirth == "" {
		validationErrors = append(validationErrors, "DateOfBirth is empty")
	}
	if !strings.EqualFold(response.Data.Status, "VALID") {
		validationErrors = append(validationErrors, fmt.Sprintf("Status is invalid: %s", response.Data.Status))
	}
	if len(validationErrors) > 0 {
		log.Printf("Validation failed: %v, Response Data: %+v", validationErrors, response.Data)
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed To verify", nil)
	}

	// Convert reference_id to string
	var refID string
	switch v := response.Data.ReferenceID.(type) {
	case string:
		refID = v
	case float64:
		refID = fmt.Sprintf("%d", int(v))
	default:
		log.Printf("Unexpected reference_id type: %T", v)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Invalid reference_id format!", nil)
	}

	// Convert pincode to string
	var pincode string
	switch v := response.Data.Address.Pincode.(type) {
	case string:
		pincode = v
	case float64:
		pincode = fmt.Sprintf("%d", int(v))
	default:
		pincode = ""
	}

	// Convert address struct to string
	addr := response.Data.Address
	var addressParts []string
	if addr.House != "" {
		addressParts = append(addressParts, addr.House)
	}
	if addr.Street != "" {
		addressParts = append(addressParts, addr.Street)
	}
	if addr.Landmark != "" {
		addressParts = append(addressParts, addr.Landmark)
	}
	if addr.Vtc != "" {
		addressParts = append(addressParts, addr.Vtc)
	}
	if addr.District != "" {
		addressParts = append(addressParts, addr.District)
	}
	if addr.State != "" {
		addressParts = append(addressParts, addr.State)
	}
	if addr.Country != "" {
		addressParts = append(addressParts, addr.Country)
	}
	if pincode != "" {
		addressParts = append(addressParts, pincode)
	}
	address := strings.Join(addressParts, ", ")

	// Prepare AadharDetails
	aadhar := models.AadharDetails{
		AadharNumber: reqData.AadharNumber,
		Name:         response.Data.Name,
		DOB:          response.Data.DateOfBirth,
		Address:      address,
		ProfileImage: response.Data.Photo,
		RefID:        refID,
		IsVerified:   true,
	}

	// Save AadharDetails
	if err := database.Database.Db.Create(&aadhar).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return middleware.JsonResponse(c, fiber.StatusConflict, false, "Aadhaar number already exists!", nil)
		}
		log.Printf("Failed to save AadharDetails: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save Aadhaar details!", nil)
	}

	// Handle UserKYC: create or update
	var userKYC models.UserKYC
	if err := database.Database.Db.Where("user_id = ?", userId).First(&userKYC).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new UserKYC, omitting PanID to allow NULL
			userKYC = models.UserKYC{
				UserID:     userId,
				AdharID:    aadhar.ID,
				IsVerified: true,
				IsDeleted:  false,
				// PanID is not set, allowing it to default to NULL in the database
			}
			if err := database.Database.Db.Omit("PanID").Create(&userKYC).Error; err != nil {
				log.Printf("Failed to create UserKYC: %v", err)
				return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create UserKYC record!", nil)
			}
		} else {
			log.Printf("Database error: %v", err)
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Database error!", nil)
		}
	} else {
		// Update existing UserKYC
		userKYC.AdharID = aadhar.ID
		userKYC.IsVerified = true
		// Leave PanID unchanged to avoid constraint violation
		if err := database.Database.Db.Omit("PanID").Save(&userKYC).Error; err != nil {
			log.Printf("Failed to update UserKYC: %v", err)
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update UserKYC record!", nil)
		}
	}

	// Return success response
	return middleware.JsonResponse(c, fiber.StatusOK, true, "Aadhaar OTP verified and details saved successfully.", nil)
}

func PanLinkStatus(c *fiber.Ctx) error {
	// Extract user ID from context
	userId, ok := c.Locals("userId").(uint)
	fmt.Print(userId)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid user ID!", nil)
	}

	// Parse request body
	reqData := new(struct {
		AadhaarNumber string `json:"aadhaarNumber"`
		PanNumber     string `json:"panNumber"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request body!", nil)
	}

	// Validate request fields
	if reqData.AadhaarNumber == "" || reqData.PanNumber == "" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Missing required fields!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = ?", userId, false).First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusNotFound, false, "User not found!", nil)
	}

	// Prepare API request
	url := config.AppConfig.SandboxApiURL + "kyc/pan-aadhaar/status"
	payload := fmt.Sprintf(`{
		"@entity": "in.co.sandbox.kyc.pan_aadhaar.status",
		"pan": "%s",
		"aadhaar_number": "%s",
		"consent": "Y",
		"reason": "Verification"
	}`, reqData.PanNumber, reqData.AadhaarNumber)

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create request!", nil)
	}

	// Set headers
	authToken, err := sandboxJwt()
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to generate auth token!", nil)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("authorization", authToken)
	req.Header.Set("x-api-key", config.AppConfig.SandboxApiKey)
	req.Header.Set("x-api-version", "2.0")
	req.Header.Set("content-type", "application/json")

	// Send API request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to send request!", nil)
	}
	defer res.Body.Close()

	// Read response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to read response!", nil)
	}

	// Check for non-200 status
	if res.StatusCode != http.StatusOK {
		var errorResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Message != "" {
			return middleware.JsonResponse(c, res.StatusCode, false, fmt.Sprintf("PAN-Aadhaar status check failed: %s", errorResp.Message), nil)
		}
		return middleware.JsonResponse(c, res.StatusCode, false, fmt.Sprintf("PAN-Aadhaar status check failed: %s", string(body)), nil)
	}

	// Parse API response
	type PanAadhaarStatusResponse struct {
		Data struct {
			Entity               string `json:"@entity"`
			AadhaarSeedingStatus string `json:"aadhaar_seeding_status"`
			Message              string `json:"message"`
		} `json:"data"`
	}

	var apiResponse PanAadhaarStatusResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("Failed to parse API response: %v, Body: %s", err, string(body))
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to parse API response!", nil)
	}

	// Check if message matches the required pattern
	isLinked := apiResponse.Data.AadhaarSeedingStatus == "y" &&
		strings.HasPrefix(apiResponse.Data.Message, "Your PAN is linked to Aadhaar Number")

	if !isLinked {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "PAN-Aadhaar link verification failed: Invalid response message!", nil)
	}

	// Use a transaction to ensure atomicity
	err = database.Database.Db.Transaction(func(tx *gorm.DB) error {
		// Prepare AadharDetails
		aadhar := models.AadharDetails{
			AadharNumber: reqData.AadhaarNumber,
			IsVerified:   isLinked,
		}
		// Save or update AadharDetails
		if err := tx.Where("aadhar_number = ?", reqData.AadhaarNumber).FirstOrCreate(&aadhar).Error; err != nil {
			log.Printf("Failed to save AadharDetails: %v", err)
			return err
		}

		// Prepare PanDetails
		pan := models.PanDetails{
			PanNumber:  reqData.PanNumber,
			IsVerified: isLinked,
		}
		// Save or update PanDetails
		if err := tx.Where("pan_number = ?", reqData.PanNumber).FirstOrCreate(&pan).Error; err != nil {
			log.Printf("Failed to save PanDetails: %v", err)
			return err
		}

		// Handle UserKYC: create or update
		var userKYC models.UserKYC
		if err := tx.Where("user_id = ?", userId).First(&userKYC).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create new UserKYC
				userKYC = models.UserKYC{
					UserID:     userId,
					AdharID:    aadhar.ID,
					PanID:      pan.ID,
					IsVerified: isLinked,
					IsDeleted:  false,
				}
				if err := tx.Create(&userKYC).Error; err != nil {
					log.Printf("Failed to create UserKYC: %v", err)
					return err
				}
			} else {
				log.Printf("Database error: %v", err)
				return err
			}
		} else {
			// Update existing UserKYC
			userKYC.AdharID = aadhar.ID
			userKYC.PanID = pan.ID
			userKYC.IsVerified = isLinked
			if err := tx.Save(&userKYC).Error; err != nil {
				log.Printf("Failed to update UserKYC: %v", err)
				return err
			}
		}

		return nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return middleware.JsonResponse(c, fiber.StatusConflict, false, "Aadhaar or PAN number already exists!", nil)
		}
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save KYC details!", nil)
	}

	// Return success response
	responseData := struct {
		IsLinked bool   `json:"is_linked"`
		Message  string `json:"message"`
	}{
		IsLinked: isLinked,
		Message:  apiResponse.Data.Message,
	}
	return middleware.JsonResponse(c, fiber.StatusOK, true, "PAN-Aadhaar link status verified and details saved successfully.", responseData)
}

func AddFolioNumber(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	reqData, ok := c.Locals("validatedFolioNumber").(*struct {
		FolioNumber string `json:"folioNumber"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid folio number data!", nil)
	}

	// Normalize folio number (trim whitespace, convert to uppercase)
	folioNumber := strings.TrimSpace(strings.ToUpper(reqData.FolioNumber))
	if folioNumber == "" {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Folio number cannot be empty!", nil)
	}

	// Begin transaction
	tx := database.Database.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if user exists
	var user models.User
	if err := tx.First(&user, userId).Error; err != nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	// Check for duplicate folio number
	var existingFolio models.Folio
	if err := tx.
		Where("user_id = ? AND folio_no = ? AND is_deleted = ?", userId, folioNumber, false).
		First(&existingFolio).Error; err == nil {
		tx.Rollback()
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Folio number already exists!", nil)
	} else if err != gorm.ErrRecordNotFound {
		tx.Rollback()
		log.Printf("Failed to check existing folio number for user %d: %v", userId, err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to check folio number!", nil)
	}

	// Create new folio entry
	folio := models.Folio{
		UserID:    userId,
		FolioNo:   folioNumber,
		IsDeleted: false,
	}

	if err := tx.Create(&folio).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to save folio number %s for user %d: %v", folioNumber, userId, err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save folio number!", nil)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Printf("Failed to commit transaction for folio number %s: %v", folioNumber, err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save folio number!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Folio Number added.", nil)
}

func FolioNoList(c *fiber.Ctx) error {

	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}
	print("call api  for amc list")
	reqData, ok := c.Locals("validatedFolioList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok || reqData.Page == nil || reqData.Limit == nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Log to verify
	log.Printf(">> FolioNoList: userId=%d, page=%d, limit=%d", userId, *reqData.Page, *reqData.Limit)

	offset := (*reqData.Page - 1) * (*reqData.Limit)
	tx := database.Database.Db.Debug().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// ... user existence check ...

	var folios []models.Folio
	var total int64

	if err := tx.Debug().
		Where("user_id = ? AND is_deleted = ?", userId, false).
		Offset(offset).
		Limit(*reqData.Limit).
		Find(&folios).
		Error; err != nil {
		tx.Rollback()
		log.Printf("Fetch error: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch folio numbers!", nil)
	}

	if err := tx.Debug().
		Model(&models.Folio{}).
		Where("user_id = ? AND is_deleted = ?", userId, false).
		Count(&total).
		Error; err != nil {
		tx.Rollback()
		log.Printf("Count error: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch folio numbers!", nil)
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Commit error: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch folio numbers!", nil)
	}

	folioNumbers := make([]string, len(folios))
	for i, f := range folios {
		folioNumbers[i] = f.FolioNo
	}

	response := map[string]interface{}{
		"folioNumbers": folioNumbers,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  *reqData.Page,
			"limit": *reqData.Limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Folio numbers retrieved.", response)
}

func Deposit(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	reqData := new(struct {
		Amount uint `json:"amount"`
		AmcId  uint `json:"amcId"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	var user models.User
	if err := database.Database.Db.First(&user, userId).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	var amc models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", reqData.AmcId, "AMC").First(&amc).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid AMC!", nil)
	}

	user.MainBalance += reqData.Amount
	if err := database.Database.Db.Save(&user).Error; err != nil {
		log.Printf("Failed to Add Main Balance: %v", err)
	}

	newTransactionDetails := models.Transactions{
		TransactionType: "DEPOSIT",
		Amount:          reqData.Amount,
		Status:          "COMPLETED",
		UserID:          userId,
		AmcID:           reqData.AmcId,
	}

	// Save the new bank account to the database
	if err := database.Database.Db.Create(&newTransactionDetails).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to Create Transaction record!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Deposite Sucess.", newTransactionDetails)
}

func Withdraw(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	reqData := new(struct {
		Amount uint `json:"amount"`
		AmcId  uint `json:"amcId"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	var user models.User
	if err := database.Database.Db.First(&user, userId).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	var amc models.User
	if err := database.Database.Db.Where("id = ? AND is_deleted = false AND role = ?", reqData.AmcId, "AMC").First(&amc).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid AMC!", nil)
	}

	if user.MainBalance < reqData.Amount {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Insufficient balance!", nil)
	}

	user.MainBalance -= reqData.Amount
	if err := database.Database.Db.Save(&user).Error; err != nil {
		log.Printf("Failed to Add Main Balance: %v", err)
	}

	newTransactionDetails := models.Transactions{
		TransactionType: "WITHDRAW",
		Amount:          reqData.Amount,
		Status:          "COMPLETED",
		UserID:          userId,
		AmcID:           reqData.AmcId,
	}

	// Save the new bank account to the database
	if err := database.Database.Db.Create(&newTransactionDetails).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to Create Transaction record!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Withdraw Sucess.", newTransactionDetails)
}

func TransactionList(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Retrieve validated request data
	reqData, ok := c.Locals("validatedTransactionList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	offset := (*reqData.Page - 1) * (*reqData.Limit)

	var transactions []models.Transactions
	var total int64

	// Fetch loginTraking with pagination
	if err := database.Database.Db.Where("user_id = ? AND is_deleted = ?", userId, false).
		Offset(offset).
		Limit(*reqData.Limit).
		Find(&transactions).
		Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Access Denied!", nil)
	}

	// Count total records
	database.Database.Db.Model(&models.Transactions{}).Where("user_id = ? AND is_deleted = ?", userId, false).Count(&total)

	// Response structure
	response := map[string]interface{}{
		"transactions": transactions,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  *reqData.Page,
			"limit": *reqData.Limit,
		},
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Transactions list.", response)
}

const (
	SYMBOLS_URL  = "https://api.truedata.in/getAllSymbols"
	USER_ID      = "tdwsp703"
	PASSWORD     = "imran@703"
	AUTH_URL     = "https://auth.truedata.in/token"
	BARS_URL     = "https://history.truedata.in/getbars"
	USERNAME     = "tdwsp703"
	MARKET_OPEN  = "09:15:00"
	MARKET_CLOSE = "15:30:00"
)

func AmcPerformance(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	var user models.User
	if err := database.Database.Db.
		Where("id = ? AND is_deleted = false", userId).
		First(&user).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Access Denied!", nil)
	}

	reqData, ok := c.Locals("validatedAmcId").(*struct {
		AmcId *int `json:"amcId"`
	})
	if !ok || reqData.AmcId == nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	var amc models.User
	if err := database.Database.Db.
		Where("id = ? AND is_deleted = false AND role = ?", reqData.AmcId, "AMC").
		First(&amc).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Invalid AMC Id!", nil)
	}

	// Join stocks with amc_stocks to get holdingPer
	type StockWithHolding struct {
		models.Stocks
		HoldingPer float32 `json:"holdingPer"`
	}

	var stocks []StockWithHolding
	if err := database.Database.Db.
		Table("amc_stocks").
		Select("stocks.*, amc_stocks.holding_per").
		Joins("JOIN stocks ON stocks.id = amc_stocks.stock_id").
		Where("amc_stocks.user_id = ? AND amc_stocks.is_deleted = false AND stocks.is_deleted = false", *reqData.AmcId).
		Order("stocks.created_at DESC").
		Scan(&stocks).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch picked stocks", nil)
	}

	client := resty.New()
	resp, err := client.R().
		SetFormData(map[string]string{
			"username":   USERNAME,
			"password":   PASSWORD,
			"grant_type": "password",
		}).
		Post(AUTH_URL)
	if err != nil || resp.StatusCode() != 200 {
		log.Printf("Auth error: %v %s", err, resp.String())
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "TrueData authentication failed", nil)
	}

	var authResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(resp.Body(), &authResp); err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Invalid auth response", nil)
	}
	token := authResp.AccessToken

	type Performance struct {
		Symbol        string  `json:"symbol"`
		OpenPrice     float64 `json:"openPrice"`
		CurrentPrice  float64 `json:"currentPrice"`
		Change        float64 `json:"change"`
		PercentChange float64 `json:"percentChange"`
		HoldingPer    float32 `json:"holdingPer"`
	}

	var performances []Performance
	today := now.BeginningOfDay()
	from := today.Format("060102") + "T" + MARKET_OPEN
	to := today.Format("060102") + "T" + MARKET_CLOSE

	for _, stock := range stocks {
		url := fmt.Sprintf("%s?symbol=%s&from=%s&to=%s&response=json&interval=1min", BARS_URL, stock.Symbol, from, to)

		resp, err := client.R().
			SetHeader("Authorization", "Bearer "+token).
			Get(url)
		if err != nil || resp.StatusCode() != 200 {
			log.Printf("Error fetching bars for %s: %v %s", stock.Symbol, err, resp.String())
			continue
		}

		var barData struct {
			Records [][]interface{} `json:"Records"`
		}
		if err := json.Unmarshal(resp.Body(), &barData); err != nil || len(barData.Records) == 0 {
			log.Printf("Invalid bar data for %s", stock.Symbol)
			continue
		}

		firstBar := barData.Records[0]
		lastBar := barData.Records[len(barData.Records)-1]
		if len(firstBar) < 5 || len(lastBar) < 5 {
			log.Printf("Invalid bar structure for %s", stock.Symbol)
			continue
		}

		openPrice, ok1 := firstBar[1].(float64)
		currentPrice, ok2 := lastBar[4].(float64)
		if !ok1 || !ok2 {
			log.Printf("Price conversion error for %s", stock.Symbol)
			continue
		}

		change := currentPrice - openPrice
		percentChange := (change / openPrice) * 100

		performances = append(performances, Performance{
			Symbol:        stock.Symbol,
			OpenPrice:     openPrice,
			CurrentPrice:  currentPrice,
			Change:        change,
			PercentChange: percentChange,
			HoldingPer:    stock.HoldingPer,
		})
	}

	// Weighted average based on HoldingPer
	var totalWeightedChange float64
	var totalHolding float64
	for _, p := range performances {
		totalWeightedChange += float64(p.HoldingPer) * p.PercentChange
		totalHolding += float64(p.HoldingPer)
	}

	var avgPercentChange float64
	if totalHolding > 0 {
		avgPercentChange = totalWeightedChange / totalHolding
	} else {
		avgPercentChange = 0
	}

	response := map[string]interface{}{
		"priceChanges":         performances,
		"averagePercentChange": fmt.Sprintf("%.2f%%", avgPercentChange),
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "AMC stock performance fetched successfully!", response)
}

func GetLatestMaintenance(c *fiber.Ctx) error {
	var maintenance models.Maintenance

	if err := database.Database.Db.Order("created_at DESC").First(&maintenance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return middleware.JsonResponse(c, fiber.StatusNotFound, false, "No maintenance record found", nil)
		}
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Database error", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Latest maintenance record", maintenance)
}

func CreateReview(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// ✅ Get validated data
	reqData, ok := c.Locals("validatedReview").(*struct {
		Rating  int    `json:"rating"`
		AmcId   uint   `json:"amcId"`
		Comment string `json:"comment"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	var review models.Review

	// ✅ Check if user already gave review for this AMC
	err := database.Database.Db.Where("user_id = ? AND amc_id = ? AND is_deleted = false", userId, reqData.AmcId).
		First(&review).Error

	if err == nil {
		// 🔄 Review exists → update it
		review.Rating = reqData.Rating
		review.Comment = reqData.Comment

		if err := database.Database.Db.Save(&review).Error; err != nil {
			return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to update review", nil)
		}
		return middleware.JsonResponse(c, fiber.StatusOK, true, "Review updated successfully", review)
	}

	// ➕ Review not found → create new one
	newReview := models.Review{
		UserID:  userId,
		AmcId:   reqData.AmcId,
		Rating:  reqData.Rating,
		Comment: reqData.Comment,
	}

	if err := database.Database.Db.Create(&newReview).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to create review", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusCreated, true, "Review created successfully", newReview)
}

func GetReviews(c *fiber.Ctx) error {
	// ✅ Get validated data from middleware
	reqData := c.Locals("validatedReviewList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
		AmcId uint `json:"amcId"`
	})

	offset := (*reqData.Page - 1) * (*reqData.Limit)

	var reviews []struct {
		ID        uint   `json:"id"`
		Rating    int    `json:"rating"`
		Comment   string `json:"comment"`
		CreatedAt string `json:"created_at"`
	}

	// Fetch reviews
	if err := database.Database.Db.
		Table("reviews").
		Where("reviews.amc_id = ? AND reviews.is_deleted = false", reqData.AmcId).
		Offset(offset).
		Limit(*reqData.Limit).
		Scan(&reviews).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch reviews", nil)
	}

	// ✅ Total reviews count
	var total int64
	database.Database.Db.Model(&models.Review{}).
		Where("amc_id = ? AND is_deleted = false", reqData.AmcId).
		Count(&total)

	// ✅ Average rating
	var avgRating float64
	database.Database.Db.
		Table("reviews").
		Where("amc_id = ? AND is_deleted = false", reqData.AmcId).
		Select("COALESCE(AVG(rating),0)"). // ensures 0 if no reviews
		Scan(&avgRating)

	// Ensure empty array instead of null
	if reviews == nil {
		reviews = []struct {
			ID        uint   `json:"id"`
			Rating    int    `json:"rating"`
			Comment   string `json:"comment"`
			CreatedAt string `json:"created_at"`
		}{}
	}

	// ✅ Response structure with avg rating
	response := map[string]interface{}{
		"reviews": reviews,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  *reqData.Page,
			"limit": *reqData.Limit,
		},
		"average_rating": avgRating,
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Reviews list fetched successfully", response)
}
