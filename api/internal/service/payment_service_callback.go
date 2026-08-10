package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/queue"
	"github.com/dujiao-next/internal/repository"

	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type notificationOrderItemCounts struct {
	Total    int
	Auto     int
	Manual   int
	Upstream int
}

func (s *PaymentService) HandleCallback(input PaymentCallbackInput) (*models.Payment, error) {
	if input.PaymentID == 0 {
		return nil, ErrPaymentInvalid
	}
	status := normalizePaymentStatus(input.Status)
	if !isPaymentStatusValid(status) {
		return nil, ErrPaymentStatusInvalid
	}

	log := paymentLogger(
		"payment_id", input.PaymentID,
		"target_status", status,
		"callback_channel_id", input.ChannelID,
		"callback_order_no", strings.TrimSpace(input.OrderNo),
		"callback_provider_ref", strings.TrimSpace(input.ProviderRef),
		"callback_currency", strings.ToUpper(strings.TrimSpace(input.Currency)),
		"callback_amount", input.Amount.String(),
	)
	log.Infow("payment_callback_received")

	payment, err := s.paymentRepo.GetByID(input.PaymentID)
	if err != nil {
		log.Errorw("payment_callback_payment_fetch_failed", "error", err)
		return nil, ErrPaymentUpdateFailed
	}
	if payment == nil {
		log.Warnw("payment_callback_payment_not_found")
		return nil, ErrPaymentNotFound
	}
	if payment.OrderID == 0 {
		log.Infow("payment_callback_wallet_recharge_flow")
		return s.handleWalletRechargeCallback(payment, status, input)
	}

	order, err := s.orderRepo.GetByID(payment.OrderID)
	if err != nil {
		log.Errorw("payment_callback_order_fetch_failed", "order_id", payment.OrderID, "error", err)
		return nil, ErrOrderFetchFailed
	}
	if order == nil {
		log.Warnw("payment_callback_order_not_found", "order_id", payment.OrderID)
		return nil, ErrOrderNotFound
	}

	if input.ChannelID != 0 && input.ChannelID != payment.ChannelID {
		log.Warnw("payment_callback_channel_mismatch",
			"stored_channel_id", payment.ChannelID,
			"callback_channel_id", input.ChannelID,
		)
		return nil, ErrPaymentInvalid
	}
	if !matchesBusinessOrderNo(input.OrderNo, order.OrderNo, payment) {
		log.Warnw("payment_callback_order_no_mismatch",
			"stored_order_no", order.OrderNo,
			"stored_gateway_order_no", payment.GatewayOrderNo,
			"callback_order_no", input.OrderNo,
		)
		return nil, ErrPaymentInvalid
	}
	if input.Currency != "" && strings.ToUpper(strings.TrimSpace(input.Currency)) != strings.ToUpper(strings.TrimSpace(payment.Currency)) {
		log.Warnw("payment_callback_currency_mismatch",
			"stored_currency", payment.Currency,
			"callback_currency", input.Currency,
		)
		return nil, ErrPaymentCurrencyMismatch
	}
	if !input.Amount.Decimal.IsZero() && input.Amount.Decimal.Cmp(payment.Amount.Decimal) != 0 {
		log.Warnw("payment_callback_amount_mismatch",
			"stored_amount", payment.Amount.String(),
			"callback_amount", input.Amount.String(),
		)
		return nil, ErrPaymentAmountMismatch
	}

	// ????:??????????
	if payment.Status == constants.PaymentStatusSuccess {
		log.Infow("payment_callback_idempotent_success",
			"current_status", payment.Status,
		)
		return s.updateCallbackMeta(payment, constants.PaymentStatusSuccess, input)
	}
	if payment.Status == status {
		log.Infow("payment_callback_idempotent_same_status",
			"current_status", payment.Status,
		)
		return s.updateCallbackMeta(payment, status, input)
	}

	previousStatus := payment.Status
	now := time.Now()
	updated, orderPaid, err := s.applyPaymentUpdate(payment, order, status, input, now)
	if err != nil {
		log.Errorw("payment_callback_apply_failed",
			"order_id", order.ID,
			"order_no", order.OrderNo,
			"current_status", payment.Status,
			"error", err,
		)
		return nil, err
	}
	if orderPaid {
		s.enqueueOrderPaidAsync(order, updated, log)
	}
	log.Infow("payment_callback_processed",
		"order_id", order.ID,
		"order_no", order.OrderNo,
		"previous_status", previousStatus,
		"new_status", updated.Status,
		"order_paid", orderPaid,
	)
	return updated, nil
}

