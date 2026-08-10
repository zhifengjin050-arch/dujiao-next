package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	walletDefaultCurrency = "CNY"
)

// WalletService ????
type WalletService struct {
	walletRepo     repository.WalletRepository
	orderRepo      repository.OrderRepository
	userRepo       repository.UserRepository
	affiliateSvc   *AffiliateService
	settingService *SettingService
}

// WalletRechargeInput ??????
type WalletRechargeInput struct {
	UserID   uint
	Amount   models.Money
	Currency string
	Remark   string
}

// WalletAdjustInput ?????????
type WalletAdjustInput struct {
	UserID   uint
	Delta    models.Money
	Currency string
	Remark   string
}

// WalletCreditInput ???????
type WalletCreditInput struct {
	UserID    uint
	Amount    models.Money
	Currency  string
	TxnType   string
	Reference string
	Remark    string
	OrderID   *uint
}

// AdminRefundToWalletInput ??????????
type AdminRefundToWalletInput struct {
	OrderID uint
	Amount  models.Money
	Remark  string
}

// NewWalletService ??????
func NewWalletService(
	walletRepo repository.WalletRepository,
	orderRepo repository.OrderRepository,
	userRepo repository.UserRepository,
	affiliateSvc *AffiliateService,
	settingService *SettingService,
) *WalletService {
	return &WalletService{
		walletRepo:     walletRepo,
		orderRepo:      orderRepo,
		userRepo:       userRepo,
		affiliateSvc:   affiliateSvc,
		settingService: settingService,
	}
}

// GetAccount ??????(????????)
func (s *WalletService) GetAccount(userID uint) (*models.WalletAccount, error) {
	if userID == 0 {
		return nil, ErrWalletAccountNotFound
	}
	return s.getOrCreateAccount(userID)
}

// ListTransactions ??????
func (s *WalletService) ListTransactions(filter repository.WalletTransactionListFilter) ([]models.WalletTransaction, int64, error) {
	return s.walletRepo.ListTransactions(filter)
}

// ListRechargeOrdersAdmin ??????????
func (s *WalletService) ListRechargeOrdersAdmin(filter repository.WalletRechargeListFilter) ([]models.WalletRechargeOrder, int64, error) {
	return s.walletRepo.ListRechargeOrdersAdmin(filter)
}

// ListUserRechargeOrders ????????????
func (s *WalletService) ListUserRechargeOrders(userID uint, page, pageSize int, status, rechargeNo string) ([]models.WalletRechargeOrder, int64, error) {
	if userID == 0 {
		return nil, 0, ErrWalletAccountNotFound
	}
	return s.walletRepo.ListRechargeOrdersAdmin(repository.WalletRechargeListFilter{
		Page:       page,
		PageSize:   pageSize,
		UserID:     userID,
		Status:     status,
		RechargeNo: rechargeNo,
	})
}

// GetRechargeOrderByRechargeNo ??????????????
func (s *WalletService) GetRechargeOrderByRechargeNo(userID uint, rechargeNo string) (*models.WalletRechargeOrder, error) {
	if userID == 0 {
		return nil, ErrWalletRechargeNotFound
	}
	order, err := s.walletRepo.GetRechargeOrderByRechargeNo(userID, rechargeNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrWalletRechargeNotFound
	}
	return order, nil
}

// GetRechargeOrderByPaymentIDAndUser ???ID????????
func (s *WalletService) GetRechargeOrderByPaymentIDAndUser(paymentID uint, userID uint) (*models.WalletRechargeOrder, error) {
	if paymentID == 0 || userID == 0 {
		return nil, ErrWalletRechargeNotFound
	}
	order, err := s.walletRepo.GetRechargeOrderByPaymentIDAndUser(paymentID, userID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrWalletRechargeNotFound
	}
	return order, nil
}

// GetBalancesByUserIDs ????????
func (s *WalletService) GetBalancesByUserIDs(userIDs []uint) (map[uint]models.Money, error) {
	result := make(map[uint]models.Money, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	accounts, err := s.walletRepo.GetAccountsByUserIDs(userIDs)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		result[account.UserID] = account.Balance
	}
	return result, nil
}

