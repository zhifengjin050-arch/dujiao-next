package service

import (
	"errors"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ExpireWalletRechargePayment ???????????????(??)?
func (s *PaymentService) ExpireWalletRechargePayment(paymentID uint) (*models.Payment, error) {
	if paymentID == 0 {
		return nil, ErrPaymentInvalid
	}
	if s == nil || s.paymentRepo == nil || s.walletRepo == nil {
		return nil, ErrPaymentUpdateFailed
	}

	var output *models.Payment
	err := s.paymentRepo.Transaction(func(tx *gorm.DB) error {
		var payment models.Payment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&payment, paymentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPaymentNotFound
			}
			return ErrPaymentUpdateFailed
		}
		// ??????????,????????????????
		if payment.OrderID != 0 {
			output = &payment
			return nil
		}

		rechargeRepo := s.walletRepo.WithTx(tx)
		recharge, err := rechargeRepo.GetRechargeOrderByPaymentIDForUpdate(payment.ID)
		if err != nil {
			return ErrPaymentUpdateFailed
		}
		if recharge == nil {
			return ErrWalletRechargeNotFound
		}
		if !canExpireWalletRechargePayment(&payment, recharge) {
			output = &payment
			return nil
		}

		now := time.Now()
		payment.Status = constants.PaymentStatusExpired
		payment.ExpiredAt = &now
		payment.UpdatedAt = now
		if err := s.paymentRepo.WithTx(tx).Update(&payment); err != nil {
			return ErrPaymentUpdateFailed
		}

		recharge.Status = constants.WalletRechargeStatusExpired
		recharge.UpdatedAt = now
		if err := rechargeRepo.UpdateRechargeOrder(recharge); err != nil {
			return ErrPaymentUpdateFailed
		}
		output = &payment
		return nil
	})
	if err != nil {
		return nil, err
	}
	return output, nil
}

func canExpireWalletRechargePayment(payment *models.Payment, recharge *models.WalletRechargeOrder) bool {
	if payment == nil || recharge == nil {
		return false
	}
	if recharge.Status != constants.WalletRechargeStatusPending {
		return false
	}
	if payment.Status == constants.PaymentStatusSuccess || recharge.Status == constants.WalletRechargeStatusSuccess {
		return false
	}
	if payment.Status == constants.PaymentStatusFailed || recharge.Status == constants.WalletRechargeStatusFailed {
		return false
	}
	if payment.Status == constants.PaymentStatusExpired || recharge.Status == constants.WalletRechargeStatusExpired {
		return false
	}
	return payment.Status == constants.PaymentStatusInitiated || payment.Status == constants.PaymentStatusPending
}
