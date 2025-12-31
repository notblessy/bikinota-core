package model

import (
	"strconv"
	"time"

	"gorm.io/gorm"
)

type InvoiceItem struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	InvoiceID   uint   `json:"invoice_id" gorm:"not null;index"`
	Name        string `json:"name" gorm:"not null"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity" gorm:"not null"`
	Price       int    `json:"price" gorm:"not null"` // Stored in smallest currency unit (cents/sen)
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type InvoiceAdjustment struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	InvoiceID   uint   `json:"invoice_id" gorm:"not null;index"`
	Description string `json:"description" gorm:"not null"`
	Type        string `json:"type" gorm:"not null"`   // "addition" or "deduction"
	Amount      int    `json:"amount" gorm:"not null"` // Stored in smallest currency unit
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type Invoice struct {
	ID               uint                `json:"id" gorm:"primaryKey"`
	UserID           uint                `json:"user_id" gorm:"not null;index"`
	InvoiceNumber    string              `json:"invoice_number" gorm:"not null;uniqueIndex"`
	CustomerName     string              `json:"customer_name" gorm:"not null"`
	CustomerEmail    string              `json:"customer_email" gorm:"not null"`
	DueDate          *time.Time          `json:"due_date"` // Optional
	TaxRate          float64             `json:"tax_rate" gorm:"not null;default:0"`
	Status           string              `json:"status" gorm:"not null;default:draft"` // "draft", "sent", "paid"
	Subtotal         int                 `json:"subtotal" gorm:"not null"`             // Stored in smallest currency unit
	TaxAmount        int                 `json:"tax_amount" gorm:"not null"`           // Stored in smallest currency unit
	AdjustmentsTotal int                 `json:"adjustments_total" gorm:"not null"`    // Stored in smallest currency unit
	Total            int                 `json:"total" gorm:"not null"`                // Stored in smallest currency unit
	BankAccountID    *uint               `json:"bank_account_id" gorm:"index"`
	Items            []InvoiceItem       `json:"items" gorm:"foreignKey:InvoiceID"`
	Adjustments      []InvoiceAdjustment `json:"adjustments" gorm:"foreignKey:InvoiceID"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
	DeletedAt        gorm.DeletedAt      `json:"deleted_at" gorm:"index"`
}

// Request DTOs
type CreateInvoiceRequest struct {
	CustomerName  string                           `json:"customer_name" validate:"required"`
	CustomerEmail string                           `json:"customer_email" validate:"required,email"`
	DueDate       *string                          `json:"due_date"` // Optional
	TaxRate       float64                          `json:"tax_rate"`
	Status        string                           `json:"status" validate:"oneof=draft sent paid"`
	Items         []CreateInvoiceItemRequest       `json:"items" validate:"required,min=1,dive"`
	Adjustments   []CreateInvoiceAdjustmentRequest `json:"adjustments"`
	BankAccountID *string                          `json:"bank_account_id"`
}

type CreateInvoiceItemRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity" validate:"required,min=1"`
	Price       float64 `json:"price" validate:"required,min=0"`
}

type CreateInvoiceAdjustmentRequest struct {
	Description string  `json:"description" validate:"required"`
	Type        string  `json:"type" validate:"required,oneof=addition deduction"`
	Amount      float64 `json:"amount" validate:"required,min=0"`
}

type UpdateInvoiceItemRequest struct {
	ID          *string  `json:"id"` // Optional: if provided, item will be updated; if not, new item will be created
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description"`
	Quantity    int      `json:"quantity" validate:"required,min=1"`
	Price       float64  `json:"price" validate:"required,min=0"`
}

type UpdateInvoiceAdjustmentRequest struct {
	ID          *string  `json:"id"` // Optional: if provided, adjustment will be updated; if not, new adjustment will be created
	Description string  `json:"description" validate:"required"`
	Type        string  `json:"type" validate:"required,oneof=addition deduction"`
	Amount      float64 `json:"amount" validate:"required,min=0"`
}

