package constants

// ??????
const (
	OrderStatusPendingPayment     = "pending_payment"
	OrderStatusPaid               = "paid"
	OrderStatusFulfilling         = "fulfilling"
	OrderStatusPartiallyDelivered = "partially_delivered"
	OrderStatusPartiallyRefunded  = "partially_refunded"
	OrderStatusDelivered          = "delivered"
	OrderStatusCompleted          = "completed"
	OrderStatusCanceled           = "canceled"
	OrderStatusRefunded           = "refunded"
)

// ??????

const (
	OrderRefundTypeManual = "manual"
	OrderRefundTypeWallet = "wallet"
)

// ?????????
const (
	FulfillmentTypeAuto        = "auto"
	FulfillmentTypeManual      = "manual"
	FulfillmentTypeUpstream    = "upstream"
	FulfillmentStatusPending   = "pending"
	FulfillmentStatusDelivered = "delivered"
)

// ??????
const (
	PaymentStatusInitiated = "initiated"
	PaymentStatusPending   = "pending"
	PaymentStatusSuccess   = "success"
	PaymentStatusFailed    = "failed"
	PaymentStatusExpired   = "expired"
)

// ???????
const (
	PaymentProviderOfficial = "official"
	PaymentProviderEpay     = "epay"
	PaymentProviderEpusdt   = "epusdt"
	PaymentProviderOkpay    = "okpay"
	PaymentProviderTokenpay = "tokenpay"
	PaymentProviderXunhupay = "xunhupay"
	PaymentProviderWallet   = "wallet"
)

// ????????
const (
	PaymentChannelTypeWechat    = "wechat"
	PaymentChannelTypeWxpay     = "wxpay"
	PaymentChannelTypeAlipay    = "alipay"
	PaymentChannelTypePaypal    = "paypal"
	PaymentChannelTypeStripe    = "stripe"
	PaymentChannelTypeQqpay     = "qqpay"
	PaymentChannelTypeUsdt      = "usdt"
	PaymentChannelTypeUsdtTrc20 = "usdt-trc20"
	PaymentChannelTypeUsdcTrc20 = "usdc-trc20"
	PaymentChannelTypeTrx       = "trx"
	PaymentChannelTypeBalance   = "balance"
)

// ??????????
const (
	PaymentRoleGuest  = "guest"
	PaymentRoleMember = "member"
)

// ????????
const (
	PaymentTypeWallet = "wallet"
	PaymentTypeOrder  = "order"
)

// ????????
const (
	PaymentInteractionQR       = "qr"
	PaymentInteractionRedirect = "redirect"
	PaymentInteractionWAP      = "wap"
	PaymentInteractionPage     = "page"
	PaymentInteractionBalance  = "balance"
)

// ????????
const (
	WalletTxnTypeRecharge    = "recharge"
	WalletTxnTypeOrderPay    = "order_pay"
	WalletTxnTypeOrderRefund = "order_refund"
	WalletTxnTypeAdminAdjust = "admin_adjust"
	WalletTxnTypeAdminRefund = "admin_refund"
	WalletTxnTypeGiftCard    = "gift_card_redeem"
)

// ????????
const (
	WalletTxnDirectionIn  = "in"
	WalletTxnDirectionOut = "out"
)

// ????????
const (
	WalletRechargeStatusPending = "pending"
	WalletRechargeStatusSuccess = "success"
	WalletRechargeStatusFailed  = "failed"
	WalletRechargeStatusExpired = "expired"
)

// ????????
const (
	AffiliateProfileStatusActive   = "active"
	AffiliateProfileStatusDisabled = "disabled"
)

// ??????????
const (
	AffiliateCommissionStatusPendingConfirm = "pending_confirm"
	AffiliateCommissionStatusAvailable      = "available"
	AffiliateCommissionStatusRejected       = "rejected"
	AffiliateCommissionStatusWithdrawn      = "withdrawn"
)

// ??????????
const (
	AffiliateCommissionTypeOrder = "order"
)

