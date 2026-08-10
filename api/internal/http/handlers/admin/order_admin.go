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

const adminOrderExportBatchSize = 500

// AdminOrderListItem ?????????
type AdminOrderListItem struct {
	models.Order
	UserEmail       string `json:"user_email,omitempty"`
	UserDisplayName string `json:"user_display_name,omitempty"`
	GatewayOrderNo  string `json:"gateway_order_no,omitempty"`
	CardSecret      string `json:"card_secret,omitempty"`
}

// AdminOrderDetail ?????????
type AdminOrderDetail struct {
	models.Order
	UserEmail       string             `json:"user_email,omitempty"`
	UserDisplayName string             `json:"user_display_name,omitempty"`
	CouponCode      string             `json:"coupon_code,omitempty"`
	PromotionName   string             `json:"promotion_name,omitempty"`
	GatewayOrderNo  string             `json:"gateway_order_no,omitempty"`
	Payments        []AdminPaymentItem `json:"payments,omitempty"`
}

// parseAdminOrderFilter ???????????????
func parseAdminOrderFilter(c *gin.Context) (repository.OrderListFilter, int, int, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	status := strings.TrimSpace(c.Query("status"))
	userKeyword := strings.TrimSpace(c.Query("user_keyword"))
	orderNo := strings.TrimSpace(c.Query("order_no"))
	guestEmail := strings.TrimSpace(c.Query("guest_email"))

	createdFrom, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("created_from")))
	if err != nil {
		return repository.OrderListFilter{}, 0, 0, err
	}
	createdTo, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("created_to")))
	if err != nil {
		return repository.OrderListFilter{}, 0, 0, err
	}

	userID, _ := shared.ParseQueryUint(c.Query("user_id"), false)
	productKeyword := strings.TrimSpace(c.Query("product_keyword"))
	productID, err := shared.ParseQueryUint(c.Query("product_id"), false)
	if err != nil {
		return repository.OrderListFilter{}, 0, 0, err
	}
	categoryID, err := shared.ParseQueryUint(c.Query("category_id"), false)
	if err != nil {
		return repository.OrderListFilter{}, 0, 0, err
	}
	skuID, err := shared.ParseQueryUint(c.Query("sku_id"), false)
	if err != nil {
		return repository.OrderListFilter{}, 0, 0, err
	}

	sortBy := strings.TrimSpace(c.Query("sort_by"))
	sortOrder := strings.TrimSpace(c.Query("sort_order"))

	return repository.OrderListFilter{
		Page:           page,
		PageSize:       pageSize,
		UserID:         userID,
		UserKeyword:    userKeyword,
		Status:         status,
		OrderNo:        orderNo,
		GuestEmail:     guestEmail,
		ProductKeyword: productKeyword,
		ProductID:      productID,
		CategoryID:     categoryID,
		SKUID:          skuID,
		CreatedFrom:    createdFrom,
		CreatedTo:      createdTo,
		SortBy:         sortBy,
		SortOrder:      sortOrder,
	}, page, pageSize, nil
}

