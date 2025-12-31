package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/notblessy/bikinota-core/model"
	"github.com/notblessy/bikinota-core/repository"
	"github.com/notblessy/bikinota-core/utils"
	"github.com/sirupsen/logrus"
)

type companyHandler struct {
	companyRepo       repository.CompanyRepository
	validate          *validator.Validate
	cloudinaryService *utils.CloudinaryService
}

func NewCompanyHandler(companyRepo repository.CompanyRepository, cloudinaryService *utils.CloudinaryService) *companyHandler {
	return &companyHandler{
		companyRepo:       companyRepo,
		validate:          validator.New(),
		cloudinaryService: cloudinaryService,
	}
}

// GetCompany retrieves the company information for the authenticated user
func (h *companyHandler) GetCompany(c echo.Context) error {
	logger := logrus.WithField("endpoint", "get_company")

	// Get user from JWT middleware
	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	company, err := h.companyRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil {
		logger.Errorf("Error finding company: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to retrieve company",
		})
	}

	// If company doesn't exist, return default/empty company
	if company == nil {
		emptyResponse := model.CompanyResponse{}
		return c.JSON(http.StatusOK, response{
			Success: true,
			Data:    emptyResponse,
		})
	}

	companyResponse := company.ToCompanyResponse()
	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    companyResponse,
	})
}

// UpdateCompany updates the company information
func (h *companyHandler) UpdateCompany(c echo.Context) error {
	logger := logrus.WithField("endpoint", "update_company")

	// Get user from JWT middleware
	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	var req model.UpdateCompanyRequest
	if err := c.Bind(&req); err != nil {
		logger.Errorf("Error parsing request: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid request body",
		})
	}

	// Find or create company
	company, err := h.companyRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil {
		logger.Errorf("Error finding company: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to retrieve company",
		})
	}

	if company == nil {
		// Create new company
		company = &model.Company{
			UserID:       userClaims.ID,
			BankAccounts: []model.BankAccount{},
		}
	}

	// Update fields if provided
	if req.Name != nil {
		company.Name = *req.Name
	}
	if req.Address != nil {
		company.Address = *req.Address
	}
	if req.City != nil {
		company.City = *req.City
	}
	if req.State != nil {
		company.State = *req.State
	}
	if req.ZipCode != nil {
		company.ZipCode = *req.ZipCode
	}
	if req.Country != nil {
		company.Country = *req.Country
	}
	if req.Email != nil {
		company.Email = *req.Email
	}
	if req.Phone != nil {
		company.Phone = *req.Phone
	}
	if req.Website != nil {
		company.Website = *req.Website
	}
	if req.Logo != nil {
		company.Logo = *req.Logo
	}

	if company.ID == 0 {
		err = h.companyRepo.Create(c.Request().Context(), company)
	} else {
		err = h.companyRepo.Update(c.Request().Context(), company)
	}

	if err != nil {
		logger.Errorf("Error saving company: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to save company",
		})
	}

	// Reload with bank accounts
	company, err = h.companyRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil {
		logger.Errorf("Error reloading company: %v", err)
	}

	companyResponse := company.ToCompanyResponse()
	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    companyResponse,
	})
}

// UploadLogo uploads a company logo to Cloudinary
func (h *companyHandler) UploadLogo(c echo.Context) error {
	logger := logrus.WithField("endpoint", "upload_logo")

	// Get user from JWT middleware
	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	// Get the uploaded file
	file, err := c.FormFile("logo")
	if err != nil {
		logger.Errorf("Error getting file: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "logo file is required",
		})
	}

	// Validate file size (5MB limit)
	if file.Size > 5*1024*1024 {
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "file size must be less than 5MB",
		})
	}

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/gif" && contentType != "image/webp" {
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "file must be an image (jpeg, png, gif, or webp)",
		})
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		logger.Errorf("Error opening file: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to read file",
		})
	}
	defer src.Close()

	// Find or create company
	company, err := h.companyRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil {
		logger.Errorf("Error finding company: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to retrieve company",
		})
	}

	if company == nil {
		// Create new company
		company = &model.Company{
			UserID:       userClaims.ID,
			BankAccounts: []model.BankAccount{},
		}
	}

	// Upload to Cloudinary
	if h.cloudinaryService == nil {
		return c.JSON(http.StatusServiceUnavailable, response{
			Success: false,
			Message: "image upload service is not configured",
		})
	}

	publicID := fmt.Sprintf("company-logo-%d", userClaims.ID)
	logoURL, err := h.cloudinaryService.UploadImage(c.Request().Context(), src, publicID)
	if err != nil {
		logger.Errorf("Error uploading to Cloudinary: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to upload logo",
		})
	}

	// Update company with logo URL
	company.Logo = logoURL
	if company.ID == 0 {
		err = h.companyRepo.Create(c.Request().Context(), company)
	} else {
		err = h.companyRepo.Update(c.Request().Context(), company)
	}

	if err != nil {
		logger.Errorf("Error saving company: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to save company",
		})
	}

	// Reload with bank accounts
	company, err = h.companyRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil {
		logger.Errorf("Error reloading company: %v", err)
	}

	companyResponse := company.ToCompanyResponse()
	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    companyResponse,
	})
}

