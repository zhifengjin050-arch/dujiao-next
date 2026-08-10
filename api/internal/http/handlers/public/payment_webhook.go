package public

import (
	"io"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// PaypalWebhook PayPal webhook ???
func (h *Handler) PaypalWebhook(c *gin.Context) {
	log := shared.RequestLog(c)
	var query PaypalWebhookQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		log.Warnw("paypal_webhook_query_invalid", "error", err)
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Warnw("paypal_webhook_body_read_failed", "channel_id", query.ChannelID, "error", err)
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	log.Infow("paypal_webhook_received",
		"channel_id", query.ChannelID,
		"client_ip", c.ClientIP(),
		"body_size", len(body),
		"paypal_transmission_id", strings.TrimSpace(c.GetHeader("Paypal-Transmission-Id")),
		"paypal_transmission_time", strings.TrimSpace(c.GetHeader("Paypal-Transmission-Time")),
		"paypal_auth_algo", strings.TrimSpace(c.GetHeader("Paypal-Auth-Algo")),
		"paypal_cert_url", truncateCallbackLogValue(strings.TrimSpace(c.GetHeader("Paypal-Cert-Url"))),
		"paypal_transmission_sig", truncateCallbackLogValue(strings.TrimSpace(c.GetHeader("Paypal-Transmission-Sig"))),
		"raw_body", callbackRawBodyForLog(body),
	)
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) == 0 {
			continue
		}
		headers[key] = values[0]
	}
	payment, eventType, err := h.PaymentService.HandlePaypalWebhook(service.WebhookCallbackInput{
		ChannelID: query.ChannelID,
		Headers:   headers,
		Body:      body,
		Context:   c.Request.Context(),
	})
	if err != nil {
		log.Warnw("paypal_webhook_handle_failed",
			"channel_id", query.ChannelID,
			"event_type", eventType,
			"error", err,
		)
		h.enqueuePaymentExceptionAlert(c, models.JSON{
			"alert_type":  "paypal_webhook_handle_failed",
			"alert_level": "error",
			"message":     strings.TrimSpace(err.Error()),
			"provider":    constants.PaymentChannelTypePaypal,
		})
		respondPaymentCallbackError(c, err)
		return
	}

	if payment == nil {
		log.Infow("paypal_webhook_accepted_no_payment",
			"channel_id", query.ChannelID,
			"event_type", eventType,
		)
		response.Success(c, gin.H{
			"accepted":   true,
			"event_type": eventType,
			"updated":    false,
		})
		return
	}

	log.Infow("paypal_webhook_processed",
		"channel_id", query.ChannelID,
		"event_type", eventType,
		"payment_id", payment.ID,
		"status", payment.Status,
	)
	response.Success(c, gin.H{
		"accepted":   true,
		"event_type": eventType,
		"updated":    true,
		"payment_id": payment.ID,
		"status":     payment.Status,
	})
}

// StripeWebhook Stripe webhook ???
func (h *Handler) StripeWebhook(c *gin.Context) {
	log := shared.RequestLog(c)
	var query StripeWebhookQuery
	_ = c.ShouldBindQuery(&query)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Warnw("stripe_webhook_body_read_failed", "channel_id", query.ChannelID, "error", err)
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	log.Infow("stripe_webhook_received",
		"channel_id", query.ChannelID,
		"client_ip", c.ClientIP(),
		"body_size", len(body),
		"stripe_signature", truncateCallbackLogValue(strings.TrimSpace(c.GetHeader("Stripe-Signature"))),
		"raw_body", callbackRawBodyForLog(body),
	)
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) == 0 {
			continue
		}
		headers[key] = values[0]
	}

	payment, eventType, err := h.PaymentService.HandleStripeWebhook(service.WebhookCallbackInput{
		ChannelID: query.ChannelID,
		Headers:   headers,
		Body:      body,
		Context:   c.Request.Context(),
	})
	if err != nil {
		log.Warnw("stripe_webhook_handle_failed",
			"channel_id", query.ChannelID,
			"event_type", eventType,
			"error", err,
		)
		h.enqueuePaymentExceptionAlert(c, models.JSON{
			"alert_type":  "stripe_webhook_handle_failed",
			"alert_level": "error",
			"message":     strings.TrimSpace(err.Error()),
			"provider":    constants.PaymentChannelTypeStripe,
		})
		respondPaymentCallbackError(c, err)
		return
	}

	if payment == nil {
		log.Infow("stripe_webhook_accepted_no_payment",
			"channel_id", query.ChannelID,
			"event_type", eventType,
		)
		response.Success(c, gin.H{
			"accepted":   true,
			"event_type": eventType,
			"updated":    false,
		})
		return
	}

	log.Infow("stripe_webhook_processed",
		"channel_id", query.ChannelID,
		"event_type", eventType,
		"payment_id", payment.ID,
		"status", payment.Status,
	)
	response.Success(c, gin.H{
		"accepted":   true,
		"event_type": eventType,
		"updated":    true,
		"payment_id": payment.ID,
		"status":     payment.Status,
	})
}
