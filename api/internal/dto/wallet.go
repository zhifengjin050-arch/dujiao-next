package dto

import (
	"time"

	"github.com/dujiao-next/internal/models"
)

// WalletAccountResp ??????
type WalletAccountResp struct {
	Balance models.Money `json:"balance"`
}

// NewWalletAccountResp ? models.WalletAccount ????
func NewWalletAccountResp(a *models.WalletAccount) WalletAccountResp {
	return WalletAccountResp{
		Balance: a.Balance,
	}
}

// WalletTransactionResp ??????
type WalletTransactionResp struct {
	ID           uint         `json:"id"`
	Type         string       `json:"type"`
	Direction    string       `json:"direction"`
	Amount       models.Money `json:"amount"`
	BalanceAfter models.Money `json:"balance_after"`
	Remark       string       `json:"remark"`
	CreatedAt    time.Time    `json:"created_at"`
}

// NewWalletTransactionResp ? models.WalletTransaction ????
func NewWalletTransactionResp(t *models.WalletTransaction) WalletTransactionResp {
	return WalletTransactionResp{
		ID:           t.ID,
		Type:         t.Type,
		Direction:    t.Direction,
		Amount:       t.Amount,
		BalanceAfter: t.BalanceAfter,
		Remark:       t.Remark,
		CreatedAt:    t.CreatedAt,
	}
	// ??:UserID?Currency?BalanceBefore?Reference?UpdatedAt
}

// NewWalletTransactionRespList ????????
func NewWalletTransactionRespList(txns []models.WalletTransaction) []WalletTransactionResp {
	result := make([]WalletTransactionResp, 0, len(txns))
	for i := range txns {
		result = append(result, NewWalletTransactionResp(&txns[i]))
	}
	return result
}

// WalletRechargeResp ???????
type WalletRechargeResp struct {
	ID            uint         `json:"id"`
	RechargeNo    string       `json:"recharge_no"`
	Amount        models.Money `json:"amount"`
	PayableAmount models.Money `json:"payable_amount"`
	FeeRate       models.Money `json:"fee_rate"`
	FeeAmount     models.Money `json:"fee_amount"`
	Currency      string       `json:"currency"`
	Status        string       `json:"status"`
	Remark        string       `json:"remark"`
	PaidAt        *time.Time   `json:"paid_at"`
	CreatedAt     time.Time    `json:"created_at"`
}

// NewWalletRechargeResp ? models.WalletRechargeOrder ????
func NewWalletRechargeResp(r *models.WalletRechargeOrder) WalletRechargeResp {
	return WalletRechargeResp{
		ID:            r.ID,
		RechargeNo:    r.RechargeNo,
		Amount:        r.Amount,
		PayableAmount: r.PayableAmount,
		FeeRate:       r.FeeRate,
		FeeAmount:     r.FeeAmount,
		Currency:      r.Currency,
		Status:        r.Status,
		Remark:        r.Remark,
		PaidAt:        r.PaidAt,
		CreatedAt:     r.CreatedAt,
	}
	// ??:UserID?PaymentID?ChannelID?ProviderType?ChannelType?InteractionMode?UpdatedAt
}

// NewWalletRechargeRespList ?????????
func NewWalletRechargeRespList(orders []models.WalletRechargeOrder) []WalletRechargeResp {
	result := make([]WalletRechargeResp, 0, len(orders))
	for i := range orders {
		result = append(result, NewWalletRechargeResp(&orders[i]))
	}
	return result
}

// WalletRechargePaymentPayload ??????????
type WalletRechargePaymentPayload struct {
	Recharge        *WalletRechargeResp `json:"recharge,omitempty"`
	RechargeNo      string              `json:"recharge_no,omitempty"`
	RechargeStatus  string              `json:"recharge_status,omitempty"`
	Account         *WalletAccountResp  `json:"account,omitempty"`
	PaymentID       *uint               `json:"payment_id,omitempty"`
	ProviderType    string              `json:"provider_type,omitempty"`
	ChannelType     string              `json:"channel_type,omitempty"`
	InteractionMode string              `json:"interaction_mode,omitempty"`
	PayURL          string              `json:"pay_url,omitempty"`
	QRCode          string              `json:"qr_code,omitempty"`
	ExpiresAt       *time.Time          `json:"expires_at,omitempty"`
	Status          string              `json:"status,omitempty"`
}

// NewWalletRechargePaymentPayload ??????????
func NewWalletRechargePaymentPayload(recharge *models.WalletRechargeOrder, payment *models.Payment, account *models.WalletAccount) WalletRechargePaymentPayload {
	p := WalletRechargePaymentPayload{}
	if recharge != nil {
		r := NewWalletRechargeResp(recharge)
		p.Recharge = &r
		p.RechargeNo = recharge.RechargeNo
		p.RechargeStatus = recharge.Status
	}
	if account != nil {
		a := NewWalletAccountResp(account)
		p.Account = &a
	}
	if payment != nil {
		p.PaymentID = &payment.ID
		p.ProviderType = payment.ProviderType
		p.ChannelType = payment.ChannelType
		p.InteractionMode = payment.InteractionMode
		p.PayURL = payment.PayURL
		p.QRCode = payment.QRCode
		p.ExpiresAt = payment.ExpiredAt
		p.Status = payment.Status
	}
	return p
	// ?? Payment ?:OrderID?ChannelID?Amount?FeeRate?FixedFee?FeeAmount?Currency?
	// ProviderRef?GatewayOrderNo?ProviderPayload?CreatedAt?UpdatedAt?PaidAt?CallbackAt
}
