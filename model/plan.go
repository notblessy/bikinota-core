package model

import (
	"time"

	"gorm.io/gorm"
)

type PlanType string

const (
	PlanFree      PlanType = "free"
	PlanUnlimited PlanType = "unlimited"
)

type Plan struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"not null;uniqueIndex"`
	PlanType  PlanType       `json:"plan_type" gorm:"type:varchar(20);not null;default:'free'"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type UpdatePlanRequest struct {
	PlanType PlanType `json:"plan_type" validate:"required,oneof=free unlimited"`
}

type PlanResponse struct {
	CurrentPlan PlanType `json:"current_plan"`
}

// ToPlanResponse converts Plan to PlanResponse
func (p *Plan) ToPlanResponse() PlanResponse {
	return PlanResponse{
		CurrentPlan: p.PlanType,
	}
}