// ??????????
const (
	AffiliateWithdrawStatusPendingReview = "pending_review"
	AffiliateWithdrawStatusRejected      = "rejected"
	AffiliateWithdrawStatusPaid          = "paid"
)

// ????????????
const (
	AffiliateWithdrawActionReject = "reject"
	AffiliateWithdrawActionPay    = "pay"
)

// ???????
const (
	EpayTradeStatusSuccess = "TRADE_SUCCESS"
	EpayCallbackSuccess    = "success"
	EpayCallbackFail       = "fail"
	EpayPayTypeQRCode      = "qrcode"
)

// ???????
const (
	AlipayTradeStatusSuccess      = "TRADE_SUCCESS"
	AlipayTradeStatusFinished     = "TRADE_FINISHED"
	AlipayTradeStatusClosed       = "TRADE_CLOSED"
	AlipayTradeStatusWaitBuyerPay = "WAIT_BUYER_PAY"
	AlipayCallbackSuccess         = "success"
	AlipayCallbackFail            = "fail"
)

// EPUSDT ????
const (
	EpusdtCallbackSuccess = "success"
	EpusdtCallbackFail    = "fail"
)

// OKPAY ????
const (
	OkpayCallbackSuccess = `{"status":"success"}`
	OkpayCallbackFail    = `{"status":"fail"}`
)

// TokenPay ????
const (
	TokenPayCallbackSuccess = "ok"
	TokenPayCallbackFail    = "fail"
)

// ??????
const (
	PostTypeBlog   = "blog"
	PostTypeNotice = "notice"
)

// ????????
const (
	ProductPurchaseGuest  = "guest"
	ProductPurchaseMember = "member"
)

// ????????
const (
	ProductStockStatusUnlimited  = "unlimited"
	ProductStockStatusInStock    = "in_stock"
	ProductStockStatusLowStock   = "low_stock"
	ProductStockStatusOutOfStock = "out_of_stock"
)

// ??????
const (
	ManualStockUnlimited = -1
)

// ???????
const (
	CouponTypeFixed   = "fixed"
	CouponTypePercent = "percent"
)

// ???????
const (
	PromotionTypeFixed        = "fixed"
	PromotionTypePercent      = "percent"
	PromotionTypeSpecialPrice = "special_price"
)

// ??????
const (
	ScopeTypeProduct = "product"
)

// ??????
const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
)

// ??????????
const (
	UserOAuthProviderTelegram = "telegram"
)

// ????????
const (
	LoginLogStatusSuccess = "success"
	LoginLogStatusFailed  = "failed"
)

// ??????????
const (
	LoginLogFailReasonBadRequest           = "bad_request"
	LoginLogFailReasonCaptchaRequired      = "captcha_required"
	LoginLogFailReasonCaptchaInvalid       = "captcha_invalid"
	LoginLogFailReasonCaptchaConfigInvalid = "captcha_config_invalid"
	LoginLogFailReasonCaptchaVerifyFailed  = "captcha_verify_failed"
	LoginLogFailReasonInvalidEmail         = "invalid_email"
	LoginLogFailReasonInvalidCredentials   = "invalid_credentials"
	LoginLogFailReasonEmailNotVerified     = "email_not_verified"
	LoginLogFailReasonUserDisabled         = "user_disabled"
	LoginLogFailReasonTelegramInvalid      = "telegram_invalid"
	LoginLogFailReasonTelegramExpired      = "telegram_expired"
	LoginLogFailReasonTelegramReplayed     = "telegram_replayed"
	LoginLogFailReasonTelegramConfig       = "telegram_config_invalid"
	LoginLogFailReasonInternalError        = "internal_error"
)

// ????????
const (
	LoginLogSourceWeb      = "web"
	LoginLogSourceTelegram = "telegram"
)

// ???????
const (
	VerifyPurposeRegister       = "register"
	VerifyPurposeReset          = "reset"
	VerifyPurposeTelegramBind   = "telegram_bind"
	VerifyPurposeChangeEmailOld = "change_email_old"
	VerifyPurposeChangeEmailNew = "change_email_new"
)

