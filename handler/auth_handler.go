package handler

import (
	"net/http"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/notblessy/bikinota-core/model"
	"github.com/notblessy/bikinota-core/repository"
	"github.com/sirupsen/logrus"
)

type response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type authHandler struct {
	userRepo repository.UserRepository
	validate *validator.Validate
}

func NewAuthHandler(userRepo repository.UserRepository) *authHandler {
	return &authHandler{
		userRepo: userRepo,
		validate: validator.New(),
	}
}

func (h *authHandler) Register(c echo.Context) error {
	logger := logrus.WithField("endpoint", "register")

	var req model.RegisterRequest
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

	// Check if user already exists
	existingUser, err := h.userRepo.FindByEmail(c.Request().Context(), req.Email)
	if err == nil && existingUser != nil {
		logger.Warnf("User with email %s already exists", req.Email)
		return c.JSON(http.StatusConflict, response{
			Success: false,
			Message: "user with this email already exists",
		})
	}

	// Create new user
	user := &model.User{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	}

	if err := h.userRepo.Create(c.Request().Context(), user); err != nil {
		logger.Errorf("Error creating user: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to create user",
		})
	}

	// Generate JWT token
	token, err := signJWTToken(user.ID, user.Email, user.Name)
	if err != nil {
		logger.Errorf("Error generating token: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to generate token",
		})
	}

	// Remove password from response
	user.Password = ""

	return c.JSON(http.StatusCreated, response{
		Success: true,
		Data: model.AuthResponse{
			Token: token,
			Type:  "Bearer",
			User:  *user,
		},
	})
}

func (h *authHandler) Login(c echo.Context) error {
	logger := logrus.WithField("endpoint", "login")

	var req model.LoginRequest
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

	// Find user by email
	user, err := h.userRepo.FindByEmail(c.Request().Context(), req.Email)
	if err != nil {
		logger.Warnf("User not found: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "invalid email or password",
		})
	}

	// Verify password
	if !repository.VerifyPassword(user.Password, req.Password) {
		logger.Warnf("Invalid password for user: %s", req.Email)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "invalid email or password",
		})
	}

	// Generate JWT token
	token, err := signJWTToken(user.ID, user.Email, user.Name)
	if err != nil {
		logger.Errorf("Error generating token: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to generate token",
		})
	}

	// Remove password from response
	user.Password = ""

	return c.JSON(http.StatusOK, response{
		Success: true,
		Data: model.AuthResponse{
			Token: token,
			Type:  "Bearer",
			User:  *user,
		},
	})
}
