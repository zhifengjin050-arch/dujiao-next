package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// SKUMappingRepository SKU ????????
type SKUMappingRepository interface {
	GetByID(id uint) (*models.SKUMapping, error)
	GetByLocalSKUID(skuID uint) (*models.SKUMapping, error)
	GetByMappingAndUpstreamSKUID(productMappingID, upstreamSKUID uint) (*models.SKUMapping, error)
	ListByProductMapping(productMappingID uint) ([]models.SKUMapping, error)
	ListByProductMappingIDs(productMappingIDs []uint) ([]models.SKUMapping, error)
	WithTx(tx *gorm.DB) SKUMappingRepository
	Create(mapping *models.SKUMapping) error
	Update(mapping *models.SKUMapping) error
	Delete(id uint) error
	DeleteByProductMapping(productMappingID uint) error
	BatchUpsert(mappings []models.SKUMapping) error
}

// GormSKUMappingRepository GORM ??
type GormSKUMappingRepository struct {
	db *gorm.DB
}

// NewSKUMappingRepository ?? SKU ????
func NewSKUMappingRepository(db *gorm.DB) *GormSKUMappingRepository {
	return &GormSKUMappingRepository{db: db}
}

func (r *GormSKUMappingRepository) WithTx(tx *gorm.DB) SKUMappingRepository {
	return &GormSKUMappingRepository{db: tx}
}

func (r *GormSKUMappingRepository) GetByID(id uint) (*models.SKUMapping, error) {
	var m models.SKUMapping
	if err := r.db.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *GormSKUMappingRepository) GetByLocalSKUID(skuID uint) (*models.SKUMapping, error) {
	var m models.SKUMapping
	if err := r.db.Where("local_sku_id = ?", skuID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *GormSKUMappingRepository) GetByMappingAndUpstreamSKUID(productMappingID, upstreamSKUID uint) (*models.SKUMapping, error) {
	var m models.SKUMapping
	if err := r.db.Where("product_mapping_id = ? AND upstream_sku_id = ?", productMappingID, upstreamSKUID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *GormSKUMappingRepository) ListByProductMapping(productMappingID uint) ([]models.SKUMapping, error) {
	var mappings []models.SKUMapping
	if err := r.db.Where("product_mapping_id = ?", productMappingID).Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *GormSKUMappingRepository) ListByProductMappingIDs(productMappingIDs []uint) ([]models.SKUMapping, error) {
	if len(productMappingIDs) == 0 {
		return nil, nil
	}
	var mappings []models.SKUMapping
	if err := r.db.Where("product_mapping_id IN ?", productMappingIDs).Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *GormSKUMappingRepository) Create(mapping *models.SKUMapping) error {
	return r.db.Create(mapping).Error
}

func (r *GormSKUMappingRepository) Update(mapping *models.SKUMapping) error {
	return r.db.Save(mapping).Error
}

func (r *GormSKUMappingRepository) Delete(id uint) error {
	return r.db.Delete(&models.SKUMapping{}, id).Error
}

func (r *GormSKUMappingRepository) DeleteByProductMapping(productMappingID uint) error {
	return r.db.Where("product_mapping_id = ?", productMappingID).Delete(&models.SKUMapping{}).Error
}

func (r *GormSKUMappingRepository) BatchUpsert(mappings []models.SKUMapping) error {
	if len(mappings) == 0 {
		return nil
	}
	return r.db.Save(&mappings).Error
}
