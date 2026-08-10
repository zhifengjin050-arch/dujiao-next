package public

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/i18n"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// mappedHandlerError ???????????????????
type mappedHandlerError struct {
	target error
	code   int
	key    string
}

func respondWithMappedError(c *gin.Context, err error, rules []mappedHandlerError, fallbackCode int, fallbackKey string) {
	// ????????:?? Retry-After header,????????
	if retryAfter := service.GetRetryAfter(err); retryAfter > 0 {
		c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
		locale := i18n.ResolveLocale(c)
		msg := fmt.Sprintf("%s (%ds)", i18n.T(locale, "error.risk_order_rate_limited"), retryAfter)
		response.Error(c, response.CodeTooManyRequests, msg)
		return
	}

	for _, rule := range rules {
		if errors.Is(err, rule.target) {
			shared.RespondError(c, rule.code, rule.key, nil)
			return
		}
	}
	shared.RespondError(c, fallbackCode, fallbackKey, err)
}

func concatMappedHandlerErrors(groups ...[]mappedHandlerError) []mappedHandlerError {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	result := make([]mappedHandlerError, 0, total)
	for _, group := range groups {
		result = append(result, group...)
	}
	return result
}

var orderRiskControlErrorRules = []mappedHandlerError{
	{target: service.ErrRiskIPBlacklisted, code: response.CodeForbidden, key: "error.risk_ip_blacklisted"},
	{target: service.ErrRiskEmailBlacklisted, code: response.CodeForbidden, key: "error.risk_email_blacklisted"},
	{target: service.ErrRiskTooManyPendingOrders, code: response.CodeTooManyRequests, key: "error.risk_too_many_pending_orders"},
	{target: service.ErrRiskOrderRateLimited, code: response.CodeTooManyRequests, key: "error.risk_order_rate_limited"},
}

var userOrderCommonErrorRules = []mappedHandlerError{
	{target: service.ErrProductSKURequired, code: response.CodeBadRequest, key: "error.order_item_invalid"},
	{target: service.ErrProductSKUInvalid, code: response.CodeBadRequest, key: "error.order_item_invalid"},
	{target: service.ErrInvalidOrderItem, code: response.CodeBadRequest, key: "error.order_item_invalid"},
	{target: service.ErrInvalidOrderAmount, code: response.CodeBadRequest, key: "error.order_amount_invalid"},
	{target: service.ErrProductPurchaseNotAllowed, code: response.CodeBadRequest, key: "error.product_purchase_not_allowed"},
	{target: service.ErrProductMaxPurchaseExceeded, code: response.CodeBadRequest, key: "error.product_max_purchase_exceeded"},
	{target: service.ErrManualStockInsufficient, code: response.CodeBadRequest, key: "error.manual_stock_insufficient"},
	{target: service.ErrCardSecretInsufficient, code: response.CodeBadRequest, key: "error.card_secret_insufficient"},
	{target: service.ErrOrderCurrencyMismatch, code: response.CodeBadRequest, key: "error.order_currency_mismatch"},
	{target: service.ErrProductPriceInvalid, code: response.CodeBadRequest, key: "error.product_price_invalid"},
	{target: service.ErrProductNotAvailable, code: response.CodeBadRequest, key: "error.product_not_available"},
	{target: service.ErrCouponInvalid, code: response.CodeBadRequest, key: "error.coupon_invalid"},
	{target: service.ErrCouponNotFound, code: response.CodeBadRequest, key: "error.coupon_not_found"},
	{target: service.ErrCouponInactive, code: response.CodeBadRequest, key: "error.coupon_inactive"},
	{target: service.ErrCouponNotStarted, code: response.CodeBadRequest, key: "error.coupon_not_started"},
	{target: service.ErrCouponExpired, code: response.CodeBadRequest, key: "error.coupon_expired"},
	{target: service.ErrCouponUsageLimit, code: response.CodeBadRequest, key: "error.coupon_usage_limit"},
	{target: service.ErrCouponPerUserLimit, code: response.CodeBadRequest, key: "error.coupon_per_user_limit"},
	{target: service.ErrCouponMinAmount, code: response.CodeBadRequest, key: "error.coupon_min_amount"},
	{target: service.ErrCouponScopeInvalid, code: response.CodeBadRequest, key: "error.coupon_scope_invalid"},
	{target: service.ErrCouponPaymentRoleNotAllowed, code: response.CodeBadRequest, key: "error.coupon_payment_role_not_allowed"},
	{target: service.ErrCouponPaymentRoleGuestOnly, code: response.CodeBadRequest, key: "error.coupon_payment_role_guest_only"},
	{target: service.ErrCouponPaymentRoleMemberOnly, code: response.CodeBadRequest, key: "error.coupon_payment_role_member_only"},
	{target: service.ErrCouponMemberLevelNotAllowed, code: response.CodeBadRequest, key: "error.coupon_member_level_not_allowed"},
	{target: service.ErrPromotionInvalid, code: response.CodeBadRequest, key: "error.promotion_invalid"},
	{target: service.ErrManualFormSchemaInvalid, code: response.CodeBadRequest, key: "error.manual_form_schema_invalid"},
	{target: service.ErrManualFormRequiredMissing, code: response.CodeBadRequest, key: "error.manual_form_required_missing"},
	{target: service.ErrManualFormFieldInvalid, code: response.CodeBadRequest, key: "error.manual_form_field_invalid"},
	{target: service.ErrManualFormTypeInvalid, code: response.CodeBadRequest, key: "error.manual_form_type_invalid"},
	{target: service.ErrManualFormOptionInvalid, code: response.CodeBadRequest, key: "error.manual_form_option_invalid"},
}

