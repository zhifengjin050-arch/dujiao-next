package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// PaymentRepository ????????
type PaymentRepository interface {
	Create(payment *models.Payment) error
	Update(payment *models.Payment) error
	GetByID(id uint) (*models.Payment, error)
	GetByIDs(ids []uint) ([]models.Payment, error)
	GetByGatewayOrderNo(gatewayOrderNo string) (*models.Payment, error)
	GetLatestByProviderRef(providerRef string) (*models.Payment, error)
	ListByOrderID(orderID uint) ([]models.Payment, error)
	ListLatestGatewayOrderNos(orderIDs []uint) (map[uint]string, error)
	GetLatestPendingByOrder(orderID uint, now time.Time) (*models.Payment, error)
	ListAdmin(filter PaymentListFilter) ([]models.Payment, int64, error)
	Transaction(fn func(tx *gorm.DB) error) error
	WithTx(tx *gorm.DB) *GormPaymentRepository
}

// GormPaymentRepository GORM ??
type GormPaymentRepository struct {
	BaseRepository
}

// NewPaymentRepository ??????
func NewPaymentRepository(db *gorm.DB) *GormPaymentRepository {
	return &GormPaymentRepository{BaseRepository: BaseRepository{db: db}}
}

// WithTx ????
func (r *GormPaymentRepository) WithTx(tx *gorm.DB) *GormPaymentRepository {
	if tx == nil {
		return r
	}
	return &GormPaymentRepository{BaseRepository: BaseRepository{db: tx}}
}

// Create ??????
func (r *GormPaymentRepository) Create(payment *models.Payment) error {
	return r.db.Create(payment).Error
}

// Update ??????
func (r *GormPaymentRepository) Update(payment *models.Payment) error {
	return r.db.Save(payment).Error
}

