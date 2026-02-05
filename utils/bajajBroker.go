package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// BajajQuoteResponse represents the response from Bajaj quote API
type BajajQuoteResponse struct {
	Data []struct {
		LastPrice float64 `json:"last_price"`
		Open      float64 `json:"open"`
		High      float64 `json:"high"`
		Low       float64 `json:"low"`
		Close     float64 `json:"close"`
		Volume    int64   `json:"volume"`
	} `json:"data"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// GetBajajQuote fetches live stock price from Bajaj Broking API
// stockToken is the exchange token for the stock (e.g., 6232 for a stock)
func GetBajajQuote(accessToken string, stockToken int) (float64, error) {
	if accessToken == "" {
		return 0, fmt.Errorf("access token is required")
	}

	url := fmt.Sprintf("https://bridgelink.bajajbroking.in/api/market/quote?exchid_token=0_%d", stockToken)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch quote: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API error: %s", string(body))
	}

	var quoteResp BajajQuoteResponse
	if err := json.Unmarshal(body, &quoteResp); err != nil {
		return 0, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(quoteResp.Data) == 0 {
		return 0, fmt.Errorf("no quote data returned")
	}

	return quoteResp.Data[0].LastPrice, nil
}

// GetBajajQuoteDetails fetches detailed stock quote from Bajaj API
func GetBajajQuoteDetails(accessToken string, stockToken int) (*BajajQuoteResponse, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}

	url := fmt.Sprintf("https://bridgelink.bajajbroking.in/api/market/quote?exchid_token=0_%d", stockToken)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var quoteResp BajajQuoteResponse
	if err := json.Unmarshal(body, &quoteResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &quoteResp, nil
}
