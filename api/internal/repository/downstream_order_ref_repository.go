package repository

import (
	"errors"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// DownstreamOrderRefRepository ????????????
type DownstreamOrderRefRepository interface {
	GetByID(id uint) (*models.DownstreamOrderRef, error)
	GetByOrderID(orderID uint) (*models.DownstreamOrderRef, error)
	GetByCredentialAndDownstreamNo(credentialID uint, downstreamOrderNo string) (*models.DownstreamOrderRef, error)
	Create(ref *models.DownstreamOrderRef) error
	Update(ref *models.DownstreamOrderRef) error
	ListPendingCallbacks(limit int) ([]models.DownstreamOrderRef, error)
	ListByCredentialID(credentialID uint, filter DownstreamOrderRefListFilter) ([]models.DownstreamOrderRef, int64, error)
}

// DownstreamOrderRefListFilter ??????????
type DownstreamOrderRefListFilter struct {
	CallbackStatus string
	Pagination
}

// GormDownstreamOrderRefRepository GORM ??
type GormDownstreamOrderRefRepository struct {
	db *gorm.DB
}

// NewDownstreamOrderRefRepository ??????????
func NewDownstreamOrderRefRepository(db *gorm.DB) *GormDownstreamOrderRefRepository {
	return &GormDownstreamOrderRefRepository{db: db}
}

// GetByID ?? ID ??
func (r *GormDownstreamOrderRefRepository) GetByID(id uint) (*models.DownstreamOrderRef, error) {
	var ref models.DownstreamOrderRef
	if err := r.db.First(&ref, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ref, nil
}

// GetByOrderID ???? ID ??
func (r *GormDownstreamOrderRefRepository) GetByOrderID(orderID uint) (*models.DownstreamOrderRef, error) {
	var ref models.DownstreamOrderRef
	if err := r.db.Where("order_id = ?", orderID).First(&ref).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ref, nil
}

// GetByCredentialAndDownstreamNo ???? ID ????????(???????)
func (r *GormDownstreamOrderRefRepository) GetByCredentialAndDownstreamNo(credentialID uint, downstreamOrderNo string) (*models.DownstreamOrderRef, error) {
	if credentialID == 0 || downstreamOrderNo == "" {
		return nil, nil
	}
	var ref models.DownstreamOrderRef
	if err := r.db.Where("api_credential_id = ? AND downstream_order_no = ?", credentialID, downstreamOrderNo).First(&ref).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ref, nil
}

// Create ????????
func (r *GormDownstreamOrderRefRepository) Create(ref *models.DownstreamOrderRef) error {
	return r.db.Create(ref).Error
}

// Update ????????
func (r *GormDownstreamOrderRefRepository) Update(ref *models.DownstreamOrderRef) error {
	return r.db.Save(ref).Error
}

// ListPendingCallbacks ??????????
func (r *GormDownstreamOrderRefRepository) ListPendingCallbacks(limit int) ([]models.DownstreamOrderRef, error) {
	var refs []models.DownstreamOrderRef
	q := r.db.Where("callback_status = ? AND callback_url != ''", constants.CallbackStatusPending).
		Order("created_at ASC").
		Limit(limit)
	if err := q.Find(&refs).Error; err != nil {
		return nil, err
	}
	return refs, nil
}

// ListByCredentialID ???? ID ????
func (r *GormDownstreamOrderRefRepository) ListByCredentialID(credentialID uint, filter DownstreamOrderRefListFilter) ([]models.DownstreamOrderRef, int64, error) {
	var refs []models.DownstreamOrderRef
	var total int64

	q := r.db.Model(&models.DownstreamOrderRef{}).Where("api_credential_id = ?", credentialID)
	if filter.CallbackStatus != "" {
		q = q.Where("callback_status = ?", filter.CallbackStatus)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q = q.Order("created_at DESC")
	if filter.Page > 0 && filter.PageSize > 0 {
		q = q.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	if err := q.Find(&refs).Error; err != nil {
		return nil, 0, err
	}
	return refs, total, nil
}
