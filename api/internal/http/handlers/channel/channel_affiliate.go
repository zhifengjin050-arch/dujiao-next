package channel

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type channelAffiliateTrackClickRequest struct {
	ChannelUserID  string `json:"channel_user_id,omitempty"`
	TelegramUserID string `json:"telegram_user_id,omitempty"`
	AffiliateCode  string `json:"affiliate_code" binding:"required"`
	VisitorKey     string `json:"visitor_key,omitempty"`
	LandingPath    string `json:"landing_path,omitempty"`
	Referrer       string `json:"referrer,omitempty"`
}

type channelAffiliateApplyWithdrawRequest struct {
	ChannelUserID  string `json:"channel_user_id,omitempty"`
	TelegramUserID string `json:"telegram_user_id,omitempty"`
	Amount         string `json:"amount" binding:"required"`
	Channel        string `json:"channel" binding:"required"`
	Account        string `json:"account" binding:"required"`
}

// OpenAffiliate POST /api/v1/channel/affiliate/open
func (h *Handler) OpenAffiliate(c *gin.Context) {
	if h.AffiliateService == nil {
		respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "internal_error", "error.internal_error", nil)
		return
	}

	var req telegramIdentityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondChannelBindError(c, err)
		return
	}

	input := buildTelegramChannelIdentityInput(req)
	if strings.TrimSpace(input.ChannelUserID) == "" {
		respondChannelError(c, http.StatusBadRequest, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(input)
	if err != nil {
		logger.Errorw("channel_affiliate_open_resolve_user", "channel_user_id", input.ChannelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	profile, err := h.AffiliateService.OpenAffiliate(userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAffiliateDisabled):
			respondChannelError(c, http.StatusBadRequest, response.CodeBadRequest, "affiliate_disabled", "error.forbidden", nil)
		case errors.Is(err, service.ErrNotFound):
			respondChannelError(c, http.StatusNotFound, response.CodeNotFound, "user_not_found", "error.user_not_found", nil)
		case errors.Is(err, service.ErrUserDisabled):
			respondChannelError(c, http.StatusUnauthorized, response.CodeUnauthorized, "user_disabled", "error.user_disabled", nil)
		default:
			logger.Errorw("channel_affiliate_open_failed", "user_id", userID, "error", err)
			respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "affiliate_open_failed", "error.save_failed", err)
		}
		return
	}

	respondChannelSuccess(c, buildChannelAffiliateProfileResponse(profile))
}

// TrackAffiliateClick POST /api/v1/channel/affiliate/click
func (h *Handler) TrackAffiliateClick(c *gin.Context) {
	if h.AffiliateService == nil {
		respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "internal_error", "error.internal_error", nil)
		return
	}

	var req channelAffiliateTrackClickRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondChannelBindError(c, err)
		return
	}

	channelUserID := channelUserIDValue(req.ChannelUserID, req.TelegramUserID)
	if channelUserID == "" {
		respondChannelError(c, http.StatusBadRequest, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	visitorKey := strings.TrimSpace(req.VisitorKey)
	if visitorKey == "" {
		visitorKey = channelUserID
	}

	if err := h.AffiliateService.TrackClick(service.AffiliateTrackClickInput{
		AffiliateCode: req.AffiliateCode,
		VisitorKey:    visitorKey,
		LandingPath:   strings.TrimSpace(req.LandingPath),
		Referrer:      strings.TrimSpace(req.Referrer),
		ClientIP:      c.ClientIP(),
		UserAgent:     c.GetHeader("User-Agent"),
	}); err != nil {
		logger.Errorw("channel_affiliate_track_click_failed", "channel_user_id", channelUserID, "affiliate_code", req.AffiliateCode, "error", err)
		respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "affiliate_track_click_failed", "error.save_failed", err)
		return
	}

	respondChannelSuccess(c, gin.H{"ok": true})
}

