package upstream

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/provider"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"
	upstreamadapter "github.com/dujiao-next/internal/upstream"

	"github.com/gin-gonic/gin"
)

// Handler ?? API ???(???? B ?????? A ????)
type Handler struct {
	*provider.Container
	downstreamRefRepo repository.DownstreamOrderRefRepository
}

// New ???????
func New(c *provider.Container, downstreamRefRepo repository.DownstreamOrderRefRepository) *Handler {
	return &Handler{
		Container:         c,
		downstreamRefRepo: downstreamRefRepo,
	}
}

// ---- context helpers (?????? router ?) ----

const (
	upstreamUserIDKey       = "upstream_user_id"
	upstreamCredentialIDKey = "upstream_credential_id"
)

func getUpstreamUserID(c *gin.Context) uint {
	v, _ := c.Get(upstreamUserIDKey)
	if id, ok := v.(uint); ok {
		return id
	}
	return 0
}

func getUpstreamCredentialID(c *gin.Context) uint {
	v, _ := c.Get(upstreamCredentialIDKey)
	if id, ok := v.(uint); ok {
		return id
	}
	return 0
}

// ---- response helpers ----

func successResponse(c *gin.Context, data interface{}) {
	if data == nil {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	c.JSON(http.StatusOK, data)
}

func errorResponse(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"ok":            false,
		"error_code":    code,
		"error_message": message,
	})
}

// ---- Ping ----

// Ping POST /api/v1/upstream/ping
func (h *Handler) Ping(c *gin.Context) {
	userID := getUpstreamUserID(c)
	if userID == 0 {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", "invalid credentials")
		return
	}

	// ????
	siteName := ""
	siteConfig, err := h.SettingService.GetByKey(constants.SettingKeySiteConfig)
	if err == nil && siteConfig != nil {
		if name, ok := siteConfig["site_name"]; ok {
			if s, ok := name.(string); ok {
				siteName = s
			}
		}
	}

	// ??????
	balanceStr := "0.00"
	account, err := h.WalletService.GetAccount(userID)
	if err == nil && account != nil {
		balanceStr = account.Balance.StringFixed(2)
	}

	// ??
	currency, _ := h.SettingService.GetSiteCurrency("CNY")

	// ??????
	var memberLevel gin.H
	user, err := h.UserRepo.GetByID(userID)
	if err == nil && user != nil && user.MemberLevelID > 0 && h.MemberLevelService != nil {
		level, levelErr := h.MemberLevelService.GetByID(user.MemberLevelID)
		if levelErr == nil && level != nil {
			memberLevel = gin.H{
				"id":   level.ID,
				"name": level.NameJSON,
				"slug": level.Slug,
				"icon": level.Icon,
			}
		}
	}

	successResponse(c, gin.H{
		"ok":               true,
		"site_name":        siteName,
		"protocol_version": "1.0",
		"user_id":          userID,
		"balance":          balanceStr,
		"currency":         currency,
		"member_level":     memberLevel,
	})
}

// ---- ListCategories ----

// upstreamCategory ????????
type upstreamCategory struct {
	ID        uint        `json:"id"`
	ParentID  uint        `json:"parent_id"`
	Slug      string      `json:"slug"`
	Name      models.JSON `json:"name"`
	Icon      string      `json:"icon"`
	SortOrder int         `json:"sort_order"`
}

// ListCategories GET /api/v1/upstream/categories
func (h *Handler) ListCategories(c *gin.Context) {
	categories, err := h.CategoryRepo.List()
	if err != nil {
		logger.Errorw("upstream_list_categories_failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to list categories")
		return
	}

	items := make([]upstreamCategory, 0, len(categories))
	for _, cat := range categories {
		items = append(items, upstreamCategory{
			ID:        cat.ID,
			ParentID:  cat.ParentID,
			Slug:      cat.Slug,
			Name:      cat.NameJSON,
			Icon:      cat.Icon,
			SortOrder: cat.SortOrder,
		})
	}

	successResponse(c, gin.H{
		"ok":         true,
		"categories": items,
	})
}