type UpdateInvoiceRequest struct {
	CustomerName  *string                          `json:"customer_name"`
	CustomerEmail *string                          `json:"customer_email"`
	DueDate       *string                          `json:"due_date"`
	TaxRate       *float64                         `json:"tax_rate"`
	Status        *string                          `json:"status" validate:"omitempty,oneof=draft sent paid"`
	Items         []UpdateInvoiceItemRequest       `json:"items"`
	Adjustments   []UpdateInvoiceAdjustmentRequest `json:"adjustments"`
	BankAccountID *string                          `json:"bank_account_id"`
}

// Response DTOs
type InvoiceItemResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

type InvoiceAdjustmentResponse struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
}

type InvoiceResponse struct {
	ID               string                      `json:"id"`
	InvoiceNumber    string                      `json:"invoice_number"`
	CustomerName     string                      `json:"customer_name"`
	CustomerEmail    string                      `json:"customer_email"`
	DueDate          string                      `json:"due_date"`
	TaxRate          float64                     `json:"tax_rate"`
	Status           string                      `json:"status"`
	Subtotal         float64                     `json:"subtotal"`
	TaxAmount        float64                     `json:"tax_amount"`
	AdjustmentsTotal float64                     `json:"adjustments_total"`
	Total            float64                     `json:"total"`
	BankAccountID    *string                     `json:"bank_account_id"`
	Items            []InvoiceItemResponse       `json:"items"`
	Adjustments      []InvoiceAdjustmentResponse `json:"adjustments"`
	CreatedAt        string                      `json:"created_at"`
}

// Helper function to convert cents to rupiah (divide by 100)
func centsToRupiah(cents int) float64 {
	return float64(cents) / 100.0
}

// Helper function to convert rupiah to cents (multiply by 100)
func rupiahToCents(rupiah float64) int {
	return int(rupiah * 100)
}

func (i *Invoice) ToInvoiceResponse() InvoiceResponse {
	items := make([]InvoiceItemResponse, len(i.Items))
	for idx, item := range i.Items {
		items[idx] = InvoiceItemResponse{
			ID:          strconv.FormatUint(uint64(item.ID), 10),
			Name:        item.Name,
			Description: item.Description,
			Quantity:    item.Quantity,
			Price:       centsToRupiah(item.Price),
		}
	}

	adjustments := make([]InvoiceAdjustmentResponse, len(i.Adjustments))
	for idx, adj := range i.Adjustments {
		adjustments[idx] = InvoiceAdjustmentResponse{
			ID:          strconv.FormatUint(uint64(adj.ID), 10),
			Description: adj.Description,
			Type:        adj.Type,
			Amount:      centsToRupiah(adj.Amount),
		}
	}

	var bankAccountID *string
	if i.BankAccountID != nil {
		idStr := strconv.FormatUint(uint64(*i.BankAccountID), 10)
		bankAccountID = &idStr
	}

	return InvoiceResponse{
		ID:               strconv.FormatUint(uint64(i.ID), 10),
		InvoiceNumber:    i.InvoiceNumber,
		CustomerName:     i.CustomerName,
		CustomerEmail:    i.CustomerEmail,
		DueDate:          func() string {
			if i.DueDate != nil {
				return i.DueDate.Format("2006-01-02")
			}
			return ""
		}(),
		TaxRate:          i.TaxRate,
		Status:           i.Status,
		Subtotal:         centsToRupiah(i.Subtotal),
		TaxAmount:        centsToRupiah(i.TaxAmount),
		AdjustmentsTotal: centsToRupiah(i.AdjustmentsTotal),
		Total:            centsToRupiah(i.Total),
		BankAccountID:    bankAccountID,
		Items:            items,
		Adjustments:      adjustments,
		CreatedAt:        i.CreatedAt.Format(time.RFC3339),
	}
}