// GetByID ?? ID ??????
func (r *GormPaymentRepository) GetByID(id uint) (*models.Payment, error) {
	var payment models.Payment
	if err := r.db.First(&payment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &payment, nil
}

// GetByIDs ?? ID ????????
func (r *GormPaymentRepository) GetByIDs(ids []uint) ([]models.Payment, error) {
	if len(ids) == 0 {
		return []models.Payment{}, nil
	}
	var payments []models.Payment
	if err := r.db.Where("id IN ?", ids).Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

// GetByGatewayOrderNo ?????????????
func (r *GormPaymentRepository) GetByGatewayOrderNo(gatewayOrderNo string) (*models.Payment, error) {
	gatewayOrderNo = strings.TrimSpace(gatewayOrderNo)
	if gatewayOrderNo == "" {
		return nil, nil
	}
	var payment models.Payment
	result := r.db.Where("gateway_order_no = ?", gatewayOrderNo).Order("id desc").Limit(1).Find(&payment)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &payment, nil
}

// GetLatestByProviderRef ????????????????
func (r *GormPaymentRepository) GetLatestByProviderRef(providerRef string) (*models.Payment, error) {
	providerRef = strings.TrimSpace(providerRef)
	if providerRef == "" {
		return nil, nil
	}
	var payment models.Payment
	result := r.db.Where("provider_ref = ?", providerRef).Order("id desc").Limit(1).Find(&payment)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &payment, nil
}

// ListByOrderID ????????
func (r *GormPaymentRepository) ListByOrderID(orderID uint) ([]models.Payment, error) {
	var payments []models.Payment
	if err := r.db.Where("order_id = ?", orderID).Order("id desc").Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

// ListLatestGatewayOrderNos ??????????????????
func (r *GormPaymentRepository) ListLatestGatewayOrderNos(orderIDs []uint) (map[uint]string, error) {
	result := make(map[uint]string)
	if len(orderIDs) == 0 {
		return result, nil
	}
	var rows []struct {
		OrderID         uint   `gorm:"column:order_id"`
		GatewayOrderNo  string `gorm:"column:gateway_order_no"`
	}
	subQuery := r.db.Model(&models.Payment{}).
		Select("order_id, MAX(id) as max_id").
		Where("order_id IN ? AND gateway_order_no IS NOT NULL AND gateway_order_no <> ''", orderIDs).
		Group("order_id")
	if err := r.db.Model(&models.Payment{}).
		Select("payments.order_id, payments.gateway_order_no").
		Joins("INNER JOIN (?) AS latest ON payments.id = latest.max_id", subQuery).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.OrderID] = row.GatewayOrderNo
	}
	return result, nil
}

// GetLatestPendingByOrder ???????????
func (r *GormPaymentRepository) GetLatestPendingByOrder(orderID uint, now time.Time) (*models.Payment, error) {
	var payment models.Payment
	result := r.db.
		Select("payments.*, payment_channels.name AS channel_name").
		Joins("LEFT JOIN payment_channels ON payment_channels.id = payments.channel_id AND payment_channels.deleted_at IS NULL").
		Where("payments.order_id = ? AND payments.status IN ? AND (payments.expired_at IS NULL OR payments.expired_at > ?) AND ((payments.pay_url IS NOT NULL AND payments.pay_url <> '') OR (payments.qr_code IS NOT NULL AND payments.qr_code <> ''))",
			orderID,
			[]string{constants.PaymentStatusInitiated, constants.PaymentStatusPending},
			now,
		).Order("payments.id desc").Limit(1).Find(&payment)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &payment, nil
}

// GetLatestPendingByOrderChannel ????+?????????
func (r *GormPaymentRepository) GetLatestPendingByOrderChannel(orderID uint, channelID uint, now time.Time) (*models.Payment, error) {
	var payment models.Payment
	result := r.db.Where("order_id = ? AND channel_id = ? AND status IN ? AND (expired_at IS NULL OR expired_at > ?) AND ((pay_url IS NOT NULL AND pay_url <> '') OR (qr_code IS NOT NULL AND qr_code <> ''))",
		orderID,
		channelID,
		[]string{constants.PaymentStatusInitiated, constants.PaymentStatusPending},
		now,
	).Order("id desc").Limit(1).Find(&payment)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &payment, nil
}

// ListAdmin ???????
func (r *GormPaymentRepository) ListAdmin(filter PaymentListFilter) ([]models.Payment, int64, error) {
	query := r.db.Model(&models.Payment{})

	if filter.UserID != 0 {
		query = query.
			Joins("LEFT JOIN orders ON orders.id = payments.order_id").
			Joins("LEFT JOIN wallet_recharge_orders ON wallet_recharge_orders.payment_id = payments.id").
			Where("(orders.user_id = ? OR wallet_recharge_orders.user_id = ?)", filter.UserID, filter.UserID)
	}
	if filter.OrderID != 0 {
		query = query.Where("payments.order_id = ?", filter.OrderID)
	}
	if filter.ChannelID != 0 {
		query = query.Where("payments.channel_id = ?", filter.ChannelID)
	}
	if filter.ProviderType != "" {
		query = query.Where("payments.provider_type = ?", filter.ProviderType)
	}
	if filter.ChannelType != "" {
		query = query.Where("payments.channel_type = ?", filter.ChannelType)
	}
	if filter.Status != "" {
		query = query.Where("payments.status = ?", filter.Status)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("payments.created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("payments.created_at <= ?", *filter.CreatedTo)
	}

	if filter.Lightweight {
		query = query.Select(
			"payments.id",
			"payments.order_id",
			"payments.channel_id",
			"payments.provider_type",
			"payments.channel_type",
			"payments.interaction_mode",
			"payments.amount",
			"payments.fee_rate",
			"payments.fee_amount",
			"payments.currency",
			"payments.status",
			"payments.provider_ref",
			"payments.created_at",
			"payments.updated_at",
			"payments.paid_at",
			"payments.expired_at",
			"payments.callback_at",
		)
	}

	var total int64
	if !filter.SkipCount {
		if err := query.Count(&total).Error; err != nil {
			return nil, 0, err
		}
	}

	query = applyPagination(query, filter.Page, filter.PageSize)

	var payments []models.Payment
	if err := query.Order("payments.id desc").Find(&payments).Error; err != nil {
		return nil, 0, err
	}
	return payments, total, nil
}
