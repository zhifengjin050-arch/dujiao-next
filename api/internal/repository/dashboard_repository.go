package repository

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// DashboardRepository ?????????
// ??:???????,????????
type DashboardRepository interface {
	GetOverview(startAt, endAt time.Time) (DashboardOverviewRow, error)
	GetOrderTrends(startAt, endAt time.Time) ([]DashboardOrderTrendRow, error)
	GetPaymentTrends(startAt, endAt time.Time) ([]DashboardPaymentTrendRow, error)
	GetProfitOverview(startAt, endAt time.Time) (DashboardProfitOverviewRow, error)
	GetProfitTrends(startAt, endAt time.Time) ([]DashboardProfitTrendRow, error)
	GetStockStats(lowStockThreshold int64) (DashboardStockStatsRow, error)
	GetInventoryAlertItems(lowStockThreshold int64) ([]DashboardInventoryAlertRow, error)
	GetTopProducts(startAt, endAt time.Time, limit int) ([]DashboardProductRankingRow, error)
	GetTopChannels(startAt, endAt time.Time, limit int) ([]DashboardChannelRankingRow, error)
	GetTotalUserBalance() (float64, error)
}

// DashboardOverviewRow ???????????
type DashboardOverviewRow struct {
	OrdersTotal          int64
	PaidOrders           int64
	CompletedOrders      int64
	PendingPaymentOrders int64
	ProcessingOrders     int64
	GMVPaid              float64
	PaymentsTotal        int64
	PaymentsSuccess      int64
	PaymentsFailed       int64
	NewUsers             int64
	ActiveProducts       int64
	Currency             string
}

// DashboardProfitOverviewRow ??????????
type DashboardProfitOverviewRow struct {
	TotalRevenue float64
	TotalCost    float64
}

// DashboardProfitTrendRow ??????
type DashboardProfitTrendRow struct {
	Day     string
	Revenue float64
	Cost    float64
}

// DashboardOrderTrendRow ??????
type DashboardOrderTrendRow struct {
	Day         string
	OrdersTotal int64
	OrdersPaid  int64
}

// DashboardPaymentTrendRow ??????
type DashboardPaymentTrendRow struct {
	Day             string
	PaymentsSuccess int64
	PaymentsFailed  int64
	GMVPaid         float64
}

// DashboardStockStatsRow ????
type DashboardStockStatsRow struct {
	OutOfStockProducts   int64
	LowStockProducts     int64
	OutOfStockSKUs       int64
	LowStockSKUs         int64
	AutoAvailableSecrets int64
	ManualAvailableUnits int64
}

// DashboardInventoryAlertRow ???????
type DashboardInventoryAlertRow struct {
	ProductID         uint
	SKUID             uint
	ProductTitleJSON  models.JSON
	SKUCode           string
	SKUSpecValuesJSON models.JSON
	FulfillmentType   string
	AlertType         string
	AvailableStock    int64
}

// DashboardProductRankingRow ???????
type DashboardProductRankingRow struct {
	ProductID  uint
	Title      string
	PaidOrders int64
	Quantity   int64
	PaidAmount float64
	TotalCost  float64
}

// DashboardChannelRankingRow ???????
type DashboardChannelRankingRow struct {
	ChannelID     uint
	ChannelName   string
	ProviderType  string
	ChannelType   string
	SuccessCount  int64
	FailedCount   int64
	SuccessAmount float64
}

// GormDashboardRepository GORM ???????
type GormDashboardRepository struct {
	db *gorm.DB
}

// NewDashboardRepository ???????
func NewDashboardRepository(db *gorm.DB) *GormDashboardRepository {
	return &GormDashboardRepository{db: db}
}

func paidOrderStatuses() []string {
	return []string{
		constants.OrderStatusPaid,
		constants.OrderStatusFulfilling,
		constants.OrderStatusPartiallyDelivered,
		constants.OrderStatusPartiallyRefunded,
		constants.OrderStatusDelivered,
		constants.OrderStatusCompleted,
	}
}

func profitOrderStatuses() []string {
	statuses := append([]string{}, paidOrderStatuses()...)
	return append(statuses, constants.OrderStatusRefunded)
}