// ????????
const (
	CaptchaProviderNone      = "none"
	CaptchaProviderImage     = "image"
	CaptchaProviderTurnstile = "turnstile"
)

// ?????????
const (
	CaptchaSceneLogin            = "login"
	CaptchaSceneRegisterSendCode = "register_send_code"
	CaptchaSceneResetSendCode    = "reset_send_code"
	CaptchaSceneGuestCreateOrder = "guest_create_order"
	CaptchaSceneGiftCardRedeem   = "gift_card_redeem"
)

// ????????
const (
	NotificationEventWalletRechargeSuccess    = "wallet_recharge_success"
	NotificationEventOrderPaidSuccess         = "order_paid_success"
	NotificationEventManualFulfillmentPending = "manual_fulfillment_pending"
	NotificationEventExceptionAlert           = "exception_alert"
	NotificationEventExceptionAlertCheck      = "exception_alert_check"
)

// ????????????
const (
	NotificationAlertTypeOutOfStockProducts = "out_of_stock_products"
	NotificationAlertTypeLowStockProducts   = "low_stock_products"
	NotificationAlertTypePendingOrders      = "pending_payment_orders"
	NotificationAlertTypePaymentsFailed     = "payments_failed"
)

// ????
const (
	QueueDefault                    = "default"
	TaskOrderStatusEmail            = "order:status_email"
	TaskOrderAutoFulfill            = "order:auto_fulfill"
	TaskOrderTimeoutCancel          = "order:timeout_cancel"
	TaskWalletRechargeExpire        = "wallet_recharge:timeout_expire"
	TaskNotificationDispatch        = "notification:dispatch"
	TaskAffiliateConfirmCommissions = "affiliate:confirm_commissions"
	TaskProcurementSubmit           = "procurement:submit"
	TaskProcurementPollStatus       = "procurement:poll_status"
	TaskProcurementSyncAccepted     = "procurement:sync_accepted"
	TaskUpstreamSyncProducts        = "upstream:sync_products"
	TaskUpstreamSyncStock           = "upstream:sync_stock"
	TaskReconciliationRun           = "reconciliation:run"
	TaskDownstreamCallback          = "downstream:callback"
	TaskBotNotify                   = "bot:notify"
	TaskTelegramBroadcast           = "telegram:broadcast"
)

// Telegram Bot ????
const (
	TelegramBroadcastRecipientTypeAll      = "all"
	TelegramBroadcastRecipientTypeSpecific = "specific"
	TelegramBroadcastStatusPending         = "pending"
	TelegramBroadcastStatusRunning         = "running"
	TelegramBroadcastStatusCompleted       = "completed"
	TelegramBroadcastStatusFailed          = "failed"
)

// ???????
const (
	ProcurementStatusPending           = "pending"
	ProcurementStatusSubmitted         = "submitted"
	ProcurementStatusAccepted          = "accepted"
	ProcurementStatusRejected          = "rejected"
	ProcurementStatusFailed            = "failed"
	ProcurementStatusPartiallyRefunded = "partially_refunded"
	ProcurementStatusFulfilled         = "fulfilled"
	ProcurementStatusCompleted         = "completed"
	ProcurementStatusRefunded          = "refunded"
	ProcurementStatusCanceled          = "canceled"
)

// ????????
const (
	ConnectionStatusPending  = "pending"
	ConnectionStatusActive   = "active"
	ConnectionStatusDisabled = "disabled"
)

// ????????
const (
	ConnectionProtocolDujiaoNext = "dujiao-next"
)

// API ??????
const (
	ApiCredentialStatusPendingReview = "pending_review"
	ApiCredentialStatusApproved      = "approved"
	ApiCredentialStatusRejected      = "rejected"
	ApiCredentialStatusDisabled      = "disabled"
)

// ????????
const (
	CallbackStatusPending = "pending"
	CallbackStatusSent    = "sent"
	CallbackStatusFailed  = "failed"
)

