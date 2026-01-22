package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/notblessy/bikinota-core/model"
	"github.com/notblessy/bikinota-core/repository"
	"github.com/notblessy/bikinota-core/standardwebhooks"
	"github.com/sirupsen/logrus"
)

type WebhookHandler struct {
	planRepo      repository.PlanRepository
	userRepo      repository.UserRepository
	webhookSecret string
}

func NewWebhookHandler(
	planRepo repository.PlanRepository,
	userRepo repository.UserRepository,
) *WebhookHandler {
	webhookSecret := os.Getenv("POLAR_WEBHOOK_SECRET")
	if webhookSecret == "" {
		logrus.Warn("POLAR_WEBHOOK_SECRET not set - webhook verification will fail")
	}

	return &WebhookHandler{
		planRepo:      planRepo,
		userRepo:      userRepo,
		webhookSecret: webhookSecret,
	}
}

// verifyWebhookSignature verifies the Polar.sh webhook signature using standard-webhooks library
func (h *WebhookHandler) verifyWebhookSignature(body []byte, headers http.Header) bool {
	if h.webhookSecret == "" {
		logrus.Warn("Webhook secret not configured, skipping verification")
		return false
	}

	// Base64 encode the secret as required by standard-webhooks
	secret := base64.StdEncoding.EncodeToString([]byte(h.webhookSecret))
	wh, err := standardwebhooks.NewWebhook(secret)
	if err != nil {
		logrus.WithError(err).Error("Failed to create webhook instance")
		return false
	}

	// Verify using the entire request headers
	err = wh.Verify(body, headers)
	if err != nil {
		logrus.WithError(err).Warn("Webhook signature verification failed")
		return false
	}

	return true
}

// validateExternalCustomerID validates that the external customer ID matches the user's ID
// Expected format: "user_{userID}"
func (h *WebhookHandler) validateExternalCustomerID(externalCustomerID string, userID uint) error {
	if externalCustomerID == "" {
		return fmt.Errorf("external_customer_id is missing")
	}

	// Expected format: "user_{id}"
	expectedID := fmt.Sprintf("user_%d", userID)
	if externalCustomerID != expectedID {
		return fmt.Errorf("external_customer_id mismatch: expected %s, got %s", expectedID, externalCustomerID)
	}

	return nil
}

// extractUserIDFromExternalID extracts user ID from external customer ID format "user_{id}"
func extractUserIDFromExternalID(externalID string) (uint, error) {
	if !strings.HasPrefix(externalID, "user_") {
		return 0, fmt.Errorf("invalid external_customer_id format: must start with 'user_'")
	}

	idStr := strings.TrimPrefix(externalID, "user_")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID in external_customer_id: %w", err)
	}

	return uint(id), nil
}

// HandleWebhook processes incoming webhook events from Polar.sh
func (h *WebhookHandler) HandleWebhook(c echo.Context) error {
	logger := logrus.WithField("endpoint", "webhook")

	// Read the raw request body for signature verification
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		logger.Errorf("Error reading request body: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "failed to read request body",
		})
	}

	// Restore body for JSON parsing
	c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Verify webhook signature using standard-webhooks library
	if !h.verifyWebhookSignature(bodyBytes, c.Request().Header) {
		logger.Warn("Invalid webhook signature")
		return c.NoContent(http.StatusForbidden)
	}

	// Parse webhook event
	var event model.WebhookEvent
	if err := c.Bind(&event); err != nil {
		logger.Errorf("Error parsing webhook event: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid event format",
		})
	}

	logger.WithFields(logrus.Fields{
		"event_type": event.Type,
	}).Info("Received webhook event")

	// Handle different event types
	switch model.WebhookEventType(event.Type) {
	case model.WebhookEventSubscriptionCreated:
		return h.handleSubscriptionCreated(c, event)
	case model.WebhookEventSubscriptionUpdated:
		return h.handleSubscriptionUpdated(c, event)
	case model.WebhookEventSubscriptionCanceled:
		return h.handleSubscriptionCanceled(c, event)
	case model.WebhookEventOrderCreated:
		return h.handleOrderCreated(c, event)
	default:
		logger.WithField("event_type", event.Type).Warn("Unhandled webhook event type")
		return c.JSON(http.StatusOK, response{
			Success: true,
			Message: "event received but not handled",
		})
	}
}

