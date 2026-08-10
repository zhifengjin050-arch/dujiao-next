package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// CardSecretBatchRepository ??????????
type CardSecretBatchRepository interface {
	Create(batch *models.CardSecretBatch) error
	GetByID(id uint) (*models.CardSecretBatch, error)
	ListByProduct(productID, skuID uint, page, pageSize int) ([]models.CardSecretBatch, int64, error)
	DeleteByProduct(productID uint) error
	WithTx(tx *gorm.DB) *GormCardSecretBatchRepository
}

// GormCardSecretBatchRepository GORM ??
type GormCardSecretBatchRepository struct {
	db *gorm.DB
}

// NewCardSecretBatchRepository ????????
func NewCardSecretBatchRepository(db *gorm.DB) *GormCardSecretBatchRepository {
	return &GormCardSecretBatchRepository{db: db}
}

// WithTx ????
func (r *GormCardSecretBatchRepository) WithTx(tx *gorm.DB) *GormCardSecretBatchRepository {
	if tx == nil {
		return r
	}
	return &GormCardSecretBatchRepository{db: tx}
}

// Create ????
func (r *GormCardSecretBatchRepository) Create(batch *models.CardSecretBatch) error {
	if batch == nil {
		return errors.New("batch is nil")
	}
	return r.db.Create(batch).Error
}

// GetByID ????
func (r *GormCardSecretBatchRepository) GetByID(id uint) (*models.CardSecretBatch, error) {
	if id == 0 {
		return nil, errors.New("invalid batch id")
	}
	var batch models.CardSecretBatch
	if err := r.db.First(&batch, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &batch, nil
}

// ListByProduct ?????????
func (r *GormCardSecretBatchRepository) ListByProduct(productID, skuID uint, page, pageSize int) ([]models.CardSecretBatch, int64, error) {
	if productID == 0 {
		return nil, 0, errors.New("invalid product id")
	}
	query := r.db.Model(&models.CardSecretBatch{}).Where("product_id = ?", productID)
	if skuID > 0 {
		query = query.Where("sku_id = ?", skuID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Limit(pageSize).Offset(offset)
	}

	var items []models.CardSecretBatch
	if err := query.Order("id desc").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// DeleteByProduct ??????????????
func (r *GormCardSecretBatchRepository) DeleteByProduct(productID uint) error {
	if productID == 0 {
		return errors.New("invalid product id")
	}
	return r.db.Where("product_id = ?", productID).Delete(&models.CardSecretBatch{}).Error
}
