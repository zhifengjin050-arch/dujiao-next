package router

import (
	"net/http"
	"strings"

	"github.com/dujiao-next/internal/constants"
	publichandlers "github.com/dujiao-next/internal/http/handlers/public"
	upstreamhandlers "github.com/dujiao-next/internal/http/handlers/upstream"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// defaultCallbackPaths ????????,?????????????????
var defaultCallbackPaths = map[string]bool{
	constants.DefaultPaymentCallbackPath:  true,
	constants.DefaultPaypalWebhookPath:    true,
	constants.DefaultStripeWebhookPath:    true,
	constants.DefaultUpstreamCallbackPath: true,
}

// CallbackRouteMiddleware ??????????
// ???????????????:
//   - ??????? ? ????? handler
//   - ???????? ? ?? 404(??????)
//   - ???????? ? ??,????????
func CallbackRouteMiddleware(
	settingService *service.SettingService,
	publicHandler *publichandlers.Handler,
	upstreamHandler *upstreamhandlers.Handler,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := strings.TrimRight(c.Request.URL.Path, "/")
		method := c.Request.Method

		// ????:???? /api/ ?????
		if !strings.HasPrefix(path, "/api/") {
			c.Next()
			return
		}

		// ????:???? POST ? GET
		if method != http.MethodPost && method != http.MethodGet {
			c.Next()
			return
		}

		routes := settingService.GetCallbackRoutesCached()
		if routes == nil {
			// ????????,????????
			c.Next()
			return
		}

		// ?????????
		switch path {
		case routes.PaymentCallback:
			if routes.PaymentCallback != "" && (method == http.MethodPost || method == http.MethodGet) {
				publicHandler.PaymentCallback(c)
				c.Abort()
				return
			}
		case routes.PaypalWebhook:
			if routes.PaypalWebhook != "" && method == http.MethodPost {
				publicHandler.PaypalWebhook(c)
				c.Abort()
				return
			}
		case routes.StripeWebhook:
			if routes.StripeWebhook != "" && method == http.MethodPost {
				publicHandler.StripeWebhook(c)
				c.Abort()
				return
			}
		case routes.UpstreamCallback:
			if routes.UpstreamCallback != "" && method == http.MethodPost {
				upstreamHandler.HandleCallback(c)
				c.Abort()
				return
			}
		}

		// ????????(?????????????)
		if defaultCallbackPaths[path] {
			shouldBlock := false
			switch path {
			case constants.DefaultPaymentCallbackPath:
				shouldBlock = routes.PaymentCallback != ""
			case constants.DefaultPaypalWebhookPath:
				shouldBlock = routes.PaypalWebhook != ""
			case constants.DefaultStripeWebhookPath:
				shouldBlock = routes.StripeWebhook != ""
			case constants.DefaultUpstreamCallbackPath:
				shouldBlock = routes.UpstreamCallback != ""
			}
			if shouldBlock {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
		}

		c.Next()
	}
}
