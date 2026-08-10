package router

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/dujiao-next/internal/authz"
	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	adminhandlers "github.com/dujiao-next/internal/http/handlers/admin"
	channelhandlers "github.com/dujiao-next/internal/http/handlers/channel"
	publichandlers "github.com/dujiao-next/internal/http/handlers/public"
	upstreamhandlers "github.com/dujiao-next/internal/http/handlers/upstream"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/provider"

	"github.com/gin-gonic/gin"
)

// SetupRouter ?????
func SetupRouter(cfg *config.Config, c *provider.Container) *gin.Engine {
	log := logger.L
	if log == nil {
		log = logger.Init(cfg.Server.Mode, cfg.Log.ToLoggerOptions())
	}
	r := gin.New()

	// ??? Handler(???/????)
	publicHandler := publichandlers.New(c)
	adminHandler := adminhandlers.New(c)
	channelHandler := channelhandlers.New(c)
	upstreamHandler := upstreamhandlers.New(c, c.DownstreamOrderRefRepo)
	redisPrefix := strings.TrimSpace(cfg.Redis.Prefix)
	if redisPrefix == "" {
		redisPrefix = constants.RedisPrefixDefault
	}
	redisClient := cache.Client()
	loginRule := RateLimitRule{
		Prefix:        fmt.Sprintf("%s:rate:login", redisPrefix),
		WindowSeconds: cfg.Security.LoginRateLimit.WindowSeconds,
		MaxRequests:   cfg.Security.LoginRateLimit.MaxAttempts,
		BlockSeconds:  cfg.Security.LoginRateLimit.BlockSeconds,
		MessageKey:    "error.login_too_many",
	}
	adminLoginRule := RateLimitRule{
		Prefix:        fmt.Sprintf("%s:rate:admin_login", redisPrefix),
		WindowSeconds: cfg.Security.LoginRateLimit.WindowSeconds,
		MaxRequests:   cfg.Security.LoginRateLimit.MaxAttempts,
		BlockSeconds:  cfg.Security.LoginRateLimit.BlockSeconds,
		MessageKey:    "error.login_too_many",
	}
	upstreamAPIRule := RateLimitRule{
		Prefix:        fmt.Sprintf("%s:rate:upstream_api", redisPrefix),
		WindowSeconds: 60,
		MaxRequests:   60,
		BlockSeconds:  30,
		MessageKey:    "error.rate_limited",
	}

	// ???
	r.Use(gin.Recovery())
	r.Use(RequestIDMiddleware())
	r.Use(LoggerMiddleware(log))
	r.Use(CORSMiddleware(cfg.CORS))
	r.Use(CallbackRouteMiddleware(c.SettingService, publicHandler, upstreamHandler))

	// ??????(?????)- ???????
	r.Static("/uploads", "./uploads")

	// API ???
	apiV1 := r.Group("/api/v1")
	{
		// ????
		public := apiV1.Group("/public")
		{
			public.GET("/config", publicHandler.GetConfig)
			public.GET("/products", publicHandler.GetProducts)
			public.GET("/products/:slug", publicHandler.GetProductBySlug)
			public.GET("/posts", publicHandler.GetPosts)
			public.GET("/posts/:slug", publicHandler.GetPostBySlug)
			public.GET("/banners", publicHandler.GetPublicBanners)
			public.GET("/categories", publicHandler.GetCategories)
			public.GET("/captcha/image", publicHandler.GetImageCaptcha)
			public.POST("/affiliate/click", publicHandler.TrackAffiliateClick)
			public.GET("/member-levels", publicHandler.GetPublicMemberLevels)
		}

		// ????
		guest := apiV1.Group("/guest")
		{
			guest.POST("/orders", publicHandler.CreateGuestOrder)
			guest.POST("/orders/create-and-pay", publicHandler.CreateGuestOrderAndPay)
			guest.POST("/orders/preview", publicHandler.PreviewGuestOrder)
			guest.GET("/orders/lookup", publicHandler.LookupGuestOrder)
			guest.GET("/orders", publicHandler.ListGuestOrders)
			guest.GET("/orders/:order_no", publicHandler.GetGuestOrderByOrderNo)
			guest.GET("/orders/:order_no/fulfillment/download", publicHandler.DownloadGuestFulfillment)
			guest.POST("/payments", publicHandler.CreateGuestPayment)
			guest.POST("/payments/:id/capture", publicHandler.CaptureGuestPayment)
			guest.GET("/payments/latest", publicHandler.GetGuestLatestPayment)
		}

		// ??????
		auth := apiV1.Group("/auth")
		{
			auth.POST("/send-verify-code", publicHandler.SendUserVerifyCode)
			auth.POST("/register", publicHandler.UserRegister)
			auth.POST("/login", RateLimitMiddleware(redisClient, loginRule, KeyByIPAndJSONField("email")), publicHandler.UserLogin)
			auth.POST("/telegram/login", RateLimitMiddleware(redisClient, loginRule, KeyByIP), publicHandler.UserTelegramLogin)
			auth.POST("/telegram/miniapp/login", RateLimitMiddleware(redisClient, loginRule, KeyByIP), publicHandler.UserTelegramMiniAppLogin)
			auth.POST("/forgot-password", publicHandler.UserForgotPassword)
		}

		// ????(???)
		user := apiV1.Group("")
		user.Use(UserJWTAuthMiddleware(cfg.UserJWT.SecretKey, c.UserRepo))
		{
			user.GET("/me", publicHandler.GetCurrentUser)
			user.GET("/me/login-logs", publicHandler.GetMyLoginLogs)
			user.PUT("/me/profile", publicHandler.UpdateUserProfile)
			user.PUT("/me/password", publicHandler.ChangeUserPassword)
			user.GET("/me/telegram", publicHandler.GetMyTelegramBinding)
			user.POST("/me/telegram/bind", publicHandler.BindMyTelegram)
			user.POST("/me/telegram/miniapp/bind", publicHandler.BindMyTelegramMiniApp)
			user.DELETE("/me/telegram/unbind", publicHandler.UnbindMyTelegram)
			user.POST("/me/email/send-verify-code", publicHandler.SendChangeEmailCode)
			user.POST("/me/email/change", publicHandler.ChangeEmail)
			user.GET("/cart", publicHandler.GetCart)
			user.POST("/cart/items", publicHandler.UpsertCartItem)
			user.DELETE("/cart/items/:product_id", publicHandler.DeleteCartItem)
			user.POST("/orders", publicHandler.CreateOrder)
			user.POST("/orders/create-and-pay", publicHandler.CreateOrderAndPay)
			user.POST("/orders/preview", publicHandler.PreviewOrder)
			user.POST("/order/payment-channels", publicHandler.GetOrderPaymentChannels)
			user.GET("/orders", publicHandler.ListOrders)
			user.GET("/orders/:order_no", publicHandler.GetOrderByOrderNo)
			user.GET("/orders/:order_no/fulfillment/download", publicHandler.DownloadFulfillment)
			user.POST("/orders/:order_no/cancel", publicHandler.CancelOrder)
			user.POST("/payments", publicHandler.CreatePayment)
			user.POST("/payments/:id/capture", publicHandler.CapturePayment)
			user.GET("/payments/latest", publicHandler.GetLatestPayment)
			user.GET("/wallet", publicHandler.GetMyWallet)
			user.GET("/wallet/transactions", publicHandler.GetMyWalletTransactions)
			user.POST("/wallet/payment-channels", publicHandler.GetMyWalletPaymentChannels)
			user.POST("/wallet/recharge", publicHandler.RechargeWallet)
			user.GET("/wallet/recharges", publicHandler.ListMyWalletRecharges)
			user.GET("/wallet/recharges/:recharge_no", publicHandler.GetMyWalletRecharge)
			user.POST("/wallet/recharge/payments/:id/capture", publicHandler.CaptureMyWalletRechargePayment)
			user.POST("/gift-cards/redeem", publicHandler.RedeemGiftCard)
			user.POST("/affiliate/open", publicHandler.OpenAffiliate)
			user.GET("/affiliate/dashboard", publicHandler.GetAffiliateDashboard)
			user.GET("/affiliate/commissions", publicHandler.ListAffiliateCommissions)
			user.GET("/affiliate/withdraws", publicHandler.ListAffiliateWithdraws)
			user.POST("/affiliate/withdraws", publicHandler.ApplyAffiliateWithdraw)

			// API ????(????)
			user.GET("/api-credential", publicHandler.GetMyApiCredential)
			user.POST("/api-credential/apply", publicHandler.ApplyApiCredential)
			user.POST("/api-credential/regenerate", publicHandler.RegenerateMyApiCredential)
			user.PUT("/api-credential/status", publicHandler.UpdateMyApiCredentialStatus)
		}

		// ?? API(???? B ??,????? A ??)
		upstreamAPI := apiV1.Group("/upstream")
		upstreamAPI.Use(RateLimitMiddleware(redisClient, upstreamAPIRule, KeyByUpstreamApiKey))
		upstreamAPI.Use(UpstreamAPIAuthMiddleware(c.ApiCredentialRepo))
		{
			upstreamAPI.POST("/ping", upstreamHandler.Ping)
			upstreamAPI.GET("/categories", upstreamHandler.ListCategories)
			upstreamAPI.GET("/products", upstreamHandler.ListProducts)
			upstreamAPI.GET("/products/:id", upstreamHandler.GetProduct)
			upstreamAPI.POST("/orders", upstreamHandler.CreateOrder)
			upstreamAPI.GET("/orders/:id", upstreamHandler.GetOrder)
			upstreamAPI.POST("/orders/:id/cancel", upstreamHandler.CancelOrder)
		}

		// ??????(???? A ??,?? B ???)
		apiV1.POST("/upstream/callback", upstreamHandler.HandleCallback)

		// ?? API(Telegram Bot ???????)
		channelAPI := apiV1.Group("/channel")
		channelAPI.Use(ChannelAPIAuthMiddleware(c))
		{
			channelAPI.GET("/telegram/config", channelHandler.GetBotConfig)
			channelAPI.POST("/telegram/heartbeat", channelHandler.ReportHeartbeat)
			channelAPI.POST("/identities/telegram/resolve", channelHandler.ResolveTelegramIdentity)
			channelAPI.POST("/identities/telegram/provision", channelHandler.ProvisionTelegramIdentity)
			channelAPI.POST("/identities/telegram/bind", channelHandler.BindTelegramIdentity)
			channelAPI.GET("/me", channelHandler.GetCurrentIdentity)
			channelAPI.POST("/affiliate/click", channelHandler.TrackAffiliateClick)
			channelAPI.POST("/affiliate/open", channelHandler.OpenAffiliate)
			channelAPI.GET("/affiliate/dashboard", channelHandler.GetAffiliateDashboard)
			channelAPI.GET("/affiliate/commissions", channelHandler.ListAffiliateCommissions)
			channelAPI.GET("/affiliate/withdraws", channelHandler.ListAffiliateWithdraws)
			channelAPI.POST("/affiliate/withdraws", channelHandler.ApplyAffiliateWithdraw)

			// Catalog ??(????)
			channelAPI.GET("/catalog/categories", channelHandler.GetCategories)
			channelAPI.GET("/catalog/products", channelHandler.GetProducts)
			channelAPI.GET("/catalog/products/:id", channelHandler.GetProductDetail)
			channelAPI.GET("/member-levels", channelHandler.GetMemberLevels)

			// Order / Payment ??(????)
			channelAPI.POST("/orders/preview", channelHandler.PreviewOrder)
			channelAPI.POST("/orders", channelHandler.CreateOrder)
			channelAPI.GET("/orders", channelHandler.ListOrders)
			channelAPI.GET("/orders/by-order-no/:order_no", channelHandler.GetOrderByOrderNo)
			channelAPI.GET("/orders/:id", channelHandler.GetOrderStatus)
			channelAPI.POST("/orders/:id/cancel", channelHandler.CancelOrder)
			channelAPI.GET("/payment-channels", channelHandler.GetPaymentChannels)
			channelAPI.GET("/payment-methods", channelHandler.GetPaymentChannels)
			channelAPI.GET("/payments/latest", channelHandler.GetLatestPayment)
			channelAPI.GET("/payments/:id", channelHandler.GetPaymentDetail)
			channelAPI.POST("/payments", channelHandler.CreatePayment)

			// Wallet ??(??)
			channelAPI.GET("/wallet", channelHandler.GetWallet)
			channelAPI.GET("/wallet/transactions", channelHandler.GetWalletTransactions)
			channelAPI.POST("/wallet/gift-card/redeem", channelHandler.RedeemGiftCard)
			channelAPI.POST("/wallet/recharge", channelHandler.CreateWalletRecharge)
		}

		apiV1.POST("/payments/callback", publicHandler.PaymentCallback)
		apiV1.GET("/payments/callback", publicHandler.PaymentCallback)
		apiV1.POST("/payment/alipay/notify", publicHandler.PaymentCallback)
		apiV1.GET("/payment/alipay/notify", publicHandler.PaymentCallback)
		apiV1.POST("/payments/webhook/paypal", publicHandler.PaypalWebhook)
		apiV1.POST("/payments/webhook/stripe", publicHandler.StripeWebhook)

		// ?????
		admin := apiV1.Group("/admin")
		{
			// ????(????)
			admin.POST("/login", RateLimitMiddleware(redisClient, adminLoginRule, KeyByIP), adminHandler.AdminLogin)

			// ???????
			authorized := admin.Use(JWTAuthMiddleware(cfg.JWT.SecretKey, c.AdminRepo), AdminRBACMiddleware(c.AuthzService))
			{
				// ???
				authorized.GET("/dashboard/overview", adminHandler.GetDashboardOverview)
				authorized.GET("/dashboard/trends", adminHandler.GetDashboardTrends)
				authorized.GET("/dashboard/rankings", adminHandler.GetDashboardRankings)
				authorized.GET("/dashboard/inventory-alerts", adminHandler.GetDashboardInventoryAlerts)

				// ????
				authorized.GET("/ads/render/:slotCode", adminHandler.GetAdRender)
				authorized.POST("/ads/impression", adminHandler.PostAdImpression)

				// ????
				authorized.GET("/products", adminHandler.GetAdminProducts)
				authorized.GET("/products/:id", adminHandler.GetAdminProduct)
				authorized.POST("/products", adminHandler.CreateProduct)
				authorized.PUT("/products/:id", adminHandler.UpdateProduct)
				authorized.PATCH("/products/:id", adminHandler.QuickUpdateProduct)
				authorized.DELETE("/products/:id", adminHandler.DeleteProduct)
				authorized.POST("/products/batch-status", adminHandler.BatchUpdateProductStatus)
				authorized.POST("/products/batch-category", adminHandler.BatchUpdateProductCategory)
				authorized.POST("/products/batch-delete", adminHandler.BatchDeleteProducts)
				authorized.GET("/product-skus", adminHandler.ListProductSKUs)

				// ????
				authorized.GET("/posts", adminHandler.GetAdminPosts)
				authorized.POST("/posts", adminHandler.CreatePost)
				authorized.PUT("/posts/:id", adminHandler.UpdatePost)
				authorized.DELETE("/posts/:id", adminHandler.DeletePost)

				// Banner ??
				authorized.GET("/banners", adminHandler.GetAdminBanners)
				authorized.GET("/banners/:id", adminHandler.GetAdminBanner)
				authorized.POST("/banners", adminHandler.CreateBanner)
				authorized.PUT("/banners/:id", adminHandler.UpdateBanner)
				authorized.DELETE("/banners/:id", adminHandler.DeleteBanner)

				// ????
				authorized.GET("/categories", adminHandler.GetAdminCategories)
				authorized.POST("/categories", adminHandler.CreateCategory)
				authorized.PUT("/categories/:id", adminHandler.UpdateCategory)
				authorized.DELETE("/categories/:id", adminHandler.DeleteCategory)

				// ????
				authorized.GET("/settings", adminHandler.GetSettings)
				authorized.PUT("/settings", adminHandler.UpdateSettings)
				authorized.GET("/settings/smtp", adminHandler.GetSMTPSettings)
				authorized.PUT("/settings/smtp", adminHandler.UpdateSMTPSettings)
				authorized.POST("/settings/smtp/test", adminHandler.TestSMTPSettings)
				authorized.GET("/settings/captcha", adminHandler.GetCaptchaSettings)
				authorized.PUT("/settings/captcha", adminHandler.UpdateCaptchaSettings)
				authorized.GET("/settings/telegram-auth", adminHandler.GetTelegramAuthSettings)
				authorized.PUT("/settings/telegram-auth", adminHandler.UpdateTelegramAuthSettings)
				authorized.GET("/settings/notification-center", adminHandler.GetNotificationCenterSettings)
				authorized.PUT("/settings/notification-center", adminHandler.UpdateNotificationCenterSettings)
				authorized.GET("/settings/notification-center/logs", adminHandler.ListNotificationLogs)
				authorized.POST("/settings/notification-center/test", adminHandler.TestNotificationCenterSettings)
				authorized.GET("/settings/notifications", adminHandler.GetNotificationCenterSettings)
				authorized.PUT("/settings/notifications", adminHandler.UpdateNotificationCenterSettings)
				authorized.GET("/settings/notifications/logs", adminHandler.ListNotificationLogs)
				authorized.POST("/settings/notifications/test", adminHandler.TestNotificationCenterSettings)
				authorized.GET("/settings/order-email-template", adminHandler.GetOrderEmailTemplateSettings)
				authorized.PUT("/settings/order-email-template", adminHandler.UpdateOrderEmailTemplateSettings)
				authorized.POST("/settings/order-email-template/reset", adminHandler.ResetOrderEmailTemplateSettings)
				authorized.GET("/settings/affiliate", adminHandler.GetAffiliateSettings)
				authorized.PUT("/settings/affiliate", adminHandler.UpdateAffiliateSettings)
				authorized.PUT("/password", adminHandler.UpdateAdminPassword) // ????

				// ????
				authorized.GET("/affiliates/users", adminHandler.ListAffiliateUsers)
				authorized.PATCH("/affiliates/users/:id/status", adminHandler.UpdateAffiliateUserStatus)
				authorized.PATCH("/affiliates/users/batch-status", adminHandler.BatchUpdateAffiliateUserStatus)
				authorized.GET("/affiliates/commissions", adminHandler.ListAffiliateCommissions)
				authorized.GET("/affiliates/withdraws", adminHandler.ListAffiliateWithdraws)
				authorized.POST("/affiliates/withdraws/:id/reject", adminHandler.RejectAffiliateWithdraw)
				authorized.POST("/affiliates/withdraws/:id/pay", adminHandler.PayAffiliateWithdraw)

				// ????
				authorized.GET("/authz/me", adminHandler.GetAuthzMe)
				authorized.GET("/authz/roles", adminHandler.ListAuthzRoles)
				authorized.GET("/authz/admins", adminHandler.ListAuthzAdmins)
				authorized.GET("/authz/audit-logs", adminHandler.ListAuthzAuditLogs)
				authorized.POST("/authz/admins", adminHandler.CreateAuthzAdmin)
				authorized.PUT("/authz/admins/:id", adminHandler.UpdateAuthzAdmin)
				authorized.DELETE("/authz/admins/:id", adminHandler.DeleteAuthzAdmin)
				authorized.GET("/authz/permissions/catalog", func(ctx *gin.Context) {
					response.Success(ctx, buildAdminPermissionCatalog(r))
				})
				authorized.POST("/authz/roles", adminHandler.CreateAuthzRole)
				authorized.DELETE("/authz/roles/:role", adminHandler.DeleteAuthzRole)
				authorized.GET("/authz/roles/:role/policies", adminHandler.GetAuthzRolePolicies)
				authorized.POST("/authz/policies", adminHandler.GrantAuthzPolicy)
				authorized.DELETE("/authz/policies", adminHandler.RevokeAuthzPolicy)
				authorized.GET("/authz/admins/:id/roles", adminHandler.GetAuthzAdminRoles)
				authorized.PUT("/authz/admins/:id/roles", adminHandler.SetAuthzAdminRoles)

				// ????
				authorized.POST("/upload", adminHandler.UploadFile)

				// ????
				authorized.GET("/media", adminHandler.GetAdminMedia)
				authorized.PUT("/media/:id", adminHandler.UpdateMedia)
				authorized.DELETE("/media/:id", adminHandler.DeleteMedia)

				// ????
				authorized.GET("/orders", adminHandler.AdminListOrders)
				authorized.GET("/orders/export", adminHandler.AdminExportOrders)
				authorized.GET("/orders/:id", adminHandler.AdminGetOrder)
				authorized.GET("/orders/:id/fulfillment/download", adminHandler.AdminDownloadFulfillment)
				authorized.PATCH("/orders/:id", adminHandler.AdminUpdateOrderStatus)
				authorized.POST("/orders/:id/refund-to-wallet", adminHandler.AdminRefundOrderToWallet)
				authorized.POST("/orders/:id/manual-refund", adminHandler.AdminManualRefundOrder)
				authorized.GET("/order-refunds", adminHandler.GetAdminOrderRefunds)
				authorized.GET("/order-refunds/:id", adminHandler.GetAdminOrderRefund)
				authorized.POST("/fulfillments", adminHandler.AdminCreateFulfillment)
				authorized.POST("/card-secrets/batch", adminHandler.CreateCardSecretBatch)
				authorized.POST("/card-secrets/import", adminHandler.ImportCardSecretCSV)
				authorized.GET("/card-secrets", adminHandler.GetCardSecrets)
				authorized.PUT("/card-secrets/:id", adminHandler.UpdateCardSecret)
				authorized.PATCH("/card-secrets/batch-status", adminHandler.BatchUpdateCardSecretStatus)
				authorized.POST("/card-secrets/batch-delete", adminHandler.BatchDeleteCardSecrets)
				authorized.POST("/card-secrets/export", adminHandler.ExportCardSecrets)
				authorized.GET("/card-secrets/stats", adminHandler.GetCardSecretStats)
				authorized.GET("/card-secrets/batches", adminHandler.GetCardSecretBatches)
				authorized.GET("/card-secrets/template", adminHandler.GetCardSecretTemplate)
				authorized.POST("/gift-cards/generate", adminHandler.GenerateGiftCards)
				authorized.GET("/gift-cards", adminHandler.GetGiftCards)
				authorized.PUT("/gift-cards/:id", adminHandler.UpdateGiftCard)
				authorized.DELETE("/gift-cards/:id", adminHandler.DeleteGiftCard)
				authorized.PATCH("/gift-cards/batch-status", adminHandler.BatchUpdateGiftCardStatus)
				authorized.POST("/gift-cards/export", adminHandler.ExportGiftCards)

				// ???????
				authorized.POST("/coupons", adminHandler.CreateCoupon)
				authorized.GET("/coupons", adminHandler.GetAdminCoupons)
				authorized.PUT("/coupons/:id", adminHandler.UpdateCoupon)
				authorized.DELETE("/coupons/:id", adminHandler.DeleteCoupon)
				authorized.POST("/promotions", adminHandler.CreatePromotion)
				authorized.GET("/promotions", adminHandler.GetAdminPromotions)
				authorized.PUT("/promotions/:id", adminHandler.UpdatePromotion)
				authorized.DELETE("/promotions/:id", adminHandler.DeletePromotion)

				// ????
				authorized.GET("/member-levels", adminHandler.GetAdminMemberLevels)
				authorized.POST("/member-levels", adminHandler.CreateMemberLevel)
				authorized.PUT("/member-levels/:id", adminHandler.UpdateMemberLevel)
				authorized.DELETE("/member-levels/:id", adminHandler.DeleteMemberLevel)
				authorized.GET("/member-level-prices", adminHandler.GetMemberLevelPrices)
				authorized.POST("/member-level-prices/batch", adminHandler.BatchUpsertMemberLevelPrices)
				authorized.DELETE("/member-level-prices/:id", adminHandler.DeleteMemberLevelPrice)
				authorized.POST("/member-levels/backfill", adminHandler.BackfillMemberLevels)

				// ?????????
				authorized.POST("/payment-channels", adminHandler.CreatePaymentChannel)
				authorized.GET("/payment-channels", adminHandler.GetPaymentChannels)
				authorized.GET("/payment-channels/:id", adminHandler.GetPaymentChannel)
				authorized.PUT("/payment-channels/:id", adminHandler.UpdatePaymentChannel)
				authorized.DELETE("/payment-channels/:id", adminHandler.DeletePaymentChannel)
				authorized.GET("/payments", adminHandler.GetAdminPayments)
				authorized.GET("/payments/export", adminHandler.ExportAdminPayments)
				authorized.GET("/payments/:id", adminHandler.GetAdminPayment)

				// ????
				authorized.GET("/users", adminHandler.GetAdminUsers)
				authorized.GET("/user-login-logs", adminHandler.GetUserLoginLogs)
				authorized.PUT("/users/batch-status", adminHandler.BatchUpdateUserStatus)
				authorized.GET("/users/:id", adminHandler.GetAdminUser)
				authorized.PUT("/users/:id", adminHandler.UpdateAdminUser)
				authorized.GET("/users/:id/coupon-usages", adminHandler.GetAdminUserCouponUsages)
				authorized.GET("/users/:id/wallet", adminHandler.GetAdminUserWallet)
				authorized.GET("/users/:id/wallet/transactions", adminHandler.GetAdminUserWalletTransactions)
				authorized.POST("/users/:id/wallet/adjust", adminHandler.AdjustAdminUserWallet)
				authorized.PUT("/users/:id/member-level", adminHandler.SetUserMemberLevel)
				authorized.GET("/wallet/recharges", adminHandler.GetAdminWalletRecharges)

				// API ??????
				authorized.GET("/api-credentials", adminHandler.GetApiCredentials)
				authorized.GET("/api-credentials/:id", adminHandler.GetApiCredential)
				authorized.POST("/api-credentials/:id/approve", adminHandler.ApproveApiCredential)
				authorized.POST("/api-credentials/:id/reject", adminHandler.RejectApiCredential)
				authorized.PUT("/api-credentials/:id/status", adminHandler.UpdateApiCredentialStatus)
				authorized.DELETE("/api-credentials/:id", adminHandler.DeleteApiCredential)

				// ????????
				authorized.GET("/site-connections", adminHandler.GetSiteConnections)
				authorized.GET("/site-connections/:id", adminHandler.GetSiteConnection)
				authorized.POST("/site-connections", adminHandler.CreateSiteConnection)
				authorized.PUT("/site-connections/:id", adminHandler.UpdateSiteConnection)
				authorized.DELETE("/site-connections/:id", adminHandler.DeleteSiteConnection)
				authorized.POST("/site-connections/:id/ping", adminHandler.PingSiteConnection)
				authorized.PUT("/site-connections/:id/status", adminHandler.UpdateSiteConnectionStatus)
				authorized.POST("/site-connections/:id/reapply-markup", adminHandler.ReapplyConnectionMarkup)

				// ??????
				authorized.GET("/product-mappings", adminHandler.GetProductMappings)
				authorized.GET("/product-mappings/:id", adminHandler.GetProductMapping)
				authorized.POST("/product-mappings/import", adminHandler.ImportUpstreamProduct)
				authorized.POST("/product-mappings/batch-import", adminHandler.BatchImportUpstreamProducts)
				authorized.POST("/product-mappings/:id/sync", adminHandler.SyncProductMapping)
				authorized.PUT("/product-mappings/:id/status", adminHandler.UpdateProductMappingStatus)
				authorized.DELETE("/product-mappings/:id", adminHandler.DeleteProductMapping)
				authorized.POST("/product-mappings/batch-sync", adminHandler.BatchSyncProductMappings)
				authorized.POST("/product-mappings/batch-status", adminHandler.BatchUpdateProductMappingStatus)
				authorized.POST("/product-mappings/batch-delete", adminHandler.BatchDeleteProductMappings)
				authorized.GET("/upstream-products", adminHandler.ListUpstreamProducts)
				authorized.GET("/upstream-categories", adminHandler.ListUpstreamCategories)
				authorized.POST("/product-mappings/batch-import-by-category", adminHandler.BatchImportByCategory)

				// ?????
				authorized.GET("/procurement-orders", adminHandler.GetProcurementOrders)
				authorized.GET("/procurement-orders/:id", adminHandler.GetProcurementOrder)
				authorized.GET("/procurement-orders/:id/upstream-payload/download", adminHandler.DownloadProcurementUpstreamPayload)
				authorized.POST("/procurement-orders/:id/retry", adminHandler.RetryProcurementOrder)
				authorized.POST("/procurement-orders/:id/cancel", adminHandler.CancelProcurementOrder)

				// ????
				authorized.POST("/reconciliation/run", adminHandler.RunReconciliation)
				authorized.GET("/reconciliation/jobs", adminHandler.GetReconciliationJobs)
				authorized.GET("/reconciliation/jobs/:id", adminHandler.GetReconciliationJob)
				authorized.PUT("/reconciliation/items/:id/resolve", adminHandler.ResolveReconciliationItem)

				// ???????
				authorized.GET("/channel-clients", adminHandler.ListChannelClients)
				authorized.POST("/channel-clients", adminHandler.CreateChannelClient)
				authorized.GET("/channel-clients/:id", adminHandler.GetChannelClient)
				authorized.PUT("/channel-clients/:id", adminHandler.UpdateChannelClient)
				authorized.PUT("/channel-clients/:id/status", adminHandler.UpdateChannelClientStatus)
				authorized.POST("/channel-clients/:id/reset-secret", adminHandler.ResetChannelClientSecret)
				authorized.DELETE("/channel-clients/:id", adminHandler.DeleteChannelClient)

				// Telegram Bot ??
				authorized.GET("/telegram-bot/broadcasts", adminHandler.ListTelegramBroadcasts)
				authorized.GET("/telegram-bot/broadcasts/:id", adminHandler.GetTelegramBroadcast)
				authorized.POST("/telegram-bot/broadcasts", adminHandler.CreateTelegramBroadcast)
				authorized.GET("/telegram-bot/users", adminHandler.ListTelegramBroadcastUsers)

				// Telegram Bot ??
				authorized.GET("/settings/telegram-bot", adminHandler.GetTelegramBotConfig)
				authorized.PUT("/settings/telegram-bot", adminHandler.UpdateTelegramBotConfig)
				authorized.GET("/settings/telegram-bot/runtime-status", adminHandler.GetTelegramBotRuntimeStatus)
			}
		}
	}

	// ????
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}

