package dto

import (
	"time"

	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/service"
)

// CreatePaymentResp ??????
type CreatePaymentResp struct {
	OrderPaid        bool         `json:"order_paid"`
	WalletPaidAmount models.Money `json:"wallet_paid_amount"`
	OnlinePayAmount  models.Money `json:"online_pay_amount"`
	PaymentID        *uint        `json:"payment_id,omitempty"`
	ChannelID        *uint        `json:"channel_id,omitempty"`
	ProviderType     string       `json:"provider_type,omitempty"`
	ChannelType      string       `json:"channel_type,omitempty"`
	InteractionMode  string       `json:"interaction_mode,omitempty"`
	PayURL           string       `json:"pay_url,omitempty"`
	QRCode           string       `json:"qr_code,omitempty"`
	ExpiresAt        *time.Time   `json:"expires_at,omitempty"`
	ChannelName      string       `json:"channel_name,omitempty"`
}

// NewCreatePaymentResp ? service.CreatePaymentResult ????
func NewCreatePaymentResp(result *service.CreatePaymentResult) CreatePaymentResp {
	resp := CreatePaymentResp{
		OrderPaid:        result.OrderPaid,
		WalletPaidAmount: result.WalletPaidAmount,
		OnlinePayAmount:  result.OnlinePayAmount,
	}
	if result.Payment != nil {
		resp.PaymentID = &result.Payment.ID
		resp.ChannelID = &result.Payment.ChannelID
		resp.ProviderType = result.Payment.ProviderType
		resp.ChannelType = result.Payment.ChannelType
		resp.InteractionMode = result.Payment.InteractionMode
		resp.PayURL = result.Payment.PayURL
		resp.QRCode = result.Payment.QRCode
		resp.ExpiresAt = result.Payment.ExpiredAt
	}
	if result.Channel != nil {
		resp.ChannelName = result.Channel.Name
	}
	return resp
}

// LatestPaymentResp ?????????
type LatestPaymentResp struct {
	PaymentID       uint       `json:"payment_id"`
	OrderNo         string     `json:"order_no"`
	ChannelID       uint       `json:"channel_id"`
	ChannelName     string     `json:"channel_name,omitempty"`
	ProviderType    string     `json:"provider_type"`
	ChannelType     string     `json:"channel_type"`
	InteractionMode string     `json:"interaction_mode"`
	PayURL          string     `json:"pay_url"`
	QRCode          string     `json:"qr_code"`
	ExpiresAt       *time.Time `json:"expires_at"`
}

// NewLatestPaymentResp ? Payment + Order ????
func NewLatestPaymentResp(payment *models.Payment, orderNo string) LatestPaymentResp {
	return LatestPaymentResp{
		PaymentID:       payment.ID,
		OrderNo:         orderNo,
		ChannelID:       payment.ChannelID,
		ChannelName:     payment.ChannelName,
		ProviderType:    payment.ProviderType,
		ChannelType:     payment.ChannelType,
		InteractionMode: payment.InteractionMode,
		PayURL:          payment.PayURL,
		QRCode:          payment.QRCode,
		ExpiresAt:       payment.ExpiredAt,
	}
	// ??:OrderID?Amount?FeeRate?FixedFee?FeeAmount?Currency?Status?
	// ProviderRef?GatewayOrderNo?ProviderPayload?CreatedAt?UpdatedAt?PaidAt?CallbackAt
}