// ??????
const (
	ReconciliationTypeStatus = "status"
	ReconciliationTypeAmount = "amount"
	ReconciliationTypeFull   = "full"
)

// ????????
const (
	ReconciliationJobStatusPending   = "pending"
	ReconciliationJobStatusRunning   = "running"
	ReconciliationJobStatusCompleted = "completed"
	ReconciliationJobStatusFailed    = "failed"
)

// ????????
const (
	RedisPrefixDefault = "dj"
)

// ?????
const (
	SettingKeySiteConfig               = "site_config"
	SettingKeyOrderConfig              = "order_config"
	SettingKeySMTPConfig               = "smtp_config"
	SettingKeyCaptchaConfig            = "captcha_config"
	SettingKeyTelegramAuthConfig       = "telegram_auth_config"
	SettingKeyDashboardConfig          = "dashboard_config"
	SettingKeyNotificationCenterConfig = "notification_center_config"
	SettingKeyAffiliateConfig          = "affiliate_config"
	SettingKeyTelegramBotConfig        = "telegram_bot_config"
	SettingKeyTelegramBotRuntimeStatus = "telegram_bot_runtime_status"
	SettingKeyOrderEmailTemplateConfig = "order_email_template_config"
	SettingFieldSiteCurrency           = "currency"
	SettingFieldPaymentExpireMinutes   = "payment_expire_minutes"

	SettingKeyNavConfig = "nav_config"

	SettingKeyWalletConfig        = "wallet_config"
	SettingFieldWalletOnlyPayment = "wallet_only_payment"

	SettingKeyRegistrationConfig         = "registration_config"
	SettingFieldRegistrationEnabled      = "registration_enabled"
	SettingFieldEmailVerificationEnabled = "email_verification_enabled"

	SettingKeyOrderRiskControlConfig = "order_risk_control_config"

	SettingKeyCallbackRoutesConfig = "callback_routes_config"
	SettingFieldPaymentCallback    = "payment_callback"
	SettingFieldPaypalWebhook      = "paypal_webhook"
	SettingFieldStripeWebhook      = "stripe_webhook"
	SettingFieldUpstreamCallback   = "upstream_callback"

	// ????????
	DefaultPaymentCallbackPath  = "/api/v1/payments/callback"
	DefaultPaypalWebhookPath    = "/api/v1/payments/webhook/paypal"
	DefaultStripeWebhookPath    = "/api/v1/payments/webhook/stripe"
	DefaultUpstreamCallbackPath = "/api/v1/upstream/callback"
)

// ????
const (
	SiteCurrencyDefault = "CNY"
)

// ??????
const (
	LocaleZhCN = "zh-CN"
	LocaleZhTW = "zh-TW"
	LocaleEnUS = "en-US"
)

// ?????????(?????)
var SupportedLocales = []string{LocaleZhCN, LocaleZhTW, LocaleEnUS}

// ????????
const (
	NotificationBizTypeOrder           = "order"
	NotificationBizTypeWalletRecharge  = "wallet_recharge"
	NotificationBizTypeDashboardAlert  = "dashboard_alert"
	NotificationBizTypePaymentCallback = "payment_callback"
	NotificationBizTypeProcurement     = "procurement"
	NotificationBizTypeReconciliation  = "reconciliation"
)

// ????????
const (
	MismatchTypeStatus = "status"
	MismatchTypeAmount = "amount"
	MismatchTypeBoth   = "both"
)

// ????????
const (
	CardSecretSourceManual = "manual"
	CardSecretSourceCSV    = "csv"
)

// ??????
const (
	ExportFormatCSV = "csv"
	ExportFormatTXT = "txt"
)

// Banner ????
const (
	BannerPositionHomeHero = "home_hero"
)

// Banner ??????
const (
	BannerLinkTypeNone     = "none"
	BannerLinkTypeInternal = "internal"
	BannerLinkTypeExternal = "external"
)