// RemoveLogo removes the company logo
func (h *companyHandler) RemoveLogo(c echo.Context) error {
	logger := logrus.WithField("endpoint", "remove_logo")

	// Get user from JWT middleware
	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	company, err := h.companyRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil {
		logger.Errorf("Error finding company: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to retrieve company",
		})
	}

	if company == nil {
		return c.JSON(http.StatusNotFound, response{
			Success: false,
			Message: "company not found",
		})
	}

	// Delete from Cloudinary if URL exists and service is available
	if company.Logo != "" && h.cloudinaryService != nil {
		publicID := fmt.Sprintf("bikinota/company-logos/company-logo-%d", userClaims.ID)
		if err := h.cloudinaryService.DeleteImage(c.Request().Context(), publicID); err != nil {
			logger.Warnf("Failed to delete image from Cloudinary: %v", err)
			// Continue with removing from database even if Cloudinary delete fails
		}
	}

	company.Logo = ""
	err = h.companyRepo.Update(c.Request().Context(), company)
	if err != nil {
		logger.Errorf("Error updating company: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to remove logo",
		})
	}

	companyResponse := company.ToCompanyResponse()
	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    companyResponse,
	})
}

// AddBankAccount adds a new bank account to the company
func (h *companyHandler) AddBankAccount(c echo.Context) error {
	logger := logrus.WithField("endpoint", "add_bank_account")

	// Get user from JWT middleware
	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	var req model.CreateBankAccountRequest
	if err := c.Bind(&req); err != nil {
		logger.Errorf("Error parsing request: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid request body",
		})
	}

	if err := h.validate.Struct(req); err != nil {
		logger.Errorf("Validation error: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: err.Error(),
		})
	}

	// Find company
	company, err := h.companyRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil {
		logger.Errorf("Error finding company: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to retrieve company",
		})
	}

	if company == nil {
		return c.JSON(http.StatusNotFound, response{
			Success: false,
			Message: "company not found. Please create company first",
		})
	}

	bankAccount := &model.BankAccount{
		CompanyID:     company.ID,
		BankName:      req.BankName,
		AccountName:   req.AccountName,
		AccountNumber: req.AccountNumber,
		SwiftCode:     req.SwiftCode,
		RoutingNumber: req.RoutingNumber,
	}

	err = h.companyRepo.AddBankAccount(c.Request().Context(), bankAccount)
	if err != nil {
		logger.Errorf("Error adding bank account: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to add bank account",
		})
	}

	bankAccountResponse := bankAccount.ToBankAccountResponse()
	return c.JSON(http.StatusCreated, response{
		Success: true,
		Data:    bankAccountResponse,
	})
}

// UpdateBankAccount updates an existing bank account
func (h *companyHandler) UpdateBankAccount(c echo.Context) error {
	logger := logrus.WithField("endpoint", "update_bank_account")

	// Get user from JWT middleware
	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	bankAccountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid bank account id",
		})
	}

	var req model.UpdateBankAccountRequest
	if err := c.Bind(&req); err != nil {
		logger.Errorf("Error parsing request: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid request body",
		})
	}

	// Find company
	company, err := h.companyRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil || company == nil {
		return c.JSON(http.StatusNotFound, response{
			Success: false,
			Message: "company not found",
		})
	}

	// Find bank account
	bankAccount, err := h.companyRepo.FindBankAccountByID(c.Request().Context(), uint(bankAccountID), company.ID)
	if err != nil {
		return c.JSON(http.StatusNotFound, response{
			Success: false,
			Message: "bank account not found",
		})
	}

	// Update fields if provided
	if req.BankName != nil {
		bankAccount.BankName = *req.BankName
	}
	if req.AccountName != nil {
		bankAccount.AccountName = *req.AccountName
	}
	if req.AccountNumber != nil {
		bankAccount.AccountNumber = *req.AccountNumber
	}
	if req.SwiftCode != nil {
		bankAccount.SwiftCode = req.SwiftCode
	}
	if req.RoutingNumber != nil {
		bankAccount.RoutingNumber = req.RoutingNumber
	}

	err = h.companyRepo.UpdateBankAccount(c.Request().Context(), bankAccount)
	if err != nil {
		logger.Errorf("Error updating bank account: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to update bank account",
		})
	}

	bankAccountResponse := bankAccount.ToBankAccountResponse()
	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    bankAccountResponse,
	})
}

// DeleteBankAccount deletes a bank account
func (h *companyHandler) DeleteBankAccount(c echo.Context) error {
	logger := logrus.WithField("endpoint", "delete_bank_account")

	// Get user from JWT middleware
	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	bankAccountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid bank account id",
		})
	}

	// Find company
	company, err := h.companyRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil || company == nil {
		return c.JSON(http.StatusNotFound, response{
			Success: false,
			Message: "company not found",
		})
	}

	err = h.companyRepo.DeleteBankAccount(c.Request().Context(), uint(bankAccountID), company.ID)
	if err != nil {
		logger.Errorf("Error deleting bank account: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to delete bank account",
		})
	}

	return c.JSON(http.StatusOK, response{
		Success: true,
		Message: "bank account deleted successfully",
	})
}

// SetDefaultBankAccount sets a bank account as the default
func (h *companyHandler) SetDefaultBankAccount(c echo.Context) error {
	logger := logrus.WithField("endpoint", "set_default_bank_account")

	// Get user from JWT middleware
	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	bankAccountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid bank account id",
		})
	}

	// Find company
	company, err := h.companyRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil || company == nil {
		return c.JSON(http.StatusNotFound, response{
			Success: false,
			Message: "company not found",
		})
	}

	err = h.companyRepo.SetDefaultBankAccount(c.Request().Context(), uint(bankAccountID), company.ID)
	if err != nil {
		logger.Errorf("Error setting default bank account: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to set default bank account",
		})
	}

	// Reload company with updated bank accounts
	company, err = h.companyRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil {
		logger.Errorf("Error reloading company: %v", err)
	}

	companyResponse := company.ToCompanyResponse()
	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    companyResponse,
	})
}
