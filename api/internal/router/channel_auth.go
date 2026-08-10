package router

import (
	"io"
	"net/http"
	"time"

	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/i18n"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/provider"
	"github.com/dujiao-next/internal/service"
	"github.com/dujiao-next/internal/upstream"

	"github.com/gin-gonic/gin"
)

const (
	channelClientIDKey = "channel_client_id"
	channelKeyCtxKey   = "channel_key"
	channelTypeCtxKey  = "channel_type"

	channelHeaderKey       = "Dujiao-Next-Channel-Key"
	channelHeaderTimestamp = "Dujiao-Next-Channel-Timestamp"
	channelHeaderSignature = "Dujiao-Next-Channel-Signature"
)

// ChannelAPIAuthMiddleware ?? API ???????
func ChannelAPIAuthMiddleware(container *provider.Container) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelKey := c.GetHeader(channelHeaderKey)
		timestampStr := c.GetHeader(channelHeaderTimestamp)
		signature := c.GetHeader(channelHeaderSignature)

		if channelKey == "" || timestampStr == "" || signature == "" {
			response.ChannelError(c, http.StatusUnauthorized, response.CodeUnauthorized, i18n.T(i18n.ResolveLocale(c), "error.unauthorized"), "channel_client_unauthorized")
			c.Abort()
			return
		}

		timestamp, err := upstream.ParseTimestamp(timestampStr)
		if err != nil {
			response.ChannelError(c, http.StatusUnauthorized, response.CodeUnauthorized, i18n.T(i18n.ResolveLocale(c), "error.unauthorized"), "channel_client_unauthorized")
			c.Abort()
			return
		}

		// ?? body ??????(???? 10MB ??????)
		var body []byte
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
			body, err = io.ReadAll(c.Request.Body)
			if err != nil {
				response.ChannelError(c, http.StatusBadRequest, response.CodeBadRequest, i18n.T(i18n.ResolveLocale(c), "error.bad_request"), "validation_error")
				c.Abort()
				return
			}
			// ?? body ??? handler ??
			c.Request.Body = io.NopCloser(&bodyReader{data: body})
		}

		method := c.Request.Method
		path := c.Request.URL.Path

		client, err := container.ChannelClientService.VerifyChannelSignature(
			channelKey, signature, timestamp, method, path, body,
		)
		if err != nil {
			switch err {
			case service.ErrChannelTimestampExpired:
				response.ChannelError(c, http.StatusUnauthorized, response.CodeUnauthorized, i18n.T(i18n.ResolveLocale(c), "error.unauthorized"), "channel_client_unauthorized")
			case service.ErrChannelClientNotFound:
				response.ChannelError(c, http.StatusUnauthorized, response.CodeUnauthorized, i18n.T(i18n.ResolveLocale(c), "error.unauthorized"), "channel_client_unauthorized")
			case service.ErrChannelClientDisabled:
				response.ChannelError(c, http.StatusForbidden, response.CodeForbidden, i18n.T(i18n.ResolveLocale(c), "error.forbidden"), "channel_client_disabled")
			case service.ErrChannelSignatureInvalid:
				response.ChannelError(c, http.StatusUnauthorized, response.CodeUnauthorized, i18n.T(i18n.ResolveLocale(c), "error.unauthorized"), "channel_client_unauthorized")
			default:
				logger.Errorw("channel_auth_error", "error", err)
				response.ChannelError(c, http.StatusInternalServerError, response.CodeInternal, i18n.T(i18n.ResolveLocale(c), "error.internal_error"), "internal_error")
			}
			c.Abort()
			return
		}

		// ???? last_used_at
		now := time.Now()
		go func() {
			client.LastUsedAt = &now
			if updateErr := container.ChannelClientRepo.Update(client); updateErr != nil {
				logger.Warnw("channel_auth_update_last_used_failed", "error", updateErr)
			}
		}()

		// ?? context keys
		c.Set(channelClientIDKey, client.ID)
		c.Set(channelKeyCtxKey, client.ChannelKey)
		c.Set(channelTypeCtxKey, client.ChannelType)

		c.Next()
	}
}
