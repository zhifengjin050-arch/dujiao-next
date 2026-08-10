package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// FulfillmentRepository ????????
type FulfillmentRepository interface {
	Create(fulfillment *models.Fulfillment) error
	GetByOrderID(orderID uint) (*models.Fulfillment, error)
}

// GormFulfillmentRepository GORM ??
type GormFulfillmentRepository struct {
	db *gorm.DB
}

// NewFulfillmentRepository ??????
func NewFulfillmentRepository(db *gorm.DB) *GormFulfillmentRepository {
	return &GormFulfillmentRepository{db: db}
}

// Create ??????
func (r *GormFulfillmentRepository) Create(fulfillment *models.Fulfillment) error {
	return r.db.Create(fulfillment).Error
}

// GetByOrderID ???? ID ??????
func (r *GormFulfillmentRepository) GetByOrderID(orderID uint) (*models.Fulfillment, error) {
	var fulfillment models.Fulfillment
	if err := r.db.Where("order_id = ?", orderID).First(&fulfillment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &fulfillment, nil
}