// buildAdminOrderListMaps ????????????
func (h *Handler) buildAdminOrderListMaps(orders []models.Order) (
	userMap map[uint]models.User,
	gatewayOrderNoMap map[uint]string,
	cardSecretMap map[uint]string,
	err error,
) {
	userMap = map[uint]models.User{}
	gatewayOrderNoMap = make(map[uint]string)
	cardSecretMap = make(map[uint]string)

	userIDs := make([]uint, 0, len(orders))
	seen := map[uint]struct{}{}
	for _, order := range orders {
		if order.UserID == 0 {
			continue
		}
		if _, ok := seen[order.UserID]; ok {
			continue
		}
		seen[order.UserID] = struct{}{}
		userIDs = append(userIDs, order.UserID)
	}
	if len(userIDs) > 0 {
		users, uErr := h.UserRepo.ListByIDs(userIDs)
		if uErr != nil {
			return nil, nil, nil, uErr
		}
		for _, user := range users {
			userMap[user.ID] = user
		}
	}

	orderIDs := make([]uint, 0, len(orders))
	for _, order := range orders {
		orderIDs = append(orderIDs, order.ID)
		for _, child := range order.Children {
			orderIDs = append(orderIDs, child.ID)
		}
	}

	if len(orderIDs) > 0 {
		gatewayOrderNoMap, err = h.PaymentRepo.ListLatestGatewayOrderNos(orderIDs)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	if len(orderIDs) > 0 {
		secrets, sErr := h.CardSecretRepo.ListByOrderIDs(orderIDs)
		if sErr != nil {
			return nil, nil, nil, sErr
		}
		for _, secret := range secrets {
			if secret.OrderID != nil {
				cardSecretMap[*secret.OrderID] = secret.Secret
			}
		}
	}

	return userMap, gatewayOrderNoMap, cardSecretMap, nil
}

// resolveAdminOrderCardSecret ?????????????
func resolveAdminOrderCardSecret(order models.Order, cardSecretMap map[uint]string) string {
	cardSecret := cardSecretMap[order.ID]
	if cardSecret == "" {
		for _, child := range order.Children {
			if childSecret, ok := cardSecretMap[child.ID]; ok && childSecret != "" {
				if cardSecret != "" {
					cardSecret += "; " + childSecret
				} else {
					cardSecret = childSecret
				}
			}
		}
	}
	return cardSecret
}

// buildAdminOrderListItems ?? AdminOrderListItem ??
func buildAdminOrderListItems(orders []models.Order, userMap map[uint]models.User, gatewayOrderNoMap, cardSecretMap map[uint]string) []AdminOrderListItem {
	items := make([]AdminOrderListItem, 0, len(orders))
	for _, order := range orders {
		var email, displayName string
		if user, ok := userMap[order.UserID]; ok {
			email = user.Email
			displayName = user.DisplayName
		}
		items = append(items, AdminOrderListItem{
			Order:           order,
			UserEmail:       email,
			UserDisplayName: displayName,
			GatewayOrderNo:  gatewayOrderNoMap[order.ID],
			CardSecret:      resolveAdminOrderCardSecret(order, cardSecretMap),
		})
	}
	return items
}

// AdminListOrders ???????
func (h *Handler) AdminListOrders(c *gin.Context) {
	filter, page, pageSize, err := parseAdminOrderFilter(c)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	orders, total, err := h.OrderService.ListOrdersForAdmin(filter)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}

	userMap, gatewayOrderNoMap, cardSecretMap, err := h.buildAdminOrderListMaps(orders)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}

	items := buildAdminOrderListItems(orders, userMap, gatewayOrderNoMap, cardSecretMap)
	response.SuccessWithPage(c, items, response.BuildPagination(page, pageSize, total))
}

