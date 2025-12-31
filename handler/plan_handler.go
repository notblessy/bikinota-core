package handler

import (
	"net/http"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/notblessy/bikinota-core/model"
	"github.com/notblessy/bikinota-core/repository"
	"github.com/sirupsen/logrus"
)

type planHandler struct {
	planRepo repository.PlanRepository
	validate *validator.Validate
}

func NewPlanHandler(planRepo repository.PlanRepository) *planHandler {
	return &planHandler{
		planRepo: planRepo,
		validate: validator.New(),
	}
}

// GetPlan retrieves the plan information for the authenticated user
func (h *planHandler) GetPlan(c echo.Context) error {
	logger := logrus.WithField("endpoint", "get_plan")

	// Get user from JWT middleware
	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	plan, err := h.planRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil {
		logger.Errorf("Error finding plan: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to retrieve plan",
		})
	}

	// If plan doesn't exist, return default free plan
	if plan == nil {
		defaultResponse := model.PlanResponse{
			CurrentPlan: model.PlanFree,
		}
		return c.JSON(http.StatusOK, response{
			Success: true,
			Data:    defaultResponse,
		})
	}

	planResponse := plan.ToPlanResponse()
	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    planResponse,
	})
}

// UpdatePlan updates the user's plan
func (h *planHandler) UpdatePlan(c echo.Context) error {
	logger := logrus.WithField("endpoint", "update_plan")

	// Get user from JWT middleware
	userClaims, err := authSession(c)
	if err != nil {
		logger.Errorf("Error getting session: %v", err)
		return c.JSON(http.StatusUnauthorized, response{
			Success: false,
			Message: "unauthorized",
		})
	}

	var req model.UpdatePlanRequest
	if err := c.Bind(&req); err != nil {
		logger.Errorf("Error parsing request: %v", err)
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid request body",
		})
	}

	// Validate plan type
	if req.PlanType != model.PlanFree && req.PlanType != model.PlanUnlimited {
		return c.JSON(http.StatusBadRequest, response{
			Success: false,
			Message: "invalid plan type. Must be 'free' or 'unlimited'",
		})
	}

	// Find or create plan
	plan, err := h.planRepo.FindByUserID(c.Request().Context(), userClaims.ID)
	if err != nil {
		logger.Errorf("Error finding plan: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to retrieve plan",
		})
	}

	if plan == nil {
		// Create new plan with default free plan
		plan = &model.Plan{
			UserID:   userClaims.ID,
			PlanType: model.PlanFree,
		}
	}

	// Update plan type
	plan.PlanType = req.PlanType

	if plan.ID == 0 {
		err = h.planRepo.Create(c.Request().Context(), plan)
	} else {
		err = h.planRepo.Update(c.Request().Context(), plan)
	}

	if err != nil {
		logger.Errorf("Error saving plan: %v", err)
		return c.JSON(http.StatusInternalServerError, response{
			Success: false,
			Message: "failed to save plan",
		})
	}

	planResponse := plan.ToPlanResponse()
	return c.JSON(http.StatusOK, response{
		Success: true,
		Data:    planResponse,
	})
}

