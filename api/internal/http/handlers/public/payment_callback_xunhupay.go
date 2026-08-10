package public

import (
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// XunhupayCallback ??????
func (h *Handler) XunhupayCallback(c *gin.Context) {
	if h.PaymentRepo == nil || h.OrderRepo == nil {
		response.Success(c, gin.H{"status": "fail"})
		return
	}

	var formMap = make(map[string][]string)
	if err := c.Request.ParseForm(); err == nil {
		for k, v := range c.Request.PostForm {
			formMap[k] = v
		}
	}
	if len(formMap) == 0 {
		response.Success(c, gin.H{"status": "fail"})
		return
	}

	tradeOrderID := strings.TrimSpace(c.PostForm("trade_order_id"))
	status := strings.TrimSpace(c.PostForm("status"))
	pid := strings.TrimSpace(c.PostForm("pid"))
	if tradeOrderID == "" || pid == "" {
		response.Success(c, gin.H{"status": "fail"})
		return
	}

	payment, err := h.PaymentRepo.GetLatestByProviderRef(tradeOrderID)
	if err != nil || payment == nil {
		response.Success(c, gin.H{"status": "fail"})
		return
	}

	if strings.EqualFold(payment.Status, constants.PaymentStatusSuccess) {
		response.Success(c, gin.H{"status": "success"})
		return
	}

	now := time.Now()
	callbackPayload := models.JSON{
		"provider":       constants.PaymentProviderXunhupay,
		"pid":            pid,
		"trade_order_id": tradeOrderID,
		"status":         status,
		"raw":            formMap,
	}

	if err := h.PaymentRepo.Transaction(func(tx *gorm.DB) error {
		payRepo := h.PaymentRepo.WithTx(tx)
		ordRepo := h.OrderRepo.WithTx(tx)

		payment.Status = constants.PaymentStatusSuccess
		payment.ProviderRef = tradeOrderID
		payment.CallbackAt = &now
		payment.PaidAt = &now
		if payment.ProviderPayload == nil {
			payment.ProviderPayload = models.JSON{}
		}
		for k, v := range callbackPayload {
			payment.ProviderPayload[k] = v
		}
		if err := payRepo.Update(payment); err != nil {
			return err
		}

		orderUpdates := map[string]interface{}{
			"paid_at": now,
		}
		if err := ordRepo.UpdateStatus(payment.OrderID, constants.OrderStatusPaid, orderUpdates); err != nil {
			return err
		}
		return nil
	}); err != nil {
		logger.Errorw("xunhupay_callback_update_failed", "error", err, "trade_order_id", tradeOrderID)
		response.Success(c, gin.H{"status": "fail"})
		return
	}

	response.Success(c, gin.H{"status": "success"})
}
