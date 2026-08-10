package repository

import (
	"errors"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// ProductSKURepository ?? SKU ??????
type ProductSKURepository interface {
	ListByProduct(productID uint, onlyActive bool) ([]models.ProductSKU, error)
	GetByID(id uint) (*models.ProductSKU, error)
	GetByProductAndCode(productID uint, skuCode string) (*models.ProductSKU, error)
	ListByIDs(ids []uint) ([]models.ProductSKU, error)
	Create(item *models.ProductSKU) error
	CreateBatch(items []models.ProductSKU) error
	Update(item *models.ProductSKU) error
	Delete(id uint) error
	DeleteByProduct(productID uint) error
	PurgeSoftDeletedByProductAndCode(productID uint, skuCode string) error
	ReserveManualStock(skuID uint, quantity int) (int64, error)
	ReleaseManualStock(skuID uint, quantity int) (int64, error)
	ConsumeManualStock(skuID uint, quantity int) (int64, error)
	WithTx(tx *gorm.DB) ProductSKURepository
}

// GormProductSKURepository GORM ??
type GormProductSKURepository struct {
	db *gorm.DB
}

// NewProductSKURepository ?? SKU ??
func NewProductSKURepository(db *gorm.DB) *GormProductSKURepository {
	return &GormProductSKURepository{db: db}
}

// WithTx ????
func (r *GormProductSKURepository) WithTx(tx *gorm.DB) ProductSKURepository {
	if tx == nil {
		return r
	}
	return &GormProductSKURepository{db: tx}
}

// ListByProduct ?????? SKU ??
func (r *GormProductSKURepository) ListByProduct(productID uint, onlyActive bool) ([]models.ProductSKU, error) {
	if productID == 0 {
		return nil, errors.New("invalid product id")
	}
	query := r.db.Model(&models.ProductSKU{}).Where("product_id = ?", productID)
	if onlyActive {
		query = query.Where("is_active = ?", true)
	}
	var items []models.ProductSKU
	if err := query.Order("sort_order DESC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetByID ?? ID ?? SKU
func (r *GormProductSKURepository) GetByID(id uint) (*models.ProductSKU, error) {
	if id == 0 {
		return nil, errors.New("invalid sku id")
	}
	var item models.ProductSKU
	if err := r.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// GetByProductAndCode ???????? SKU
func (r *GormProductSKURepository) GetByProductAndCode(productID uint, skuCode string) (*models.ProductSKU, error) {
	if productID == 0 {
		return nil, errors.New("invalid product id")
	}
	code := strings.TrimSpace(skuCode)
	if code == "" {
		return nil, errors.New("invalid sku code")
	}

	var item models.ProductSKU
	if err := r.db.Where("product_id = ? AND sku_code = ?", productID, code).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// ListByIDs ???? SKU
func (r *GormProductSKURepository) ListByIDs(ids []uint) ([]models.ProductSKU, error) {
	if len(ids) == 0 {
		return []models.ProductSKU{}, nil
	}
	var items []models.ProductSKU
	if err := r.db.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// Create ?? SKU
func (r *GormProductSKURepository) Create(item *models.ProductSKU) error {
	if item == nil {
		return errors.New("sku is nil")
	}
	return r.db.Create(item).Error
}

// CreateBatch ???? SKU
func (r *GormProductSKURepository) CreateBatch(items []models.ProductSKU) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.Create(&items).Error
}

// Update ?? SKU
func (r *GormProductSKURepository) Update(item *models.ProductSKU) error {
	if item == nil {
		return errors.New("sku is nil")
	}
	return r.db.Save(item).Error
}

// Delete ????? SKU(?????,????????)
func (r *GormProductSKURepository) Delete(id uint) error {
	if id == 0 {
		return errors.New("invalid sku id")
	}
	return r.db.Unscoped().Delete(&models.ProductSKU{}, id).Error
}

// PurgeSoftDeletedByProductAndCode ???????? sku_code ????????
func (r *GormProductSKURepository) PurgeSoftDeletedByProductAndCode(productID uint, skuCode string) error {
	return r.db.Unscoped().
		Where("product_id = ? AND sku_code = ? AND deleted_at IS NOT NULL", productID, skuCode).
		Delete(&models.ProductSKU{}).Error
}

// DeleteByProduct ???????? SKU
func (r *GormProductSKURepository) DeleteByProduct(productID uint) error {
	if productID == 0 {
		return errors.New("invalid product id")
	}
	return r.db.Where("product_id = ?", productID).Delete(&models.ProductSKU{}).Error
}

// ReserveManualStock ??????
func (r *GormProductSKURepository) ReserveManualStock(skuID uint, quantity int) (int64, error) {
	if skuID == 0 || quantity <= 0 {
		return 0, errors.New("invalid manual stock reserve params")
	}
	result := r.db.Model(&models.ProductSKU{}).
		Where("id = ? AND manual_stock_total >= 0 AND manual_stock_total >= ?", skuID, quantity).
		Updates(map[string]interface{}{
			"manual_stock_total":  gorm.Expr("manual_stock_total - ?", quantity),
			"manual_stock_locked": gorm.Expr("manual_stock_locked + ?", quantity),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// ReleaseManualStock ????????
func (r *GormProductSKURepository) ReleaseManualStock(skuID uint, quantity int) (int64, error) {
	if skuID == 0 || quantity <= 0 {
		return 0, errors.New("invalid manual stock release params")
	}
	result := r.db.Model(&models.ProductSKU{}).
		Where("id = ? AND manual_stock_total >= 0 AND manual_stock_locked >= ?", skuID, quantity).
		Updates(map[string]interface{}{
			"manual_stock_total":  gorm.Expr("manual_stock_total + ?", quantity),
			"manual_stock_locked": gorm.Expr("manual_stock_locked - ?", quantity),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// ConsumeManualStock ??????(??????????)
func (r *GormProductSKURepository) ConsumeManualStock(skuID uint, quantity int) (int64, error) {
	if skuID == 0 || quantity <= 0 {
		return 0, errors.New("invalid manual stock consume params")
	}
	result := r.db.Model(&models.ProductSKU{}).
		Where("id = ? AND manual_stock_total >= ? AND (manual_stock_locked >= ? OR (manual_stock_locked < ? AND manual_stock_total >= (? - manual_stock_locked)))",
			skuID, constants.ManualStockUnlimited+1, quantity, quantity, quantity).
		Updates(map[string]interface{}{
			// ?????????:????????????????
			"manual_stock_total":  gorm.Expr("manual_stock_total - CASE WHEN manual_stock_locked >= ? THEN 0 ELSE ? - manual_stock_locked END", quantity, quantity),
			"manual_stock_locked": gorm.Expr("CASE WHEN manual_stock_locked >= ? THEN manual_stock_locked - ? ELSE 0 END", quantity, quantity),
			"manual_stock_sold":   gorm.Expr("manual_stock_sold + ?", quantity),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