func onlinePaymentBase(db *gorm.DB, startAt, endAt time.Time) *gorm.DB {
	return db.Model(&models.Payment{}).
		Where("created_at >= ? AND created_at < ? AND provider_type <> ?", startAt, endAt, constants.PaymentProviderWallet)
}

func resolveDashboardManualAvailableStock(product models.Product) (int64, bool) {
	activeSKUs := activeProductSKUs(product.SKUs)
	if len(activeSKUs) == 0 {
		if product.ManualStockTotal == constants.ManualStockUnlimited {
			return 0, true
		}
		available := int64(product.ManualStockTotal)
		if available < 0 {
			available = 0
		}
		return available, false
	}

	total := int64(0)
	for _, sku := range activeSKUs {
		if sku.ManualStockTotal == constants.ManualStockUnlimited {
			return 0, true
		}
		available := int64(sku.ManualStockTotal)
		if available < 0 {
			available = 0
		}
		total += available
	}
	return total, false
}

// GetOverview ??????
func (r *GormDashboardRepository) GetOverview(startAt, endAt time.Time) (DashboardOverviewRow, error) {
	result := DashboardOverviewRow{}

	// ????:? 6 ??? COUNT/SUM ????? 1 ?
	paidIn := quotedStatusList(paidOrderStatuses())
	processingStatuses := []string{
		constants.OrderStatusPaid,
		constants.OrderStatusFulfilling,
		constants.OrderStatusPartiallyDelivered,
		constants.OrderStatusDelivered,
	}
	processingIn := quotedStatusList(processingStatuses)

	var orderAgg struct {
		OrdersTotal          int64   `gorm:"column:orders_total"`
		PaidOrders           int64   `gorm:"column:paid_orders"`
		CompletedOrders      int64   `gorm:"column:completed_orders"`
		PendingPaymentOrders int64   `gorm:"column:pending_payment_orders"`
		ProcessingOrders     int64   `gorm:"column:processing_orders"`
		GMVPaid              float64 `gorm:"column:gmv_paid"`
	}
	orderSelectSQL := fmt.Sprintf(`
		COUNT(*) as orders_total,
		COALESCE(SUM(CASE WHEN status IN (%s) THEN 1 ELSE 0 END), 0) as paid_orders,
		COALESCE(SUM(CASE WHEN status = '%s' THEN 1 ELSE 0 END), 0) as completed_orders,
		COALESCE(SUM(CASE WHEN status = '%s' THEN 1 ELSE 0 END), 0) as pending_payment_orders,
		COALESCE(SUM(CASE WHEN status IN (%s) THEN 1 ELSE 0 END), 0) as processing_orders,
		COALESCE(SUM(CASE WHEN status IN (%s) THEN total_amount ELSE 0 END), 0) as gmv_paid
	`, paidIn, constants.OrderStatusCompleted, constants.OrderStatusPendingPayment, processingIn, paidIn)

	if err := r.db.Model(&models.Order{}).
		Select(orderSelectSQL).
		Where("parent_id IS NULL AND created_at >= ? AND created_at < ?", startAt, endAt).
		Scan(&orderAgg).Error; err != nil {
		return result, err
	}
	result.OrdersTotal = orderAgg.OrdersTotal
	result.PaidOrders = orderAgg.PaidOrders
	result.CompletedOrders = orderAgg.CompletedOrders
	result.PendingPaymentOrders = orderAgg.PendingPaymentOrders
	result.ProcessingOrders = orderAgg.ProcessingOrders
	result.GMVPaid = orderAgg.GMVPaid

	// ????:? 3 ??? COUNT ????? 1 ?
	var paymentAgg struct {
		PaymentsTotal   int64 `gorm:"column:payments_total"`
		PaymentsSuccess int64 `gorm:"column:payments_success"`
		PaymentsFailed  int64 `gorm:"column:payments_failed"`
	}
	paymentSelectSQL := fmt.Sprintf(`
		COUNT(*) as payments_total,
		COALESCE(SUM(CASE WHEN status = '%s' THEN 1 ELSE 0 END), 0) as payments_success,
		COALESCE(SUM(CASE WHEN status = '%s' THEN 1 ELSE 0 END), 0) as payments_failed
	`, constants.PaymentStatusSuccess, constants.PaymentStatusFailed)

	if err := onlinePaymentBase(r.db, startAt, endAt).
		Select(paymentSelectSQL).
		Scan(&paymentAgg).Error; err != nil {
		return result, err
	}
	result.PaymentsTotal = paymentAgg.PaymentsTotal
	result.PaymentsSuccess = paymentAgg.PaymentsSuccess
	result.PaymentsFailed = paymentAgg.PaymentsFailed

	if err := r.db.Model(&models.User{}).
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Count(&result.NewUsers).Error; err != nil {
		return result, err
	}

	if err := r.db.Model(&models.Product{}).
		Where("is_active = ?", true).
		Count(&result.ActiveProducts).Error; err != nil {
		return result, err
	}

	_ = r.db.Model(&models.Order{}).
		Where("parent_id IS NULL AND created_at >= ? AND created_at < ? AND currency <> ''", startAt, endAt).
		Order("id DESC").
		Limit(1).
		Pluck("currency", &result.Currency).Error
	// ?????????,????????????
	if result.Currency == "" {
		_ = r.db.Model(&models.Order{}).
			Where("parent_id IS NULL AND currency <> ''").
			Order("id DESC").
			Limit(1).
			Pluck("currency", &result.Currency).Error
	}

	return result, nil
}

