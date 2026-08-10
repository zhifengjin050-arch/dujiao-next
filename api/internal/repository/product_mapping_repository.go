package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// ProductMappingRepository ??????????
type ProductMappingRepository interface {
	GetByID(id uint) (*models.ProductMapping, error)
	GetByLocalProductID(productID uint) (*models.ProductMapping, error)
	GetByConnectionAndUpstreamID(connectionID, upstreamProductID uint) (*models.ProductMapping, error)
	WithTx(tx *gorm.DB) ProductMappingRepository
	Create(mapping *models.ProductMapping) error
	Update(mapping *models.ProductMapping) error
	Delete(id uint) error
	DeleteByLocalProduct(productID uint) error
	List(filter ProductMappingListFilter) ([]models.ProductMapping, int64, error)
	ListActiveByConnection(connectionID uint) ([]models.ProductMapping, error)
	ListAllActive() ([]models.ProductMapping, error)
	ListByLocalProductIDs(productIDs []uint) ([]models.ProductMapping, error)
	ListUpstreamIDsByConnection(connectionID uint) ([]uint, error)
}

// ProductMappingListFilter ??????
type ProductMappingListFilter struct {
	ConnectionID uint
	Pagination
}

// GormProductMappingRepository GORM ??
type GormProductMappingRepository struct {
	db *gorm.DB
}

// NewProductMappingRepository ????????
func NewProductMappingRepository(db *gorm.DB) *GormProductMappingRepository {
	return &GormProductMappingRepository{db: db}
}

func (r *GormProductMappingRepository) WithTx(tx *gorm.DB) ProductMappingRepository {
	return &GormProductMappingRepository{db: tx}
}

func (r *GormProductMappingRepository) GetByID(id uint) (*models.ProductMapping, error) {
	var m models.ProductMapping
	if err := r.db.Preload("Connection").Preload("Product").First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *GormProductMappingRepository) GetByLocalProductID(productID uint) (*models.ProductMapping, error) {
	var m models.ProductMapping
	if err := r.db.Where("local_product_id = ?", productID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *GormProductMappingRepository) GetByConnectionAndUpstreamID(connectionID, upstreamProductID uint) (*models.ProductMapping, error) {
	var m models.ProductMapping
	if err := r.db.Where("connection_id = ? AND upstream_product_id = ?", connectionID, upstreamProductID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *GormProductMappingRepository) Create(mapping *models.ProductMapping) error {
	return r.db.Create(mapping).Error
}

func (r *GormProductMappingRepository) Update(mapping *models.ProductMapping) error {
	return r.db.Save(mapping).Error
}

func (r *GormProductMappingRepository) Delete(id uint) error {
	return r.db.Delete(&models.ProductMapping{}, id).Error
}

// DeleteByLocalProduct ??????????????? SKU ??
func (r *GormProductMappingRepository) DeleteByLocalProduct(productID uint) error {
	// ?????? SKU ??
	if err := r.db.Where("product_mapping_id IN (?)",
		r.db.Model(&models.ProductMapping{}).Select("id").Where("local_product_id = ?", productID),
	).Delete(&models.SKUMapping{}).Error; err != nil {
		return err
	}
	return r.db.Where("local_product_id = ?", productID).Delete(&models.ProductMapping{}).Error
}

func (r *GormProductMappingRepository) List(filter ProductMappingListFilter) ([]models.ProductMapping, int64, error) {
	var mappings []models.ProductMapping
	var total int64

	q := r.db.Model(&models.ProductMapping{})
	if filter.ConnectionID > 0 {
		q = q.Where("connection_id = ?", filter.ConnectionID)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q = q.Preload("Connection").Preload("Product").Preload("Product.SKUs").Order("created_at DESC")
	if filter.Page > 0 && filter.PageSize > 0 {
		q = q.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	if err := q.Find(&mappings).Error; err != nil {
		return nil, 0, err
	}

	return mappings, total, nil
}

func (r *GormProductMappingRepository) ListActiveByConnection(connectionID uint) ([]models.ProductMapping, error) {
	var mappings []models.ProductMapping
	if err := r.db.Where("connection_id = ? AND is_active = ?", connectionID, true).Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *GormProductMappingRepository) ListAllActive() ([]models.ProductMapping, error) {
	var mappings []models.ProductMapping
	if err := r.db.Where("is_active = ?", true).Preload("Connection").Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *GormProductMappingRepository) ListByLocalProductIDs(productIDs []uint) ([]models.ProductMapping, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}
	var mappings []models.ProductMapping
	if err := r.db.Where("local_product_id IN ?", productIDs).Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *GormProductMappingRepository) ListUpstreamIDsByConnection(connectionID uint) ([]uint, error) {
	var ids []uint
	if err := r.db.Model(&models.ProductMapping{}).
		Where("connection_id = ?", connectionID).
		Pluck("upstream_product_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
