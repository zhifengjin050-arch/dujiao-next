package repository

import (
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// CouponUsageRepository ?????????????
type CouponUsageRepository interface {
	Create(usage *models.CouponUsage) error
	CountByUser(couponID, userID uint) (int64, error)
	ListByOrderID(orderID uint) ([]models.CouponUsage, error)
	ListByUser(filter CouponUsageListFilter) ([]models.CouponUsage, int64, error)
	DeleteByOrderID(orderID uint) error
	WithTx(tx *gorm.DB) *GormCouponUsageRepository
}

// GormCouponUsageRepository GORM ??
type GormCouponUsageRepository struct {
	db *gorm.DB
}

// NewCouponUsageRepository ???????????
func NewCouponUsageRepository(db *gorm.DB) *GormCouponUsageRepository {
	return &GormCouponUsageRepository{db: db}
}

// WithTx ????
func (r *GormCouponUsageRepository) WithTx(tx *gorm.DB) *GormCouponUsageRepository {
	if tx == nil {
		return r
	}
	return &GormCouponUsageRepository{db: tx}
}

// Create ??????
func (r *GormCouponUsageRepository) Create(usage *models.CouponUsage) error {
	return r.db.Create(usage).Error
}

// CountByUser ????????
func (r *GormCouponUsageRepository) CountByUser(couponID, userID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&models.CouponUsage{}).
		Where("coupon_id = ? AND user_id = ?", couponID, userID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ListByOrderID ????????
func (r *GormCouponUsageRepository) ListByOrderID(orderID uint) ([]models.CouponUsage, error) {
	var usages []models.CouponUsage
	if err := r.db.Where("order_id = ?", orderID).Find(&usages).Error; err != nil {
		return nil, err
	}
	return usages, nil
}

// ListByUser ????????
func (r *GormCouponUsageRepository) ListByUser(filter CouponUsageListFilter) ([]models.CouponUsage, int64, error) {
	query := r.db.Model(&models.CouponUsage{}).Where("user_id = ?", filter.UserID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, filter.Page, filter.PageSize)

	var usages []models.CouponUsage
	if err := query.Order("id desc").Find(&usages).Error; err != nil {
		return nil, 0, err
	}
	return usages, total, nil
}

// DeleteByOrderID ????????
func (r *GormCouponUsageRepository) DeleteByOrderID(orderID uint) error {
	return r.db.Where("order_id = ?", orderID).Delete(&models.CouponUsage{}).Error
}
