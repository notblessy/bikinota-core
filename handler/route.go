package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/notblessy/bikinota-core/repository"
	"github.com/notblessy/bikinota-core/utils"
)

func SetupRoutes(e *echo.Echo, userRepo repository.UserRepository, companyRepo repository.CompanyRepository, planRepo repository.PlanRepository, invoiceRepo repository.InvoiceRepository, cloudinaryService interface{}) {
	// CORS middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
		},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.PATCH, echo.OPTIONS},
	}))

	// Logger middleware
	e.Use(middleware.Logger())

	// Recover middleware
	e.Use(middleware.Recover())

	// Health check
	e.GET("/ping", func(c echo.Context) error {
		return c.JSON(200, response{
			Success: true,
			Data:    "pong",
		})
	})

	// Auth routes
	authHandler := NewAuthHandler(userRepo)
	auth := e.Group("/api/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

	// Protected routes (require JWT)
	protected := e.Group("/api")
	protected.Use(NewJWTMiddleware().ValidateJWT)

	// Company routes
	var cloudinarySvc *utils.CloudinaryService
	if cloudinaryService != nil {
		cloudinarySvc = cloudinaryService.(*utils.CloudinaryService)
	}
	companyHandler := NewCompanyHandler(companyRepo, cloudinarySvc)
	company := protected.Group("/company")
	company.GET("", companyHandler.GetCompany)
	company.PUT("", companyHandler.UpdateCompany)
	company.POST("/logo", companyHandler.UploadLogo)
	company.DELETE("/logo", companyHandler.RemoveLogo)

	// Bank account routes
	bankAccounts := company.Group("/bank-accounts")
	bankAccounts.POST("", companyHandler.AddBankAccount)
	bankAccounts.PUT("/:id", companyHandler.UpdateBankAccount)
	bankAccounts.DELETE("/:id", companyHandler.DeleteBankAccount)
	bankAccounts.PUT("/:id/default", companyHandler.SetDefaultBankAccount)

	// Plan routes
	planHandler := NewPlanHandler(planRepo)
	plan := protected.Group("/plan")
	plan.GET("", planHandler.GetPlan)
	plan.PUT("", planHandler.UpdatePlan)

	// Invoice routes
	invoiceHandler := NewInvoiceHandler(invoiceRepo)
	invoice := protected.Group("/invoice")
	invoice.GET("", invoiceHandler.GetInvoices)
	invoice.GET("/:id", invoiceHandler.GetInvoice)
	invoice.POST("", invoiceHandler.CreateInvoice)
	invoice.PUT("/:id", invoiceHandler.UpdateInvoice)
	invoice.DELETE("/:id", invoiceHandler.DeleteInvoice)
}