var userOrderPreviewExtraErrorRules = []mappedHandlerError{
	{target: service.ErrQueueUnavailable, code: response.CodeInternal, key: "error.queue_unavailable"},
}

var guestOrderCommonErrorRules = []mappedHandlerError{
	{target: service.ErrProductSKURequired, code: response.CodeBadRequest, key: "error.order_item_invalid"},
	{target: service.ErrProductSKUInvalid, code: response.CodeBadRequest, key: "error.order_item_invalid"},
	{target: service.ErrGuestEmailRequired, code: response.CodeBadRequest, key: "error.guest_email_required"},
	{target: service.ErrGuestPasswordRequired, code: response.CodeBadRequest, key: "error.guest_password_required"},
	{target: service.ErrInvalidEmail, code: response.CodeBadRequest, key: "error.email_invalid"},
	{target: service.ErrProductPurchaseNotAllowed, code: response.CodeBadRequest, key: "error.product_purchase_not_allowed"},
	{target: service.ErrProductMaxPurchaseExceeded, code: response.CodeBadRequest, key: "error.product_max_purchase_exceeded"},
	{target: service.ErrGuestCouponNotAllowed, code: response.CodeBadRequest, key: "error.guest_coupon_not_allowed"},
	{target: service.ErrInvalidOrderItem, code: response.CodeBadRequest, key: "error.order_item_invalid"},
	{target: service.ErrInvalidOrderAmount, code: response.CodeBadRequest, key: "error.order_amount_invalid"},
	{target: service.ErrManualStockInsufficient, code: response.CodeBadRequest, key: "error.manual_stock_insufficient"},
	{target: service.ErrCardSecretInsufficient, code: response.CodeBadRequest, key: "error.card_secret_insufficient"},
	{target: service.ErrOrderCurrencyMismatch, code: response.CodeBadRequest, key: "error.order_currency_mismatch"},
	{target: service.ErrProductPriceInvalid, code: response.CodeBadRequest, key: "error.product_price_invalid"},
	{target: service.ErrProductNotAvailable, code: response.CodeBadRequest, key: "error.product_not_available"},
	{target: service.ErrManualFormSchemaInvalid, code: response.CodeBadRequest, key: "error.manual_form_schema_invalid"},
	{target: service.ErrManualFormRequiredMissing, code: response.CodeBadRequest, key: "error.manual_form_required_missing"},
	{target: service.ErrManualFormFieldInvalid, code: response.CodeBadRequest, key: "error.manual_form_field_invalid"},
	{target: service.ErrManualFormTypeInvalid, code: response.CodeBadRequest, key: "error.manual_form_type_invalid"},
	{target: service.ErrManualFormOptionInvalid, code: response.CodeBadRequest, key: "error.manual_form_option_invalid"},
}

