package model

// Polar.sh webhook event types
type WebhookEventType string

const (
	WebhookEventSubscriptionCreated  WebhookEventType = "subscription.created"
	WebhookEventSubscriptionUpdated  WebhookEventType = "subscription.updated"
	WebhookEventSubscriptionCanceled WebhookEventType = "subscription.canceled"
	WebhookEventOrderCreated         WebhookEventType = "order.created"
)

// WebhookEvent represents a Polar.sh webhook event
type WebhookEvent struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// CheckoutData represents checkout data from webhook
type CheckoutData struct {
	ID            string                 `json:"id"`
	Status        string                 `json:"status"`
	CustomerEmail *string                `json:"customer_email"`
	Customer      *CustomerData          `json:"customer,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// CustomerData represents customer data from webhook
type CustomerData struct {
	ID            string                 `json:"id"`
	ExternalID    string                 `json:"external_id,omitempty"`
	Email         string                 `json:"email"`
	EmailVerified bool                   `json:"email_verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SubscriptionData represents subscription data from webhook
type SubscriptionData struct {
	ID                 string                 `json:"id"`
	CheckoutID         string                 `json:"checkout_id"`
	Status             string                 `json:"status"`
	CustomerEmail      *string                `json:"customer_email"`
	Customer           *CustomerData          `json:"customer,omitempty"`
	CurrentPeriodStart *string                `json:"current_period_start"`
	CurrentPeriodEnd   *string                `json:"current_period_end"`
	CancelAtPeriodEnd  bool                   `json:"cancel_at_period_end"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// OrderData represents order data from webhook
type OrderData struct {
	ID            string                 `json:"id"`
	CheckoutID    string                 `json:"checkout_id"`
	Status        string                 `json:"status"`
	CustomerEmail *string                `json:"customer_email"`
	Customer      *CustomerData          `json:"customer,omitempty"`
	Amount        int                    `json:"amount"`
	Currency      string                 `json:"currency"`
	Metadata      map[string]interface{} `json:"metadata"`
}
