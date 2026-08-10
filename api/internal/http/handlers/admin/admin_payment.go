package admin

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminPaymentItem ??????
type AdminPaymentItem struct {
	models.Payment
	ChannelName    string `json:"channel_name"`
	RechargeNo     string `json:"recharge_no,omitempty"`
	RechargeStatus string `json:"recharge_status,omitempty"`
	RechargeUserID uint   `json:"recharge_user_id,omitempty"`
}

const adminPaymentExportBatchSize = 500

// GetAdminPayments ????????
func (h *Handler) GetAdminPayments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	filter, err := buildAdminPaymentFilter(c, page, pageSize)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	payments, total, err := h.PaymentService.ListPayments(filter)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	channelNameMap, err := h.resolvePaymentChannelNames(payments)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", err)
		return
	}
	rechargeMetaMap, err := h.resolvePaymentRechargeMeta(payments)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", err)
		return
	}

	items := make([]AdminPaymentItem, 0, len(payments))
	for _, payment := range payments {
		rechargeMeta := rechargeMetaMap[payment.ID]
		items = append(items, AdminPaymentItem{
			Payment:        payment,
			ChannelName:    channelNameMap[payment.ChannelID],
			RechargeNo:     rechargeMeta.RechargeNo,
			RechargeStatus: rechargeMeta.Status,
			RechargeUserID: rechargeMeta.UserID,
		})
	}

	response.SuccessWithPage(c, items, pagination)
}

// ExportAdminPayments ?????? CSV
func (h *Handler) ExportAdminPayments(c *gin.Context) {
	filter, err := buildAdminPaymentFilter(c, 1, adminPaymentExportBatchSize)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	filter.SkipCount = true
	filter.Lightweight = true

	payments, _, err := h.PaymentService.ListPayments(filter)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", err)
		return
	}

	filename := fmt.Sprintf("payments_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	writer := csv.NewWriter(c.Writer)
	if err := writer.Write([]string{
		"id",
		"order_id",
		"recharge_no",
		"recharge_status",
		"recharge_user_id",
		"channel_id",
		"provider_type",
		"channel_type",
		"status",
		"amount",
		"currency",
		"created_at",
		"paid_at",
		"expired_at",
		"provider_ref",
	}); err != nil {
		shared.RequestLog(c).Errorw("admin_payment_export_header_write_failed", "error", err)
		return
	}

	page := 1
	for {
		if len(payments) > 0 {
			if err := h.writeAdminPaymentCSVRows(writer, payments); err != nil {
				shared.RequestLog(c).Errorw("admin_payment_export_rows_write_failed", "page", page, "error", err)
				return
			}
			writer.Flush()
			if err := writer.Error(); err != nil {
				shared.RequestLog(c).Errorw("admin_payment_export_flush_failed", "page", page, "error", err)
				return
			}
		}
		if len(payments) < adminPaymentExportBatchSize {
			break
		}
		page++
		filter.Page = page
		payments, _, err = h.PaymentService.ListPayments(filter)
		if err != nil {
			shared.RequestLog(c).Errorw("admin_payment_export_batch_fetch_failed", "page", page, "error", err)
			return
		}
	}
}

// GetAdminPayment ????????
func (h *Handler) GetAdminPayment(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.payment_invalid", nil)
		return
	}

	payment, err := h.PaymentService.GetPayment(id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPaymentNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.payment_not_found", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", err)
		}
		return
	}

	channelNameMap, err := h.resolvePaymentChannelNames([]models.Payment{*payment})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", err)
		return
	}
	rechargeMetaMap, err := h.resolvePaymentRechargeMeta([]models.Payment{*payment})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", err)
		return
	}
	rechargeMeta := rechargeMetaMap[payment.ID]
	response.Success(c, AdminPaymentItem{
		Payment:        *payment,
		ChannelName:    channelNameMap[payment.ChannelID],
		RechargeNo:     rechargeMeta.RechargeNo,
		RechargeStatus: rechargeMeta.Status,
		RechargeUserID: rechargeMeta.UserID,
	})
}

func formatTimeNullable(raw *time.Time) string {
	if raw == nil {
		return ""
	}
	return raw.Format(time.RFC3339)
}

