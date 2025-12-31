package repository

import (
	"context"
	"errors"

	"github.com/notblessy/bikinota-core/model"
	"gorm.io/gorm"
)

type CompanyRepository interface {
	FindByUserID(ctx context.Context, userID uint) (*model.Company, error)
	Create(ctx context.Context, company *model.Company) error
	Update(ctx context.Context, company *model.Company) error
	AddBankAccount(ctx context.Context, bankAccount *model.BankAccount) error
	FindBankAccountByID(ctx context.Context, bankAccountID uint, companyID uint) (*model.BankAccount, error)
	UpdateBankAccount(ctx context.Context, bankAccount *model.BankAccount) error
	DeleteBankAccount(ctx context.Context, bankAccountID uint, companyID uint) error
	GetBankAccounts(ctx context.Context, companyID uint) ([]model.BankAccount, error)
	SetDefaultBankAccount(ctx context.Context, bankAccountID uint, companyID uint) error
}

type companyRepository struct {
	db *gorm.DB
}

func NewCompanyRepository(db *gorm.DB) CompanyRepository {
	return &companyRepository{db: db}
}

func (r *companyRepository) FindByUserID(ctx context.Context, userID uint) (*model.Company, error) {
	var company model.Company
	err := r.db.WithContext(ctx).
		Preload("BankAccounts").
		Where("user_id = ?", userID).
		First(&company).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if not found (not an error, just no company yet)
		}
		return nil, err
	}
	return &company, nil
}

func (r *companyRepository) Create(ctx context.Context, company *model.Company) error {
	return r.db.WithContext(ctx).Create(company).Error
}

func (r *companyRepository) Update(ctx context.Context, company *model.Company) error {
	return r.db.WithContext(ctx).Save(company).Error
}

func (r *companyRepository) AddBankAccount(ctx context.Context, bankAccount *model.BankAccount) error {
	// Check if this is the first bank account, make it default
	var count int64
	r.db.WithContext(ctx).Model(&model.BankAccount{}).
		Where("company_id = ?", bankAccount.CompanyID).
		Count(&count)
	
	if count == 0 {
		bankAccount.IsDefault = true
	}

	return r.db.WithContext(ctx).Create(bankAccount).Error
}

func (r *companyRepository) FindBankAccountByID(ctx context.Context, bankAccountID uint, companyID uint) (*model.BankAccount, error) {
	var bankAccount model.BankAccount
	err := r.db.WithContext(ctx).
		Where("id = ? AND company_id = ?", bankAccountID, companyID).
		First(&bankAccount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("bank account not found")
		}
		return nil, err
	}
	return &bankAccount, nil
}

func (r *companyRepository) UpdateBankAccount(ctx context.Context, bankAccount *model.BankAccount) error {
	return r.db.WithContext(ctx).Save(bankAccount).Error
}

func (r *companyRepository) DeleteBankAccount(ctx context.Context, bankAccountID uint, companyID uint) error {
	// Check if this is the default account
	bankAccount, err := r.FindBankAccountByID(ctx, bankAccountID, companyID)
	if err != nil {
		return err
	}

	// Delete the account
	err = r.db.WithContext(ctx).Delete(&model.BankAccount{}, bankAccountID).Error
	if err != nil {
		return err
	}

	// If it was the default, set the first remaining account as default
	if bankAccount.IsDefault {
		var remainingAccounts []model.BankAccount
		r.db.WithContext(ctx).
			Where("company_id = ?", companyID).
			Order("created_at ASC").
			Limit(1).
			Find(&remainingAccounts)
		
		if len(remainingAccounts) > 0 {
			remainingAccounts[0].IsDefault = true
			r.db.WithContext(ctx).Save(&remainingAccounts[0])
		}
	}

	return nil
}

func (r *companyRepository) GetBankAccounts(ctx context.Context, companyID uint) ([]model.BankAccount, error) {
	var bankAccounts []model.BankAccount
	err := r.db.WithContext(ctx).
		Where("company_id = ?", companyID).
		Find(&bankAccounts).Error
	return bankAccounts, err
}

func (r *companyRepository) SetDefaultBankAccount(ctx context.Context, bankAccountID uint, companyID uint) error {
	// First, unset all default accounts for this company
	err := r.db.WithContext(ctx).
		Model(&model.BankAccount{}).
		Where("company_id = ?", companyID).
		Update("is_default", false).Error
	if err != nil {
		return err
	}

	// Then set the specified account as default
	err = r.db.WithContext(ctx).
		Model(&model.BankAccount{}).
		Where("id = ? AND company_id = ?", bankAccountID, companyID).
		Update("is_default", true).Error
	return err
}

