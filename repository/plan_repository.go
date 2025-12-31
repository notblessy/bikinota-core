package repository

import (
	"context"
	"errors"

	"github.com/notblessy/bikinota-core/model"
	"gorm.io/gorm"
)

type PlanRepository interface {
	FindByUserID(ctx context.Context, userID uint) (*model.Plan, error)
	Create(ctx context.Context, plan *model.Plan) error
	Update(ctx context.Context, plan *model.Plan) error
}

type planRepository struct {
	db *gorm.DB
}

func NewPlanRepository(db *gorm.DB) PlanRepository {
	return &planRepository{db: db}
}

func (r *planRepository) FindByUserID(ctx context.Context, userID uint) (*model.Plan, error) {
	var plan model.Plan
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if not found (not an error, just no plan yet)
		}
		return nil, err
	}
	return &plan, nil
}

func (r *planRepository) Create(ctx context.Context, plan *model.Plan) error {
	return r.db.WithContext(ctx).Create(plan).Error
}

func (r *planRepository) Update(ctx context.Context, plan *model.Plan) error {
	return r.db.WithContext(ctx).Save(plan).Error
}
