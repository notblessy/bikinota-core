package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/notblessy/bikinota-core/model"
	"github.com/notblessy/bikinota-core/repository"
	"github.com/sirupsen/logrus"
)

// Helper function to convert rupiah to cents (multiply by 100)
func rupiahToCents(rupiah float64) int {
	return int(rupiah * 100)
}

type invoiceHandler struct {
	invoiceRepo repository.InvoiceRepository
	validate    *validator.Validate
}

func NewInvoiceHandler(invoiceRepo repository.InvoiceRepository) *invoiceHandler {
	return &invoiceHandler{
		invoiceRepo: invoiceRepo,
		validate:    validator.New(),
	}
}

// GetInvoices retrieves all invoices for the authenticated user
func (h *invoiceHandler) GetInvoices(c echo.Context) error {
	logger := logrus.WithField("endpoint", "get_invoices")

	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	invoices, err := h.invoiceRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil {
		logger.Errorf("Error finding invoices: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to retrieve invoices",
		})
	}

	invoiceResponses := make([]model.InvoiceResponse, len(invoices))
	for i, inv := range invoices {
		invoiceResponses[i] = inv.ToInvoiceResponse()
	}

	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    invoiceResponses,
	})
}

// GetInvoice retrieves a single invoice by ID
func (h *invoiceHandler) GetInvoice(c echo.Context) error {
	logger := logrus.WithField("endpoint", "get_invoice")

	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid invoice id",
		})
	}

	invoice, err := h.invoiceRepo.FindByID(c.Request().Context(), uint(id))
	if err != nil {
		logger.Errorf("Error finding invoice: %v", err)
		return c.JSON(http.StatusNotFound, response{
			Success: false,
			Message: "invoice not found",
		})
	}

	// Verify invoice belongs to user
	if invoice.UserID != userClaims.ID {
		return c.JSON(http.StatusForbidden, response{
			Success: false,
			Message: "access denied",
		})
	}

	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    invoice.ToInvoiceResponse(),
	})
}

// CreateInvoice creates a new invoice
func (h *invoiceHandler) CreateInvoice(c echo.Context) error {
	logger := logrus.WithField("endpoint", "create_invoice")

	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	var req model.CreateInvoiceRequest
	if err := c.Bind(&req); err != nil {
		logger.Errorf("Error binding request: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid request",
		})
	}

	if err := h.validate.Struct(req); err != nil {
		logger.Errorf("Validation error: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "validation failed",
		})
	}

	// Parse due date (optional)
	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		parsedDate, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			return c.JSON(http.StatusBadRequest, response{
				Success: false,
				Message: "invalid due date format",
			})
		}
		dueDate = &parsedDate
	}

	// Convert items
	items := make([]model.InvoiceItem, len(req.Items))
	for i, itemReq := range req.Items {
		items[i] = model.InvoiceItem{
			Name:        itemReq.Name,
			Description: itemReq.Description,
			Quantity:    itemReq.Quantity,
			Price:       rupiahToCents(itemReq.Price),
		}
	}

	// Convert adjustments
	adjustments := make([]model.InvoiceAdjustment, len(req.Adjustments))
	for i, adjReq := range req.Adjustments {
		adjustments[i] = model.InvoiceAdjustment{
			Description: adjReq.Description,
			Type:        adjReq.Type,
			Amount:      rupiahToCents(adjReq.Amount),
		}
	}

	// Calculate totals
	subtotal := 0
	for _, item := range items {
		subtotal += item.Quantity * item.Price
	}

	adjustmentsTotal := 0
	for _, adj := range adjustments {
		if adj.Type == "addition" {
			adjustmentsTotal += adj.Amount
		} else {
			adjustmentsTotal -= adj.Amount
		}
	}

	taxAmount := int(float64(subtotal) * req.TaxRate / 100.0)
	total := subtotal + taxAmount + adjustmentsTotal

	// Parse bank account ID if provided
	var bankAccountID *uint
	if req.BankAccountID != nil && *req.BankAccountID != "" {
		id, err := strconv.ParseUint(*req.BankAccountID, 10, 32)
		if err == nil {
			uid := uint(id)
			bankAccountID = &uid
		}
	}

	invoice := &model.Invoice{
		UserID:           userClaims.ID,
		CustomerName:     req.CustomerName,
		CustomerEmail:    req.CustomerEmail,
		DueDate:          dueDate,
		TaxRate:          req.TaxRate,
		Status:           req.Status,
		Subtotal:         subtotal,
		TaxAmount:        taxAmount,
		AdjustmentsTotal: adjustmentsTotal,
		Total:            total,
		BankAccountID:    bankAccountID,
		Items:            items,
		Adjustments:      adjustments,
	}

	if err := h.invoiceRepo.Create(c.Request().Context(), invoice); err != nil {
		logger.Errorf("Error creating invoice: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to create invoice",
		})
	}

	return c.JSON(http.StatusCreated, response{
		Success: true,
		Data:    invoice.ToInvoiceResponse(),
	})
}

