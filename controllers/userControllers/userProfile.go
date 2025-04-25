package userController

import (
	"encoding/json"
	"fib/config"
	"fib/database"
	"fib/middleware"
	"fib/models"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
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
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Access Denied!", nil)
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
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, fmt.Sprintf("Invalid or incomplete Aadhaar data from API: %s", strings.Join(validationErrors, "; ")), nil)
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
		Status     bool   `json:"status"`
		Message    string `json:"message"`
		IsLinked   bool   `json:"is_linked"`
		LinkedDate string `json:"linked_date,omitempty"`
	}
	var apiResponse PanAadhaarStatusResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("Failed to parse API response: %v, Body: %s", err, string(body))
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to parse API response!", nil)
	}

	// Validate critical fields
	// var validationErrors []string
	// if apiResponse.Message == "" {
	// 	validationErrors = append(validationErrors, "Message is empty")
	// }
	// if !apiResponse.Status {
	// 	validationErrors = append(validationErrors, "API status is false")
	// }
	// if len(validationErrors) > 0 {
	// 	log.Printf("Validation failed: %v, Response Data: %+v", validationErrors, apiResponse)
	// 	return middleware.JsonResponse(c, fiber.StatusBadRequest, false, fmt.Sprintf("Invalid or incomplete API response: %s", strings.Join(validationErrors, "; ")), nil)
	// }

	// Use a transaction to ensure atomicity
	err = database.Database.Db.Transaction(func(tx *gorm.DB) error {
		// Prepare AadharDetails
		aadhar := models.AadharDetails{
			AadharNumber: reqData.AadhaarNumber,
			IsVerified:   apiResponse.IsLinked, // Set based on link status
		}
		// Save or update AadharDetails
		if err := tx.Where("aadhar_number = ?", reqData.AadhaarNumber).FirstOrCreate(&aadhar).Error; err != nil {
			log.Printf("Failed to save AadharDetails: %v", err)
			return err
		}

		// Prepare PanDetails
		pan := models.PanDetails{
			PanNumber:  reqData.PanNumber,
			IsVerified: apiResponse.IsLinked,
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
					IsVerified: apiResponse.IsLinked,
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
			userKYC.IsVerified = apiResponse.IsLinked
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
	return middleware.JsonResponse(c, fiber.StatusOK, true, "PAN-Aadhaar link status verified and details saved successfully.", apiResponse)
}

func AddFolioNumber(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	reqData, ok := c.Locals("validatedFolioNumber").(*struct {
		FolioNumber string `json:"folioNumber"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid folio number data!", nil)
	}

	// Check if user exists
	var user models.User
	if err := database.Database.Db.First(&user, userId).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	var existingFolio models.Folio
	if err := database.Database.Db.
		Where("user_id = ? AND folio_no = ? AND is_deleted = ?", userId, reqData.FolioNumber, false).
		First(&existingFolio).Error; err == nil {
		// Folio number already exists
		return middleware.JsonResponse(c, fiber.StatusConflict, false, "Folio number already exists!", nil)
	} else if err != gorm.ErrRecordNotFound {
		// Handle unexpected database errors
		log.Printf("Failed to check existing folio number: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to check folio number!", nil)
	}

	// Create a new folio entry
	folio := models.Folio{
		UserID:    userId,
		FolioNo:   reqData.FolioNumber,
		IsDeleted: false,
	}

	if err := database.Database.Db.Create(&folio).Error; err != nil {
		log.Printf("Failed to save folio number: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to save folio number!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Folio Number added.", nil)
}

func FolioNoList(c *fiber.Ctx) error {
	// Retrieve userId from JWT middleware
	userId, ok := c.Locals("userId").(uint)
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "Unauthorized!", nil)
	}

	// Retrieve validated request data
	reqData, ok := c.Locals("validatedFolioList").(*struct {
		Page  *int `json:"page"`
		Limit *int `json:"limit"`
	})
	if !ok {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Invalid request data!", nil)
	}

	// Calculate offset
	offset := (*reqData.Page - 1) * (*reqData.Limit)

	// Check if user exists
	var user models.User
	if err := database.Database.Db.First(&user, userId).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
	}

	// Fetch folio numbers with pagination
	var folios []models.Folio
	var total int64

	if err := database.Database.Db.
		Where("user_id = ? AND is_deleted = ?", userId, false).
		Offset(offset).
		Limit(*reqData.Limit).
		Find(&folios).
		Error; err != nil {
		log.Printf("Failed to fetch folio numbers: %v", err)
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to fetch folio numbers!", nil)
	}

	// Count total records
	database.Database.Db.Model(&models.Folio{}).
		Where("user_id = ? AND is_deleted = ?", userId, false).
		Count(&total)

	// Prepare response data (only include FolioNo)
	folioNumbers := make([]string, len(folios))
	for i, folio := range folios {
		folioNumbers[i] = folio.FolioNo
	}

	// Response structure
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
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	var user models.User
	if err := database.Database.Db.First(&user, userId).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
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
	}

	// Save the new bank account to the database
	if err := database.Database.Db.Create(&newTransactionDetails).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to Create Transaction record!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Deposite Sucess.", nil)
}

func Withdraw(c *fiber.Ctx) error {
	userId := c.Locals("userId").(uint)

	reqData := new(struct {
		Amount uint `json:"amount"`
	})

	if err := c.BodyParser(reqData); err != nil {
		return middleware.JsonResponse(c, fiber.StatusBadRequest, false, "Failed to parse request body!", nil)
	}

	var user models.User
	if err := database.Database.Db.First(&user, userId).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusUnauthorized, false, "User not found!", nil)
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
	}

	// Save the new bank account to the database
	if err := database.Database.Db.Create(&newTransactionDetails).Error; err != nil {
		return middleware.JsonResponse(c, fiber.StatusInternalServerError, false, "Failed to Create Transaction record!", nil)
	}

	return middleware.JsonResponse(c, fiber.StatusOK, true, "Withdraw Sucess.", nil)
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