var guestOrderCreateExtraErrorRules = []mappedHandlerError{
	{target: service.ErrQueueUnavailable, code: response.CodeInternal, key: "error.queue_unavailable"},
}

var guestOrderPreviewExtraErrorRules = []mappedHandlerError{
	{target: service.ErrCouponInvalid, code: response.CodeBadRequest, key: "error.coupon_invalid"},
	{target: service.ErrCouponNotFound, code: response.CodeBadRequest, key: "error.coupon_not_found"},
	{target: service.ErrCouponInactive, code: response.CodeBadRequest, key: "error.coupon_inactive"},
	{target: service.ErrCouponNotStarted, code: response.CodeBadRequest, key: "error.coupon_not_started"},
	{target: service.ErrCouponExpired, code: response.CodeBadRequest, key: "error.coupon_expired"},
	{target: service.ErrCouponUsageLimit, code: response.CodeBadRequest, key: "error.coupon_usage_limit"},
	{target: service.ErrCouponPerUserLimit, code: response.CodeBadRequest, key: "error.coupon_per_user_limit"},
	{target: service.ErrCouponMinAmount, code: response.CodeBadRequest, key: "error.coupon_min_amount"},
	{target: service.ErrCouponScopeInvalid, code: response.CodeBadRequest, key: "error.coupon_scope_invalid"},
	{target: service.ErrCouponPaymentRoleNotAllowed, code: response.CodeBadRequest, key: "error.coupon_payment_role_not_allowed"},
	{target: service.ErrCouponPaymentRoleGuestOnly, code: response.CodeBadRequest, key: "error.coupon_payment_role_guest_only"},
	{target: service.ErrCouponPaymentRoleMemberOnly, code: response.CodeBadRequest, key: "error.coupon_payment_role_member_only"},
	{target: service.ErrCouponMemberLevelNotAllowed, code: response.CodeBadRequest, key: "error.coupon_member_level_not_allowed"},
	{target: service.ErrPromotionInvalid, code: response.CodeBadRequest, key: "error.promotion_invalid"},
}

var paymentCreateErrorRules = []mappedHandlerError{
	{target: service.ErrPaymentInvalid, code: response.CodeBadRequest, key: "error.payment_invalid"},
	{target: service.ErrOrderNotFound, code: response.CodeNotFound, key: "error.order_not_found"},
	{target: service.ErrOrderStatusInvalid, code: response.CodeBadRequest, key: "error.order_status_invalid"},
	{target: service.ErrPaymentChannelNotFound, code: response.CodeNotFound, key: "error.payment_channel_not_found"},
	{target: service.ErrPaymentChannelInactive, code: response.CodeBadRequest, key: "error.payment_channel_inactive"},
	{target: service.ErrPaymentProviderNotSupported, code: response.CodeBadRequest, key: "error.payment_provider_not_supported"},
	{target: service.ErrPaymentChannelConfigInvalid, code: response.CodeBadRequest, key: "error.payment_channel_config_invalid"},
	{target: service.ErrPaymentGatewayRequestFailed, code: response.CodeBadRequest, key: "error.payment_gateway_request_failed"},
	{target: service.ErrPaymentGatewayResponseInvalid, code: response.CodeBadRequest, key: "error.payment_gateway_response_invalid"},
	{target: service.ErrPaymentCurrencyMismatch, code: response.CodeBadRequest, key: "error.payment_currency_mismatch"},
	{target: service.ErrWalletNotSupportedForGuest, code: response.CodeBadRequest, key: "error.payment_invalid"},
	{target: service.ErrPaymentChannelNotAllowedForProduct, code: response.CodeBadRequest, key: "error.payment_channel_not_allowed_for_product"},
	{target: service.ErrPaymentChannelNotAllowedForRecharge, code: response.CodeBadRequest, key: "error.payment_channel_not_allowed_for_recharge"},
	{target: service.ErrWalletOnlyPaymentRequired, code: response.CodeBadRequest, key: "error.wallet_only_payment_required"},
}