// GetAffiliateDashboard GET /api/v1/channel/affiliate/dashboard
func (h *Handler) GetAffiliateDashboard(c *gin.Context) {
	if h.AffiliateService == nil {
		respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "internal_error", "error.internal_error", nil)
		return
	}

	userID, channelUserID, ok := h.resolveChannelAffiliateUserID(c)
	if !ok {
		return
	}

	dashboard, err := h.AffiliateService.GetUserDashboard(userID)
	if err != nil {
		logger.Errorw("channel_affiliate_dashboard_failed", "user_id", userID, "channel_user_id", channelUserID, "error", err)
		respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "affiliate_dashboard_failed", "error.user_fetch_failed", err)
		return
	}
	setting := service.AffiliateDefaultSetting()
	if h.SettingService != nil {
		loadedSetting, settingErr := h.SettingService.GetAffiliateSetting()
		if settingErr != nil {
			logger.Errorw("channel_affiliate_dashboard_setting_failed", "user_id", userID, "channel_user_id", channelUserID, "error", settingErr)
			respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "affiliate_dashboard_failed", "error.user_fetch_failed", settingErr)
			return
		}
		setting = loadedSetting
	}

	respondChannelSuccess(c, gin.H{
		"opened":               dashboard.Opened,
		"affiliate_code":       dashboard.AffiliateCode,
		"promotion_path":       dashboard.PromotionPath,
		"click_count":          dashboard.ClickCount,
		"valid_order_count":    dashboard.ValidOrderCount,
		"conversion_rate":      dashboard.ConversionRate,
		"pending_commission":   dashboard.PendingCommission,
		"available_commission": dashboard.AvailableCommission,
		"withdrawn_commission": dashboard.WithdrawnCommission,
		"min_withdraw_amount":  setting.MinWithdrawAmount,
		"withdraw_channels":    setting.WithdrawChannels,
	})
}

