package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// PaymentChannelRepository ??????????
type PaymentChannelRepository interface {
	Create(channel *models.PaymentChannel) error
	Update(channel *models.PaymentChannel) error
	Delete(id uint) error
	GetByID(id uint) (*models.PaymentChannel, error)
	ListByIDs(ids []uint) ([]models.PaymentChannel, error)
	List(filter PaymentChannelListFilter) ([]models.PaymentChannel, int64, error)
	WithTx(tx *gorm.DB) *GormPaymentChannelRepository
}

// GormPaymentChannelRepository GORM ??
type GormPaymentChannelRepository struct {
	db *gorm.DB
}

// NewPaymentChannelRepository ????????
func NewPaymentChannelRepository(db *gorm.DB) *GormPaymentChannelRepository {
	return &GormPaymentChannelRepository{db: db}
}

// WithTx ????
func (r *GormPaymentChannelRepository) WithTx(tx *gorm.DB) *GormPaymentChannelRepository {
	if tx == nil {
		return r
	}
	return &GormPaymentChannelRepository{db: tx}
}

// Create ??????
func (r *GormPaymentChannelRepository) Create(channel *models.PaymentChannel) error {
	return r.db.Create(channel).Error
}

// Update ??????
func (r *GormPaymentChannelRepository) Update(channel *models.PaymentChannel) error {
	return r.db.Save(channel).Error
}

// Delete ??????
func (r *GormPaymentChannelRepository) Delete(id uint) error {
	return r.db.Delete(&models.PaymentChannel{}, id).Error
}

// GetByID ?? ID ??????
func (r *GormPaymentChannelRepository) GetByID(id uint) (*models.PaymentChannel, error) {
	var channel models.PaymentChannel
	if err := r.db.First(&channel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &channel, nil
}

// ListByIDs ?? ID ????????
func (r *GormPaymentChannelRepository) ListByIDs(ids []uint) ([]models.PaymentChannel, error) {
	if len(ids) == 0 {
		return []models.PaymentChannel{}, nil
	}
	var channels []models.PaymentChannel
	if err := r.db.Where("id IN ?", ids).Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

// List ??????
func (r *GormPaymentChannelRepository) List(filter PaymentChannelListFilter) ([]models.PaymentChannel, int64, error) {
	query := r.db.Model(&models.PaymentChannel{})

	if filter.ProviderType != "" {
		query = query.Where("provider_type = ?", filter.ProviderType)
	}
	if filter.ChannelType != "" {
		query = query.Where("channel_type = ?", filter.ChannelType)
	}
	if filter.ActiveOnly {
		query = query.Where("is_active = ?", true)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, filter.Page, filter.PageSize)

	var channels []models.PaymentChannel
	if err := query.Order("sort_order DESC, id ASC").Find(&channels).Error; err != nil {
		return nil, 0, err
	}
	return channels, total, nil
}
