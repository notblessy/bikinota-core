package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/notblessy/bikinota-core/service"
	"github.com/sirupsen/logrus"
)

type PaymentHandler struct {
	polarService *service.PolarService
	validate     *validator.Validate
}

func NewPaymentHandler() *PaymentHandler {
	return &PaymentHandler{
		polarService: service.NewPolarService(),
		validate:     validator.New(),
	}
}

// CreateCheckoutLink creates a new Polar checkout link dynamically
func (h *PaymentHandler) CreateCheckoutLink(c echo.Context) error {
	logger := logrus.WithField("endpoint", "create_checkout_link")

	user, err := authSession(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: err.Error(),
		})
	}

	// Get product ID from environment variable
	productID := os.Getenv("POLAR_PRODUCT_ID")
	if productID == "" {
		logger.Error("POLAR_PRODUCT_ID not configured")
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "payment configuration error",
		})
	}

	successURL := os.Getenv("POLAR_SUCCESS_URL")
	if successURL == "" {
		logger.Error("POLAR_SUCCESS_URL not configured")
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "payment configuration error",
		})
	}

	// Create checkout session via Polar API
	// Pass customer email and external customer ID (user ID) to:
	// 1. Autofill the email field
	// 2. Disable email editing by linking to an existing customer
	externalCustomerID := fmt.Sprintf("user_%d", user.ID)
	checkoutURL, err := h.polarService.CreateCheckoutLink(c.Request().Context(), productID, successURL, user.Email, externalCustomerID)
	if err != nil {
		logger.WithError(err).Error("Error creating checkout link")
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: fmt.Sprintf("failed to create checkout link: %v", err),
		})
	}

	// Validate URL before proceeding
	if checkoutURL == "" {
		logger.Error("Received empty checkout URL from Polar service")
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "invalid checkout URL received",
		})
	}

	// Validate URL format
	_, err = url.Parse(checkoutURL)
	if err != nil {
		logger.Errorf("Invalid checkout URL format: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "invalid checkout URL format",
		})
	}

	// Return response with checkout URL in data field
	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    checkoutURL,
	})
}
