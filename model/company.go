package model

import (
	"strconv"
	"time"

	"gorm.io/gorm"
)

type BankAccount struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	CompanyID     uint           `json:"company_id" gorm:"not null;index"`
	BankName      string         `json:"bank_name" gorm:"not null"`
	AccountName   string         `json:"account_name" gorm:"not null"`
	AccountNumber string         `json:"account_number" gorm:"not null"`
	SwiftCode     *string        `json:"swift_code,omitempty"`
	RoutingNumber *string        `json:"routing_number,omitempty"`
	IsDefault     bool           `json:"is_default" gorm:"default:false"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type Company struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	UserID       uint           `json:"user_id" gorm:"not null;uniqueIndex"`
	Name         string         `json:"name" gorm:"not null"`
	Address      string         `json:"address" gorm:"not null"`
	City         string         `json:"city" gorm:"not null"`
	State        string         `json:"state" gorm:"not null"`
	ZipCode      string         `json:"zip_code" gorm:"not null"`
	Country      string         `json:"country" gorm:"not null"`
	Email        string         `json:"email" gorm:"not null"`
	Phone        string         `json:"phone" gorm:"not null"`
	Website      string         `json:"website" gorm:"not null"`
	Logo         string         `json:"logo" gorm:"type:text"` // base64 encoded image
	BankAccounts []BankAccount  `json:"bank_accounts" gorm:"foreignKey:CompanyID"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type UpdateCompanyRequest struct {
	Name    *string `json:"name,omitempty"`
	Address *string `json:"address,omitempty"`
	City    *string `json:"city,omitempty"`
	State   *string `json:"state,omitempty"`
	ZipCode *string `json:"zip_code,omitempty"`
	Country *string `json:"country,omitempty"`
	Email   *string `json:"email,omitempty"`
	Phone   *string `json:"phone,omitempty"`
	Website *string `json:"website,omitempty"`
	Logo    *string `json:"logo,omitempty"`
}

type CreateBankAccountRequest struct {
	BankName      string  `json:"bank_name" validate:"required"`
	AccountName   string  `json:"account_name" validate:"required"`
	AccountNumber string  `json:"account_number" validate:"required"`
	SwiftCode     *string `json:"swift_code,omitempty"`
	RoutingNumber *string `json:"routing_number,omitempty"`
}

type UpdateBankAccountRequest struct {
	BankName      *string `json:"bank_name,omitempty"`
	AccountName   *string `json:"account_name,omitempty"`
	AccountNumber *string `json:"account_number,omitempty"`
	SwiftCode     *string `json:"swift_code,omitempty"`
	RoutingNumber *string `json:"routing_number,omitempty"`
}

// Response DTOs with string IDs to match frontend
type BankAccountResponse struct {
	ID            string  `json:"id"`
	BankName      string  `json:"bank_name"`
	AccountName   string  `json:"account_name"`
	AccountNumber string  `json:"account_number"`
	SwiftCode     *string `json:"swift_code,omitempty"`
	RoutingNumber *string `json:"routing_number,omitempty"`
	IsDefault     bool    `json:"is_default"`
}

type CompanyResponse struct {
	Name         string                `json:"name"`
	Address      string                `json:"address"`
	City         string                `json:"city"`
	State        string                `json:"state"`
	ZipCode      string                `json:"zip_code"`
	Country      string                `json:"country"`
	Email        string                `json:"email"`
	Phone        string                `json:"phone"`
	Website      string                `json:"website"`
	Logo         string                `json:"logo"`
	BankAccounts []BankAccountResponse `json:"bank_accounts"`
}

// ToBankAccountResponse converts BankAccount to BankAccountResponse
func (ba *BankAccount) ToBankAccountResponse() BankAccountResponse {
	return BankAccountResponse{
		ID:            convertUintToString(ba.ID),
		BankName:      ba.BankName,
		AccountName:   ba.AccountName,
		AccountNumber: ba.AccountNumber,
		SwiftCode:     ba.SwiftCode,
		RoutingNumber: ba.RoutingNumber,
		IsDefault:     ba.IsDefault,
	}
}

// ToCompanyResponse converts Company to CompanyResponse
func (c *Company) ToCompanyResponse() CompanyResponse {
	bankAccounts := make([]BankAccountResponse, len(c.BankAccounts))
	for i, ba := range c.BankAccounts {
		bankAccounts[i] = ba.ToBankAccountResponse()
	}

	return CompanyResponse{
		Name:         c.Name,
		Address:      c.Address,
		City:         c.City,
		State:        c.State,
		ZipCode:      c.ZipCode,
		Country:      c.Country,
		Email:        c.Email,
		Phone:        c.Phone,
		Website:      c.Website,
		Logo:         c.Logo,
		BankAccounts: bankAccounts,
	}
}

// Helper function to convert uint to string
func convertUintToString(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
