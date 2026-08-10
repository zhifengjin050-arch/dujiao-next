package admin

import (
	"errors"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminCreateFulfillmentRequest ?????????
type AdminCreateFulfillmentRequest struct {
	OrderID      uint        `json:"order_id" binding:"required"`
	Payload      string      `json:"payload"`
	DeliveryData models.JSON `json:"delivery_data"`
}

// AdminCreateFulfillment ?????????
func (h *Handler) AdminCreateFulfillment(c *gin.Context) {
	adminID, ok := shared.GetAdminID(c)
	if !ok {
		return
	}

	var req AdminCreateFulfillmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	fulfillment, err := h.FulfillmentService.CreateManual(service.CreateManualInput{
		OrderID:      req.OrderID,
		AdminID:      adminID,
		Payload:      req.Payload,
		DeliveryData: req.DeliveryData,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFulfillmentExists):
			shared.RespondError(c, response.CodeBadRequest, "error.fulfillment_exists", nil)
		case errors.Is(err, service.ErrFulfillmentInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.fulfillment_invalid", nil)
		case errors.Is(err, service.ErrOrderStatusInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.order_status_invalid", nil)
		case errors.Is(err, service.ErrOrderNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.fulfillment_create_failed", err)
		}
		return
	}

	response.Success(c, fulfillment)
}
