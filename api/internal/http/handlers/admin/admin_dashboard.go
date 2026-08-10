package admin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// GetDashboardOverview ?????????
func (h *Handler) GetDashboardOverview(c *gin.Context) {
	input, err := parseDashboardQuery(c)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	data, err := h.DashboardService.GetOverview(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrDashboardRangeInvalid) {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.dashboard_fetch_failed", err)
		return
	}

	response.Success(c, data)
}

// GetDashboardRankings ??????????
func (h *Handler) GetDashboardRankings(c *gin.Context) {
	input, err := parseDashboardQuery(c)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	data, err := h.DashboardService.GetRankings(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrDashboardRangeInvalid) {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.dashboard_fetch_failed", err)
		return
	}

	response.Success(c, data)
}

// GetDashboardTrends ?????????
func (h *Handler) GetDashboardTrends(c *gin.Context) {
	input, err := parseDashboardQuery(c)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	data, err := h.DashboardService.GetTrends(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrDashboardRangeInvalid) {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.dashboard_fetch_failed", err)
		return
	}

	response.Success(c, data)
}

// GetDashboardInventoryAlerts ?? SKU ????????
func (h *Handler) GetDashboardInventoryAlerts(c *gin.Context) {
	setting := h.DashboardService.LoadDashboardAlertSetting()
	items, err := h.DashboardService.GetInventoryAlertItems(c.Request.Context(), setting.LowStockThreshold)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.dashboard_fetch_failed", err)
		return
	}

	type inventoryAlertResponse struct {
		ProductID       uint                   `json:"product_id"`
		SKUID           uint                   `json:"sku_id,omitempty"`
		ProductTitle    map[string]interface{} `json:"product_title"`
		SKUCode         string                 `json:"sku_code,omitempty"`
		SKUSpecValues   map[string]interface{} `json:"sku_spec_values,omitempty"`
		FulfillmentType string                 `json:"fulfillment_type"`
		AlertType       string                 `json:"alert_type"`
		AvailableStock  int64                  `json:"available_stock"`
	}

	result := make([]inventoryAlertResponse, 0, len(items))
	for _, item := range items {
		row := inventoryAlertResponse{
			ProductID:       item.ProductID,
			SKUID:           item.SKUID,
			ProductTitle:    item.ProductTitleJSON,
			SKUCode:         item.SKUCode,
			FulfillmentType: item.FulfillmentType,
			AlertType:       item.AlertType,
			AvailableStock:  item.AvailableStock,
		}
		if item.SKUSpecValuesJSON != nil {
			row.SKUSpecValues = item.SKUSpecValuesJSON
		}
		result = append(result, row)
	}
	response.Success(c, result)
}

func parseDashboardQuery(c *gin.Context) (service.DashboardQueryInput, error) {
	rangeRaw := strings.TrimSpace(c.DefaultQuery("range", "7d"))
	fromRaw := strings.TrimSpace(c.Query("from"))
	toRaw := strings.TrimSpace(c.Query("to"))
	timezone := strings.TrimSpace(c.Query("tz"))
	forceRefreshRaw := strings.TrimSpace(c.Query("force_refresh"))

	from, err := shared.ParseTimeNullable(fromRaw)
	if err != nil {
		return service.DashboardQueryInput{}, err
	}
	to, err := shared.ParseTimeNullable(toRaw)
	if err != nil {
		return service.DashboardQueryInput{}, err
	}

	forceRefresh := false
	if forceRefreshRaw != "" {
		parsed, err := strconv.ParseBool(forceRefreshRaw)
		if err != nil {
			return service.DashboardQueryInput{}, err
		}
		forceRefresh = parsed
	}

	return service.DashboardQueryInput{
		Range:        rangeRaw,
		From:         from,
		To:           to,
		Timezone:     timezone,
		ForceRefresh: forceRefresh,
	}, nil
}
