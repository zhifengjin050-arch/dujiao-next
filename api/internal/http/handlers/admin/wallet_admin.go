package admin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// AdminAdjustUserWalletRequest ???????????
type AdminAdjustUserWalletRequest struct {
	Amount    string `json:"amount" binding:"required"`
	Operation string `json:"operation"` // add/subtract
	Currency  string `json:"currency"`
	Remark    string `json:"remark"`
}

type adminWalletRechargeUser struct {
	ID          uint   `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type adminWalletRechargeItem struct {
	models.WalletRechargeOrder
	User          *adminWalletRechargeUser `json:"user,omitempty"`
	ChannelName   string                   `json:"channel_name,omitempty"`
	PaymentStatus string                   `json:"payment_status,omitempty"`
}

// GetAdminUserWallet ???????????
func (h *Handler) GetAdminUserWallet(c *gin.Context) {
	userID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.user_id_invalid", nil)
		return
	}
	user, err := h.UserRepo.GetByID(userID)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", err)
		return
	}
	if user == nil {
		shared.RespondError(c, response.CodeNotFound, "error.user_not_found", nil)
		return
	}
	account, err := h.WalletService.GetAccount(userID)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", err)
		return
	}
	response.Success(c, gin.H{
		"user":    user,
		"account": account,
	})
}

// GetAdminUserWalletTransactions ???????????
func (h *Handler) GetAdminUserWalletTransactions(c *gin.Context) {
	userID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.user_id_invalid", nil)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	filter := repository.WalletTransactionListFilter{
		Page:      page,
		PageSize:  pageSize,
		UserID:    userID,
		Type:      strings.TrimSpace(c.Query("type")),
		Direction: strings.TrimSpace(c.Query("direction")),
	}
	transactions, total, err := h.WalletService.ListTransactions(filter)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", err)
		return
	}
	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, transactions, pagination)
}

// GetAdminWalletRecharges ?????????????
func (h *Handler) GetAdminWalletRecharges(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	userID, err := shared.ParseQueryUint(c.Query("user_id"), false)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	paymentID, err := shared.ParseQueryUint(c.Query("payment_id"), false)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	channelID, err := shared.ParseQueryUint(c.Query("channel_id"), false)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	createdFrom, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("created_from")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	createdTo, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("created_to")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	paidFrom, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("paid_from")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	paidTo, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("paid_to")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	recharges, total, err := h.WalletService.ListRechargeOrdersAdmin(repository.WalletRechargeListFilter{
		Page:         page,
		PageSize:     pageSize,
		RechargeNo:   strings.TrimSpace(c.Query("recharge_no")),
		UserID:       userID,
		UserKeyword:  strings.TrimSpace(c.Query("user_keyword")),
		PaymentID:    paymentID,
		ChannelID:    channelID,
		ProviderType: strings.TrimSpace(strings.ToLower(c.Query("provider_type"))),
		ChannelType:  strings.TrimSpace(strings.ToLower(c.Query("channel_type"))),
		Status:       strings.TrimSpace(strings.ToLower(c.Query("status"))),
		CreatedFrom:  createdFrom,
		CreatedTo:    createdTo,
		PaidFrom:     paidFrom,
		PaidTo:       paidTo,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", err)
		return
	}

	userIDs := make([]uint, 0, len(recharges))
	channelIDs := make([]uint, 0, len(recharges))
	paymentIDs := make([]uint, 0, len(recharges))
	seenUsers := make(map[uint]struct{})
	seenChannels := make(map[uint]struct{})
	seenPayments := make(map[uint]struct{})
	for _, recharge := range recharges {
		if recharge.UserID != 0 {
			if _, ok := seenUsers[recharge.UserID]; !ok {
				seenUsers[recharge.UserID] = struct{}{}
				userIDs = append(userIDs, recharge.UserID)
			}
		}
		if recharge.ChannelID != 0 {
			if _, ok := seenChannels[recharge.ChannelID]; !ok {
				seenChannels[recharge.ChannelID] = struct{}{}
				channelIDs = append(channelIDs, recharge.ChannelID)
			}
		}
		if recharge.PaymentID != 0 {
			if _, ok := seenPayments[recharge.PaymentID]; !ok {
				seenPayments[recharge.PaymentID] = struct{}{}
				paymentIDs = append(paymentIDs, recharge.PaymentID)
			}
		}
	}

	userMap := make(map[uint]models.User, len(userIDs))
	if len(userIDs) > 0 {
		users, userErr := h.UserRepo.ListByIDs(userIDs)
		if userErr != nil {
			shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", userErr)
			return
		}
		for _, user := range users {
			userMap[user.ID] = user
		}
	}

	channelNameMap := make(map[uint]string, len(channelIDs))
	if len(channelIDs) > 0 {
		channels, channelErr := h.PaymentChannelRepo.ListByIDs(channelIDs)
		if channelErr != nil {
			shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", channelErr)
			return
		}
		for _, channel := range channels {
			channelNameMap[channel.ID] = channel.Name
		}
	}

	paymentStatusMap := make(map[uint]string, len(paymentIDs))
	if len(paymentIDs) > 0 {
		payments, paymentErr := h.PaymentRepo.GetByIDs(paymentIDs)
		if paymentErr != nil {
			shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", paymentErr)
			return
		}
		for _, payment := range payments {
			paymentStatusMap[payment.ID] = payment.Status
		}
	}

	items := make([]adminWalletRechargeItem, 0, len(recharges))
	for _, recharge := range recharges {
		item := adminWalletRechargeItem{
			WalletRechargeOrder: recharge,
			ChannelName:         channelNameMap[recharge.ChannelID],
			PaymentStatus:       paymentStatusMap[recharge.PaymentID],
		}
		if user, ok := userMap[recharge.UserID]; ok {
			item.User = &adminWalletRechargeUser{
				ID:          user.ID,
				Email:       user.Email,
				DisplayName: user.DisplayName,
			}
		}
		items = append(items, item)
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, items, pagination)
}

// AdjustAdminUserWallet ?????????
func (h *Handler) AdjustAdminUserWallet(c *gin.Context) {
	userID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.user_id_invalid", nil)
		return
	}
	var req AdminAdjustUserWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	delta := amount
	if op == "" {
		op = "add"
	}
	if op == "subtract" {
		delta = amount.Neg()
	}
	if op != "add" && op != "subtract" {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" && h.SettingService != nil {
		siteCurrency, currencyErr := h.SettingService.GetSiteCurrency(constants.SiteCurrencyDefault)
		if currencyErr == nil {
			currency = siteCurrency
		}
	}

	account, txn, err := h.WalletService.AdminAdjustBalance(service.WalletAdjustInput{
		UserID:   userID,
		Delta:    models.NewMoneyFromDecimal(delta),
		Currency: currency,
		Remark:   strings.TrimSpace(req.Remark),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWalletInvalidAmount):
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		case errors.Is(err, service.ErrWalletInsufficientBalance):
			shared.RespondError(c, response.CodeBadRequest, "error.payment_amount_mismatch", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.user_update_failed", err)
		}
		return
	}

	response.Success(c, gin.H{
		"account":     account,
		"transaction": txn,
	})
}