// GetOrderTrends ??????
func (r *GormDashboardRepository) GetOrderTrends(startAt, endAt time.Time) ([]DashboardOrderTrendRow, error) {
	dayExpr := dateGroupExpr(r.db, "created_at", startAt.Location(), startAt)
	paidIn := quotedStatusList(paidOrderStatuses())

	rows := make([]DashboardOrderTrendRow, 0)
	selectSQL := fmt.Sprintf(`
		%s as day,
		COUNT(*) as orders_total,
		COALESCE(SUM(CASE WHEN status IN (%s) THEN 1 ELSE 0 END), 0) as orders_paid
	`, dayExpr, paidIn)

	if err := r.db.Model(&models.Order{}).
		Select(selectSQL).
		Where("parent_id IS NULL AND created_at >= ? AND created_at < ?", startAt, endAt).
		Group(dayExpr).
		Order("day ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetPaymentTrends ??????
func (r *GormDashboardRepository) GetPaymentTrends(startAt, endAt time.Time) ([]DashboardPaymentTrendRow, error) {
	dayExpr := dateGroupExpr(r.db, "created_at", startAt.Location(), startAt)

	rows := make([]DashboardPaymentTrendRow, 0)
	selectSQL := fmt.Sprintf(`
		%s as day,
		COALESCE(SUM(CASE WHEN status = '%s' THEN 1 ELSE 0 END), 0) as payments_success,
		COALESCE(SUM(CASE WHEN status = '%s' THEN 1 ELSE 0 END), 0) as payments_failed,
		COALESCE(SUM(CASE WHEN status = '%s' THEN amount ELSE 0 END), 0) as gmv_paid
	`, dayExpr, constants.PaymentStatusSuccess, constants.PaymentStatusFailed, constants.PaymentStatusSuccess)

	if err := onlinePaymentBase(r.db, startAt, endAt).
		Select(selectSQL).
		Group(dayExpr).
		Order("day ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetStockStats ????????
func (r *GormDashboardRepository) GetStockStats(lowStockThreshold int64) (DashboardStockStatsRow, error) {
	result := DashboardStockStatsRow{}

	products := make([]models.Product, 0)
	if err := r.db.
		Preload("SKUs", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).Order("sort_order DESC, created_at ASC")
		}).
		Where("is_active = ?", true).
		Find(&products).Error; err != nil {
		return result, err
	}

	autoProductIDs := make([]uint, 0)
	allActiveSKUIDs := make([]uint, 0)
	autoProductActiveSKUs := make(map[uint][]uint) // product_id -> active sku_ids
	for _, product := range products {
		fulfillmentType := strings.TrimSpace(product.FulfillmentType)
		if fulfillmentType == constants.FulfillmentTypeAuto {
			autoProductIDs = append(autoProductIDs, product.ID)
			skuIDs := make([]uint, 0, len(product.SKUs))
			for _, sku := range activeProductSKUs(product.SKUs) {
				skuIDs = append(skuIDs, sku.ID)
				allActiveSKUIDs = append(allActiveSKUIDs, sku.ID)
			}
			autoProductActiveSKUs[product.ID] = skuIDs
			continue
		}
		if fulfillmentType != constants.FulfillmentTypeManual {
			continue
		}
		available, unlimited := resolveDashboardManualAvailableStock(product)
		if unlimited {
			continue
		}
		result.ManualAvailableUnits += available
		switch classifyInventoryAlertType(available, lowStockThreshold) {
		case constants.NotificationAlertTypeOutOfStockProducts:
			result.OutOfStockProducts += 1
		case constants.NotificationAlertTypeLowStockProducts:
			result.LowStockProducts += 1
		}
		// ???? SKU ????
		activeSKUs := activeProductSKUs(product.SKUs)
		for _, sku := range activeSKUs {
			if sku.ManualStockTotal == constants.ManualStockUnlimited {
				continue
			}
			skuAvail := int64(sku.ManualStockTotal)
			if skuAvail < 0 {
				skuAvail = 0
			}
			switch classifyInventoryAlertType(skuAvail, lowStockThreshold) {
			case constants.NotificationAlertTypeOutOfStockProducts:
				result.OutOfStockSKUs += 1
			case constants.NotificationAlertTypeLowStockProducts:
				result.LowStockSKUs += 1
			}
		}
	}

	if len(autoProductIDs) == 0 {
		return result, nil
	}

	// ? product_id + sku_id ????,????? SKU ? sku_id=0(??)???
	type countRow struct {
		ProductID uint
		SKUID     uint `gorm:"column:sku_id"`
		Total     int64
	}
	var rows []countRow
	query := r.db.Model(&models.CardSecret{}).
		Select("product_id, sku_id, COUNT(*) as total").
		Where("product_id IN ? AND status = ?", autoProductIDs, models.CardSecretStatusAvailable)
	if len(allActiveSKUIDs) > 0 {
		query = query.Where("sku_id = 0 OR sku_id IN ?", allActiveSKUIDs)
	} else {
		query = query.Where("sku_id = 0")
	}
	if err := query.Group("product_id, sku_id").Scan(&rows).Error; err != nil {
		return result, err
	}

	// ????????,??? SKU ??
	productAvailableMap := make(map[uint]int64)
	skuAvailableMap := make(map[uint]map[uint]int64) // product_id -> sku_id -> total
	for _, item := range rows {
		productAvailableMap[item.ProductID] += item.Total
		result.AutoAvailableSecrets += item.Total
		if skuAvailableMap[item.ProductID] == nil {
			skuAvailableMap[item.ProductID] = make(map[uint]int64)
		}
		skuAvailableMap[item.ProductID][item.SKUID] = item.Total
	}

	// ??????
	for _, productID := range autoProductIDs {
		available := productAvailableMap[productID]
		switch classifyInventoryAlertType(available, lowStockThreshold) {
		case constants.NotificationAlertTypeOutOfStockProducts:
			result.OutOfStockProducts += 1
		case constants.NotificationAlertTypeLowStockProducts:
			result.LowStockProducts += 1
		}
	}

	// SKU ????
	for productID, skuIDs := range autoProductActiveSKUs {
		skuMap := skuAvailableMap[productID]
		legacyTargetSKUID := uint(0)
		if len(skuIDs) > 0 {
			legacyTargetSKUID = skuIDs[0] // ????:sku_id=0 ????????? SKU
		}
		for _, skuID := range skuIDs {
			skuAvail := int64(0)
			if skuMap != nil {
				skuAvail = skuMap[skuID]
			}
			if skuID == legacyTargetSKUID && skuMap != nil {
				skuAvail += skuMap[0]
			}
			switch classifyInventoryAlertType(skuAvail, lowStockThreshold) {
			case constants.NotificationAlertTypeOutOfStockProducts:
				result.OutOfStockSKUs += 1
			case constants.NotificationAlertTypeLowStockProducts:
				result.LowStockSKUs += 1
			}
		}
	}

	return result, nil
}

// GetInventoryAlertItems ????????
func (r *GormDashboardRepository) GetInventoryAlertItems(lowStockThreshold int64) ([]DashboardInventoryAlertRow, error) {
	products := make([]models.Product, 0)
	if err := r.db.
		Preload("SKUs", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).Order("sort_order DESC, created_at ASC")
		}).
		Where("is_active = ?", true).
		Order("sort_order DESC, created_at DESC").
		Find(&products).Error; err != nil {
		return nil, err
	}

	autoProductIDs := make([]uint, 0)
	for _, product := range products {
		if strings.TrimSpace(product.FulfillmentType) == constants.FulfillmentTypeAuto {
			autoProductIDs = append(autoProductIDs, product.ID)
		}
	}

	autoAvailableMap := make(map[uint]map[uint]int64)
	if len(autoProductIDs) > 0 {
		type countRow struct {
			ProductID uint
			SKUID     uint `gorm:"column:sku_id"`
			Total     int64
		}
		rows := make([]countRow, 0)
		if err := r.db.Model(&models.CardSecret{}).
			Select("product_id, sku_id, COUNT(*) as total").
			Where("product_id IN ? AND status = ?", autoProductIDs, models.CardSecretStatusAvailable).
			Group("product_id, sku_id").
			Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			if autoAvailableMap[row.ProductID] == nil {
				autoAvailableMap[row.ProductID] = make(map[uint]int64)
			}
			autoAvailableMap[row.ProductID][row.SKUID] = row.Total
		}
	}

	result := make([]DashboardInventoryAlertRow, 0)
	for _, product := range products {
		switch strings.TrimSpace(product.FulfillmentType) {
		case constants.FulfillmentTypeAuto:
			result = append(result, collectAutoInventoryAlertRows(product, autoAvailableMap[product.ID], lowStockThreshold)...)
		case constants.FulfillmentTypeManual:
			result = append(result, collectManualInventoryAlertRows(product, lowStockThreshold)...)
		}
	}
	return result, nil
}