// AdminGetOrder ???????
func (h *Handler) AdminGetOrder(c *gin.Context) {
	orderID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}

	order, err := h.OrderService.GetOrderForAdmin(orderID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		}
		return
	}
	var email, displayName string
	if order.UserID != 0 {
		user, err := h.UserRepo.GetByID(order.UserID)
		if err != nil {
			shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
			return
		}
		if user != nil {
			email = user.Email
			displayName = user.DisplayName
		}
	}

	var couponCode string
	if order.CouponID != nil && *order.CouponID > 0 {
		coupon, err := h.CouponRepo.GetByID(*order.CouponID)
		if err != nil {
			shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
			return
		}
		if coupon != nil {
			couponCode = coupon.Code
		}
	}

	var promotionName string
	if order.PromotionID != nil && *order.PromotionID > 0 {
		promotion, err := h.PromotionRepo.GetByID(*order.PromotionID)
		if err != nil {
			shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
			return
		}
		if promotion != nil {
			promotionName = promotion.Name
		}
	}

	promotionNameMap := make(map[uint]string)
	for i := range order.Items {
		item := order.Items[i]
		if item.PromotionID == nil || *item.PromotionID == 0 {
			continue
		}
		promotionID := *item.PromotionID
		if _, ok := promotionNameMap[promotionID]; ok {
			continue
		}
		promotion, err := h.PromotionRepo.GetByID(promotionID)
		if err != nil {
			shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
			return
		}
		if promotion != nil {
			promotionNameMap[promotionID] = promotion.Name
		} else {
			promotionNameMap[promotionID] = ""
		}
	}
	for i := range order.Children {
		for _, item := range order.Children[i].Items {
			if item.PromotionID == nil || *item.PromotionID == 0 {
				continue
			}
			promotionID := *item.PromotionID
			if _, ok := promotionNameMap[promotionID]; ok {
				continue
			}
			promotion, err := h.PromotionRepo.GetByID(promotionID)
			if err != nil {
				shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
				return
			}
			if promotion != nil {
				promotionNameMap[promotionID] = promotion.Name
			} else {
				promotionNameMap[promotionID] = ""
			}
		}
	}
	for i := range order.Items {
		item := &order.Items[i]
		if item.PromotionID == nil || *item.PromotionID == 0 {
			continue
		}
		item.PromotionName = promotionNameMap[*item.PromotionID]
	}
	for i := range order.Children {
		for j := range order.Children[i].Items {
			item := &order.Children[i].Items[j]
			if item.PromotionID == nil || *item.PromotionID == 0 {
				continue
			}
			item.PromotionName = promotionNameMap[*item.PromotionID]
		}
	}

	payments, err := h.PaymentRepo.ListByOrderID(order.ID)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	channelNameMap, err := h.resolvePaymentChannelNames(payments)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	paymentItems := make([]AdminPaymentItem, 0, len(payments))
	for _, payment := range payments {
		paymentItems = append(paymentItems, AdminPaymentItem{
			Payment:     payment,
			ChannelName: channelNameMap[payment.ChannelID],
		})
	}

	var gatewayOrderNo string
	if len(payments) > 0 {
		gatewayOrderNo = payments[0].GatewayOrderNo
	}

	order.TruncateFulfillmentPayload()
	response.Success(c, AdminOrderDetail{
		Order:           *order,
		UserEmail:       email,
		UserDisplayName: displayName,
		CouponCode:      couponCode,
		PromotionName:   promotionName,
		GatewayOrderNo:  gatewayOrderNo,
		Payments:        paymentItems,
	})
}

// AdminUpdateOrderStatusRequest ???????????
type AdminUpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// AdminUpdateOrderStatus ?????????
func (h *Handler) AdminUpdateOrderStatus(c *gin.Context) {
	orderID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}

	var req AdminUpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	order, err := h.OrderService.UpdateOrderStatus(orderID, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		case errors.Is(err, service.ErrOrderStatusInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.order_status_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.order_update_failed", err)
		}
		return
	}

	response.Success(c, order)
}

// AdminDownloadFulfillment ???????????
func (h *Handler) AdminDownloadFulfillment(c *gin.Context) {
	orderID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	order, err := h.OrderService.GetOrderForAdmin(orderID)
	if err != nil || order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		return
	}
	payload := collectAdminFulfillmentPayload(order)
	if payload == "" {
		shared.RespondError(c, response.CodeNotFound, "error.fulfillment_not_found", nil)
		return
	}
	filename := "fulfillment-" + order.OrderNo + ".txt"
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(200, "text/plain; charset=utf-8", []byte(payload))
}

func collectAdminFulfillmentPayload(order *models.Order) string {
	if order.Fulfillment != nil && order.Fulfillment.Payload != "" {
		return order.Fulfillment.Payload
	}
	var parts []string
	for _, child := range order.Children {
		if child.Fulfillment != nil && child.Fulfillment.Payload != "" {
			parts = append(parts, child.Fulfillment.Payload)
		}
	}
	return strings.Join(parts, "\n")
}

