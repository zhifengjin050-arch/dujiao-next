package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// MemberLevelPriceRepository ????????????
type MemberLevelPriceRepository interface {
	GetByID(id uint) (*models.MemberLevelPrice, error)
	GetByLevelAndProductAndSKU(levelID, productID, skuID uint) (*models.MemberLevelPrice, error)
	ListByProduct(productID uint) ([]models.MemberLevelPrice, error)
	ListByLevelAndProducts(levelID uint, productIDs []uint) ([]models.MemberLevelPrice, error)
	BatchUpsert(prices []models.MemberLevelPrice) error
	Delete(id uint) error
	DeleteByProduct(productID uint) error
	WithTx(tx *gorm.DB) *GormMemberLevelPriceRepository
}

// GormMemberLevelPriceRepository GORM ??
type GormMemberLevelPriceRepository struct {
	db *gorm.DB
}

// NewMemberLevelPriceRepository ??????????
func NewMemberLevelPriceRepository(db *gorm.DB) *GormMemberLevelPriceRepository {
	return &GormMemberLevelPriceRepository{db: db}
}

func (r *GormMemberLevelPriceRepository) WithTx(tx *gorm.DB) *GormMemberLevelPriceRepository {
	if tx == nil {
		return r
	}
	return &GormMemberLevelPriceRepository{db: tx}
}

func (r *GormMemberLevelPriceRepository) GetByID(id uint) (*models.MemberLevelPrice, error) {
	var price models.MemberLevelPrice
	if err := r.db.First(&price, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &price, nil
}

func (r *GormMemberLevelPriceRepository) GetByLevelAndProductAndSKU(levelID, productID, skuID uint) (*models.MemberLevelPrice, error) {
	var price models.MemberLevelPrice
	if err := r.db.Where("member_level_id = ? AND product_id = ? AND sku_id = ?", levelID, productID, skuID).First(&price).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &price, nil
}

// ListByProduct ??????????
func (r *GormMemberLevelPriceRepository) ListByProduct(productID uint) ([]models.MemberLevelPrice, error) {
	var prices []models.MemberLevelPrice
	if err := r.db.Where("product_id = ?", productID).Order("member_level_id asc, sku_id asc").Find(&prices).Error; err != nil {
		return nil, err
	}
	return prices, nil
}

// ListByLevelAndProducts ???????????????
func (r *GormMemberLevelPriceRepository) ListByLevelAndProducts(levelID uint, productIDs []uint) ([]models.MemberLevelPrice, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}
	var prices []models.MemberLevelPrice
	if err := r.db.Where("member_level_id = ? AND product_id IN ?", levelID, productIDs).Find(&prices).Error; err != nil {
		return nil, err
	}
	return prices, nil
}

// BatchUpsert ????/?????
func (r *GormMemberLevelPriceRepository) BatchUpsert(prices []models.MemberLevelPrice) error {
	if len(prices) == 0 {
		return nil
	}
	for _, p := range prices {
		existing, err := r.GetByLevelAndProductAndSKU(p.MemberLevelID, p.ProductID, p.SKUID)
		if err != nil {
			return err
		}
		if existing != nil {
			existing.PriceAmount = p.PriceAmount
			if err := r.db.Save(existing).Error; err != nil {
				return err
			}
		} else {
			if err := r.db.Create(&p).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *GormMemberLevelPriceRepository) Delete(id uint) error {
	return r.db.Delete(&models.MemberLevelPrice{}, id).Error
}

func (r *GormMemberLevelPriceRepository) DeleteByProduct(productID uint) error {
	return r.db.Where("product_id = ?", productID).Delete(&models.MemberLevelPrice{}).Error
}
