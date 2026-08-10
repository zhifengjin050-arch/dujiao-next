package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// CartRepository ?????????
type CartRepository interface {
	ListByUser(userID uint) ([]models.CartItem, error)
	Upsert(item *models.CartItem) error
	DeleteByProduct(productID uint) error
	DeleteByUserAndProduct(userID, productID uint) error
	DeleteByUserProductSKU(userID, productID, skuID uint) error
	ClearByUser(userID uint) error
	WithTx(tx *gorm.DB) *GormCartRepository
}

// GormCartRepository GORM ??
type GormCartRepository struct {
	db *gorm.DB
}

// NewCartRepository ???????
func NewCartRepository(db *gorm.DB) *GormCartRepository {
	return &GormCartRepository{db: db}
}

// WithTx ????
func (r *GormCartRepository) WithTx(tx *gorm.DB) *GormCartRepository {
	if tx == nil {
		return r
	}
	return &GormCartRepository{db: tx}
}

// ListByUser ????????
func (r *GormCartRepository) ListByUser(userID uint) ([]models.CartItem, error) {
	var items []models.CartItem
	if err := r.db.Preload("Product").Preload("SKU").Where("user_id = ?", userID).Order("updated_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// Upsert ?????????
func (r *GormCartRepository) Upsert(item *models.CartItem) error {
	if item == nil {
		return nil
	}
	var existing models.CartItem
	err := r.db.Where("user_id = ? AND product_id = ? AND sku_id = ?", item.UserID, item.ProductID, item.SKUID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(item).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]interface{}{
		"sku_id":           item.SKUID,
		"quantity":         item.Quantity,
		"fulfillment_type": item.FulfillmentType,
		"updated_at":       item.UpdatedAt,
	}
	return r.db.Model(&existing).Updates(updates).Error
}

// DeleteByProduct ?????????????
func (r *GormCartRepository) DeleteByProduct(productID uint) error {
	return r.db.Where("product_id = ?", productID).Delete(&models.CartItem{}).Error
}

// DeleteByUserAndProduct ??????
func (r *GormCartRepository) DeleteByUserAndProduct(userID, productID uint) error {
	return r.db.Where("user_id = ? AND product_id = ?", userID, productID).Delete(&models.CartItem{}).Error
}

// DeleteByUserProductSKU ???+??+SKU??????
func (r *GormCartRepository) DeleteByUserProductSKU(userID, productID, skuID uint) error {
	if skuID == 0 {
		return r.DeleteByUserAndProduct(userID, productID)
	}
	return r.db.Where("user_id = ? AND product_id = ? AND sku_id = ?", userID, productID, skuID).Delete(&models.CartItem{}).Error
}

// ClearByUser ?????
func (r *GormCartRepository) ClearByUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.CartItem{}).Error
}