// handleSubscriptionCreated handles subscription.created events
func (h *WebhookHandler) handleSubscriptionCreated(c echo.Context, event model.WebhookEvent) error {
	logger := logrus.WithField("handler", "subscription_created")

	subscriptionDataBytes, err := json.Marshal(event.Data)
	if err != nil {
		logger.Errorf("Error marshaling subscription data: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid subscription data",
		})
	}

	var subscriptionData model.SubscriptionData
	if err := json.Unmarshal(subscriptionDataBytes, &subscriptionData); err != nil {
		logger.Errorf("Error unmarshaling subscription data: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid subscription data format",
		})
	}

	logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionData.ID,
		"checkout_id":     subscriptionData.CheckoutID,
		"status":          subscriptionData.Status,
	}).Info("Processing subscription.created event")

	// Validate external customer ID to prevent email changes
	var user *model.User

	// First, try to find user by external_customer_id from customer data
	if subscriptionData.Customer != nil && subscriptionData.Customer.ExternalID != "" {
		userID, err := extractUserIDFromExternalID(subscriptionData.Customer.ExternalID)
		if err != nil {
			logger.WithField("external_id", subscriptionData.Customer.ExternalID).Errorf("Invalid external_customer_id format: %v", err)
			return c.JSON(http.StatusBadRequest, response{
				Success: false,
				Message: "invalid external_customer_id format",
			})
		}

		// Find user by ID
		user, err = h.userRepo.FindByID(c.Request().Context(), userID)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"external_id": subscriptionData.Customer.ExternalID,
				"user_id":     userID,
			}).Error("User not found for external_customer_id")
			return c.JSON(http.StatusBadRequest, response{
				Success: false,
				Message: "user not found for external_customer_id",
			})
		}

		// Validate that the email in the webhook matches the user's email
		if subscriptionData.CustomerEmail != nil && *subscriptionData.CustomerEmail != user.Email {
			logger.WithFields(logrus.Fields{
				"user_email":    user.Email,
				"webhook_email": *subscriptionData.CustomerEmail,
				"external_id":   subscriptionData.Customer.ExternalID,
			}).Error("Email mismatch: webhook email does not match user email")
			return c.JSON(http.StatusBadRequest, response{
				Success: false,
				Message: "email mismatch: customer email changed",
			})
		}

		// Validate external customer ID matches
		if err := h.validateExternalCustomerID(subscriptionData.Customer.ExternalID, user.ID); err != nil {
			logger.WithFields(logrus.Fields{
				"external_id": subscriptionData.Customer.ExternalID,
				"user_id":     user.ID,
			}).Errorf("External customer ID validation failed: %v", err)
			return c.JSON(http.StatusBadRequest, response{
				Success: false,
				Message: fmt.Sprintf("external_customer_id validation failed: %v", err),
			})
		}
	} else {
		// Fallback: Find user by email if external_customer_id is not available
		if subscriptionData.CustomerEmail == nil || *subscriptionData.CustomerEmail == "" {
			logger.Warn("No customer email or external_customer_id in subscription data")
			return c.JSON(http.StatusOK, response{
				Success: true,
				Message: "subscription processed but no customer information",
			})
		}

		user, err = h.userRepo.FindByEmail(c.Request().Context(), *subscriptionData.CustomerEmail)
		if err != nil {
			logger.WithField("email", *subscriptionData.CustomerEmail).Warn("User not found for subscription")
			return c.JSON(http.StatusOK, response{
				Success: true,
				Message: "subscription processed but user not found",
			})
		}

		logger.Warn("External customer ID not found in webhook - using email fallback (less secure)")
	}

	// Update user's plan to unlimited if subscription is active
	if subscriptionData.Status == "active" {
		plan, err := h.planRepo.FindByUserID(c.Request().Context(), user.ID)
		if err != nil {
			logger.Errorf("Error finding plan: %v", err)
		} else {
			if plan == nil {
				plan = &model.Plan{
					UserID:   user.ID,
					PlanType: model.PlanUnlimited,
				}
				if err := h.planRepo.Create(c.Request().Context(), plan); err != nil {
					logger.Errorf("Error creating plan: %v", err)
				}
			} else {
				plan.PlanType = model.PlanUnlimited
				if err := h.planRepo.Update(c.Request().Context(), plan); err != nil {
					logger.Errorf("Error updating plan: %v", err)
				}
			}
		}
	}

	logger.WithFields(logrus.Fields{
		"user_id":         user.ID,
		"subscription_id": subscriptionData.ID,
	}).Info("Subscription created from webhook")

	return c.JSON(http.StatusOK, response{
		Success: true,
		Message: "subscription created successfully",
	})
}