// Recharge ??????
func (s *WalletService) Recharge(input WalletRechargeInput) (*models.WalletAccount, *models.WalletTransaction, error) {
	if input.UserID == 0 {
		return nil, nil, ErrWalletAccountNotFound
	}
	amount := input.Amount.Decimal.Round(2)
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, nil, ErrWalletInvalidAmount
	}
	reference := buildWalletReference("recharge", input.UserID)
	remark := cleanWalletRemark(input.Remark, "????")
	currency := normalizeWalletCurrency(input.Currency)
	return s.changeBalance(input.UserID, amount, constants.WalletTxnTypeRecharge, nil, reference, remark, currency)
}

// AdminAdjustBalance ?????????
func (s *WalletService) AdminAdjustBalance(input WalletAdjustInput) (*models.WalletAccount, *models.WalletTransaction, error) {
	if input.UserID == 0 {
		return nil, nil, ErrWalletAccountNotFound
	}
	delta := input.Delta.Decimal.Round(2)
	if delta.IsZero() {
		return nil, nil, ErrWalletInvalidAmount
	}
	reference := buildWalletReference("admin_adjust", input.UserID)
	remark := cleanWalletRemark(input.Remark, "???????")
	currency := normalizeWalletCurrency(input.Currency)
	return s.changeBalance(input.UserID, delta, constants.WalletTxnTypeAdminAdjust, nil, reference, remark, currency)
}

// AdminRefundToWallet ??????????
func (s *WalletService) AdminRefundToWallet(input AdminRefundToWalletInput) (*models.Order, *models.WalletTransaction, *models.OrderRefundRecord, error) {
	if input.OrderID == 0 {
		return nil, nil, nil, ErrOrderNotFound
	}
	amount := input.Amount.Decimal.Round(2)
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, nil, nil, ErrWalletInvalidAmount
	}
	reference := buildWalletReference(fmt.Sprintf("order:%d:admin_refund", input.OrderID), input.OrderID)
	remark := cleanWalletRemark(input.Remark, "????????")
	recordRemark := strings.TrimSpace(input.Remark)

	var txnResult *models.WalletTransaction
	var refundRecordResult *models.OrderRefundRecord
	if err := s.walletRepo.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&order, input.OrderID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrOrderNotFound
			}
			return err
		}
		if order.UserID == 0 {
			return ErrWalletNotSupportedForGuest
		}
		if order.PaidAt == nil {
			return ErrOrderStatusInvalid
		}
		cfg := DefaultOrderRefundConfig()
		if s.settingService != nil {
			cfgLoaded, cfgErr := s.settingService.GetOrderRefundConfig()
			if cfgErr != nil {
				return cfgErr
			}
			cfg = cfgLoaded
		}
		if isOrderRefundWindowExpired(&order, cfg.MaxRefundDays, time.Now()) {
			return ErrOrderRefundExpired
		}
		if order.TotalAmount.Decimal.LessThanOrEqual(decimal.Zero) {
			return ErrOrderStatusInvalid
		}
		refundedBefore := order.RefundedAmount.Decimal.Round(2)
		refundable := order.TotalAmount.Decimal.Sub(refundedBefore).Round(2)
		if amount.GreaterThan(refundable) {
			return ErrWalletRefundExceeded
		}

		repo := s.walletRepo.WithTx(tx)
		account, err := s.ensureAccountForUpdate(repo, order.UserID, time.Now())
		if err != nil {
			return err
		}
		before := account.Balance.Decimal.Round(2)
		after := before.Add(amount).Round(2)
		account.Balance = models.NewMoneyFromDecimal(after)
		account.UpdatedAt = time.Now()
		if err := repo.UpdateAccount(account); err != nil {
			return ErrWalletAccountUpdateFailed
		}

		newRefunded := refundedBefore.Add(amount).Round(2)
		now := time.Now()
		updates := map[string]interface{}{
			"refunded_amount": models.NewMoneyFromDecimal(newRefunded),
			"updated_at":      now,
		}
		markRefunded := newRefunded.GreaterThanOrEqual(order.TotalAmount.Decimal.Round(2))
		if markRefunded {
			updates["status"] = constants.OrderStatusRefunded
		} else {
			updates["status"] = constants.OrderStatusPartiallyRefunded
		}
		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(updates).Error; err != nil {
			return ErrOrderUpdateFailed
		}
		if order.ParentID == nil {
			targetStatus := constants.OrderStatusPartiallyRefunded
			if markRefunded {
				targetStatus = constants.OrderStatusRefunded
			}
			if err := applyParentRefundChildStatusUpdatesTx(tx, order.ID, targetStatus, now); err != nil {
				return ErrOrderUpdateFailed
			}
		}
		if order.ParentID != nil {
			if _, err := syncParentStatus(s.orderRepo.WithTx(tx), *order.ParentID, now); err != nil {
				return ErrOrderUpdateFailed
			}
		}
		if s.affiliateSvc != nil {
			if err := s.affiliateSvc.HandleOrderRefundedTx(
				tx,
				&order,
				amount,
				refundedBefore,
				"order_refunded_to_wallet",
			); err != nil {
				return err
			}
		}

		txn := &models.WalletTransaction{
			UserID:        order.UserID,
			OrderID:       &order.ID,
			Type:          constants.WalletTxnTypeAdminRefund,
			Direction:     constants.WalletTxnDirectionIn,
			Amount:        models.NewMoneyFromDecimal(amount),
			BalanceBefore: models.NewMoneyFromDecimal(before),
			BalanceAfter:  models.NewMoneyFromDecimal(after),
			Currency:      normalizeWalletCurrency(order.Currency),
			Reference:     reference,
			Remark:        remark,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := repo.CreateTransaction(txn); err != nil {
			return ErrWalletTransactionCreateFailed
		}
		record := &models.OrderRefundRecord{
			UserID:     order.UserID,
			GuestEmail: order.GuestEmail,
			OrderID:    order.ID,
			Type:       constants.OrderRefundTypeWallet,
			Amount:     models.NewMoneyFromDecimal(amount),
			Currency:   normalizeWalletCurrency(order.Currency),
			Remark:     recordRemark,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := tx.Create(record).Error; err != nil {
			return ErrRefundRecordCreateFailed
		}
		txnResult = txn
		refundRecordResult = record
		return nil
	}); err != nil {
		return nil, nil, nil, err
	}

	order, err := s.orderRepo.GetByID(input.OrderID)
	if err != nil {
		return nil, nil, nil, ErrOrderFetchFailed
	}
	if order == nil {
		return nil, nil, nil, ErrOrderNotFound
	}
	return order, txnResult, refundRecordResult, nil
}

