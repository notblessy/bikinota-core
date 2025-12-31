package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/notblessy/bikinota-core/db"
	"github.com/notblessy/bikinota-core/handler"
	"github.com/notblessy/bikinota-core/model"
	"github.com/notblessy/bikinota-core/repository"
	"github.com/notblessy/bikinota-core/utils"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		logrus.Warn("cannot load .env file")
	}

	// Initialize database
	postgres := db.NewPostgres()

	// Auto-migrate models
	err = postgres.AutoMigrate(
		&model.User{},
		&model.Company{},
		&model.BankAccount{},
		&model.Plan{},
		&model.Invoice{},
		&model.InvoiceItem{},
		&model.InvoiceAdjustment{},
	)
	if err != nil {
		logrus.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(postgres)
	companyRepo := repository.NewCompanyRepository(postgres)
	planRepo := repository.NewPlanRepository(postgres)
	invoiceRepo := repository.NewInvoiceRepository(postgres)

	// Initialize Cloudinary service (optional - will work without it but uploads will fail)
	var cloudinaryService *utils.CloudinaryService
	cloudinaryService, err = utils.NewCloudinaryService()
	if err != nil {
		logrus.Warnf("Cloudinary not configured: %v. Logo uploads will not work.", err)
		cloudinaryService = nil
	}

	// Initialize Echo
	e := echo.New()

	// Setup routes
	handler.SetupRoutes(e, userRepo, companyRepo, planRepo, invoiceRepo, cloudinaryService)

	// Shared context with cancel
	_, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		logrus.Info("HTTP server starting on :8080")

		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("HTTP server error: %v", err)
		}
	}()

	// Signal handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutdown signal received")

	// Initiate graceful shutdown
	cancel()
	ctxTimeout, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(ctxTimeout); err != nil {
		logrus.Errorf("Server shutdown error: %v", err)
	}

	wg.Wait()
	logrus.Info("All services shut down gracefully")
}