var paymentCaptureErrorRules = []mappedHandlerError{
	{target: service.ErrPaymentInvalid, code: response.CodeBadRequest, key: "error.payment_invalid"},
	{target: service.ErrPaymentNotFound, code: response.CodeNotFound, key: "error.payment_not_found"},
	{target: service.ErrPaymentChannelNotFound, code: response.CodeNotFound, key: "error.payment_channel_not_found"},
	{target: service.ErrPaymentProviderNotSupported, code: response.CodeBadRequest, key: "error.payment_provider_not_supported"},
	{target: service.ErrPaymentChannelConfigInvalid, code: response.CodeBadRequest, key: "error.payment_channel_config_invalid"},
	{target: service.ErrPaymentGatewayRequestFailed, code: response.CodeBadRequest, key: "error.payment_gateway_request_failed"},
	{target: service.ErrPaymentGatewayResponseInvalid, code: response.CodeBadRequest, key: "error.payment_gateway_response_invalid"},
	{target: service.ErrPaymentStatusInvalid, code: response.CodeBadRequest, key: "error.payment_status_invalid"},
	{target: service.ErrPaymentAmountMismatch, code: response.CodeBadRequest, key: "error.payment_amount_mismatch"},
	{target: service.ErrPaymentCurrencyMismatch, code: response.CodeBadRequest, key: "error.payment_currency_mismatch"},
	{target: service.ErrOrderNotFound, code: response.CodeNotFound, key: "error.order_not_found"},
}

var paymentCallbackErrorRules = []mappedHandlerError{
	{target: service.ErrPaymentInvalid, code: response.CodeBadRequest, key: "error.payment_invalid"},
	{target: service.ErrPaymentNotFound, code: response.CodeNotFound, key: "error.payment_not_found"},
	{target: service.ErrPaymentStatusInvalid, code: response.CodeBadRequest, key: "error.payment_status_invalid"},
	{target: service.ErrPaymentAmountMismatch, code: response.CodeBadRequest, key: "error.payment_amount_mismatch"},
	{target: service.ErrPaymentCurrencyMismatch, code: response.CodeBadRequest, key: "error.payment_currency_mismatch"},
	{target: service.ErrPaymentChannelNotFound, code: response.CodeNotFound, key: "error.payment_channel_not_found"},
	{target: service.ErrPaymentProviderNotSupported, code: response.CodeBadRequest, key: "error.payment_provider_not_supported"},
	{target: service.ErrPaymentChannelConfigInvalid, code: response.CodeBadRequest, key: "error.payment_channel_config_invalid"},
	{target: service.ErrPaymentGatewayRequestFailed, code: response.CodeBadRequest, key: "error.payment_gateway_request_failed"},
	{target: service.ErrPaymentGatewayResponseInvalid, code: response.CodeBadRequest, key: "error.payment_gateway_response_invalid"},
}

func respondUserOrderPreviewError(c *gin.Context, err error) {
	respondWithMappedError(c, err, concatMappedHandlerErrors(userOrderCommonErrorRules, userOrderPreviewExtraErrorRules), response.CodeInternal, "error.order_create_failed")
}

func respondUserOrderCreateError(c *gin.Context, err error) {
	respondWithMappedError(c, err, concatMappedHandlerErrors(orderRiskControlErrorRules, userOrderCommonErrorRules), response.CodeInternal, "error.order_create_failed")
}

func respondGuestOrderCreateError(c *gin.Context, err error) {
	respondWithMappedError(c, err, concatMappedHandlerErrors(orderRiskControlErrorRules, guestOrderCommonErrorRules, guestOrderCreateExtraErrorRules), response.CodeInternal, "error.order_create_failed")
}

func respondGuestOrderPreviewError(c *gin.Context, err error) {
	respondWithMappedError(c, err, concatMappedHandlerErrors(guestOrderCommonErrorRules, guestOrderPreviewExtraErrorRules), response.CodeInternal, "error.order_create_failed")
}

func respondPaymentCallbackError(c *gin.Context, err error) {
	respondWithMappedError(c, err, paymentCallbackErrorRules, response.CodeInternal, "error.payment_callback_failed")
}