func buildAdminPaymentFilter(c *gin.Context, page, pageSize int) (repository.PaymentListFilter, error) {
	orderID, err := shared.ParseQueryUint(c.Query("order_id"), true)
	if err != nil {
		return repository.PaymentListFilter{}, err
	}
	userID, err := shared.ParseQueryUint(c.Query("user_id"), true)
	if err != nil {
		return repository.PaymentListFilter{}, err
	}
	channelID, err := shared.ParseQueryUint(c.Query("channel_id"), true)
	if err != nil {
		return repository.PaymentListFilter{}, err
	}

	createdFrom, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("created_from")))
	if err != nil {
		return repository.PaymentListFilter{}, err
	}
	createdTo, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("created_to")))
	if err != nil {
		return repository.PaymentListFilter{}, err
	}

	return repository.PaymentListFilter{
		Page:         page,
		PageSize:     pageSize,
		UserID:       userID,
		OrderID:      orderID,
		ChannelID:    channelID,
		ProviderType: strings.TrimSpace(c.Query("provider_type")),
		ChannelType:  strings.TrimSpace(c.Query("channel_type")),
		Status:       strings.TrimSpace(c.Query("status")),
		CreatedFrom:  createdFrom,
		CreatedTo:    createdTo,
	}, nil
}

func (h *Handler) writeAdminPaymentCSVRows(writer *csv.Writer, payments []models.Payment) error {
	rechargeMetaMap, err := h.resolvePaymentRechargeMeta(payments)
	if err != nil {
		return err
	}
	for _, payment := range payments {
		rechargeMeta := rechargeMetaMap[payment.ID]
		if err := writer.Write([]string{
			strconv.FormatUint(uint64(payment.ID), 10),
			strconv.FormatUint(uint64(payment.OrderID), 10),
			rechargeMeta.RechargeNo,
			rechargeMeta.Status,
			strconv.FormatUint(uint64(rechargeMeta.UserID), 10),
			strconv.FormatUint(uint64(payment.ChannelID), 10),
			payment.ProviderType,
			payment.ChannelType,
			payment.Status,
			payment.Amount.String(),
			payment.Currency,
			payment.CreatedAt.Format(time.RFC3339),
			formatTimeNullable(payment.PaidAt),
			formatTimeNullable(payment.ExpiredAt),
			payment.ProviderRef,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) resolvePaymentChannelNames(payments []models.Payment) (map[uint]string, error) {
	channelIDs := make([]uint, 0, len(payments))
	seen := make(map[uint]struct{})
	for _, payment := range payments {
		if payment.ChannelID == 0 {
			continue
		}
		if _, ok := seen[payment.ChannelID]; ok {
			continue
		}
		seen[payment.ChannelID] = struct{}{}
		channelIDs = append(channelIDs, payment.ChannelID)
	}
	result := make(map[uint]string)
	if len(channelIDs) == 0 {
		return result, nil
	}
	channels, err := h.PaymentChannelRepo.ListByIDs(channelIDs)
	if err != nil {
		return nil, err
	}
	for _, channel := range channels {
		result[channel.ID] = channel.Name
	}
	return result, nil
}

type paymentRechargeMeta struct {
	RechargeNo string
	Status     string
	UserID     uint
}

func (h *Handler) resolvePaymentRechargeMeta(payments []models.Payment) (map[uint]paymentRechargeMeta, error) {
	paymentIDs := make([]uint, 0, len(payments))
	seen := make(map[uint]struct{})
	for _, payment := range payments {
		if payment.ID == 0 {
			continue
		}
		if _, ok := seen[payment.ID]; ok {
			continue
		}
		seen[payment.ID] = struct{}{}
		paymentIDs = append(paymentIDs, payment.ID)
	}
	result := make(map[uint]paymentRechargeMeta)
	if len(paymentIDs) == 0 || h.WalletRepo == nil {
		return result, nil
	}
	orders, err := h.WalletRepo.GetRechargeOrdersByPaymentIDs(paymentIDs)
	if err != nil {
		return nil, err
	}
	for _, order := range orders {
		result[order.PaymentID] = paymentRechargeMeta{
			RechargeNo: strings.TrimSpace(order.RechargeNo),
			Status:     strings.TrimSpace(order.Status),
			UserID:     order.UserID,
		}
	}
	return result, nil
}
