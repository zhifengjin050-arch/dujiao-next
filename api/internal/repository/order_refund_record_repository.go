package repository

import (
	"errors"
	"strings"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// OrderRefundRecordRepository ??????????
type OrderRefundRecordRepository interface {
	Create(record *models.OrderRefundRecord) error
	GetByID(id uint) (*models.OrderRefundRecord, error)
	ListByOrderIDs(orderIDs []uint) ([]models.OrderRefundRecord, error)
	ListAdmin(filter OrderRefundRecordListFilter) ([]models.OrderRefundRecord, int64, error)
	WithTx(tx *gorm.DB) *GormOrderRefundRecordRepository
}

// GormOrderRefundRecordRepository GORM ??????
type GormOrderRefundRecordRepository struct {
	BaseRepository
}

// NewOrderRefundRecordRepository ????????
func NewOrderRefundRecordRepository(db *gorm.DB) *GormOrderRefundRecordRepository {
	return &GormOrderRefundRecordRepository{BaseRepository: BaseRepository{db: db}}
}

// WithTx ????
func (r *GormOrderRefundRecordRepository) WithTx(tx *gorm.DB) *GormOrderRefundRecordRepository {
	if tx == nil {
		return r
	}
	return &GormOrderRefundRecordRepository{BaseRepository: BaseRepository{db: tx}}
}

// Create ??????
func (r *GormOrderRefundRecordRepository) Create(record *models.OrderRefundRecord) error {
	if record == nil {
		return nil
	}
	return r.db.Create(record).Error
}

// GetByID ?? ID ??????
func (r *GormOrderRefundRecordRepository) GetByID(id uint) (*models.OrderRefundRecord, error) {
	if id == 0 {
		return nil, nil
	}
	var record models.OrderRefundRecord
	if err := r.db.First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// ListByOrderIDs ???ID????????(???????)
func (r *GormOrderRefundRecordRepository) ListByOrderIDs(orderIDs []uint) ([]models.OrderRefundRecord, error) {
	records := make([]models.OrderRefundRecord, 0)
	if len(orderIDs) == 0 {
		return records, nil
	}
	if err := r.db.
		Where("order_id IN ?", orderIDs).
		Order("created_at DESC, id DESC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// ListAdmin ?????????
func (r *GormOrderRefundRecordRepository) ListAdmin(filter OrderRefundRecordListFilter) ([]models.OrderRefundRecord, int64, error) {
	records := make([]models.OrderRefundRecord, 0)
	query := r.db.Model(&models.OrderRefundRecord{}).
		Joins("LEFT JOIN orders ON orders.id = order_refund_records.order_id")

	if filter.UserID != 0 {
		query = query.Where("orders.user_id = ?", filter.UserID)
	}
	if keyword := strings.TrimSpace(filter.UserKeyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"orders.user_id IN ("+
				"SELECT users.id FROM users "+
				"WHERE users.deleted_at IS NULL AND ("+
				"users.email LIKE ? OR "+
				"users.display_name LIKE ? OR "+
				"EXISTS ("+
				"SELECT 1 FROM user_oauth_identities "+
				"WHERE user_oauth_identities.user_id = users.id AND ("+
				"user_oauth_identities.provider LIKE ? OR "+
				"user_oauth_identities.provider_user_id LIKE ? OR "+
				"user_oauth_identities.username LIKE ?"+
				")"+
				")"+
				")"+
				")",
			like, like, like, like, like,
		)
	}
	if orderNo := strings.TrimSpace(filter.OrderNo); orderNo != "" {
		query = query.Where("orders.order_no = ?", orderNo)
	}
	if guestEmail := strings.TrimSpace(filter.GuestEmail); guestEmail != "" {
		query = query.Where("(order_refund_records.guest_email = ? OR orders.guest_email = ?)", guestEmail, guestEmail)
	}
	if keyword := strings.TrimSpace(filter.ProductKeyword); keyword != "" {
		like := "%" + keyword + "%"
		cond, argCount := buildLocalizedLikeCondition(r.db, nil, []string{"oi.title_json"})
		if cond != "" {
			args := repeatLikeArgs(like, argCount)
			query = query.Where(
				"EXISTS (SELECT 1 FROM order_items oi WHERE oi.order_id = order_refund_records.order_id AND ("+cond+"))",
				args...,
			)
		}
	}
	if filter.CreatedFrom != nil {
		query = query.Where("order_refund_records.created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("order_refund_records.created_at <= ?", *filter.CreatedTo)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := applyPagination(query.Session(&gorm.Session{}), filter.Page, filter.PageSize)
	if err := dataQuery.
		Order("order_refund_records.id DESC").
		Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}