// ApplyOrderBalance ????????????????,??????
func (s *WalletService) ApplyOrderBalance(tx *gorm.DB, order *models.Order, useBalance bool) (decimal.Decimal, error) {
	if tx == nil {
		return decimal.Zero, ErrOrderUpdateFailed
	}
	if order == nil {
		return decimal.Zero, ErrOrderNotFound
	}
	if !useBalance {
		return order.WalletPaidAmount.Decimal.Round(2), nil
	}
	if order.UserID == 0 {
		return decimal.Zero, ErrWalletNotSupportedForGuest
	}
	existing := order.WalletPaidAmount.Decimal.Round(2)
	if existing.GreaterThan(decimal.Zero) {
		return existing, nil
	}
	if order.TotalAmount.Decimal.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, nil
	}

	now := time.Now()
	repo := s.walletRepo.WithTx(tx)
	account, err := s.ensureAccountForUpdate(repo, order.UserID, now)
	if err != nil {
		return decimal.Zero, err
	}

	available := account.Balance.Decimal.Round(2)
	if available.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, nil
	}
	deduct := available
	if deduct.GreaterThan(order.TotalAmount.Decimal) {
		deduct = order.TotalAmount.Decimal.Round(2)
	}
	if deduct.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, nil
	}

	reference := buildOrderWalletReference(order.ID, constants.WalletTxnTypeOrderPay)
	exists, err := repo.GetTransactionByReference(reference)
	if err != nil {
		return decimal.Zero, err
	}
	if exists != nil {
		return exists.Amount.Decimal.Round(2), nil
	}

	before := account.Balance.Decimal.Round(2)
	after := before.Sub(deduct).Round(2)
	if after.LessThan(decimal.Zero) {
		return decimal.Zero, ErrWalletInsufficientBalance
	}
	account.Balance = models.NewMoneyFromDecimal(after)
	account.UpdatedAt = now
	if err := repo.UpdateAccount(account); err != nil {
		return decimal.Zero, ErrWalletAccountUpdateFailed
	}

	txn := &models.WalletTransaction{
		UserID:        order.UserID,
		OrderID:       &order.ID,
		Type:          constants.WalletTxnTypeOrderPay,
		Direction:     constants.WalletTxnDirectionOut,
		Amount:        models.NewMoneyFromDecimal(deduct),
		BalanceBefore: models.NewMoneyFromDecimal(before),
		BalanceAfter:  models.NewMoneyFromDecimal(after),
		Currency:      normalizeWalletCurrency(order.Currency),
		Reference:     reference,
		Remark:        "??????",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.CreateTransaction(txn); err != nil {
		return decimal.Zero, ErrWalletTransactionCreateFailed
	}

	onlineAmount := normalizeOrderAmount(order.TotalAmount.Decimal.Sub(deduct))
	if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"wallet_paid_amount": models.NewMoneyFromDecimal(deduct),
		"online_paid_amount": models.NewMoneyFromDecimal(onlineAmount),
		"updated_at":         now,
	}).Error; err != nil {
		return decimal.Zero, ErrOrderUpdateFailed
	}
	order.WalletPaidAmount = models.NewMoneyFromDecimal(deduct)
	order.OnlinePaidAmount = models.NewMoneyFromDecimal(onlineAmount)
	order.UpdatedAt = now
	return deduct, nil
}

