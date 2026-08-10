package dto

import (
	"time"

	"github.com/dujiao-next/internal/models"
)

// AffiliateProfileResp ????????
type AffiliateProfileResp struct {
	ID            uint      `json:"id"`
	AffiliateCode string    `json:"code"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// NewAffiliateProfileResp ? models.AffiliateProfile ????
func NewAffiliateProfileResp(p *models.AffiliateProfile) AffiliateProfileResp {
	return AffiliateProfileResp{
		ID:            p.ID,
		AffiliateCode: p.AffiliateCode,
		Status:        p.Status,
		CreatedAt:     p.CreatedAt,
	}
	// ??:UserID?User?UpdatedAt
}

// AffiliateCommissionResp ??????
type AffiliateCommissionResp struct {
	ID               uint         `json:"id"`
	CommissionType   string       `json:"commission_type"`
	CommissionAmount models.Money `json:"commission_amount"`
	Status           string       `json:"status"`
	ConfirmAt        *time.Time   `json:"confirm_at,omitempty"`
	AvailableAt      *time.Time   `json:"available_at,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
}

// NewAffiliateCommissionResp ? models.AffiliateCommission ????
func NewAffiliateCommissionResp(c *models.AffiliateCommission) AffiliateCommissionResp {
	return AffiliateCommissionResp{
		ID:               c.ID,
		CommissionType:   c.CommissionType,
		CommissionAmount: c.CommissionAmount,
		Status:           c.Status,
		ConfirmAt:        c.ConfirmAt,
		AvailableAt:      c.AvailableAt,
		CreatedAt:        c.CreatedAt,
	}
	// ??:AffiliateProfileID?OrderItemID?BaseAmount?RatePercent?
	// WithdrawRequestID?InvalidReason?UpdatedAt??? Order/AffiliateProfile/WithdrawRequest
}

// NewAffiliateCommissionRespList ????????
func NewAffiliateCommissionRespList(commissions []models.AffiliateCommission) []AffiliateCommissionResp {
	result := make([]AffiliateCommissionResp, 0, len(commissions))
	for i := range commissions {
		result = append(result, NewAffiliateCommissionResp(&commissions[i]))
	}
	return result
}

// AffiliateWithdrawResp ??????
type AffiliateWithdrawResp struct {
	ID           uint         `json:"id"`
	Amount       models.Money `json:"amount"`
	Channel      string       `json:"channel"`
	Account      string       `json:"account"`
	Status       string       `json:"status"`
	RejectReason string       `json:"reject_reason,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
}

// NewAffiliateWithdrawResp ? models.AffiliateWithdrawRequest ????
func NewAffiliateWithdrawResp(w *models.AffiliateWithdrawRequest) AffiliateWithdrawResp {
	return AffiliateWithdrawResp{
		ID:           w.ID,
		Amount:       w.Amount,
		Channel:      w.Channel,
		Account:      w.Account,
		Status:       w.Status,
		RejectReason: w.RejectReason,
		CreatedAt:    w.CreatedAt,
	}
	// ??:AffiliateProfileID?ProcessedBy?ProcessedAt?UpdatedAt???
}

// NewAffiliateWithdrawRespList ????????
func NewAffiliateWithdrawRespList(withdraws []models.AffiliateWithdrawRequest) []AffiliateWithdrawResp {
	result := make([]AffiliateWithdrawResp, 0, len(withdraws))
	for i := range withdraws {
		result = append(result, NewAffiliateWithdrawResp(&withdraws[i]))
	}
	return result
}