type adminPermissionCatalogItem struct {
	Module     string `json:"module"`
	Method     string `json:"method"`
	Object     string `json:"object"`
	Permission string `json:"permission"`
}

func buildAdminPermissionCatalog(engine *gin.Engine) []adminPermissionCatalogItem {
	if engine == nil {
		return []adminPermissionCatalogItem{}
	}

	routes := engine.Routes()
	seen := make(map[string]struct{}, len(routes))
	items := make([]adminPermissionCatalogItem, 0, len(routes))

	for _, item := range routes {
		method := strings.ToUpper(strings.TrimSpace(item.Method))
		if method == "" || method == http.MethodOptions || method == http.MethodHead {
			continue
		}
		if !strings.HasPrefix(item.Path, "/api/v1/admin/") {
			continue
		}
		if item.Path == "/api/v1/admin/login" {
			continue
		}
		object := authz.NormalizeObject(item.Path)
		permission := method + ":" + object
		if _, exists := seen[permission]; exists {
			continue
		}
		seen[permission] = struct{}{}
		items = append(items, adminPermissionCatalogItem{
			Module:     deriveAdminPermissionModule(object),
			Method:     method,
			Object:     object,
			Permission: permission,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Module == items[j].Module {
			if items[i].Object == items[j].Object {
				return items[i].Method < items[j].Method
			}
			return items[i].Object < items[j].Object
		}
		return items[i].Module < items[j].Module
	})

	return items
}

func deriveAdminPermissionModule(object string) string {
	normalized := strings.TrimPrefix(strings.TrimSpace(object), "/")
	if normalized == "" {
		return "system"
	}
	segments := strings.Split(normalized, "/")
	if len(segments) <= 1 {
		return segments[0]
	}
	if segments[0] != "admin" {
		return segments[0]
	}
	if segments[1] == "authz" {
		return "authz"
	}
	return segments[1]
}