func (s *PaymentService) handleWalletRechargeCallback(payment *models.Payment, status string, input PaymentCallbackInput) (*models.Payment, error) {
	log := paymentLogger(
		"payment_id", payment.ID,
		"recharge_no", strings.TrimSpace(input.OrderNo),
		"target_status", status,
		"callback_channel_id", input.ChannelID,
		"callback_provider_ref", strings.TrimSpace(input.ProviderRef),
		"callback_currency", strings.ToUpper(strings.TrimSpace(input.Currency)),
		"callback_amount", input.Amount.String(),
	)
	if s.walletRepo == nil {
		log.Errorw("wallet_recharge_callback_wallet_repo_nil")
		return nil, ErrPaymentUpdateFailed
	}
	recharge, err := s.walletRepo.GetRechargeOrderByPaymentID(payment.ID)
	if err != nil {
		log.Errorw("wallet_recharge_callback_recharge_fetch_failed", "error", err)
		return nil, ErrPaymentUpdateFailed
	}
	if recharge == nil {
		log.Warnw("wallet_recharge_callback_recharge_not_found")
		return nil, ErrWalletRechargeNotFound
	}

	if input.ChannelID != 0 && input.ChannelID != payment.ChannelID {
		log.Warnw("wallet_recharge_callback_channel_mismatch",
			"stored_channel_id", payment.ChannelID,
			"callback_channel_id", input.ChannelID,
		)
		return nil, ErrPaymentInvalid
	}
	if !matchesBusinessOrderNo(input.OrderNo, recharge.RechargeNo, payment) {
		log.Warnw("wallet_recharge_callback_order_no_mismatch",
			"stored_recharge_no", recharge.RechargeNo,
			"stored_gateway_order_no", payment.GatewayOrderNo,
			"callback_order_no", input.OrderNo,
		)
		return nil, ErrPaymentInvalid
	}
	if input.Currency != "" && strings.ToUpper(strings.TrimSpace(input.Currency)) != strings.ToUpper(strings.TrimSpace(payment.Currency)) {
		log.Warnw("wallet_recharge_callback_currency_mismatch",
			"stored_currency", payment.Currency,
			"callback_currency", input.Currency,
		)
		return nil, ErrPaymentCurrencyMismatch
	}
	if !input.Amount.Decimal.IsZero() && input.Amount.Decimal.Cmp(payment.Amount.Decimal) != 0 {
		log.Warnw("wallet_recharge_callback_amount_mismatch",
			"stored_amount", payment.Amount.String(),
			"callback_amount", input.Amount.String(),
		)
		return nil, ErrPaymentAmountMismatch
	}

	// ????:??????????????
	if payment.Status == constants.PaymentStatusSuccess {
		log.Infow("wallet_recharge_callback_idempotent_success",
			"current_status", payment.Status,
		)
		return s.updateCallbackMeta(payment, constants.PaymentStatusSuccess, input)
	}
	if payment.Status == status {
		log.Infow("wallet_recharge_callback_idempotent_same_status",
			"current_status", payment.Status,
		)
		return s.updateCallbackMeta(payment, status, input)
	}
	if !canApplyWalletRechargeCallback(payment.Status, recharge.Status, status) {
		log.Infow("wallet_recharge_callback_ignored_terminal_transition",
			"current_payment_status", payment.Status,
			"current_recharge_status", recharge.Status,
			"target_status", status,
		)
		return s.updateCallbackMeta(payment, payment.Status, input)
	}

	now := time.Now()
	updated, err := s.applyWalletRechargePaymentUpdate(payment, status, input, now)
	if err != nil {
		log.Errorw("wallet_recharge_callback_apply_failed", "error", err)
		return nil, err
	}
	log.Infow("wallet_recharge_callback_processed",
		"new_status", updated.Status,
	)
	if updated.Status == constants.PaymentStatusSuccess {
		s.enqueueWalletRechargeSuccessAsync(recharge, updated, log)
		s.enqueueWalletRechargeBotNotifyAsync(recharge, log)
	}
	return updated, nil
}