// GetTopProducts ???????
func (r *GormDashboardRepository) GetTopProducts(startAt, endAt time.Time, limit int) ([]DashboardProductRankingRow, error) {
	if limit <= 0 {
		limit = 5
	}
	rows := make([]DashboardProductRankingRow, 0)
	titleExpr := localizedJSONCoalesceExpr(r.db, "order_items.title_json")
	if err := r.db.Model(&models.OrderItem{}).
		Select(fmt.Sprintf(`
			order_items.product_id as product_id,
			%s as title,
			COUNT(DISTINCT order_items.order_id) as paid_orders,
			COALESCE(SUM(order_items.quantity), 0) as quantity,
			COALESCE(SUM(order_items.total_price - order_items.coupon_discount), 0) as paid_amount,
			COALESCE(SUM(CASE WHEN order_items.cost_price > 0 THEN order_items.cost_price * order_items.quantity ELSE 0 END), 0) as total_cost
		`, titleExpr)).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.created_at >= ? AND orders.created_at < ? AND orders.status IN ?", startAt, endAt, paidOrderStatuses()).
		Group("order_items.product_id, title").
		Order("paid_amount DESC, quantity DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func collectManualInventoryAlertRows(product models.Product, lowStockThreshold int64) []DashboardInventoryAlertRow {
	result := make([]DashboardInventoryAlertRow, 0)
	activeSKUs := activeProductSKUs(product.SKUs)
	if len(activeSKUs) == 0 {
		if product.ManualStockTotal == constants.ManualStockUnlimited {
			return result
		}
		available := int64(product.ManualStockTotal)
		if available < 0 {
			available = 0
		}
		if alertType := classifyInventoryAlertType(available, lowStockThreshold); alertType != "" {
			result = append(result, DashboardInventoryAlertRow{
				ProductID:        product.ID,
				ProductTitleJSON: product.TitleJSON,
				FulfillmentType:  constants.FulfillmentTypeManual,
				AlertType:        alertType,
				AvailableStock:   available,
			})
		}
		return result
	}

	for _, sku := range activeSKUs {
		if sku.ManualStockTotal == constants.ManualStockUnlimited {
			continue
		}
		available := int64(sku.ManualStockTotal)
		if available < 0 {
			available = 0
		}
		if alertType := classifyInventoryAlertType(available, lowStockThreshold); alertType != "" {
			result = append(result, DashboardInventoryAlertRow{
				ProductID:         product.ID,
				SKUID:             sku.ID,
				ProductTitleJSON:  product.TitleJSON,
				SKUCode:           strings.TrimSpace(sku.SKUCode),
				SKUSpecValuesJSON: sku.SpecValuesJSON,
				FulfillmentType:   constants.FulfillmentTypeManual,
				AlertType:         alertType,
				AvailableStock:    available,
			})
		}
	}
	return result
}

func collectAutoInventoryAlertRows(product models.Product, availableMap map[uint]int64, lowStockThreshold int64) []DashboardInventoryAlertRow {
	result := make([]DashboardInventoryAlertRow, 0)
	activeSKUs := activeProductSKUs(product.SKUs)
	totalAvailable := int64(0)
	activeSKUSet := make(map[uint]struct{}, len(activeSKUs))
	for _, sku := range activeSKUs {
		activeSKUSet[sku.ID] = struct{}{}
	}
	legacyInactiveAvailable := int64(0)
	for skuID, total := range availableMap {
		totalAvailable += total
		if skuID == 0 {
			continue
		}
		if _, ok := activeSKUSet[skuID]; ok {
			continue
		}
		legacyInactiveAvailable += total
	}
	if len(activeSKUs) == 0 {
		if alertType := classifyInventoryAlertType(totalAvailable, lowStockThreshold); alertType != "" {
			result = append(result, DashboardInventoryAlertRow{
				ProductID:        product.ID,
				ProductTitleJSON: product.TitleJSON,
				FulfillmentType:  constants.FulfillmentTypeAuto,
				AlertType:        alertType,
				AvailableStock:   totalAvailable,
			})
		}
		return result
	}

	legacyTargetIdx := resolveDashboardLegacyStockTargetSKUIndex(activeSKUs)
	hasPositiveActive := false
	for idx, sku := range activeSKUs {
		available := availableMap[sku.ID]
		if idx == legacyTargetIdx {
			available += availableMap[0]
		}
		if len(activeSKUs) == 1 {
			available += legacyInactiveAvailable
		}
		if available > 0 {
			hasPositiveActive = true
		}
		if alertType := classifyInventoryAlertType(available, lowStockThreshold); alertType != "" {
			result = append(result, DashboardInventoryAlertRow{
				ProductID:         product.ID,
				SKUID:             sku.ID,
				ProductTitleJSON:  product.TitleJSON,
				SKUCode:           strings.TrimSpace(sku.SKUCode),
				SKUSpecValuesJSON: sku.SpecValuesJSON,
				FulfillmentType:   constants.FulfillmentTypeAuto,
				AlertType:         alertType,
				AvailableStock:    available,
			})
		}
	}
	if hasPositiveActive || legacyInactiveAvailable <= 0 {
		return result
	}
	if fallbackType := classifyInventoryAlertType(totalAvailable, lowStockThreshold); fallbackType != "" {
		return []DashboardInventoryAlertRow{
			{
				ProductID:        product.ID,
				ProductTitleJSON: product.TitleJSON,
				FulfillmentType:  constants.FulfillmentTypeAuto,
				AlertType:        fallbackType,
				AvailableStock:   totalAvailable,
			},
		}
	}
	return result
}

func activeProductSKUs(items []models.ProductSKU) []models.ProductSKU {
	result := make([]models.ProductSKU, 0, len(items))
	for _, item := range items {
		if !item.IsActive {
			continue
		}
		result = append(result, item)
	}
	return result
}

func classifyInventoryAlertType(available int64, lowStockThreshold int64) string {
	switch {
	case available <= 0:
		return constants.NotificationAlertTypeOutOfStockProducts
	case available <= lowStockThreshold:
		return constants.NotificationAlertTypeLowStockProducts
	default:
		return ""
	}
}

func resolveDashboardLegacyStockTargetSKUIndex(skus []models.ProductSKU) int {
	if len(skus) == 0 {
		return -1
	}
	defaultCode := strings.ToUpper(strings.TrimSpace(models.DefaultSKUCode))
	firstActiveIdx := -1
	for idx := range skus {
		if !skus[idx].IsActive {
			continue
		}
		if firstActiveIdx < 0 {
			firstActiveIdx = idx
		}
		if strings.ToUpper(strings.TrimSpace(skus[idx].SKUCode)) == defaultCode {
			return idx
		}
	}
	return firstActiveIdx
}

// GetProfitOverview ????????
func (r *GormDashboardRepository) GetProfitOverview(startAt, endAt time.Time) (DashboardProfitOverviewRow, error) {
	result := DashboardProfitOverviewRow{}
	if err := r.db.Model(&models.OrderItem{}).
		Select(`
			COALESCE(SUM(order_items.total_price - order_items.coupon_discount), 0) as total_revenue,
			COALESCE(SUM(order_items.cost_price * order_items.quantity), 0) as total_cost
		`).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("order_items.cost_price > 0 AND orders.created_at >= ? AND orders.created_at < ? AND orders.status IN ?", startAt, endAt, profitOrderStatuses()).
		Scan(&result).Error; err != nil {
		return result, err
	}

	var refundedAmount float64
	if err := r.db.Model(&models.OrderRefundRecord{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Scan(&refundedAmount).Error; err != nil {
		return result, err
	}
	result.TotalRevenue -= refundedAmount
	return result, nil
}

// GetProfitTrends ??????
func (r *GormDashboardRepository) GetProfitTrends(startAt, endAt time.Time) ([]DashboardProfitTrendRow, error) {
	orderDayExpr := dateGroupExpr(r.db, "orders.created_at", startAt.Location(), startAt)

	rows := make([]DashboardProfitTrendRow, 0)
	if err := r.db.Model(&models.OrderItem{}).Select(fmt.Sprintf(`
		%s as day,
		COALESCE(SUM(order_items.total_price - order_items.coupon_discount), 0) as revenue,
		COALESCE(SUM(order_items.cost_price * order_items.quantity), 0) as cost
	`, orderDayExpr)).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("order_items.cost_price > 0 AND orders.created_at >= ? AND orders.created_at < ? AND orders.status IN ?", startAt, endAt, profitOrderStatuses()).
		Group(orderDayExpr).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	type refundTrendRow struct {
		Day          string
		RefundAmount float64 `gorm:"column:refund_amount"`
	}
	refundRows := make([]refundTrendRow, 0)
	refundDayExpr := dateGroupExpr(r.db, "created_at", startAt.Location(), startAt)
	if err := r.db.Model(&models.OrderRefundRecord{}).
		Select(fmt.Sprintf(`
			%s as day,
			COALESCE(SUM(amount), 0) as refund_amount
		`, refundDayExpr)).
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Group(refundDayExpr).
		Scan(&refundRows).Error; err != nil {
		return nil, err
	}

	byDay := make(map[string]DashboardProfitTrendRow, len(rows)+len(refundRows))
	for _, row := range rows {
		byDay[row.Day] = row
	}
	for _, refundRow := range refundRows {
		day := strings.TrimSpace(refundRow.Day)
		if day == "" {
			continue
		}
		row := byDay[day]
		row.Day = day
		row.Revenue -= refundRow.RefundAmount
		byDay[day] = row
	}

	merged := make([]DashboardProfitTrendRow, 0, len(byDay))
	for _, row := range byDay {
		merged = append(merged, row)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Day < merged[j].Day
	})
	return merged, nil
}

// GetTopChannels ?????????
func (r *GormDashboardRepository) GetTopChannels(startAt, endAt time.Time, limit int) ([]DashboardChannelRankingRow, error) {
	if limit <= 0 {
		limit = 5
	}
	rows := make([]DashboardChannelRankingRow, 0)
	if err := r.db.Model(&models.Payment{}).
		Select(`
			payments.channel_id as channel_id,
			COALESCE(payment_channels.name, '') as channel_name,
			payments.provider_type as provider_type,
			payments.channel_type as channel_type,
			SUM(CASE WHEN payments.status = 'success' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN payments.status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			COALESCE(SUM(CASE WHEN payments.status = 'success' THEN payments.amount ELSE 0 END), 0) as success_amount
		`).
		Joins("LEFT JOIN payment_channels ON payment_channels.id = payments.channel_id").
		Where("payments.created_at >= ? AND payments.created_at < ? AND payments.provider_type <> ?", startAt, endAt, constants.PaymentProviderWallet).
		Group("payments.channel_id, payment_channels.name, payments.provider_type, payments.channel_type").
		Order("success_amount DESC, success_count DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetTotalUserBalance ??????????
func (r *GormDashboardRepository) GetTotalUserBalance() (float64, error) {
	var total float64
	if err := r.db.Model(&models.WalletAccount{}).
		Select("COALESCE(SUM(balance), 0)").
		Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