// ListAffiliateCommissions GET /api/v1/channel/affiliate/commissions
func (h *Handler) ListAffiliateCommissions(c *gin.Context) {
	if h.AffiliateService == nil {
		respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "internal_error", "error.internal_error", nil)
		return
	}

	userID, channelUserID, ok := h.resolveChannelAffiliateUserID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)
	status := strings.TrimSpace(c.Query("status"))

	rows, total, err := h.AffiliateService.ListUserCommissions(userID, page, pageSize, status)
	if err != nil {
		logger.Errorw("channel_affiliate_commissions_failed", "user_id", userID, "channel_user_id", channelUserID, "error", err)
		respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "affiliate_commissions_failed", "error.user_fetch_failed", err)
		return
	}

	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, gin.H{
			"id":                   row.ID,
			"affiliate_profile_id": row.AffiliateProfileID,
			"order_id":             row.OrderID,
			"order_no":             strings.TrimSpace(row.Order.OrderNo),
			"order_item_id":        channelAffiliateUintValue(row.OrderItemID),
			"commission_type":      row.CommissionType,
			"base_amount":          row.BaseAmount,
			"rate_percent":         row.RatePercent,
			"commission_amount":    row.CommissionAmount,
			"status":               row.Status,
			"confirm_at":           channelAffiliateTimeValue(row.ConfirmAt),
			"available_at":         channelAffiliateTimeValue(row.AvailableAt),
			"withdraw_request_id":  channelAffiliateUintValue(row.WithdrawRequestID),
			"invalid_reason":       row.InvalidReason,
			"created_at":           row.CreatedAt,
			"updated_at":           row.UpdatedAt,
		})
	}

	respondChannelSuccess(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// ListAffiliateWithdraws GET /api/v1/channel/affiliate/withdraws
func (h *Handler) ListAffiliateWithdraws(c *gin.Context) {
	if h.AffiliateService == nil {
		respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "internal_error", "error.internal_error", nil)
		return
	}

	userID, channelUserID, ok := h.resolveChannelAffiliateUserID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)
	status := strings.TrimSpace(c.Query("status"))

	rows, total, err := h.AffiliateService.ListUserWithdraws(userID, page, pageSize, status)
	if err != nil {
		logger.Errorw("channel_affiliate_withdraws_failed", "user_id", userID, "channel_user_id", channelUserID, "error", err)
		respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "affiliate_withdraws_failed", "error.user_fetch_failed", err)
		return
	}

	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, gin.H{
			"id":                   row.ID,
			"affiliate_profile_id": row.AffiliateProfileID,
			"amount":               row.Amount,
			"channel":              row.Channel,
			"account":              row.Account,
			"status":               row.Status,
			"reject_reason":        row.RejectReason,
			"processed_by":         channelAffiliateUintValue(row.ProcessedBy),
			"processed_at":         channelAffiliateTimeValue(row.ProcessedAt),
			"created_at":           row.CreatedAt,
			"updated_at":           row.UpdatedAt,
		})
	}

	respondChannelSuccess(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// ApplyAffiliateWithdraw POST /api/v1/channel/affiliate/withdraws
func (h *Handler) ApplyAffiliateWithdraw(c *gin.Context) {
	if h.AffiliateService == nil {
		respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "internal_error", "error.internal_error", nil)
		return
	}

	var req channelAffiliateApplyWithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondChannelBindError(c, err)
		return
	}

	channelUserID := channelUserIDValue(req.ChannelUserID, req.TelegramUserID)
	if channelUserID == "" {
		respondChannelError(c, http.StatusBadRequest, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil {
		respondChannelError(c, http.StatusBadRequest, response.CodeBadRequest, "affiliate_withdraw_amount_invalid", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_affiliate_apply_withdraw_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	row, err := h.AffiliateService.ApplyWithdraw(userID, service.AffiliateWithdrawApplyInput{
		Amount:  amount,
		Channel: strings.TrimSpace(req.Channel),
		Account: strings.TrimSpace(req.Account),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAffiliateDisabled):
			respondChannelError(c, http.StatusBadRequest, response.CodeBadRequest, "affiliate_disabled", "error.forbidden", nil)
		case errors.Is(err, service.ErrAffiliateNotOpened):
			respondChannelError(c, http.StatusBadRequest, response.CodeBadRequest, "affiliate_not_opened", "error.bad_request", nil)
		case errors.Is(err, service.ErrAffiliateWithdrawAmountInvalid):
			respondChannelError(c, http.StatusBadRequest, response.CodeBadRequest, "affiliate_withdraw_amount_invalid", "error.bad_request", nil)
		case errors.Is(err, service.ErrAffiliateWithdrawChannelInvalid):
			respondChannelError(c, http.StatusBadRequest, response.CodeBadRequest, "affiliate_withdraw_channel_invalid", "error.bad_request", nil)
		case errors.Is(err, service.ErrAffiliateWithdrawInsufficient):
			respondChannelError(c, http.StatusBadRequest, response.CodeBadRequest, "affiliate_withdraw_insufficient", "error.bad_request", nil)
		default:
			logger.Errorw("channel_affiliate_apply_withdraw_failed", "user_id", userID, "channel_user_id", channelUserID, "error", err)
			respondChannelError(c, http.StatusInternalServerError, response.CodeInternal, "affiliate_withdraw_apply_failed", "error.save_failed", err)
		}
		return
	}

	respondChannelSuccess(c, gin.H{
		"id":                   row.ID,
		"affiliate_profile_id": row.AffiliateProfileID,
		"amount":               row.Amount,
		"channel":              row.Channel,
		"account":              row.Account,
		"status":               row.Status,
		"reject_reason":        row.RejectReason,
		"processed_by":         channelAffiliateUintValue(row.ProcessedBy),
		"processed_at":         channelAffiliateTimeValue(row.ProcessedAt),
		"created_at":           row.CreatedAt,
		"updated_at":           row.UpdatedAt,
	})
}

func (h *Handler) resolveChannelAffiliateUserID(c *gin.Context) (uint, string, bool) {
	channelUserID := channelUserIDFromQuery(c)
	if channelUserID == "" {
		respondChannelError(c, http.StatusBadRequest, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return 0, "", false
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_affiliate_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return 0, channelUserID, false
	}
	return userID, channelUserID, true
}

func buildChannelAffiliateProfileResponse(profile *models.AffiliateProfile) gin.H {
	if profile == nil {
		return gin.H{}
	}
	return gin.H{
		"id":         profile.ID,
		"user_id":    profile.UserID,
		"code":       profile.AffiliateCode,
		"status":     profile.Status,
		"created_at": profile.CreatedAt,
		"updated_at": profile.UpdatedAt,
	}
}

func channelAffiliateUintValue(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

func channelAffiliateTimeValue(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}