// ---- ListProducts ----

// upstreamProduct ????????
type upstreamProduct struct {
	ID               uint               `json:"id"`
	Slug             string             `json:"slug"`
	SeoMeta          models.JSON        `json:"seo_meta"`
	Title            models.JSON        `json:"title"`
	Description      models.JSON        `json:"description"`
	Content          models.JSON        `json:"content"`
	Images           models.StringArray `json:"images"`
	Tags             models.StringArray `json:"tags"`
	PriceAmount      string             `json:"price_amount"`
	OriginalPrice    string             `json:"original_price,omitempty"`
	MemberPrice      string             `json:"member_price,omitempty"`
	FulfillmentType  string             `json:"fulfillment_type"`
	ManualFormSchema models.JSON        `json:"manual_form_schema"`
	IsActive         bool               `json:"is_active"`
	CategoryID       uint               `json:"category_id"`
	SKUs             []upstreamSKU      `json:"skus"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

type upstreamSKU struct {
	ID            uint        `json:"id"`
	SKUCode       string      `json:"sku_code"`
	SpecValues    models.JSON `json:"spec_values"`
	PriceAmount   string      `json:"price_amount"`
	OriginalPrice string      `json:"original_price,omitempty"`
	MemberPrice   string      `json:"member_price,omitempty"`
	StockStatus   string      `json:"stock_status"`
	StockQuantity int         `json:"stock_quantity"`
	IsActive      bool        `json:"is_active"`
}

// ListProducts GET /api/v1/upstream/products
func (h *Handler) ListProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 50
	}

	// ??????:??????????????
	var products []models.Product
	var total int64
	var err error
	if updatedAfterStr := c.Query("updated_after"); updatedAfterStr != "" {
		if t, parseErr := time.Parse(time.RFC3339, updatedAfterStr); parseErr == nil {
			products, total, err = h.ProductService.ListPublicUpdatedAfter(&t, page, pageSize)
		} else {
			products, total, err = h.ProductService.ListPublic("", "", page, pageSize)
		}
	} else {
		products, total, err = h.ProductService.ListPublic("", "", page, pageSize)
	}
	if err != nil {
		logger.Errorw("upstream_list_products_failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to list products")
		return
	}

	// ??????????
	if err := h.ProductService.ApplyAutoStockCounts(products); err != nil {
		logger.Warnw("upstream_apply_stock_counts_failed", "error", err)
	}

	// ????????? SKU ??
	h.applyUpstreamStockToProducts(products)

	// ???????????
	userID := getUpstreamUserID(c)
	var memberLevelID uint
	if userID > 0 {
		user, err := h.UserRepo.GetByID(userID)
		if err == nil && user != nil {
			memberLevelID = user.MemberLevelID
		}
	}

	// ???????????????
	fulfillmentTypeMap := h.resolveEffectiveFulfillmentTypes(products)

	items := make([]upstreamProduct, 0, len(products))
	for _, p := range products {
		items = append(items, h.toUpstreamProductWithMemberPrice(p, memberLevelID, fulfillmentTypeMap))
	}

	successResponse(c, gin.H{
		"ok":        true,
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetProduct GET /api/v1/upstream/products/:id
func (h *Handler) GetProduct(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		errorResponse(c, http.StatusBadRequest, "bad_request", "product id is required")
		return
	}

	product, err := h.ProductService.GetAdminByID(id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			errorResponse(c, http.StatusNotFound, "product_not_found", "product not found")
			return
		}
		logger.Errorw("upstream_get_product_failed", "id", id, "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to get product")
		return
	}

	if !product.IsActive {
		errorResponse(c, http.StatusNotFound, "product_unavailable", "product is not active")
		return
	}

	// ??????????
	products := []models.Product{*product}
	if err := h.ProductService.ApplyAutoStockCounts(products); err != nil {
		logger.Warnw("upstream_apply_stock_counts_failed", "error", err)
	}

	// ????????? SKU ??
	h.applyUpstreamStockToProducts(products)

	// ???????????
	userID := getUpstreamUserID(c)
	var memberLevelID uint
	if userID > 0 {
		user, err := h.UserRepo.GetByID(userID)
		if err == nil && user != nil {
			memberLevelID = user.MemberLevelID
		}
	}

	// ?????????????
	fulfillmentTypeMap := h.resolveEffectiveFulfillmentTypes(products)

	successResponse(c, gin.H{
		"ok":      true,
		"product": h.toUpstreamProductWithMemberPrice(products[0], memberLevelID, fulfillmentTypeMap),
	})
}

// ---- CreateOrder ----

type createOrderRequest struct {
	SKUID             uint        `json:"sku_id" binding:"required"`
	Quantity          int         `json:"quantity" binding:"required,min=1"`
	ManualFormData    models.JSON `json:"manual_form_data"`
	DownstreamOrderNo string      `json:"downstream_order_no"`
	TraceID           string      `json:"trace_id"`
	CallbackURL       string      `json:"callback_url"`
}

// CreateOrder POST /api/v1/upstream/orders
func (h *Handler) CreateOrder(c *gin.Context) {
	userID := getUpstreamUserID(c)
	credentialID := getUpstreamCredentialID(c)
	if userID == 0 || credentialID == 0 {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", "invalid credentials")
		return
	}

	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "bad_request", "invalid request body: "+err.Error())
		return
	}

	// ?? callback URL(?? SSRF)
	if req.CallbackURL != "" {
		if err := validateCallbackURL(req.CallbackURL); err != nil {
			errorResponse(c, http.StatusBadRequest, "invalid_callback_url", err.Error())
			return
		}
	}

	// ?????:?? credential ??? downstream_order_no ???????
	if strings.TrimSpace(req.DownstreamOrderNo) != "" {
		existingRef, _ := h.downstreamRefRepo.GetByCredentialAndDownstreamNo(credentialID, req.DownstreamOrderNo)
		if existingRef != nil {
			// ????????????,????????
			existingOrder, orderErr := h.OrderService.GetOrderByUser(existingRef.OrderID, userID)
			if orderErr == nil && existingOrder != nil {
				currency, _ := h.SettingService.GetSiteCurrency("CNY")
				successResponse(c, gin.H{
					"ok":       true,
					"order_id": existingOrder.ID,
					"order_no": existingOrder.OrderNo,
					"status":   existingOrder.Status,
					"amount":   existingOrder.TotalAmount.StringFixed(2),
					"currency": currency,
				})
				return
			}
		}
	}

	// ?? SKU ?????? ID
	sku, err := h.ProductSKURepo.GetByID(req.SKUID)
	if err != nil || sku == nil {
		errorResponse(c, http.StatusBadRequest, "sku_unavailable", "sku not found")
		return
	}
	if !sku.IsActive {
		errorResponse(c, http.StatusBadRequest, "sku_unavailable", "sku is not active")
		return
	}

	// ????????
	product, err := h.ProductRepo.GetByID(fmt.Sprintf("%d", sku.ProductID))
	if err != nil || product == nil || !product.IsActive {
		errorResponse(c, http.StatusBadRequest, "product_unavailable", "product is not available")
		return
	}

	// ????????
	var manualFormData map[string]models.JSON
	if req.ManualFormData != nil && product.FulfillmentType == constants.FulfillmentTypeManual {
		manualFormData = map[string]models.JSON{
			fmt.Sprintf("%d", sku.ProductID): req.ManualFormData,
		}
	}

	// ????(????????:??? API ????,??????,????????)
	input := service.CreateOrderInput{
		UserID: userID,
		Items: []service.CreateOrderItem{
			{
				ProductID:       sku.ProductID,
				SKUID:           req.SKUID,
				Quantity:        req.Quantity,
				FulfillmentType: product.FulfillmentType,
			},
		},
		ClientIP:        c.ClientIP(),
		ManualFormData:  manualFormData,
		SkipRiskControl: true,
	}

	order, err := h.OrderService.CreateOrder(input)
	if err != nil {
		mapOrderErrorToResponse(c, err)
		return
	}

	// ??????????(????????)
	ref := &models.DownstreamOrderRef{
		OrderID:           order.ID,
		ApiCredentialID:   credentialID,
		DownstreamOrderNo: req.DownstreamOrderNo,
		CallbackURL:       req.CallbackURL,
		TraceID:           req.TraceID,
		CallbackStatus:    "pending",
	}
	if createErr := h.downstreamRefRepo.Create(ref); createErr != nil {
		logger.Errorw("upstream_create_downstream_ref_failed",
			"order_id", order.ID,
			"credential_id", credentialID,
			"downstream_order_no", req.DownstreamOrderNo,
			"callback_url", req.CallbackURL,
			"trace_id", req.TraceID,
			"error", createErr,
		)
	} else {
		logger.Infow("upstream_downstream_ref_created",
			"ref_id", ref.ID,
			"order_id", order.ID,
			"callback_url", req.CallbackURL,
		)
	}

	// ??????????(?? API ????????)
	payResult, payErr := h.PaymentService.CreatePayment(service.CreatePaymentInput{
		OrderID:    order.ID,
		UseBalance: true,
		ClientIP:   c.ClientIP(),
	})
	if payErr != nil {
		logger.Errorw("upstream_auto_wallet_pay_failed",
			"order_id", order.ID,
			"error", payErr,
		)
		// ????,???????????????
		if _, cancelErr := h.OrderService.CancelOrder(order.ID, userID); cancelErr != nil {
			logger.Warnw("upstream_cancel_unpaid_order_failed", "order_id", order.ID, "error", cancelErr)
		}
		// ???????????,?? 200 + ok:false ? A ????????
		c.JSON(http.StatusOK, gin.H{
			"ok":            false,
			"order_id":      order.ID,
			"order_no":      order.OrderNo,
			"status":        constants.OrderStatusCanceled,
			"error_code":    "payment_failed",
			"error_message": fmt.Sprintf("wallet payment failed: %s", payErr.Error()),
		})
		return
	}

	// ??????
	finalStatus := order.Status
	if payResult != nil && payResult.OrderPaid {
		finalStatus = constants.OrderStatusPaid
	}

	// ??
	currency, _ := h.SettingService.GetSiteCurrency("CNY")

	successResponse(c, gin.H{
		"ok":       true,
		"order_id": order.ID,
		"order_no": order.OrderNo,
		"status":   finalStatus,
		"amount":   order.TotalAmount.StringFixed(2),
		"currency": currency,
	})
}

// ---- GetOrder ----

// GetOrder GET /api/v1/upstream/orders/:id
func (h *Handler) GetOrder(c *gin.Context) {
	userID := getUpstreamUserID(c)
	if userID == 0 {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", "invalid credentials")
		return
	}

	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "bad_request", "invalid order id")
		return
	}

	order, err := h.OrderService.GetOrderByUser(uint(orderID), userID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			errorResponse(c, http.StatusNotFound, "order_not_found", "order not found")
			return
		}
		logger.Errorw("upstream_get_order_failed", "order_id", orderID, "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to get order")
		return
	}

	status := strings.ToLower(strings.TrimSpace(order.Status))
	localRefundRecords := make([]models.JSON, 0)

	// ????????????????,?????????????????
	if h.ProcurementOrderService != nil {
		procOrder, procErr := h.ProcurementOrderService.GetByLocalOrderNo(order.OrderNo)
		if procErr == nil && procOrder != nil {
			switch strings.ToLower(strings.TrimSpace(procOrder.Status)) {
			case constants.ProcurementStatusPartiallyRefunded, constants.ProcurementStatusRefunded:
				status = strings.ToLower(strings.TrimSpace(procOrder.Status))
			}
		}
	}

	// refund_records ??????????(??????),?????????
	if records, recordsErr := h.OrderService.BuildLocalRefundRecordsForOrder(order); recordsErr != nil {
		logger.Warnw("upstream_get_order_refund_records_failed", "order_id", order.ID, "error", recordsErr)
	} else {
		localRefundRecords = records
	}

	resp := gin.H{
		"ok":              true,
		"order_id":        order.ID,
		"order_no":        order.OrderNo,
		"status":          status,
		"amount":          order.TotalAmount.StringFixed(2),
		"refunded_amount": order.RefundedAmount.StringFixed(2),
		"currency":        order.Currency,
		"refund_records":  localRefundRecords,
	}

	// ????,??????(????????? fulfillment,????????)
	sourceFulfillment := order.Fulfillment
	if sourceFulfillment == nil && len(order.Children) > 0 {
		for i := range order.Children {
			if order.Children[i].Fulfillment != nil {
				sourceFulfillment = order.Children[i].Fulfillment
				break
			}
		}
	}
	if sourceFulfillment != nil && sourceFulfillment.Status == constants.FulfillmentStatusDelivered {
		resp["fulfillment"] = gin.H{
			"type":          sourceFulfillment.Type,
			"status":        sourceFulfillment.Status,
			"payload":       sourceFulfillment.Payload,
			"delivery_data": sourceFulfillment.LogisticsJSON,
			"delivered_at":  sourceFulfillment.DeliveredAt,
		}
	}

	// ?????
	if len(order.Items) > 0 {
		items := make([]gin.H, 0, len(order.Items))
		for _, item := range order.Items {
			items = append(items, gin.H{
				"product_id":       item.ProductID,
				"sku_id":           item.SKUID,
				"title":            item.TitleJSON,
				"quantity":         item.Quantity,
				"unit_price":       item.UnitPrice.StringFixed(2),
				"total_price":      item.TotalPrice.StringFixed(2),
				"fulfillment_type": item.FulfillmentType,
			})
		}
		resp["items"] = items
	}

	successResponse(c, resp)
}

// ---- CancelOrder ----

// CancelOrder POST /api/v1/upstream/orders/:id/cancel
func (h *Handler) CancelOrder(c *gin.Context) {
	userID := getUpstreamUserID(c)
	if userID == 0 {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", "invalid credentials")
		return
	}

	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "bad_request", "invalid order id")
		return
	}

	order, err := h.OrderService.CancelOrder(uint(orderID), userID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			errorResponse(c, http.StatusNotFound, "order_not_found", "order not found")
			return
		}
		if errors.Is(err, service.ErrOrderCancelNotAllowed) {
			errorResponse(c, http.StatusConflict, "cancel_not_allowed", "order cannot be canceled in current status")
			return
		}
		logger.Errorw("upstream_cancel_order_failed", "order_id", orderID, "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to cancel order")
		return
	}

	successResponse(c, gin.H{
		"ok":       true,
		"order_id": order.ID,
		"order_no": order.OrderNo,
		"status":   order.Status,
	})
}

// ---- HandleCallback (A ??? B ???) ----

type callbackPayload struct {
	Event             string `json:"event"`
	OrderID           uint   `json:"order_id"`
	OrderNo           string `json:"order_no"`
	DownstreamOrderNo string `json:"downstream_order_no"`
	Status            string `json:"status"`
	Fulfillment       *struct {
		Type         string      `json:"type"`
		Status       string      `json:"status"`
		Payload      string      `json:"payload"`
		DeliveryData models.JSON `json:"delivery_data"`
		DeliveredAt  *time.Time  `json:"delivered_at"`
	} `json:"fulfillment,omitempty"`
	Timestamp int64 `json:"timestamp"`
}

// HandleCallback POST /api/v1/upstream/callback (A ???? B ???)
func (h *Handler) HandleCallback(c *gin.Context) {
	// ---- ???? ----
	apiKey := c.GetHeader(upstreamadapter.HeaderApiKey)
	timestampStr := c.GetHeader(upstreamadapter.HeaderTimestamp)
	signature := c.GetHeader(upstreamadapter.HeaderSignature)

	if apiKey == "" || timestampStr == "" || signature == "" {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "missing authentication headers"})
		return
	}

	timestamp, err := upstreamadapter.ParseTimestamp(timestampStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "invalid timestamp"})
		return
	}

	if !upstreamadapter.IsTimestampValid(timestamp) {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "timestamp expired"})
		return
	}

	// ?? api_key ?????????
	conn, err := h.SiteConnectionRepo.GetByApiKey(apiKey)
	if err != nil {
		logger.Errorw("upstream_callback_lookup_connection_failed", "api_key", apiKey, "error", err)
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "internal error"})
		return
	}
	if conn == nil || conn.Status != "active" {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "invalid api key"})
		return
	}

	// ?? body ??????
	var body []byte
	if c.Request.Body != nil {
		body, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "message": "failed to read request body"})
			return
		}
		c.Request.Body = io.NopCloser(&bodyBuf{data: body})
	}

	// ?? api_secret ?????
	apiSecret := conn.ApiSecret
	if h.SiteConnectionService != nil {
		if decrypted, decErr := h.SiteConnectionService.DecryptSecret(apiSecret); decErr == nil {
			apiSecret = decrypted
		}
	}

	if !upstreamadapter.Verify(apiSecret, "POST", "/api/v1/upstream/callback", signature, timestamp, body) {
		logger.Warnw("upstream_callback_signature_invalid", "api_key", apiKey)
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "signature verification failed"})
		return
	}

	// ---- ?? payload ----
	var payload callbackPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "invalid request body"})
		return
	}

	if payload.DownstreamOrderNo == "" || payload.Status == "" {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "missing required fields"})
		return
	}

	if h.ProcurementOrderService == nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "service not available"})
		return
	}

	// ?? downstream_order_no(???? local_order_no)????????
	procOrder, err := h.ProcurementOrderService.GetByLocalOrderNo(payload.DownstreamOrderNo)
	if err != nil || procOrder == nil {
		logger.Warnw("upstream_callback_procurement_not_found",
			"downstream_order_no", payload.DownstreamOrderNo,
			"upstream_order_id", payload.OrderID,
		)
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "procurement order not found"})
		return
	}

	// ?????????
	var uf *upstreamadapter.UpstreamFulfillment
	if payload.Fulfillment != nil {
		uf = &upstreamadapter.UpstreamFulfillment{
			Type:         payload.Fulfillment.Type,
			Status:       payload.Fulfillment.Status,
			Payload:      payload.Fulfillment.Payload,
			DeliveryData: payload.Fulfillment.DeliveryData,
			DeliveredAt:  payload.Fulfillment.DeliveredAt,
		}
	}

	upstreamStatus := mapCallbackStatus(payload.Status)
	if err := h.ProcurementOrderService.HandleUpstreamCallback(procOrder.ID, upstreamStatus, uf); err != nil {
		logger.Warnw("upstream_callback_handle_failed",
			"procurement_order_id", procOrder.ID,
			"upstream_status", upstreamStatus,
			"error", err,
		)
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": "callback processing failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "received"})
}

// bodyBuf ???? body
type bodyBuf struct {
	data   []byte
	offset int
}

func (b *bodyBuf) Read(p []byte) (n int, err error) {
	if b.offset >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.offset:])
	b.offset += n
	return n, nil
}

// mapCallbackStatus ????????????????
func mapCallbackStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case "delivered", "completed", "fulfilled":
		return "delivered"
	case "canceled", "cancelled":
		return "canceled"
	default:
		return normalized
	}
}

// ---- helpers ----

// applyUpstreamStockToProducts ? upstream ????? SKU ????????
// ? SKU ????? UpstreamStock,?? ProductSKU ?????,? computeSKUStock ??
func (h *Handler) applyUpstreamStockToProducts(products []models.Product) {
	for i := range products {
		p := &products[i]
		if p.FulfillmentType != constants.FulfillmentTypeUpstream {
			continue
		}
		for j := range p.SKUs {
			sku := &p.SKUs[j]
			mapping, err := h.SKUMappingRepo.GetByLocalSKUID(sku.ID)
			if err != nil || mapping == nil {
				sku.UpstreamStock = 0
				continue
			}
			sku.UpstreamStock = mapping.UpstreamStock
		}
	}
}

// resolveEffectiveFulfillmentTypes ???????????????
// ??????(FulfillmentType="upstream"),?? ProductMapping ??????????
func (h *Handler) resolveEffectiveFulfillmentTypes(products []models.Product) map[uint]string {
	result := make(map[uint]string)
	var mappedIDs []uint
	for _, p := range products {
		if p.IsMapped && p.FulfillmentType == constants.FulfillmentTypeUpstream {
			mappedIDs = append(mappedIDs, p.ID)
		}
	}
	if len(mappedIDs) == 0 {
		return result
	}
	mappings, err := h.ProductMappingRepo.ListByLocalProductIDs(mappedIDs)
	if err != nil {
		logger.Warnw("resolve_effective_fulfillment_types_failed", "error", err)
		return result
	}
	for _, m := range mappings {
		ft := m.UpstreamFulfillmentType
		if ft != constants.FulfillmentTypeAuto {
			ft = constants.FulfillmentTypeManual
		}
		result[m.LocalProductID] = ft
	}
	return result
}

func (h *Handler) toUpstreamProductWithMemberPrice(p models.Product, memberLevelID uint, fulfillmentTypeMap map[uint]string) upstreamProduct {
	skus := make([]upstreamSKU, 0, len(p.SKUs))
	for _, s := range p.SKUs {
		if !s.IsActive {
			continue
		}
		stockStatus, stockQuantity := computeSKUStock(p, s)
		si := upstreamSKU{
			ID:            s.ID,
			SKUCode:       s.SKUCode,
			SpecValues:    s.SpecValuesJSON,
			PriceAmount:   s.PriceAmount.StringFixed(2),
			StockStatus:   stockStatus,
			StockQuantity: stockQuantity,
			IsActive:      s.IsActive,
		}
		if memberLevelID > 0 && h.MemberLevelService != nil {
			mp, _ := h.MemberLevelService.ResolveMemberPrice(memberLevelID, p.ID, s.ID, s.PriceAmount.Decimal)
			if mp.LessThan(s.PriceAmount.Decimal) {
				si.OriginalPrice = si.PriceAmount
				si.MemberPrice = models.NewMoneyFromDecimal(mp).StringFixed(2)
				si.PriceAmount = si.MemberPrice // price_amount ?????(???)
			}
		}
		skus = append(skus, si)
	}

	// ????????????,?????????????
	effectiveFulfillmentType := p.FulfillmentType
	if ft, ok := fulfillmentTypeMap[p.ID]; ok {
		effectiveFulfillmentType = ft
	}

	result := upstreamProduct{
		ID:               p.ID,
		Slug:             p.Slug,
		SeoMeta:          p.SeoMetaJSON,
		Title:            p.TitleJSON,
		Description:      p.DescriptionJSON,
		Content:          p.ContentJSON,
		Images:           p.Images,
		Tags:             p.Tags,
		PriceAmount:      p.PriceAmount.StringFixed(2),
		FulfillmentType:  effectiveFulfillmentType,
		ManualFormSchema: p.ManualFormSchemaJSON,
		IsActive:         p.IsActive,
		CategoryID:       p.CategoryID,
		SKUs:             skus,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}

	if memberLevelID > 0 && h.MemberLevelService != nil {
		mp, _ := h.MemberLevelService.ResolveMemberPrice(memberLevelID, p.ID, 0, p.PriceAmount.Decimal)
		if mp.LessThan(p.PriceAmount.Decimal) {
			result.OriginalPrice = result.PriceAmount
			result.MemberPrice = models.NewMoneyFromDecimal(mp).StringFixed(2)
			result.PriceAmount = result.MemberPrice
		}
	}

	return result
}

// computeSKUStock ?? SKU ???????????
func computeSKUStock(p models.Product, s models.ProductSKU) (status string, quantity int) {
	if p.FulfillmentType == constants.FulfillmentTypeManual {
		// ????:?? SKU ????????
		skuTotal := s.ManualStockTotal
		if skuTotal == constants.ManualStockUnlimited {
			return constants.ProductStockStatusUnlimited, -1
		}
		available := skuTotal - s.ManualStockLocked
		if available <= 0 {
			return constants.ProductStockStatusOutOfStock, 0
		}
		if available <= 20 {
			return constants.ProductStockStatusLowStock, available
		}
		return constants.ProductStockStatusInStock, available
	}

	if p.FulfillmentType == constants.FulfillmentTypeUpstream {
		// ??????:?? SKU ????????(?????? UpstreamStock ??)
		available := s.UpstreamStock
		if available < 0 {
			return constants.ProductStockStatusUnlimited, -1
		}
		if available == 0 {
			return constants.ProductStockStatusOutOfStock, 0
		}
		if available <= 20 {
			return constants.ProductStockStatusLowStock, available
		}
		return constants.ProductStockStatusInStock, available
	}

	// ????:????????
	available := int(s.AutoStockAvailable)
	if available <= 0 {
		return constants.ProductStockStatusOutOfStock, 0
	}
	if available <= 20 {
		return constants.ProductStockStatusLowStock, available
	}
	return constants.ProductStockStatusInStock, available
}

// mapOrderErrorToResponse ???????????? API ????
// validateCallbackURL ???? URL ????(?? SSRF)
func validateCallbackURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url format")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("callback url must use http or https")
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("callback url must have a host")
	}
	// ?? localhost ?????
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0" {
		return fmt.Errorf("callback url must not point to localhost")
	}
	// ??????? IP
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("callback url must not point to private network")
		}
	}
	return nil
}

func mapOrderErrorToResponse(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrWalletInsufficientBalance):
		errorResponse(c, http.StatusPaymentRequired, "insufficient_balance", "wallet balance is insufficient")
	case errors.Is(err, service.ErrCardSecretInsufficient),
		errors.Is(err, service.ErrManualStockInsufficient):
		errorResponse(c, http.StatusConflict, "insufficient_stock", "product stock is insufficient")
	case errors.Is(err, service.ErrProductNotAvailable),
		errors.Is(err, service.ErrProductNotFound):
		errorResponse(c, http.StatusBadRequest, "product_unavailable", "product is not available")
	case errors.Is(err, service.ErrProductSKUInvalid),
		errors.Is(err, service.ErrProductSKURequired):
		errorResponse(c, http.StatusBadRequest, "sku_unavailable", "sku is invalid or not available")
	case errors.Is(err, service.ErrInvalidOrderItem):
		errorResponse(c, http.StatusBadRequest, "bad_request", "invalid order parameters")
	case errors.Is(err, service.ErrManualFormRequiredMissing),
		errors.Is(err, service.ErrManualFormFieldInvalid),
		errors.Is(err, service.ErrManualFormTypeInvalid),
		errors.Is(err, service.ErrManualFormOptionInvalid):
		errorResponse(c, http.StatusBadRequest, "bad_request", "manual form data is invalid: "+err.Error())
	default:
		logger.Errorw("upstream_create_order_failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to create order")
	}
}