// UpdateInvoice updates an existing invoice
func (h *invoiceHandler) UpdateInvoice(c echo.Context) error {
	logger := logrus.WithField("endpoint", "update_invoice")

	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid invoice id",
		})
	}

	invoice, err := h.invoiceRepo.FindByID(c.Request().Context(), uint(id))
	if err != nil {
		logger.Errorf("Error finding invoice: %v", err)
		return c.JSON(http.StatusNotFound, response{
			Success: false,
			Message: "invoice not found",
		})
	}

	// Verify invoice belongs to user
	if invoice.UserID != userClaims.ID {
		return c.JSON(http.StatusForbidden, response{
			Success: false,
			Message: "access denied",
		})
	}

	var req model.UpdateInvoiceRequest
	if err := c.Bind(&req); err != nil {
		logger.Errorf("Error binding request: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid request",
		})
	}

	// Update fields
	if req.CustomerName != nil {
		invoice.CustomerName = *req.CustomerName
	}
	if req.CustomerEmail != nil {
		invoice.CustomerEmail = *req.CustomerEmail
	}
	if req.DueDate != nil {
		if *req.DueDate == "" {
			// Clear due date if empty string is sent
			invoice.DueDate = nil
		} else {
			dueDate, err := time.Parse("2006-01-02", *req.DueDate)
			if err != nil {
				return c.JSON(http.StatusBadRequest, response{
					Success: false,
					Message: "invalid due date format",
				})
			}
			invoice.DueDate = &dueDate
		}
	}
	if req.TaxRate != nil {
		invoice.TaxRate = *req.TaxRate
	}
	if req.Status != nil {
		invoice.Status = *req.Status
	}

	// Update items if provided
	if req.Items != nil {
		items := make([]model.InvoiceItem, len(req.Items))
		for i, itemReq := range req.Items {
			item := model.InvoiceItem{
				Name:        itemReq.Name,
				Description: itemReq.Description,
				Quantity:    itemReq.Quantity,
				Price:       rupiahToCents(itemReq.Price),
			}
			// If ID is provided, parse it and set it (for updating existing items)
			if itemReq.ID != nil && *itemReq.ID != "" {
				id, err := strconv.ParseUint(*itemReq.ID, 10, 32)
				if err == nil {
					item.ID = uint(id)
				}
			}
			items[i] = item
		}
		invoice.Items = items
	}

	// Update adjustments if provided
	if req.Adjustments != nil {
		adjustments := make([]model.InvoiceAdjustment, len(req.Adjustments))
		for i, adjReq := range req.Adjustments {
			adj := model.InvoiceAdjustment{
				Description: adjReq.Description,
				Type:        adjReq.Type,
				Amount:      rupiahToCents(adjReq.Amount),
			}
			// If ID is provided, parse it and set it (for updating existing adjustments)
			if adjReq.ID != nil && *adjReq.ID != "" {
				id, err := strconv.ParseUint(*adjReq.ID, 10, 32)
				if err == nil {
					adj.ID = uint(id)
				}
			}
			adjustments[i] = adj
		}
		invoice.Adjustments = adjustments
	}

	// Recalculate totals if items, adjustments, or tax rate changed
	if req.Items != nil || req.Adjustments != nil || req.TaxRate != nil {
		subtotal := 0
		for _, item := range invoice.Items {
			subtotal += item.Quantity * item.Price
		}

		adjustmentsTotal := 0
		for _, adj := range invoice.Adjustments {
			if adj.Type == "addition" {
				adjustmentsTotal += adj.Amount
			} else {
				adjustmentsTotal -= adj.Amount
			}
		}

		taxAmount := int(float64(subtotal) * invoice.TaxRate / 100.0)
		invoice.Subtotal = subtotal
		invoice.TaxAmount = taxAmount
		invoice.AdjustmentsTotal = adjustmentsTotal
		invoice.Total = subtotal + taxAmount + adjustmentsTotal
	}

	// Update bank account ID if provided
	if req.BankAccountID != nil {
		if *req.BankAccountID == "" {
			invoice.BankAccountID = nil
		} else {
			id, err := strconv.ParseUint(*req.BankAccountID, 10, 32)
			if err == nil {
				uid := uint(id)
				invoice.BankAccountID = &uid
			}
		}
	}

	if err := h.invoiceRepo.Update(c.Request().Context(), invoice); err != nil {
		logger.Errorf("Error updating invoice: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to update invoice",
		})
	}

	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    invoice.ToInvoiceResponse(),
	})
}

// DeleteInvoice deletes an invoice
func (h *invoiceHandler) DeleteInvoice(c echo.Context) error {
	logger := logrus.WithField("endpoint", "delete_invoice")

	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid invoice id",
		})
	}

	invoice, err := h.invoiceRepo.FindByID(c.Request().Context(), uint(id))
	if err != nil {
		logger.Errorf("Error finding invoice: %v", err)
		return c.JSON(http.StatusNotFound, response{
			Success: false,
			Message: "invoice not found",
		})
	}

	// Verify invoice belongs to user
	if invoice.UserID != userClaims.ID {
		return c.JSON(http.StatusForbidden, response{
			Success: false,
			Message: "access denied",
		})
	}

	if err := h.invoiceRepo.Delete(c.Request().Context(), uint(id)); err != nil {
		logger.Errorf("Error deleting invoice: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to delete invoice",
		})
	}

	return c.JSON(http.StatusOK, response{
		Success: true,
		Message: "invoice deleted successfully",
	})
}