// handleSubscriptionUpdated handles subscription.updated events
func (h *WebhookHandler) handleSubscriptionUpdated(c echo.Context, event model.WebhookEvent) error {
	logger := logrus.WithField("handler", "subscription_updated")

	subscriptionDataBytes, err := json.Marshal(event.Data)
	if err != nil {
		logger.Errorf("Error marshaling subscription data: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid subscription data",
		})
	}

	var subscriptionData model.SubscriptionData
	if err := json.Unmarshal(subscriptionDataBytes, &subscriptionData); err != nil {
		logger.Errorf("Error unmarshaling subscription data: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid subscription data format",
		})
	}

	logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionData.ID,
		"status":          subscriptionData.Status,
	}).Info("Processing subscription.updated event")

	// Find user by external_customer_id or email
	var user *model.User
	if subscriptionData.Customer != nil && subscriptionData.Customer.ExternalID != "" {
		userID, err := extractUserIDFromExternalID(subscriptionData.Customer.ExternalID)
		if err == nil {
			user, err = h.userRepo.FindByID(c.Request().Context(), userID)
		}
	}

	if user == nil && subscriptionData.CustomerEmail != nil {
		user, _ = h.userRepo.FindByEmail(c.Request().Context(), *subscriptionData.CustomerEmail)
	}

	if user == nil {
		logger.Warn("User not found for subscription update")
		return c.JSON(http.StatusOK, response{
			Success: true,
			Message: "subscription updated but user not found",
		})
	}

	// Update user's plan based on subscription status
	plan, err := h.planRepo.FindByUserID(c.Request().Context(), user.ID)
	if err != nil {
		logger.Errorf("Error finding plan: %v", err)
	} else {
		if plan == nil {
			plan = &model.Plan{
				UserID:   user.ID,
				PlanType: model.PlanFree,
			}
		}

		if subscriptionData.Status == "active" {
			plan.PlanType = model.PlanUnlimited
		} else if subscriptionData.Status == "canceled" || subscriptionData.Status == "expired" {
			plan.PlanType = model.PlanFree
		}

		if plan.ID == 0 {
			if err := h.planRepo.Create(c.Request().Context(), plan); err != nil {
				logger.Errorf("Error creating plan: %v", err)
			}
		} else {
			if err := h.planRepo.Update(c.Request().Context(), plan); err != nil {
				logger.Errorf("Error updating plan: %v", err)
			}
		}
	}

	logger.WithField("subscription_id", subscriptionData.ID).Info("Subscription updated from webhook")

	return c.JSON(http.StatusOK, response{
		Success: true,
		Message: "subscription updated successfully",
	})
}

// handleSubscriptionCanceled handles subscription.canceled events
func (h *WebhookHandler) handleSubscriptionCanceled(c echo.Context, event model.WebhookEvent) error {
	logger := logrus.WithField("handler", "subscription_canceled")

	subscriptionDataBytes, err := json.Marshal(event.Data)
	if err != nil {
		logger.Errorf("Error marshaling subscription data: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid subscription data",
		})
	}

	var subscriptionData model.SubscriptionData
	if err := json.Unmarshal(subscriptionDataBytes, &subscriptionData); err != nil {
		logger.Errorf("Error unmarshaling subscription data: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid subscription data format",
		})
	}

	logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionData.ID,
		"checkout_id":     subscriptionData.CheckoutID,
	}).Info("Processing subscription.canceled event")

	// Find user by external_customer_id or email
	var user *model.User
	if subscriptionData.Customer != nil && subscriptionData.Customer.ExternalID != "" {
		userID, err := extractUserIDFromExternalID(subscriptionData.Customer.ExternalID)
		if err == nil {
			user, err = h.userRepo.FindByID(c.Request().Context(), userID)
		}
	}

	if user == nil && subscriptionData.CustomerEmail != nil {
		user, _ = h.userRepo.FindByEmail(c.Request().Context(), *subscriptionData.CustomerEmail)
	}

	if user == nil {
		logger.Warn("User not found for subscription cancellation")
		return c.JSON(http.StatusOK, response{
			Success: true,
			Message: "subscription canceled but user not found",
		})
	}

	// Update user's plan to free
	plan, err := h.planRepo.FindByUserID(c.Request().Context(), user.ID)
	if err != nil {
		logger.Errorf("Error finding plan: %v", err)
	} else {
		if plan == nil {
			plan = &model.Plan{
				UserID:   user.ID,
				PlanType: model.PlanFree,
			}
			if err := h.planRepo.Create(c.Request().Context(), plan); err != nil {
				logger.Errorf("Error creating plan: %v", err)
			}
		} else {
			plan.PlanType = model.PlanFree
			if err := h.planRepo.Update(c.Request().Context(), plan); err != nil {
				logger.Errorf("Error updating plan: %v", err)
			}
		}
	}

	logger.WithField("subscription_id", subscriptionData.ID).Info("Subscription canceled from webhook")

	return c.JSON(http.StatusOK, response{
		Success: true,
		Message: "subscription canceled successfully",
	})
}

// handleOrderCreated handles order.created events
func (h *WebhookHandler) handleOrderCreated(c echo.Context, event model.WebhookEvent) error {
	logger := logrus.WithField("handler", "order_created")

	orderDataBytes, err := json.Marshal(event.Data)
	if err != nil {
		logger.Errorf("Error marshaling order data: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid order data",
		})
	}

	var orderData model.OrderData
	if err := json.Unmarshal(orderDataBytes, &orderData); err != nil {
		logger.Errorf("Error unmarshaling order data: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid order data format",
		})
	}

	logger.WithFields(logrus.Fields{
		"order_id":    orderData.ID,
		"checkout_id": orderData.CheckoutID,
		"amount":      orderData.Amount,
	}).Info("Processing order.created event")

	// For now, we just log order events. In the future, you might want to track one-time payments
	// This could be used for one-time purchases or upgrades

	return c.JSON(http.StatusOK, response{
		Success: true,
		Message: "order processed successfully",
	})
}
