package public

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	wechatCallbackRespCodeSuccess = "SUCCESS"
	wechatCallbackRespCodeFail    = "FAIL"
	wechatCallbackRespMsgSuccess  = "??"
	wechatCallbackRespMsgFail     = "??"
)

func (h *Handler) HandleWechatCallback(c *gin.Context) bool {
	log := shared.RequestLog(c)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Warnw("wechat_callback_body_read_failed", "error", err)
		return false
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	if !isWechatCallbackRequest(c, body) {
		log.Debugw("wechat_callback_not_matched")
		return false
	}

	var query WechatCallbackQuery
	_ = c.ShouldBindQuery(&query)
	log.Infow("wechat_callback_received",
		"channel_id", query.ChannelID,
		"client_ip", c.ClientIP(),
		"body_size", len(body),
		"wechatpay_signature", truncateCallbackLogValue(strings.TrimSpace(c.GetHeader("Wechatpay-Signature"))),
		"wechatpay_timestamp", strings.TrimSpace(c.GetHeader("Wechatpay-Timestamp")),
		"wechatpay_nonce", truncateCallbackLogValue(strings.TrimSpace(c.GetHeader("Wechatpay-Nonce"))),
		"wechatpay_serial", strings.TrimSpace(c.GetHeader("Wechatpay-Serial")),
		"raw_body", callbackRawBodyForLog(body),
	)

	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) == 0 {
			continue
		}
		headers[key] = values[0]
	}

	payment, _, err := h.PaymentService.HandleWechatWebhook(service.WebhookCallbackInput{
		ChannelID: query.ChannelID,
		Headers:   headers,
		Body:      body,
		Context:   c.Request.Context(),
	})
	if err != nil {
		log.Warnw("wechat_callback_handle_failed",
			"channel_id", query.ChannelID,
			"error", err,
		)
		h.enqueuePaymentExceptionAlert(c, models.JSON{
			"alert_type":  "wechat_callback_handle_failed",
			"alert_level": "error",
			"message":     strings.TrimSpace(err.Error()),
			"provider":    constants.PaymentChannelTypeWechat,
		})
		respondWechatCallback(c, false)
		return true
	}
	if payment == nil {
		log.Infow("wechat_callback_accepted_no_payment", "channel_id", query.ChannelID)
		respondWechatCallback(c, true)
		return true
	}
	log.Infow("wechat_callback_processed",
		"channel_id", query.ChannelID,
		"payment_id", payment.ID,
		"status", payment.Status,
	)
	respondWechatCallback(c, true)
	return true
}

func isWechatCallbackRequest(c *gin.Context, body []byte) bool {
	if strings.TrimSpace(c.GetHeader("Wechatpay-Signature")) == "" {
		return false
	}
	if strings.TrimSpace(c.GetHeader("Wechatpay-Timestamp")) == "" {
		return false
	}
	if strings.TrimSpace(c.GetHeader("Wechatpay-Nonce")) == "" {
		return false
	}
	if strings.TrimSpace(c.GetHeader("Wechatpay-Serial")) == "" {
		return false
	}

	payload := map[string]interface{}{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	resourceRaw, ok := payload["resource"]
	if !ok {
		return false
	}
	_, ok = resourceRaw.(map[string]interface{})
	return ok
}

func respondWechatCallback(c *gin.Context, success bool) {
	if success {
		c.JSON(http.StatusOK, gin.H{
			"code":    wechatCallbackRespCodeSuccess,
			"message": wechatCallbackRespMsgSuccess,
		})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"code":    wechatCallbackRespCodeFail,
		"message": wechatCallbackRespMsgFail,
	})
}
