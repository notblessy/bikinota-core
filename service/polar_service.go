package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"

	"github.com/sirupsen/logrus"
)

type PolarService struct {
	apiToken string
	apiURL   string
	client   *http.Client
}

// CreateCheckoutSessionRequest represents the request to create a checkout session
type CreateCheckoutSessionRequest struct {
	Products           []string               `json:"products"`
	CustomerEmail      string                 `json:"customer_email,omitempty"`
	ExternalCustomerID string                 `json:"external_customer_id,omitempty"`
	SuccessURL         string                 `json:"success_url,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// CheckoutSessionResponse represents the checkout session response
type CheckoutSessionResponse struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

func NewPolarService() *PolarService {
	apiToken := os.Getenv("POLAR_API_ACCESS_TOKEN")
	apiURL := os.Getenv("POLAR_API_URL")
	if apiURL == "" {
		apiURL = "https://api.polar.sh"
	}

	return &PolarService{
		apiToken: apiToken,
		apiURL:   apiURL,
		client:   &http.Client{},
	}
}

// CreateCheckoutLink creates a new checkout session using Polar API
// This uses Checkout Sessions API which returns polar.sh/checkout/... URLs that support email autofill
// The customer_email parameter will prefill the email field in the checkout form
// The external_customer_id parameter will disable email editing by linking to an existing customer
func (s *PolarService) CreateCheckoutLink(ctx context.Context, productID, successURL, customerEmail, externalCustomerID string) (string, error) {
	logger := logrus.WithContext(ctx).WithFields(logrus.Fields{
		"product_id":           productID,
		"success_url":          successURL,
		"customer_email":       customerEmail,
		"external_customer_id": externalCustomerID,
	})

	if s.apiToken == "" {
		return "", fmt.Errorf("POLAR_API_ACCESS_TOKEN not configured")
	}

	if productID == "" {
		return "", fmt.Errorf("product_id is required")
	}

	// Create checkout session request with products array, customer email, and external customer ID
	// Using external_customer_id will disable email editing in the checkout form
	reqBody := CreateCheckoutSessionRequest{
		Products:           []string{productID},
		CustomerEmail:      customerEmail,
		ExternalCustomerID: externalCustomerID,
		SuccessURL:         successURL,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logger.Errorf("Error marshaling request: %v", err)
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log request for debugging
	logger.WithField("request_body", string(jsonData)).Debug("Sending request to Polar API")

	// Use checkouts endpoint instead of checkout-links
	url := fmt.Sprintf("%s/v1/checkouts", s.apiURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Errorf("Error creating request: %v", err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		logger.Errorf("Error making request to Polar API: %v", err)
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("Error reading response: %v", err)
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("Polar API returned error")
		return "", fmt.Errorf("Polar API error: status %d, response: %s", resp.StatusCode, string(body))
	}

	var checkoutResponse CheckoutSessionResponse
	if err := json.Unmarshal(body, &checkoutResponse); err != nil {
		logger.WithField("response_body", string(body)).Errorf("Error unmarshaling response: %v", err)
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Log the parsed response structure for debugging
	logger.WithFields(logrus.Fields{
		"checkout_id":  checkoutResponse.ID,
		"checkout_url": checkoutResponse.URL,
		"has_url":      checkoutResponse.URL != "",
	}).Info("Parsed checkout session response")

	checkoutURL := checkoutResponse.URL
	if checkoutURL == "" {
		// Try to find URL in alternative locations
		var rawResponse map[string]interface{}
		if err := json.Unmarshal(body, &rawResponse); err == nil {
			// Check if URL is at root level
			if urlVal, ok := rawResponse["url"].(string); ok && urlVal != "" {
				checkoutURL = urlVal
				logger.Info("Found URL at root level of response")
			}
		}

		if checkoutURL == "" {
			logger.WithField("response_body", string(body)).Error("Polar API returned empty checkout URL - could not find URL in response")
			return "", fmt.Errorf("Polar API returned empty checkout URL. Response: %s", string(body))
		}
	}

	// Validate URL format and ensure it's absolute
	parsedURL, err := neturl.Parse(checkoutURL)
	if err != nil {
		logger.WithField("checkout_url", checkoutURL).Errorf("Invalid URL format: %v", err)
		return "", fmt.Errorf("invalid checkout URL format: %w", err)
	}

	// Ensure URL is absolute (has scheme)
	if parsedURL.Scheme == "" {
		logger.WithField("checkout_url", checkoutURL).Error("Checkout URL is not absolute (missing scheme)")
		return "", fmt.Errorf("checkout URL is not absolute: %s", checkoutURL)
	}

	// Ensure URL uses http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		logger.WithField("checkout_url", checkoutURL).Errorf("Checkout URL uses unsupported scheme: %s", parsedURL.Scheme)
		return "", fmt.Errorf("checkout URL uses unsupported scheme: %s", parsedURL.Scheme)
	}

	logger.WithField("checkout_url", checkoutURL).Info("Checkout session created successfully")
	return checkoutURL, nil
}
