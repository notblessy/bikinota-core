package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/notblessy/bikinota-core/model"
	"gorm.io/gorm"
)

type InvoiceRepository interface {
	FindByUserID(ctx context.Context, userID uint) ([]*model.Invoice, error)
	FindByID(ctx context.Context, id uint) (*model.Invoice, error)
	Create(ctx context.Context, invoice *model.Invoice) error
	Update(ctx context.Context, invoice *model.Invoice) error
	Delete(ctx context.Context, id uint) error
}

type invoiceRepository struct {
	db *gorm.DB
}

func NewInvoiceRepository(db *gorm.DB) InvoiceRepository {
	return &invoiceRepository{db: db}
}

func (r *invoiceRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.Invoice, error) {
	var invoices []*model.Invoice
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Adjustments").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&invoices).Error
	return invoices, err
}

func (r *invoiceRepository) FindByID(ctx context.Context, id uint) (*model.Invoice, error) {
	var invoice model.Invoice
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Adjustments").
		First(&invoice, id).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *invoiceRepository) Create(ctx context.Context, invoice *model.Invoice) error {
	// Generate invoice number
	year := time.Now().Year()
	month := int(time.Now().Month())
	
	// Count invoices for this user this month
	var count int64
	r.db.WithContext(ctx).Model(&model.Invoice{}).
		Where("user_id = ? AND EXTRACT(YEAR FROM created_at) = ? AND EXTRACT(MONTH FROM created_at) = ?", 
			invoice.UserID, year, month).
		Count(&count)
	
	invoice.InvoiceNumber = fmt.Sprintf("INV-%d%02d-%03d", year, month, count+1)
	
	return r.db.WithContext(ctx).Create(invoice).Error
}

func (r *invoiceRepository) Update(ctx context.Context, invoice *model.Invoice) error {
	// Use transaction to ensure atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get existing items and adjustments
		var existingItems []model.InvoiceItem
		var existingAdjustments []model.InvoiceAdjustment
		tx.Where("invoice_id = ?", invoice.ID).Find(&existingItems)
		tx.Where("invoice_id = ?", invoice.ID).Find(&existingAdjustments)

		// Create maps of existing items/adjustments by ID for quick lookup
		existingItemsMap := make(map[uint]bool)
		for _, item := range existingItems {
			existingItemsMap[item.ID] = true
		}
		existingAdjustmentsMap := make(map[uint]bool)
		for _, adj := range existingAdjustments {
			existingAdjustmentsMap[adj.ID] = true
		}

		// Track which items/adjustments are being kept
		keptItemsMap := make(map[uint]bool)
		keptAdjustmentsMap := make(map[uint]bool)

		// Update or create items
		if len(invoice.Items) > 0 {
			for i := range invoice.Items {
				invoice.Items[i].InvoiceID = invoice.ID
				// If item has ID and exists, update it; otherwise create new
				if invoice.Items[i].ID != 0 && existingItemsMap[invoice.Items[i].ID] {
					keptItemsMap[invoice.Items[i].ID] = true
					if err := tx.Save(&invoice.Items[i]).Error; err != nil {
						return err
					}
				} else {
					// Clear ID to create new item
					invoice.Items[i].ID = 0
					if err := tx.Create(&invoice.Items[i]).Error; err != nil {
						return err
					}
				}
			}
		}

		// Update or create adjustments
		if len(invoice.Adjustments) > 0 {
			for i := range invoice.Adjustments {
				invoice.Adjustments[i].InvoiceID = invoice.ID
				// If adjustment has ID and exists, update it; otherwise create new
				if invoice.Adjustments[i].ID != 0 && existingAdjustmentsMap[invoice.Adjustments[i].ID] {
					keptAdjustmentsMap[invoice.Adjustments[i].ID] = true
					if err := tx.Save(&invoice.Adjustments[i]).Error; err != nil {
						return err
					}
				} else {
					// Clear ID to create new adjustment
					invoice.Adjustments[i].ID = 0
					if err := tx.Create(&invoice.Adjustments[i]).Error; err != nil {
						return err
					}
				}
			}
		}

		// Delete items that are no longer in the list
		for _, existingItem := range existingItems {
			if !keptItemsMap[existingItem.ID] {
				if err := tx.Delete(&existingItem).Error; err != nil {
					return err
				}
			}
		}

		// Delete adjustments that are no longer in the list
		for _, existingAdj := range existingAdjustments {
			if !keptAdjustmentsMap[existingAdj.ID] {
				if err := tx.Delete(&existingAdj).Error; err != nil {
					return err
				}
			}
		}

		// Update invoice (without items and adjustments to avoid conflicts)
		invoiceCopy := *invoice
		invoiceCopy.Items = nil
		invoiceCopy.Adjustments = nil
		if err := tx.Save(&invoiceCopy).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *invoiceRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Invoice{}, id).Error
}

// Helper function to convert string ID to uint
func parseUintID(idStr string) (uint, error) {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