// ReleaseOrderBalance ???????????????,??????
func (s *WalletService) ReleaseOrderBalance(tx *gorm.DB, order *models.Order, txnType string, remark string) (decimal.Decimal, error) {
	if tx == nil {
		return decimal.Zero, ErrOrderUpdateFailed
	}
	if order == nil || order.UserID == 0 {
		return decimal.Zero, nil
	}
	amount := order.WalletPaidAmount.Decimal.Round(2)
	if amount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, nil
	}
	now := time.Now()
	reference := buildOrderWalletReference(order.ID, txnType)
	repo := s.walletRepo.WithTx(tx)

	exists, err := repo.GetTransactionByReference(reference)
	if err != nil {
		return decimal.Zero, err
	}
	if exists != nil {
		return exists.Amount.Decimal.Round(2), nil
	}

	result := tx.Model(&models.Order{}).Where("id = ? AND wallet_paid_amount > 0", order.ID).Updates(map[string]interface{}{
		"wallet_paid_amount": models.NewMoneyFromDecimal(decimal.Zero),
		"online_paid_amount": models.NewMoneyFromDecimal(order.TotalAmount.Decimal.Round(2)),
		"updated_at":         now,
	})
	if result.Error != nil {
		return decimal.Zero, ErrOrderUpdateFailed
	}
	if result.RowsAffected == 0 {
		return decimal.Zero, nil
	}

	account, err := s.ensureAccountForUpdate(repo, order.UserID, now)
	if err != nil {
		return decimal.Zero, err
	}
	before := account.Balance.Decimal.Round(2)
	after := before.Add(amount).Round(2)
	account.Balance = models.NewMoneyFromDecimal(after)
	account.UpdatedAt = now
	if err := repo.UpdateAccount(account); err != nil {
		return decimal.Zero, ErrWalletAccountUpdateFailed
	}

	txn := &models.WalletTransaction{
		UserID:        order.UserID,
		OrderID:       &order.ID,
		Type:          txnType,
		Direction:     constants.WalletTxnDirectionIn,
		Amount:        models.NewMoneyFromDecimal(amount),
		BalanceBefore: models.NewMoneyFromDecimal(before),
		BalanceAfter:  models.NewMoneyFromDecimal(after),
		Currency:      normalizeWalletCurrency(order.Currency),
		Reference:     reference,
		Remark:        cleanWalletRemark(remark, "??????"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.CreateTransaction(txn); err != nil {
		return decimal.Zero, ErrWalletTransactionCreateFailed
	}

	order.WalletPaidAmount = models.NewMoneyFromDecimal(decimal.Zero)
	order.OnlinePaidAmount = models.NewMoneyFromDecimal(order.TotalAmount.Decimal.Round(2))
	order.UpdatedAt = now
	return amount, nil
}

// ApplyRechargePayment ?????????????????
func (s *WalletService) ApplyRechargePayment(tx *gorm.DB, recharge *models.WalletRechargeOrder) (*models.WalletTransaction, error) {
	if tx == nil {
		return nil, ErrWalletRechargeStatusInvalid
	}
	if recharge == nil || recharge.ID == 0 || recharge.UserID == 0 {
		return nil, ErrWalletRechargeNotFound
	}
	amount := recharge.Amount.Decimal.Round(2)
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrWalletInvalidAmount
	}

	reference := fmt.Sprintf("recharge:%d:success", recharge.ID)
	repo := s.walletRepo.WithTx(tx)
	exists, err := repo.GetTransactionByReference(reference)
	if err != nil {
		return nil, err
	}
	if exists != nil {
		return exists, nil
	}

	now := time.Now()
	account, err := s.ensureAccountForUpdate(repo, recharge.UserID, now)
	if err != nil {
		return nil, err
	}
	before := account.Balance.Decimal.Round(2)
	after := before.Add(amount).Round(2)
	account.Balance = models.NewMoneyFromDecimal(after)
	account.UpdatedAt = now
	if err := repo.UpdateAccount(account); err != nil {
		return nil, ErrWalletAccountUpdateFailed
	}

	txn := &models.WalletTransaction{
		UserID:        recharge.UserID,
		OrderID:       nil,
		Type:          constants.WalletTxnTypeRecharge,
		Direction:     constants.WalletTxnDirectionIn,
		Amount:        models.NewMoneyFromDecimal(amount),
		BalanceBefore: models.NewMoneyFromDecimal(before),
		BalanceAfter:  models.NewMoneyFromDecimal(after),
		Currency:      normalizeWalletCurrency(recharge.Currency),
		Reference:     reference,
		Remark:        cleanWalletRemark(recharge.Remark, "??????"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.CreateTransaction(txn); err != nil {
		return nil, ErrWalletTransactionCreateFailed
	}
	return txn, nil
}

// CreditInTx ????????????????????
func (s *WalletService) CreditInTx(tx *gorm.DB, input WalletCreditInput) (*models.WalletAccount, *models.WalletTransaction, error) {
	if tx == nil {
		return nil, nil, ErrOrderUpdateFailed
	}
	if input.UserID == 0 {
		return nil, nil, ErrWalletAccountNotFound
	}
	amount := input.Amount.Decimal.Round(2)
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, nil, ErrWalletInvalidAmount
	}
	reference := strings.TrimSpace(input.Reference)
	if reference == "" {
		return nil, nil, ErrWalletTransactionCreateFailed
	}
	txnType := strings.TrimSpace(input.TxnType)
	if txnType == "" {
		txnType = constants.WalletTxnTypeRecharge
	}
	remark := cleanWalletRemark(input.Remark, "????")
	now := time.Now()
	repo := s.walletRepo.WithTx(tx)

	exists, err := repo.GetTransactionByReference(reference)
	if err != nil {
		return nil, nil, err
	}
	if exists != nil {
		account, accountErr := repo.GetAccountByUserID(input.UserID)
		if accountErr != nil {
			return nil, nil, accountErr
		}
		if account == nil {
			account, accountErr = s.ensureAccountForUpdate(repo, input.UserID, now)
			if accountErr != nil {
				return nil, nil, accountErr
			}
		}
		return account, exists, nil
	}

	account, err := s.ensureAccountForUpdate(repo, input.UserID, now)
	if err != nil {
		return nil, nil, err
	}
	before := account.Balance.Decimal.Round(2)
	after := before.Add(amount).Round(2)
	account.Balance = models.NewMoneyFromDecimal(after)
	account.UpdatedAt = now
	if err := repo.UpdateAccount(account); err != nil {
		return nil, nil, ErrWalletAccountUpdateFailed
	}

	txn := &models.WalletTransaction{
		UserID:        input.UserID,
		OrderID:       input.OrderID,
		Type:          txnType,
		Direction:     constants.WalletTxnDirectionIn,
		Amount:        models.NewMoneyFromDecimal(amount),
		BalanceBefore: models.NewMoneyFromDecimal(before),
		BalanceAfter:  models.NewMoneyFromDecimal(after),
		Currency:      normalizeWalletCurrency(input.Currency),
		Reference:     reference,
		Remark:        remark,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.CreateTransaction(txn); err != nil {
		return nil, nil, ErrWalletTransactionCreateFailed
	}
	return account, txn, nil
}

func (s *WalletService) changeBalance(userID uint, delta decimal.Decimal, txnType string, orderID *uint, reference, remark, currency string) (*models.WalletAccount, *models.WalletTransaction, error) {
	var accountResult *models.WalletAccount
	var txnResult *models.WalletTransaction
	if err := s.walletRepo.Transaction(func(tx *gorm.DB) error {
		repo := s.walletRepo.WithTx(tx)
		now := time.Now()
		account, err := s.ensureAccountForUpdate(repo, userID, now)
		if err != nil {
			return err
		}

		before := account.Balance.Decimal.Round(2)
		after := before.Add(delta).Round(2)
		if after.LessThan(decimal.Zero) {
			return ErrWalletInsufficientBalance
		}
		direction := constants.WalletTxnDirectionIn
		amount := delta.Round(2)
		if delta.LessThan(decimal.Zero) {
			direction = constants.WalletTxnDirectionOut
			amount = delta.Abs().Round(2)
		}

		account.Balance = models.NewMoneyFromDecimal(after)
		account.UpdatedAt = now
		if err := repo.UpdateAccount(account); err != nil {
			return ErrWalletAccountUpdateFailed
		}

		txn := &models.WalletTransaction{
			UserID:        userID,
			OrderID:       orderID,
			Type:          txnType,
			Direction:     direction,
			Amount:        models.NewMoneyFromDecimal(amount),
			BalanceBefore: models.NewMoneyFromDecimal(before),
			BalanceAfter:  models.NewMoneyFromDecimal(after),
			Currency:      normalizeWalletCurrency(currency),
			Reference:     strings.TrimSpace(reference),
			Remark:        remark,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := repo.CreateTransaction(txn); err != nil {
			return ErrWalletTransactionCreateFailed
		}

		accountResult = account
		txnResult = txn
		return nil
	}); err != nil {
		return nil, nil, err
	}
	return accountResult, txnResult, nil
}

func (s *WalletService) getOrCreateAccount(userID uint) (*models.WalletAccount, error) {
	account, err := s.walletRepo.GetAccountByUserID(userID)
	if err != nil {
		return nil, err
	}
	if account != nil {
		return account, nil
	}
	now := time.Now()
	account = &models.WalletAccount{
		UserID:    userID,
		Balance:   models.NewMoneyFromDecimal(decimal.Zero),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.walletRepo.CreateAccount(account); err != nil {
		created, queryErr := s.walletRepo.GetAccountByUserID(userID)
		if queryErr == nil && created != nil {
			return created, nil
		}
		return nil, ErrWalletAccountCreateFailed
	}
	return account, nil
}

func (s *WalletService) ensureAccountForUpdate(repo *repository.GormWalletRepository, userID uint, now time.Time) (*models.WalletAccount, error) {
	account, err := repo.GetAccountByUserIDForUpdate(userID)
	if err != nil {
		return nil, err
	}
	if account != nil {
		return account, nil
	}
	account = &models.WalletAccount{
		UserID:    userID,
		Balance:   models.NewMoneyFromDecimal(decimal.Zero),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateAccount(account); err != nil {
		created, queryErr := repo.GetAccountByUserIDForUpdate(userID)
		if queryErr == nil && created != nil {
			return created, nil
		}
		return nil, ErrWalletAccountCreateFailed
	}
	return account, nil
}

func normalizeWalletCurrency(currency string) string {
	normalized := strings.ToUpper(strings.TrimSpace(currency))
	if normalized == "" {
		return walletDefaultCurrency
	}
	return normalized
}

func cleanWalletRemark(raw string, fallback string) string {
	remark := strings.TrimSpace(raw)
	if remark == "" {
		return fallback
	}
	return remark
}

func buildOrderWalletReference(orderID uint, action string) string {
	action = strings.TrimSpace(action)
	if action == "" {
		action = "wallet"
	}
	return fmt.Sprintf("order:%d:%s", orderID, action)
}

func buildWalletReference(prefix string, id uint) string {
	normalized := strings.TrimSpace(prefix)
	if normalized == "" {
		normalized = "wallet"
	}
	return fmt.Sprintf("%s:%d:%d", normalized, id, time.Now().UnixNano())
}
