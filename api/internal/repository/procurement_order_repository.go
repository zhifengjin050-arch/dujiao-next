package repository

import (
	"errors"
	"time"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// ProcurementOrderRepository ?????????
type ProcurementOrderRepository interface {
	GetByID(id uint) (*models.ProcurementOrder, error)
	GetByLocalOrderID(localOrderID uint) (*models.ProcurementOrder, error)
	GetByLocalOrderNo(localOrderNo string) (*models.ProcurementOrder, error)
	GetByUpstreamOrderID(connectionID, upstreamOrderID uint) (*models.ProcurementOrder, error)
	Create(order *models.ProcurementOrder) error
	Update(order *models.ProcurementOrder) error
	UpdateStatus(id uint, status string, updates map[string]interface{}) error
	List(filter ProcurementOrderListFilter) ([]models.ProcurementOrder, int64, error)
	ListRetriable(now time.Time, limit int) ([]models.ProcurementOrder, error)
	ListByLocalOrderIDs(localOrderIDs []uint) ([]models.ProcurementOrder, error)
	ListByConnectionAndTimeRange(connectionID uint, start, end time.Time) ([]models.ProcurementOrder, error)
	Transaction(fn func(tx *gorm.DB) error) error
	WithTx(tx *gorm.DB) *GormProcurementOrderRepository
}

// ProcurementOrderListFilter ???????
type ProcurementOrderListFilter struct {
	ConnectionID    uint
	Status          string
	LocalOrderNo    string
	UpstreamOrderNo string
	CreatedFrom     string
	CreatedTo       string
	Pagination
}

// GormProcurementOrderRepository GORM ??
type GormProcurementOrderRepository struct {
	db *gorm.DB
}

// NewProcurementOrderRepository ???????
func NewProcurementOrderRepository(db *gorm.DB) *GormProcurementOrderRepository {
	return &GormProcurementOrderRepository{db: db}
}

// WithTx ????
func (r *GormProcurementOrderRepository) WithTx(tx *gorm.DB) *GormProcurementOrderRepository {
	if tx == nil {
		return r
	}
	return &GormProcurementOrderRepository{db: tx}
}

// Transaction ????
func (r *GormProcurementOrderRepository) Transaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}

// GetByID ?? ID ??
func (r *GormProcurementOrderRepository) GetByID(id uint) (*models.ProcurementOrder, error) {
	var order models.ProcurementOrder
	if err := r.db.Preload("Connection").Preload("LocalOrder").Preload("LocalOrder.Items").First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetByLocalOrderID ?????? ID ??
func (r *GormProcurementOrderRepository) GetByLocalOrderID(localOrderID uint) (*models.ProcurementOrder, error) {
	var order models.ProcurementOrder
	if err := r.db.Where("local_order_id = ?", localOrderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetByLocalOrderNo ?????????
func (r *GormProcurementOrderRepository) GetByLocalOrderNo(localOrderNo string) (*models.ProcurementOrder, error) {
	var order models.ProcurementOrder
	if err := r.db.Where("local_order_no = ?", localOrderNo).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetByUpstreamOrderID ????+???? ID ??
func (r *GormProcurementOrderRepository) GetByUpstreamOrderID(connectionID, upstreamOrderID uint) (*models.ProcurementOrder, error) {
	var order models.ProcurementOrder
	if err := r.db.Where("connection_id = ? AND upstream_order_id = ?", connectionID, upstreamOrderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// Create ?????
func (r *GormProcurementOrderRepository) Create(order *models.ProcurementOrder) error {
	return r.db.Create(order).Error
}

// Update ?????
func (r *GormProcurementOrderRepository) Update(order *models.ProcurementOrder) error {
	return r.db.Save(order).Error
}

// UpdateStatus ???????
func (r *GormProcurementOrderRepository) UpdateStatus(id uint, status string, updates map[string]interface{}) error {
	if updates == nil {
		updates = map[string]interface{}{}
	}
	updates["status"] = status
	return r.db.Model(&models.ProcurementOrder{}).Where("id = ?", id).Updates(updates).Error
}

// List ????
func (r *GormProcurementOrderRepository) List(filter ProcurementOrderListFilter) ([]models.ProcurementOrder, int64, error) {
	var orders []models.ProcurementOrder
	var total int64

	q := r.db.Model(&models.ProcurementOrder{})
	if filter.ConnectionID > 0 {
		q = q.Where("connection_id = ?", filter.ConnectionID)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.LocalOrderNo != "" {
		q = q.Where("local_order_no = ?", filter.LocalOrderNo)
	}
	if filter.UpstreamOrderNo != "" {
		q = q.Where("upstream_order_no = ?", filter.UpstreamOrderNo)
	}
	if filter.CreatedFrom != "" {
		q = q.Where("created_at >= ?", filter.CreatedFrom)
	}
	if filter.CreatedTo != "" {
		q = q.Where("created_at <= ?", filter.CreatedTo)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q = q.Order("created_at DESC").Preload("Connection").Preload("LocalOrder").Preload("LocalOrder.Items")
	if filter.Page > 0 && filter.PageSize > 0 {
		q = q.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	if err := q.Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ListRetriable ??????????
func (r *GormProcurementOrderRepository) ListRetriable(now time.Time, limit int) ([]models.ProcurementOrder, error) {
	var orders []models.ProcurementOrder
	q := r.db.Where("status IN (?, ?) AND next_retry_at IS NOT NULL AND next_retry_at <= ?",
		"pending", "failed", now).
		Order("next_retry_at ASC").
		Limit(limit)
	if err := q.Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// ListByLocalOrderIDs ?????? ID ????
func (r *GormProcurementOrderRepository) ListByLocalOrderIDs(localOrderIDs []uint) ([]models.ProcurementOrder, error) {
	if len(localOrderIDs) == 0 {
		return nil, nil
	}
	var orders []models.ProcurementOrder
	if err := r.db.Where("local_order_id IN ?", localOrderIDs).Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// ListByConnectionAndTimeRange ?????????????
func (r *GormProcurementOrderRepository) ListByConnectionAndTimeRange(connectionID uint, start, end time.Time) ([]models.ProcurementOrder, error) {
	var orders []models.ProcurementOrder
	q := r.db.Preload("LocalOrder").Where("connection_id = ? AND created_at >= ? AND created_at <= ?", connectionID, start, end)
	if err := q.Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}