func (s *PaymentService) applyWalletRechargePaymentUpdate(payment *models.Payment, status string, input PaymentCallbackInput, now time.Time) (*models.Payment, error) {
	paymentVal := payment

	switch status {
	case constants.PaymentStatusSuccess:
		paidAt := now
		if input.PaidAt != nil {
			paidAt = *input.PaidAt
		}
		payment.PaidAt = &paidAt
	case constants.PaymentStatusExpired:
		payment.ExpiredAt = &now
	}

	payment.Status = status
	payment.CallbackAt = &now
	payment.UpdatedAt = now
	if input.ProviderRef != "" {
		payment.ProviderRef = input.ProviderRef
	}
	if input.Payload != nil {
		payment.ProviderPayload = input.Payload
	}

	err := s.paymentRepo.Transaction(func(tx *gorm.DB) error {
		paymentRepo := s.paymentRepo.WithTx(tx)
		rechargeRepo := s.walletRepo.WithTx(tx)

		if err := paymentRepo.Update(payment); err != nil {
			return ErrPaymentUpdateFailed
		}
		recharge, err := rechargeRepo.GetRechargeOrderByPaymentIDForUpdate(payment.ID)
		if err != nil {
			return ErrPaymentUpdateFailed
		}
		if recharge == nil {
			return ErrWalletRechargeNotFound
		}
		if recharge.Status == constants.WalletRechargeStatusSuccess {
			return nil
		}

		switch status {
		case constants.PaymentStatusSuccess:
			if s.walletSvc == nil {
				return ErrWalletAccountNotFound
			}
			if _, err := s.walletSvc.ApplyRechargePayment(tx, recharge); err != nil {
				return err
			}
			recharge.Status = constants.WalletRechargeStatusSuccess
			paidAt := now
			if payment.PaidAt != nil {
				paidAt = *payment.PaidAt
			}
			recharge.PaidAt = &paidAt
		case constants.PaymentStatusFailed:
			recharge.Status = constants.WalletRechargeStatusFailed
		case constants.PaymentStatusExpired:
			recharge.Status = constants.WalletRechargeStatusExpired
		default:
			recharge.Status = constants.WalletRechargeStatusPending
		}
		recharge.UpdatedAt = now
		if err := rechargeRepo.UpdateRechargeOrder(recharge); err != nil {
			return ErrPaymentUpdateFailed
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// ???????????????(?????)
	if status == constants.PaymentStatusSuccess && s.memberLevelSvc != nil {
		if recharge, _ := s.walletRepo.GetRechargeOrderByPaymentID(payment.ID); recharge != nil &&
			recharge.Status == constants.WalletRechargeStatusSuccess && recharge.UserID > 0 {
			if err := s.memberLevelSvc.OnRechargeCompleted(recharge.UserID, recharge.Amount.Decimal); err != nil {
				paymentLogger().Warnw("member_level_recharge_completed_failed",
					"payment_id", payment.ID,
					"user_id", recharge.UserID,
					"amount", recharge.Amount.Decimal.String(),
					"error", err,
				)
			}
		}
	}

	return paymentVal, nil
}

func canApplyWalletRechargeCallback(paymentStatus string, rechargeStatus string, targetStatus string) bool {
	// ??????????(??????????????)?
	if targetStatus == constants.PaymentStatusSuccess {
		return true
	}
	// ??????????????,?? expired/failed/success ????????
	if paymentStatus == constants.PaymentStatusSuccess || rechargeStatus == constants.WalletRechargeStatusSuccess {
		return false
	}
	if paymentStatus == constants.PaymentStatusFailed || rechargeStatus == constants.WalletRechargeStatusFailed {
		return false
	}
	if paymentStatus == constants.PaymentStatusExpired || rechargeStatus == constants.WalletRechargeStatusExpired {
		return false
	}
	return true
}

func (s *PaymentService) updateCallbackMeta(payment *models.Payment, status string, input PaymentCallbackInput) (*models.Payment, error) {
	updated := false
	if input.ProviderRef != "" && payment.ProviderRef == "" {
		payment.ProviderRef = input.ProviderRef
		updated = true
	}
	if input.Payload != nil {
		payment.ProviderPayload = input.Payload
		updated = true
	}
	if status != "" && payment.Status != status {
		payment.Status = status
		updated = true
	}
	if payment.Status == constants.PaymentStatusSuccess && payment.PaidAt == nil && input.PaidAt != nil {
		payment.PaidAt = input.PaidAt
		updated = true
	}
	if updated {
		now := time.Now()
		payment.CallbackAt = &now
		payment.UpdatedAt = now
		if err := s.paymentRepo.Update(payment); err != nil {
			return nil, ErrPaymentUpdateFailed
		}
	}
	return payment, nil
}

func (s *PaymentService) applyPaymentUpdate(payment *models.Payment, order *models.Order, status string, input PaymentCallbackInput, now time.Time) (*models.Payment, bool, error) {
	returnVal := payment
	orderPaid := false

	switch status {
	case constants.PaymentStatusSuccess:
		paidAt := now
		if input.PaidAt != nil {
			paidAt = *input.PaidAt
		}
		payment.PaidAt = &paidAt
	case constants.PaymentStatusExpired:
		payment.ExpiredAt = &now
	}

	payment.Status = status
	payment.CallbackAt = &now
	payment.UpdatedAt = now
	if input.ProviderRef != "" {
		payment.ProviderRef = input.ProviderRef
	}
	if input.Payload != nil {
		payment.ProviderPayload = input.Payload
	}

	err := s.paymentRepo.Transaction(func(tx *gorm.DB) error {
		paymentRepo := s.paymentRepo.WithTx(tx)

		if err := paymentRepo.Update(payment); err != nil {
			return ErrPaymentUpdateFailed
		}

		if status == constants.PaymentStatusSuccess && order.Status != constants.OrderStatusPaid {
			if err := s.markOrderPaid(tx, order, now); err != nil {
				return err
			}
			orderPaid = true
		}
		if (status == constants.PaymentStatusFailed || status == constants.PaymentStatusExpired) && order.Status == constants.OrderStatusPendingPayment && s.walletSvc != nil {
			if _, err := s.walletSvc.ReleaseOrderBalance(tx, order, constants.WalletTxnTypeOrderRefund, "??????,????"); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return returnVal, orderPaid, nil
}

// markOrderPaid ??????????????????
func (s *PaymentService) markOrderPaid(tx *gorm.DB, order *models.Order, now time.Time) error {
	if order == nil {
		return ErrOrderNotFound
	}
	if !isTransitionAllowed(order.Status, constants.OrderStatusPaid) {
		return ErrOrderStatusInvalid
	}
	orderRepo := s.orderRepo.WithTx(tx)
	productRepo := s.productRepo.WithTx(tx)
	var productSKURepo repository.ProductSKURepository
	if s.productSKURepo != nil {
		productSKURepo = s.productSKURepo.WithTx(tx)
	}

	onlineAmount := normalizeOrderAmount(order.TotalAmount.Decimal.Sub(order.WalletPaidAmount.Decimal))
	orderUpdates := map[string]interface{}{
		"paid_at":            now,
		"online_paid_amount": models.NewMoneyFromDecimal(onlineAmount),
		"updated_at":         now,
	}
	if err := orderRepo.UpdateStatus(order.ID, constants.OrderStatusPaid, orderUpdates); err != nil {
		return ErrOrderUpdateFailed
	}
	order.Status = constants.OrderStatusPaid
	order.PaidAt = &now
	order.OnlinePaidAmount = models.NewMoneyFromDecimal(onlineAmount)
	order.UpdatedAt = now

	if len(order.Children) > 0 {
		for idx := range order.Children {
			child := &order.Children[idx]
			childStatus := constants.OrderStatusPaid
			if shouldMarkFulfilling(child) {
				childStatus = constants.OrderStatusFulfilling
			}
			if err := orderRepo.UpdateStatus(child.ID, childStatus, map[string]interface{}{
				"paid_at":    now,
				"updated_at": now,
			}); err != nil {
				return ErrOrderUpdateFailed
			}
			if err := consumeManualStockByItems(productRepo, productSKURepo, child.Items); err != nil {
				return err
			}
			child.Status = childStatus
			child.PaidAt = &now
			child.UpdatedAt = now
		}
		parentStatus := calcParentStatus(order.Children, constants.OrderStatusPaid)
		if parentStatus != "" && parentStatus != constants.OrderStatusPaid {
			if err := orderRepo.UpdateStatus(order.ID, parentStatus, map[string]interface{}{
				"online_paid_amount": models.NewMoneyFromDecimal(onlineAmount),
				"updated_at":         now,
			}); err != nil {
				return ErrOrderUpdateFailed
			}
			order.Status = parentStatus
		}
		return nil
	}

	if err := consumeManualStockByItems(productRepo, productSKURepo, order.Items); err != nil {
		return err
	}
	return nil
}

func (s *PaymentService) enqueueOrderPaidAsync(order *models.Order, payment *models.Payment, log *zap.SugaredLogger) {
	if order == nil {
		return
	}
	if s.affiliateSvc != nil {
		if err := s.affiliateSvc.HandleOrderPaid(order.ID); err != nil {
			log.Warnw("affiliate_handle_order_paid_failed",
				"order_id", order.ID,
				"order_no", order.OrderNo,
				"error", err,
			)
		}
	}
	if s.queueClient != nil {
		if _, err := enqueueOrderStatusEmailTaskIfEligible(s.orderRepo, s.queueClient, s.settingService, s.defaultEmailConfig, order.ID, constants.OrderStatusPaid); err != nil {
			log.Warnw("payment_enqueue_status_email_failed",
				"order_id", order.ID,
				"order_no", order.OrderNo,
				"status", constants.OrderStatusPaid,
				"error", err,
			)
		}
	}
	s.enqueueOrderPaidNotificationAsync(order, payment, log)
	s.enqueueOrderPaidBotNotifyAsync(order, log)

	// ?????????????????
	if s.memberLevelSvc != nil && order.UserID > 0 {
		if err := s.memberLevelSvc.OnOrderPaid(order.UserID, order.TotalAmount.Decimal); err != nil {
			log.Warnw("member_level_order_paid_failed",
				"order_id", order.ID,
				"user_id", order.UserID,
				"amount", order.TotalAmount.Decimal.String(),
				"error", err,
			)
		}
	}

	// ????????:?????????,???????
	s.processAutoFulfillment(order, log)

	if s.queueClient == nil {
		return
	}
	// ????:?????????????????
	s.enqueueProcurementAsync(order, log)
	// B ?:?????????????????
	s.enqueueDownstreamCallbackAsync(order, log)
}

// processAutoFulfillment ????????
// ??????,?????????;??????,????????
func (s *PaymentService) processAutoFulfillment(order *models.Order, log *zap.SugaredLogger) {
	if s.fulfillmentSvc == nil {
		log.Warnw("payment_auto_fulfillment_skip_service_nil",
			"order_id", order.ID,
			"order_no", order.OrderNo,
		)
		return
	}

	queueEnabled := s.queueClient != nil && s.queueClient.Enabled()

	processOrder := func(targetOrder *models.Order) {
		if targetOrder.Status == constants.OrderStatusFulfilling {
			s.enqueueManualFulfillmentPendingAsync(targetOrder, order, log)
		}
		if !shouldAutoFulfill(targetOrder) {
			return
		}
		if queueEnabled {
			// ????:????
			if err := s.queueClient.EnqueueOrderAutoFulfill(queue.OrderAutoFulfillPayload{
				OrderID: targetOrder.ID,
			}, asynq.MaxRetry(3)); err != nil {
				log.Warnw("payment_enqueue_auto_fulfill_failed",
					"order_id", order.ID,
					"target_order_id", targetOrder.ID,
					"order_no", order.OrderNo,
					"error", err,
				)
			}
		} else {
			// ????:????????
			if _, err := s.fulfillmentSvc.CreateAuto(targetOrder.ID); err != nil {
				switch {
				case errors.Is(err, ErrFulfillmentExists):
					log.Debugw("payment_auto_fulfillment_skip_exists",
						"order_id", targetOrder.ID,
						"order_no", order.OrderNo,
					)
				case errors.Is(err, ErrFulfillmentNotAuto):
					log.Debugw("payment_auto_fulfillment_skip_not_auto",
						"order_id", targetOrder.ID,
						"order_no", order.OrderNo,
					)
				case errors.Is(err, ErrOrderStatusInvalid):
					log.Debugw("payment_auto_fulfillment_skip_invalid_status",
						"order_id", targetOrder.ID,
						"order_no", order.OrderNo,
					)
				case errors.Is(err, ErrOrderNotFound):
					log.Debugw("payment_auto_fulfillment_skip_order_not_found",
						"order_id", targetOrder.ID,
						"order_no", order.OrderNo,
					)
				default:
					log.Warnw("payment_auto_fulfillment_failed",
						"order_id", targetOrder.ID,
						"order_no", order.OrderNo,
						"error", err,
					)
				}
			} else {
				log.Infow("payment_auto_fulfillment_sync_success",
					"order_id", targetOrder.ID,
					"order_no", order.OrderNo,
				)
			}
		}
	}

	if len(order.Children) > 0 {
		for _, child := range order.Children {
			processOrder(&child)
		}
	} else {
		processOrder(order)
	}
}

// enqueueProcurementAsync ??????????????,?????
func (s *PaymentService) enqueueProcurementAsync(order *models.Order, log *zap.SugaredLogger) {
	if s.procurementSvc == nil || order == nil {
		return
	}
	if err := s.procurementSvc.CreateForOrder(order.ID); err != nil {
		if !errors.Is(err, ErrProcurementExists) {
			log.Warnw("payment_enqueue_procurement_failed",
				"order_id", order.ID,
				"order_no", order.OrderNo,
				"error", err,
			)
		}
	}
}

// enqueueDownstreamCallbackAsync B ?:???? A ???????
func (s *PaymentService) enqueueDownstreamCallbackAsync(order *models.Order, log *zap.SugaredLogger) {
	if s.downstreamCallbackSvc == nil || order == nil {
		return
	}
	s.downstreamCallbackSvc.EnqueueCallback(order.ID)
}

func (s *PaymentService) enqueueOrderPaidNotificationAsync(order *models.Order, payment *models.Payment, log *zap.SugaredLogger) {
	if s.notificationSvc == nil || order == nil {
		return
	}
	payload := s.buildOrderNotificationPayload(order, payment)
	if err := s.notificationSvc.Enqueue(NotificationEnqueueInput{
		EventType: constants.NotificationEventOrderPaidSuccess,
		BizType:   constants.NotificationBizTypeOrder,
		BizID:     order.ID,
		Data:      payload,
	}); err != nil {
		log.Warnw("notification_enqueue_order_paid_failed",
			"order_id", order.ID,
			"order_no", order.OrderNo,
			"error", err,
		)
	}
}

func (s *PaymentService) enqueueWalletRechargeSuccessAsync(recharge *models.WalletRechargeOrder, payment *models.Payment, log *zap.SugaredLogger) {
	if s.notificationSvc == nil || recharge == nil {
		return
	}
	payload := s.buildWalletRechargeNotificationPayload(recharge, payment)
	if err := s.notificationSvc.Enqueue(NotificationEnqueueInput{
		EventType: constants.NotificationEventWalletRechargeSuccess,
		BizType:   constants.NotificationBizTypeWalletRecharge,
		BizID:     recharge.ID,
		Data:      payload,
	}); err != nil {
		log.Warnw("notification_enqueue_wallet_recharge_failed",
			"recharge_id", recharge.ID,
			"recharge_no", recharge.RechargeNo,
			"error", err,
		)
	}
}

func (s *PaymentService) enqueueOrderPaidBotNotifyAsync(order *models.Order, log *zap.SugaredLogger) {
	if s.queueClient == nil || order == nil || order.UserID == 0 || s.userOAuthIdentityRepo == nil {
		return
	}

	identity, err := s.userOAuthIdentityRepo.GetByUserProvider(order.UserID, constants.UserOAuthProviderTelegram)
	if err != nil {
		log.Warnw("order_paid_notify_bot_fetch_identity_failed",
			"order_id", order.ID,
			"user_id", order.UserID,
			"error", err,
		)
		return
	}
	if identity == nil || strings.TrimSpace(identity.ProviderUserID) == "" {
		return
	}

	if err := s.queueClient.EnqueueBotNotify(queue.BotNotifyPayload{
		EventType:      queue.BotNotifyEventOrderPaid,
		OrderID:        order.ID,
		TelegramUserID: strings.TrimSpace(identity.ProviderUserID),
	}); err != nil {
		log.Warnw("order_paid_notify_bot_enqueue_failed",
			"order_id", order.ID,
			"order_no", order.OrderNo,
			"user_id", order.UserID,
			"error", err,
		)
	}
}

func (s *PaymentService) enqueueWalletRechargeBotNotifyAsync(recharge *models.WalletRechargeOrder, log *zap.SugaredLogger) {
	if s.queueClient == nil || recharge == nil || recharge.UserID == 0 || s.userOAuthIdentityRepo == nil {
		return
	}

	identity, err := s.userOAuthIdentityRepo.GetByUserProvider(recharge.UserID, constants.UserOAuthProviderTelegram)
	if err != nil {
		log.Warnw("wallet_recharge_notify_bot_fetch_identity_failed",
			"recharge_id", recharge.ID,
			"user_id", recharge.UserID,
			"error", err,
		)
		return
	}
	if identity == nil || strings.TrimSpace(identity.ProviderUserID) == "" {
		return
	}

	if err := s.queueClient.EnqueueBotNotify(queue.BotNotifyPayload{
		EventType:      queue.BotNotifyEventWalletRechargeSucceeded,
		TelegramUserID: strings.TrimSpace(identity.ProviderUserID),
		RechargeNo:     strings.TrimSpace(recharge.RechargeNo),
		Amount:         recharge.Amount.String(),
		Currency:       strings.ToUpper(strings.TrimSpace(recharge.Currency)),
	}); err != nil {
		log.Warnw("wallet_recharge_notify_bot_enqueue_failed",
			"recharge_id", recharge.ID,
			"recharge_no", recharge.RechargeNo,
			"user_id", recharge.UserID,
			"error", err,
		)
	}
}

func (s *PaymentService) enqueueManualFulfillmentPendingAsync(order *models.Order, parent *models.Order, log *zap.SugaredLogger) {
	if s.notificationSvc == nil || order == nil {
		return
	}
	payload := s.buildManualFulfillmentNotificationPayload(order, parent)
	if err := s.notificationSvc.Enqueue(NotificationEnqueueInput{
		EventType: constants.NotificationEventManualFulfillmentPending,
		BizType:   constants.NotificationBizTypeOrder,
		BizID:     order.ID,
		Data:      payload,
	}); err != nil {
		log.Warnw("notification_enqueue_manual_pending_failed",
			"order_id", order.ID,
			"order_no", order.OrderNo,
			"error", err,
		)
	}
}

func (s *PaymentService) buildOrderNotificationPayload(order *models.Order, payment *models.Payment) models.JSON {
	locale := s.notificationTemplateLocale()
	customerEmail, customerLabel, customerType := s.resolveNotificationCustomer(order)
	// ??????????????????,???????????????
	fillOrderItemsFromChildren(order)
	itemsSummary, fulfillmentItemsSummary, counts := buildNotificationOrderItemSummaries(order.Items, locale)
	providerType, channelType, paymentChannel := notificationPaymentChannel(order, payment)

	payload := models.JSON{
		"order_id":                  fmt.Sprintf("%d", order.ID),
		"order_no":                  strings.TrimSpace(order.OrderNo),
		"user_id":                   fmt.Sprintf("%d", order.UserID),
		"guest_email":               strings.TrimSpace(order.GuestEmail),
		"amount":                    order.TotalAmount.String(),
		"currency":                  strings.ToUpper(strings.TrimSpace(order.Currency)),
		"order_status":              strings.TrimSpace(order.Status),
		"customer_email":            customerEmail,
		"customer_label":            customerLabel,
		"customer_type":             customerType,
		"items_summary":             itemsSummary,
		"fulfillment_items_summary": fulfillmentItemsSummary,
		"delivery_summary":          buildNotificationDeliverySummary(locale, counts),
		"item_count":                fmt.Sprintf("%d", counts.Total),
		"auto_item_count":           fmt.Sprintf("%d", counts.Auto),
		"manual_item_count":         fmt.Sprintf("%d", counts.Manual),
		"upstream_item_count":       fmt.Sprintf("%d", counts.Upstream),
		"payment_channel":           paymentChannel,
	}
	if payment != nil {
		payload["payment_id"] = fmt.Sprintf("%d", payment.ID)
	}
	if providerType != "" {
		payload["provider_type"] = providerType
	}
	if channelType != "" {
		payload["channel_type"] = channelType
	}
	return payload
}

func (s *PaymentService) buildWalletRechargeNotificationPayload(recharge *models.WalletRechargeOrder, payment *models.Payment) models.JSON {
	customerEmail, customerLabel := s.resolveUserNotificationIdentity(recharge.UserID)
	providerType := strings.TrimSpace(recharge.ProviderType)
	channelType := strings.TrimSpace(recharge.ChannelType)
	paymentChannel := providerType
	if paymentChannel != "" && channelType != "" {
		paymentChannel += "/" + channelType
	} else if paymentChannel == "" {
		paymentChannel = channelType
	}

	payload := models.JSON{
		"user_id":         fmt.Sprintf("%d", recharge.UserID),
		"recharge_id":     fmt.Sprintf("%d", recharge.ID),
		"recharge_no":     strings.TrimSpace(recharge.RechargeNo),
		"amount":          recharge.Amount.String(),
		"currency":        strings.ToUpper(strings.TrimSpace(recharge.Currency)),
		"provider_type":   providerType,
		"channel_type":    channelType,
		"payment_channel": paymentChannel,
		"customer_email":  customerEmail,
		"customer_label":  customerLabel,
	}
	if payment != nil {
		payload["payment_id"] = fmt.Sprintf("%d", payment.ID)
	}
	return payload
}

func (s *PaymentService) buildManualFulfillmentNotificationPayload(order *models.Order, parent *models.Order) models.JSON {
	payload := s.buildOrderNotificationPayload(order, nil)
	if parent != nil {
		payload["parent_order_id"] = fmt.Sprintf("%d", parent.ID)
		payload["parent_order_no"] = strings.TrimSpace(parent.OrderNo)
	}
	return payload
}

func (s *PaymentService) notificationTemplateLocale() string {
	if s == nil || s.settingService == nil {
		return constants.LocaleZhCN
	}
	setting, err := s.settingService.GetNotificationCenterSetting()
	if err != nil {
		return constants.LocaleZhCN
	}
	return normalizeNotificationLocale(setting.DefaultLocale)
}

func (s *PaymentService) resolveNotificationCustomer(order *models.Order) (string, string, string) {
	if order == nil {
		return "", "", "guest"
	}
	if order.UserID == 0 {
		guestEmail := strings.TrimSpace(order.GuestEmail)
		return guestEmail, guestEmail, "guest"
	}
	email, label := s.resolveUserNotificationIdentity(order.UserID)
	if email == "" {
		email = strings.TrimSpace(order.GuestEmail)
	}
	if label == "" {
		label = email
	}
	if label == "" {
		label = fmt.Sprintf("user#%d", order.UserID)
	}
	return email, label, "registered"
}

func (s *PaymentService) resolveUserNotificationIdentity(userID uint) (string, string) {
	if userID == 0 || s == nil || s.userRepo == nil {
		return "", ""
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return "", ""
	}
	email := strings.TrimSpace(user.Email)
	displayName := strings.TrimSpace(user.DisplayName)
	switch {
	case displayName != "" && email != "":
		return email, displayName + " <" + email + ">"
	case email != "":
		return email, email
	case displayName != "":
		return "", displayName
	default:
		return "", fmt.Sprintf("user#%d", userID)
	}
}

func notificationPaymentChannel(order *models.Order, payment *models.Payment) (string, string, string) {
	providerType := ""
	channelType := ""
	if payment != nil {
		providerType = strings.TrimSpace(payment.ProviderType)
		channelType = strings.TrimSpace(payment.ChannelType)
	}
	if providerType == "" && order != nil && order.WalletPaidAmount.Decimal.GreaterThan(decimal.Zero) {
		providerType = constants.PaymentProviderWallet
		channelType = constants.PaymentChannelTypeBalance
	}
	paymentChannel := providerType
	if paymentChannel != "" && channelType != "" {
		paymentChannel += "/" + channelType
	} else if paymentChannel == "" {
		paymentChannel = channelType
	}
	return providerType, channelType, paymentChannel
}

func buildNotificationOrderItemSummaries(items []models.OrderItem, locale string) (string, string, notificationOrderItemCounts) {
	counts := notificationOrderItemCounts{Total: len(items)}
	if len(items) == 0 {
		empty := localizedNotificationText(locale, "??????", "??????", "No item details")
		return empty, empty, counts
	}

	allLines := make([]string, 0, len(items))
	fulfillmentLines := make([]string, 0, len(items))
	for idx, item := range items {
		line := buildNotificationOrderItemLine(idx, item, locale)
		allLines = append(allLines, line)

		switch normalizeNotificationFulfillmentType(item.FulfillmentType) {
		case constants.FulfillmentTypeAuto:
			counts.Auto++
		case constants.FulfillmentTypeUpstream:
			counts.Upstream++
			fulfillmentLines = append(fulfillmentLines, line)
		default:
			counts.Manual++
			fulfillmentLines = append(fulfillmentLines, line)
		}
	}

	if len(fulfillmentLines) == 0 {
		fulfillmentLines = append(fulfillmentLines, localizedNotificationText(locale, "??????", "??????", "No manual follow-up required"))
	}
	return strings.Join(allLines, "\n"), strings.Join(fulfillmentLines, "\n"), counts
}

func buildNotificationOrderItemLine(index int, item models.OrderItem, locale string) string {
	title := resolveNotificationLocalizedJSON(item.TitleJSON, locale, constants.LocaleZhCN)
	if title == "" {
		title = localizedNotificationText(locale, "?????", "?????", "Unnamed item")
	}

	skuText := buildNotificationSKUSummary(item.SKUSnapshotJSON, locale)
	fulfillmentLabel := notificationFulfillmentLabel(locale, item.FulfillmentType)
	line := fmt.Sprintf("%d. %s", index+1, title)
	if skuText != "" {
		line += " / " + skuText
	}
	line += fmt.Sprintf(" x%d", item.Quantity)
	if fulfillmentLabel != "" {
		line += " [" + fulfillmentLabel + "]"
	}
	return line
}

func buildNotificationSKUSummary(snapshot models.JSON, locale string) string {
	if len(snapshot) == 0 {
		return ""
	}
	specText := notificationInterfaceText(snapshot["spec_values"], locale, constants.LocaleZhCN)
	if specText != "" {
		return specText
	}
	code := strings.TrimSpace(fmt.Sprintf("%v", snapshot["sku_code"]))
	if code == "" || code == "<nil>" {
		return ""
	}
	return code
}

func buildNotificationDeliverySummary(locale string, counts notificationOrderItemCounts) string {
	return localizedNotificationText(
		locale,
		fmt.Sprintf("?%d?,????%d?,????%d?,????%d?", counts.Total, counts.Auto, counts.Manual, counts.Upstream),
		fmt.Sprintf("?%d?,????%d?,????%d?,????%d?", counts.Total, counts.Auto, counts.Manual, counts.Upstream),
		fmt.Sprintf("Total %d items, auto %d, manual %d, upstream %d", counts.Total, counts.Auto, counts.Manual, counts.Upstream),
	)
}

func notificationFulfillmentLabel(locale, fulfillmentType string) string {
	switch normalizeNotificationFulfillmentType(fulfillmentType) {
	case constants.FulfillmentTypeAuto:
		return localizedNotificationText(locale, "????", "????", "Auto")
	case constants.FulfillmentTypeUpstream:
		return localizedNotificationText(locale, "????", "????", "Upstream")
	default:
		return localizedNotificationText(locale, "????", "????", "Manual")
	}
}

func normalizeNotificationFulfillmentType(fulfillmentType string) string {
	switch strings.ToLower(strings.TrimSpace(fulfillmentType)) {
	case constants.FulfillmentTypeAuto:
		return constants.FulfillmentTypeAuto
	case constants.FulfillmentTypeUpstream:
		return constants.FulfillmentTypeUpstream
	default:
		return constants.FulfillmentTypeManual
	}
}

func notificationInterfaceText(value interface{}, locale, defaultLocale string) string {
	switch typed := value.(type) {
	case models.JSON:
		return resolveNotificationLocalizedJSON(typed, locale, defaultLocale)
	case map[string]interface{}:
		return resolveNotificationLocalizedJSON(models.JSON(typed), locale, defaultLocale)
	case nil:
		return ""
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", typed))
		if text == "<nil>" {
			return ""
		}
		return text
	}
}

func resolveNotificationLocalizedJSON(value models.JSON, locale, defaultLocale string) string {
	if len(value) == 0 {
		return ""
	}
	if text := strings.TrimSpace(fmt.Sprintf("%v", value[locale])); text != "" && text != "<nil>" {
		return text
	}
	if text := strings.TrimSpace(fmt.Sprintf("%v", value[defaultLocale])); text != "" && text != "<nil>" {
		return text
	}
	for _, item := range value {
		if text := strings.TrimSpace(fmt.Sprintf("%v", item)); text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func localizedNotificationText(locale, zhCN, zhTW, enUS string) string {
	switch normalizeNotificationLocale(locale) {
	case constants.LocaleZhTW:
		return zhTW
	case constants.LocaleEnUS:
		return enUS
	default:
		return zhCN
	}
}
