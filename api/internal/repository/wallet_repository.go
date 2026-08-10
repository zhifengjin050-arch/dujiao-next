package repository

import (
	"errors"
	"strings"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WalletRepository ????????
type WalletRepository interface {
	GetAccountByUserID(userID uint) (*models.WalletAccount, error)
	GetAccountByUserIDForUpdate(userID uint) (*models.WalletAccount, error)
	GetAccountsByUserIDs(userIDs []uint) ([]models.WalletAccount, error)
	CreateAccount(account *models.WalletAccount) error
	UpdateAccount(account *models.WalletAccount) error
	ListAccounts(filter WalletAccountListFilter) ([]models.WalletAccount, int64, error)
	CreateTransaction(txn *models.WalletTransaction) error
	GetTransactionByReference(reference string) (*models.WalletTransaction, error)
	ListTransactions(filter WalletTransactionListFilter) ([]models.WalletTransaction, int64, error)
	CreateRechargeOrder(order *models.WalletRechargeOrder) error
	UpdateRechargeOrder(order *models.WalletRechargeOrder) error
	GetRechargeOrderByRechargeNo(userID uint, rechargeNo string) (*models.WalletRechargeOrder, error)
	GetRechargeOrderByPaymentID(paymentID uint) (*models.WalletRechargeOrder, error)
	GetRechargeOrderByPaymentIDAndUser(paymentID uint, userID uint) (*models.WalletRechargeOrder, error)
	GetRechargeOrderByPaymentIDForUpdate(paymentID uint) (*models.WalletRechargeOrder, error)
	ListRechargeOrdersAdmin(filter WalletRechargeListFilter) ([]models.WalletRechargeOrder, int64, error)
	GetRechargeOrdersByPaymentIDs(paymentIDs []uint) ([]models.WalletRechargeOrder, error)
	Transaction(fn func(tx *gorm.DB) error) error
	WithTx(tx *gorm.DB) *GormWalletRepository
}

// GormWalletRepository GORM ??????
type GormWalletRepository struct {
	BaseRepository
}

// NewWalletRepository ??????
func NewWalletRepository(db *gorm.DB) *GormWalletRepository {
	return &GormWalletRepository{BaseRepository: BaseRepository{db: db}}
}

// WithTx ????
func (r *GormWalletRepository) WithTx(tx *gorm.DB) *GormWalletRepository {
	if tx == nil {
		return r
	}
	return &GormWalletRepository{BaseRepository: BaseRepository{db: tx}}
}

// GetAccountByUserID ???ID??????
func (r *GormWalletRepository) GetAccountByUserID(userID uint) (*models.WalletAccount, error) {
	if userID == 0 {
		return nil, nil
	}
	var account models.WalletAccount
	if err := r.db.Where("user_id = ?", userID).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// GetAccountByUserIDForUpdate ???ID????????
func (r *GormWalletRepository) GetAccountByUserIDForUpdate(userID uint) (*models.WalletAccount, error) {
	if userID == 0 {
		return nil, nil
	}
	var account models.WalletAccount
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// GetAccountsByUserIDs ????????
func (r *GormWalletRepository) GetAccountsByUserIDs(userIDs []uint) ([]models.WalletAccount, error) {
	if len(userIDs) == 0 {
		return []models.WalletAccount{}, nil
	}
	var accounts []models.WalletAccount
	if err := r.db.Where("user_id IN ?", userIDs).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// CreateAccount ??????
func (r *GormWalletRepository) CreateAccount(account *models.WalletAccount) error {
	return r.db.Create(account).Error
}

// UpdateAccount ??????
func (r *GormWalletRepository) UpdateAccount(account *models.WalletAccount) error {
	return r.db.Save(account).Error
}

// ListAccounts ????????
func (r *GormWalletRepository) ListAccounts(filter WalletAccountListFilter) ([]models.WalletAccount, int64, error) {
	query := r.db.Model(&models.WalletAccount{})
	if filter.UserID != 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, filter.Page, filter.PageSize)

	var accounts []models.WalletAccount
	if err := query.Order("id desc").Find(&accounts).Error; err != nil {
		return nil, 0, err
	}
	return accounts, total, nil
}

// CreateTransaction ??????
func (r *GormWalletRepository) CreateTransaction(txn *models.WalletTransaction) error {
	return r.db.Create(txn).Error
}

// GetTransactionByReference ????????
func (r *GormWalletRepository) GetTransactionByReference(reference string) (*models.WalletTransaction, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return nil, nil
	}
	var txn models.WalletTransaction
	if err := r.db.Where("reference = ?", reference).First(&txn).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &txn, nil
}

// ListTransactions ????????
func (r *GormWalletRepository) ListTransactions(filter WalletTransactionListFilter) ([]models.WalletTransaction, int64, error) {
	query := r.db.Model(&models.WalletTransaction{})
	if filter.UserID != 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.OrderID != 0 {
		query = query.Where("order_id = ?", filter.OrderID)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Direction != "" {
		query = query.Where("direction = ?", filter.Direction)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, filter.Page, filter.PageSize)

	var txns []models.WalletTransaction
	if err := query.Order("id desc").Find(&txns).Error; err != nil {
		return nil, 0, err
	}
	return txns, total, nil
}

// CreateRechargeOrder ?????????
func (r *GormWalletRepository) CreateRechargeOrder(order *models.WalletRechargeOrder) error {
	return r.db.Create(order).Error
}

// UpdateRechargeOrder ?????????
func (r *GormWalletRepository) UpdateRechargeOrder(order *models.WalletRechargeOrder) error {
	return r.db.Save(order).Error
}

// GetRechargeOrderByRechargeNo ????????????
func (r *GormWalletRepository) GetRechargeOrderByRechargeNo(userID uint, rechargeNo string) (*models.WalletRechargeOrder, error) {
	if userID == 0 {
		return nil, nil
	}
	rechargeNo = strings.TrimSpace(rechargeNo)
	if rechargeNo == "" {
		return nil, nil
	}
	var order models.WalletRechargeOrder
	if err := r.db.Where("user_id = ? AND recharge_no = ?", userID, rechargeNo).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetRechargeOrderByPaymentID ???ID???????
func (r *GormWalletRepository) GetRechargeOrderByPaymentID(paymentID uint) (*models.WalletRechargeOrder, error) {
	if paymentID == 0 {
		return nil, nil
	}
	var order models.WalletRechargeOrder
	if err := r.db.Where("payment_id = ?", paymentID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetRechargeOrderByPaymentIDAndUser ???ID???ID???????
func (r *GormWalletRepository) GetRechargeOrderByPaymentIDAndUser(paymentID uint, userID uint) (*models.WalletRechargeOrder, error) {
	if paymentID == 0 || userID == 0 {
		return nil, nil
	}
	var order models.WalletRechargeOrder
	if err := r.db.Where("payment_id = ? AND user_id = ?", paymentID, userID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetRechargeOrderByPaymentIDForUpdate ???ID?????????
func (r *GormWalletRepository) GetRechargeOrderByPaymentIDForUpdate(paymentID uint) (*models.WalletRechargeOrder, error) {
	if paymentID == 0 {
		return nil, nil
	}
	var order models.WalletRechargeOrder
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("payment_id = ?", paymentID).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// ListRechargeOrdersAdmin ????????????
func (r *GormWalletRepository) ListRechargeOrdersAdmin(filter WalletRechargeListFilter) ([]models.WalletRechargeOrder, int64, error) {
	query := r.db.Model(&models.WalletRechargeOrder{})

	if filter.RechargeNo != "" {
		query = query.Where("wallet_recharge_orders.recharge_no LIKE ?", "%"+filter.RechargeNo+"%")
	}
	if filter.UserID != 0 {
		query = query.Where("wallet_recharge_orders.user_id = ?", filter.UserID)
	}
	if filter.UserKeyword != "" {
		like := "%" + filter.UserKeyword + "%"
		query = query.
			Joins("LEFT JOIN users ON users.id = wallet_recharge_orders.user_id").
			Where("(users.email LIKE ? OR users.display_name LIKE ?)", like, like)
	}
	if filter.PaymentID != 0 {
		query = query.Where("wallet_recharge_orders.payment_id = ?", filter.PaymentID)
	}
	if filter.ChannelID != 0 {
		query = query.Where("wallet_recharge_orders.channel_id = ?", filter.ChannelID)
	}
	if filter.ProviderType != "" {
		query = query.Where("wallet_recharge_orders.provider_type = ?", filter.ProviderType)
	}
	if filter.ChannelType != "" {
		query = query.Where("wallet_recharge_orders.channel_type = ?", filter.ChannelType)
	}
	if filter.Status != "" {
		query = query.Where("wallet_recharge_orders.status = ?", filter.Status)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("wallet_recharge_orders.created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("wallet_recharge_orders.created_at <= ?", *filter.CreatedTo)
	}
	if filter.PaidFrom != nil {
		query = query.Where("wallet_recharge_orders.paid_at >= ?", *filter.PaidFrom)
	}
	if filter.PaidTo != nil {
		query = query.Where("wallet_recharge_orders.paid_at <= ?", *filter.PaidTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, filter.Page, filter.PageSize)

	var orders []models.WalletRechargeOrder
	if err := query.Order("wallet_recharge_orders.id DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// GetRechargeOrdersByPaymentIDs ???ID?????????
func (r *GormWalletRepository) GetRechargeOrdersByPaymentIDs(paymentIDs []uint) ([]models.WalletRechargeOrder, error) {
	if len(paymentIDs) == 0 {
		return []models.WalletRechargeOrder{}, nil
	}
	var orders []models.WalletRechargeOrder
	if err := r.db.Where("payment_id IN ?", paymentIDs).Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}