// AdminExportOrders ???? CSV
func (h *Handler) AdminExportOrders(c *gin.Context) {
	filter, _, _, err := parseAdminOrderFilter(c)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	filter.Page = 1
	filter.PageSize = adminOrderExportBatchSize

	orders, _, err := h.OrderService.ListOrdersForAdmin(filter)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}

	filename := fmt.Sprintf("orders_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(c.Writer)
	if err := writer.Write([]string{
		"ID",
		"???",
		"?????",
		"??",
		"??",
		"????",
		"????",
		"??",
		"??",
		"??ID",
		"????",
		"????",
		"????",
		"???IP",
		"????",
		"????",
	}); err != nil {
		shared.RequestLog(c).Errorw("admin_order_export_header_write_failed", "error", err)
		return
	}

	batch := 1
	for {
		if len(orders) == 0 {
			break
		}

		userMap, gatewayOrderNoMap, cardSecretMap, mapErr := h.buildAdminOrderListMaps(orders)
		if mapErr != nil {
			shared.RequestLog(c).Errorw("admin_order_export_maps_failed", "error", mapErr)
			return
		}

		for _, order := range orders {
			email := ""
			displayName := ""
			if user, ok := userMap[order.UserID]; ok {
				email = user.Email
				displayName = user.DisplayName
			}
			cardSecret := resolveAdminOrderCardSecret(order, cardSecretMap)

			productNames := collectOrderProductNames(order)
			productSpecs := collectOrderProductSpecs(order)

			row := []string{
				strconv.FormatUint(uint64(order.ID), 10),
				order.OrderNo,
				gatewayOrderNoMap[order.ID],
				cardSecret,
				order.Status,
				productNames,
				productSpecs,
				order.TotalAmount.String(),
				order.Currency,
				strconv.FormatUint(uint64(order.UserID), 10),
				email,
				displayName,
				order.GuestEmail,
				order.ClientIP,
				order.CreatedAt.Format("2006-01-02") + " " + order.CreatedAt.Format("15:04:05"),
				order.UpdatedAt.Format("2006-01-02") + " " + order.UpdatedAt.Format("15:04:05"),
			}
			if err := writer.Write(row); err != nil {
				shared.RequestLog(c).Errorw("admin_order_export_row_write_failed", "error", err)
				return
			}
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			shared.RequestLog(c).Errorw("admin_order_export_flush_failed", "batch", batch, "error", err)
			return
		}

		if len(orders) < adminOrderExportBatchSize {
			break
		}
		batch++
		filter.Page = batch
		orders, _, err = h.OrderService.ListOrdersForAdmin(filter)
		if err != nil {
			shared.RequestLog(c).Errorw("admin_order_export_batch_fetch_failed", "page", batch, "error", err)
			return
		}
	}
}

func collectOrderProductNames(order models.Order) string {
	var parts []string
	for _, item := range order.Items {
		name := localizeAdminTitle(item.TitleJSON)
		if name == "" {
			name = fmt.Sprintf("#%d", item.ProductID)
		}
		parts = append(parts, name+" x"+strconv.Itoa(item.Quantity))
	}
	return strings.Join(parts, "; ")
}

func collectOrderProductSpecs(order models.Order) string {
	var specs []string
	for _, item := range order.Items {
		spec := collectItemSpecLabel(item)
		if spec != "" {
			specs = append(specs, spec)
			continue
		}
		for _, child := range order.Children {
			for _, childItem := range child.Items {
				if childItem.ProductID == item.ProductID {
					spec = collectItemSpecLabel(childItem)
					if spec != "" {
						specs = append(specs, spec)
						break
					}
				}
			}
			if spec != "" {
				break
			}
		}
	}
	return strings.Join(specs, "; ")
}

func collectItemSpecLabel(item models.OrderItem) string {
	if item.SKUSnapshotJSON == nil {
		return ""
	}
	code, _ := stringFromJSON(item.SKUSnapshotJSON, "sku_code")
	specValuesRaw := item.SKUSnapshotJSON["spec_values"]
	if specValuesRaw == nil {
		if code != "" {
			return code
		}
		return ""
	}
	specMap, ok := specValuesRaw.(map[string]interface{})
	if !ok {
		if code != "" {
			return code
		}
		return ""
	}
	var parts []string
	for k, v := range specMap {
		s, ok := v.(string)
		if !ok {
			continue
		}
		if s == "" {
			continue
		}
		parts = append(parts, k+":"+s)
	}
	result := strings.Join(parts, " ")
	if result == "" && code != "" {
		return code
	}
	return result
}

func stringFromJSON(j models.JSON, key string) (string, bool) {
	if j == nil {
		return "", false
	}
	raw, ok := j[key]
	if !ok {
		return "", false
	}
	s, ok := raw.(string)
	return s, ok
}

func localizeAdminTitle(titleJSON models.JSON) string {
	if titleJSON == nil {
		return ""
	}
	for _, key := range []string{"zh-CN", "en-US", "zh-TW"} {
		if val, ok := titleJSON[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}
	for _, val := range titleJSON {
		if s, ok := val.(string); ok && s != "" {
			return s
		}
	}
	return ""
}