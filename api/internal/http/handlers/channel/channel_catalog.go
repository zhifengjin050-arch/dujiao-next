package channel

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// GetCategories GET /api/v1/channel/catalog/categories?locale=zh-CN
func (h *Handler) GetCategories(c *gin.Context) {
	locale := c.DefaultQuery("locale", "zh-CN")
	defaultLocale := "zh-CN"

	categories, err := h.CategoryService.List()
	if err != nil {
		logger.Errorw("channel_catalog_list_categories", "error", err)
		respondChannelError(c, 500, 500, "internal_error", "error.internal_error", err)
		return
	}

	type categoryItem struct {
		ID           uint   `json:"id"`
		ParentID     uint   `json:"parent_id"`
		Name         string `json:"name"`
		Icon         string `json:"icon"`
		Slug         string `json:"slug"`
		ProductCount int64  `json:"product_count"`
	}

	// ????????????
	directCounts := make(map[uint]int64, len(categories))
	// ???? parentID ????
	hasChildren := make(map[uint]struct{})
	for _, cat := range categories {
		count, err := h.CategoryRepo.CountActiveProducts(fmt.Sprintf("%d", cat.ID))
		if err != nil {
			logger.Warnw("channel_catalog_count_products", "category_id", cat.ID, "error", err)
			count = 0
		}
		directCounts[cat.ID] = count
		if cat.ParentID != 0 {
			hasChildren[cat.ParentID] = struct{}{}
		}
	}

	// ????????:
	// - ????:????? ? ???? ? ??
	// - ????:????????????(?????????????)
	visibleParentIDs := make(map[uint]struct{})
	for _, cat := range categories {
		if cat.ParentID == 0 {
			count := directCounts[cat.ID]
			_, hasChild := hasChildren[cat.ID]
			if count > 0 || hasChild {
				visibleParentIDs[cat.ID] = struct{}{}
			}
		}
	}

	var items []categoryItem
	for _, cat := range categories {
		if cat.ParentID == 0 {
			if _, ok := visibleParentIDs[cat.ID]; !ok {
				continue
			}
		} else {
			if _, ok := visibleParentIDs[cat.ParentID]; !ok {
				continue
			}
		}
		items = append(items, categoryItem{
			ID:           cat.ID,
			ParentID:     cat.ParentID,
			Name:         resolveLocalizedJSON(cat.NameJSON, locale, defaultLocale),
			Icon:         cat.Icon,
			Slug:         cat.Slug,
			ProductCount: directCounts[cat.ID],
		})
	}

	respondChannelSuccess(c, gin.H{"items": items})
}

// GetProducts GET /api/v1/channel/catalog/products?locale=zh-CN&category_id=1&page=1&page_size=5
func (h *Handler) GetProducts(c *gin.Context) {
	locale := c.DefaultQuery("locale", "zh-CN")
	defaultLocale := "zh-CN"
	categoryID := c.DefaultQuery("category_id", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "5"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 20 {
		pageSize = 5
	}
	exact := c.DefaultQuery("exact", "") == "1"

	var products []models.Product
	var total int64
	var err error
	if exact {
		products, total, err = h.ProductService.ListPublicExact(categoryID, page, pageSize)
	} else {
		products, total, err = h.ProductService.ListPublic(categoryID, "", page, pageSize)
	}
	if err != nil {
		logger.Errorw("channel_catalog_list_products", "error", err)
		respondChannelError(c, 500, 500, "internal_error", "error.internal_error", err)
		return
	}

	if err := h.ProductService.ApplyAutoStockCounts(products); err != nil {
		logger.Warnw("channel_catalog_apply_stock", "error", err)
	}

	// ???????????????,???? SKU ???????????
	fulfillmentTypeMap := h.applyUpstreamMappings(products)

	currency, err := h.SettingService.GetSiteCurrency("CNY")
	if err != nil {
		logger.Warnw("channel_catalog_get_currency", "error", err)
		currency = "CNY"
	}

	// ??:?? channel_user_id ????????
	var memberLevelID uint
	if cuid := channelUserIDFromQuery(c); cuid != "" {
		user, _, err := h.UserAuthService.ResolveTelegramChannelIdentity(service.TelegramChannelIdentityInput{
			ChannelUserID: cuid,
		})
		if err == nil && user != nil {
			memberLevelID = user.MemberLevelID
		}
	}

	type productItem struct {
		ID              uint   `json:"id"`
		Title           string `json:"title"`
		Summary         string `json:"summary"`
		ImageURL        string `json:"image_url"`
		PriceFrom       string `json:"price_from"`
		MemberPriceFrom string `json:"member_price_from,omitempty"`
		Currency        string `json:"currency"`
		StockStatus     string `json:"stock_status"`
		StockCount      int64  `json:"stock_count"`
		CategoryName    string `json:"category_name"`
	}

	items := make([]productItem, 0, len(products))
	for _, p := range products {
		title := resolveLocalizedJSON(p.TitleJSON, locale, defaultLocale)
		desc := resolveLocalizedJSON(p.DescriptionJSON, locale, defaultLocale)
		summary := truncate(stripHTML(desc), 100)

		var imageURL string
		if len(p.Images) > 0 {
			imageURL = string(p.Images[0])
		}

		// ????????????????
		ft := p.FulfillmentType
		if eft, ok := fulfillmentTypeMap[p.ID]; ok {
			ft = eft
		}

		item := productItem{
			ID:           p.ID,
			Title:        title,
			Summary:      summary,
			ImageURL:     imageURL,
			PriceFrom:    p.PriceAmount.String(),
			Currency:     currency,
			StockStatus:  computeStockStatus(ft, p.AutoStockAvailable, p.ManualStockTotal),
			StockCount:   computeStockCount(ft, p.AutoStockAvailable, p.ManualStockTotal),
			CategoryName: resolveLocalizedJSON(p.Category.NameJSON, locale, defaultLocale),
		}

		// ?????
		if memberLevelID > 0 && h.MemberLevelService != nil {
			memberPrice, _ := h.MemberLevelService.ResolveMemberPrice(memberLevelID, p.ID, 0, p.PriceAmount.Decimal)
			if memberPrice.LessThan(p.PriceAmount.Decimal) {
				item.MemberPriceFrom = models.NewMoneyFromDecimal(memberPrice).String()
			}
		}

		items = append(items, item)
	}

	totalPages := int64(math.Ceil(float64(total) / float64(pageSize)))

	respondChannelSuccess(c, gin.H{
		"items":      items,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": totalPages,
	})
}

// GetProductDetail GET /api/v1/channel/catalog/products/:id?locale=zh-CN
func (h *Handler) GetProductDetail(c *gin.Context) {
	locale := c.DefaultQuery("locale", "zh-CN")
	defaultLocale := "zh-CN"
	id := c.Param("id")

	product, err := h.ProductRepo.GetByID(id)
	if err != nil {
		logger.Errorw("channel_catalog_get_product", "id", id, "error", err)
		respondChannelError(c, 500, 500, "internal_error", "error.internal_error", err)
		return
	}
	if product == nil || !product.IsActive {
		respondChannelError(c, 404, 404, "product_not_found", "error.product_not_found", nil)
		return
	}

	// ????(ApplyAutoStockCounts ?? []models.Product ??? slice ??)
	stockSlice := []models.Product{*product}
	if err := h.ProductService.ApplyAutoStockCounts(stockSlice); err != nil {
		logger.Warnw("channel_catalog_apply_stock_detail", "error", err)
	}
	fulfillmentTypeMap := h.applyUpstreamMappings(stockSlice)
	*product = stockSlice[0]

	currency, err := h.SettingService.GetSiteCurrency("CNY")
	if err != nil {
		logger.Warnw("channel_catalog_get_currency_detail", "error", err)
		currency = "CNY"
	}

	effectiveFT := product.FulfillmentType
	if eft, ok := fulfillmentTypeMap[product.ID]; ok {
		effectiveFT = eft
	}

	title := resolveLocalizedJSON(product.TitleJSON, locale, defaultLocale)
	description := stripHTML(resolveLocalizedJSON(product.ContentJSON, locale, defaultLocale))

	var imageURL string
	if len(product.Images) > 0 {
		imageURL = string(product.Images[0])
	}

	// ??:?? channel_user_id ????????
	var memberLevelID uint
	if cuid := channelUserIDFromQuery(c); cuid != "" {
		user, _, err := h.UserAuthService.ResolveTelegramChannelIdentity(service.TelegramChannelIdentityInput{
			ChannelUserID: cuid,
		})
		if err == nil && user != nil {
			memberLevelID = user.MemberLevelID
		}
	}

	type skuItem struct {
		ID          uint   `json:"id"`
		SKUCode     string `json:"sku_code"`
		SpecValues  string `json:"spec_values"`
		Price       string `json:"price"`
		MemberPrice string `json:"member_price,omitempty"`
		StockStatus string `json:"stock_status"`
		StockCount  int64  `json:"stock_count"`
	}

	skus := make([]skuItem, 0, len(product.SKUs))
	for _, sku := range product.SKUs {
		if !sku.IsActive {
			continue
		}
		specValues := resolveLocalizedJSON(sku.SpecValuesJSON, locale, defaultLocale)
		si := skuItem{
			ID:          sku.ID,
			SKUCode:     sku.SKUCode,
			SpecValues:  specValues,
			Price:       sku.PriceAmount.String(),
			StockStatus: computeStockStatus(effectiveFT, sku.AutoStockAvailable, sku.ManualStockTotal),
			StockCount:  computeStockCount(effectiveFT, sku.AutoStockAvailable, sku.ManualStockTotal),
		}
		if memberLevelID > 0 && h.MemberLevelService != nil {
			memberPrice, _ := h.MemberLevelService.ResolveMemberPrice(memberLevelID, product.ID, sku.ID, sku.PriceAmount.Decimal)
			if memberPrice.LessThan(sku.PriceAmount.Decimal) {
				si.MemberPrice = models.NewMoneyFromDecimal(memberPrice).String()
			}
		}
		skus = append(skus, si)
	}

	// ??????
	var memberPriceFrom string
	if memberLevelID > 0 && h.MemberLevelService != nil {
		mp, _ := h.MemberLevelService.ResolveMemberPrice(memberLevelID, product.ID, 0, product.PriceAmount.Decimal)
		if mp.LessThan(product.PriceAmount.Decimal) {
			memberPriceFrom = models.NewMoneyFromDecimal(mp).String()
		}
	}

	respondChannelSuccess(c, gin.H{
		"id":                    product.ID,
		"title":                 title,
		"description":           description,
		"image_url":             imageURL,
		"price_from":            product.PriceAmount.String(),
		"member_price_from":     memberPriceFrom,
		"currency":              currency,
		"stock_status":          computeStockStatus(effectiveFT, product.AutoStockAvailable, product.ManualStockTotal),
		"stock_count":           computeStockCount(effectiveFT, product.AutoStockAvailable, product.ManualStockTotal),
		"category_name":         resolveLocalizedJSON(product.Category.NameJSON, locale, defaultLocale),
		"fulfillment_type":      effectiveFT,
		"max_purchase_quantity": normalizeChannelMaxPurchaseQuantity(product.MaxPurchaseQuantity),
		"manual_form_schema":    normalizeChannelManualFormSchema(product.ManualFormSchemaJSON, locale, defaultLocale),
		"purchase_note":         "",
		"skus":                  skus,
	})
}

// GetMemberLevels GET /api/v1/channel/member-levels?locale=zh-CN
func (h *Handler) GetMemberLevels(c *gin.Context) {
	locale := c.DefaultQuery("locale", "zh-CN")
	defaultLocale := "zh-CN"

	levels, err := h.MemberLevelService.ListActiveLevels()
	if err != nil {
		logger.Errorw("channel_member_levels_list", "error", err)
		respondChannelError(c, 500, 500, "internal_error", "error.internal_error", err)
		return
	}

	type levelItem struct {
		ID                uint    `json:"id"`
		Name              string  `json:"name"`
		Slug              string  `json:"slug"`
		Icon              string  `json:"icon"`
		DiscountRate      float64 `json:"discount_rate"`
		RechargeThreshold float64 `json:"recharge_threshold"`
		SpendThreshold    float64 `json:"spend_threshold"`
		IsDefault         bool    `json:"is_default"`
		SortOrder         int     `json:"sort_order"`
	}

	items := make([]levelItem, 0, len(levels))
	for _, l := range levels {
		items = append(items, levelItem{
			ID:                l.ID,
			Name:              resolveLocalizedJSON(l.NameJSON, locale, defaultLocale),
			Slug:              l.Slug,
			Icon:              l.Icon,
			DiscountRate:      l.DiscountRate.Decimal.InexactFloat64(),
			RechargeThreshold: l.RechargeThreshold.Decimal.InexactFloat64(),
			SpendThreshold:    l.SpendThreshold.Decimal.InexactFloat64(),
			IsDefault:         l.IsDefault,
			SortOrder:         l.SortOrder,
		})
	}

	respondChannelSuccess(c, gin.H{"items": items})
}

func normalizeChannelManualFormSchema(schema models.JSON, locale, defaultLocale string) gin.H {
	fieldsRaw, ok := schema["fields"]
	if !ok {
		return gin.H{"fields": []gin.H{}}
	}

	fieldList, ok := fieldsRaw.([]interface{})
	if !ok {
		return gin.H{"fields": []gin.H{}}
	}

	fields := make([]gin.H, 0, len(fieldList))
	for _, rawField := range fieldList {
		fieldMap, ok := rawField.(map[string]interface{})
		if !ok {
			continue
		}

		field := gin.H{}
		if key, ok := fieldMap["key"].(string); ok {
			field["key"] = key
		}
		if typeValue, ok := fieldMap["type"].(string); ok {
			field["type"] = typeValue
		}
		if required, ok := fieldMap["required"].(bool); ok {
			field["required"] = required
		}
		if label := localizedFieldText(fieldMap["label"], locale, defaultLocale); label != "" {
			field["label"] = label
		}
		if placeholder := localizedFieldText(fieldMap["placeholder"], locale, defaultLocale); placeholder != "" {
			field["placeholder"] = placeholder
		}
		if regex, ok := fieldMap["regex"].(string); ok && strings.TrimSpace(regex) != "" {
			field["regex"] = regex
		}
		if minValue, ok := fieldMap["min"]; ok {
			field["min"] = minValue
		}
		if maxValue, ok := fieldMap["max"]; ok {
			field["max"] = maxValue
		}
		if maxLen, ok := fieldMap["max_len"]; ok {
			field["max_len"] = maxLen
		}
		if options, ok := fieldMap["options"].([]string); ok {
			field["options"] = options
		} else if optionsRaw, ok := fieldMap["options"].([]interface{}); ok {
			options := make([]string, 0, len(optionsRaw))
			for _, rawOption := range optionsRaw {
				option := strings.TrimSpace(fmt.Sprintf("%v", rawOption))
				if option == "" || option == "<nil>" {
					continue
				}
				options = append(options, option)
			}
			if len(options) > 0 {
				field["options"] = options
			}
		}

		fields = append(fields, field)
	}

	return gin.H{"fields": fields}
}

func localizedFieldText(raw interface{}, locale, defaultLocale string) string {
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value)
	case models.JSON:
		return strings.TrimSpace(resolveLocalizedJSON(value, locale, defaultLocale))
	case map[string]interface{}:
		return strings.TrimSpace(resolveLocalizedJSON(models.JSON(value), locale, defaultLocale))
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if text == "<nil>" {
			return ""
		}
		return text
	}
}

// computeStockCount ????????(-1 ??????)
func computeStockCount(fulfillmentType string, autoStockAvailable int64, manualStockTotal int) int64 {
	if fulfillmentType == "auto" {
		if autoStockAvailable < 0 {
			return -1
		}
		return autoStockAvailable
	}
	return int64(manualStockTotal)
}

func normalizeChannelMaxPurchaseQuantity(value int) int {
	if value <= 0 {
		return 0
	}
	return value
}

// computeStockStatus ???????
// ? auto ? manual ????,??? < 0 ????????(in_stock)?
func computeStockStatus(fulfillmentType string, autoStockAvailable int64, manualStockTotal int) string {
	if fulfillmentType == "auto" {
		if autoStockAvailable < 0 || autoStockAvailable > 0 {
			return "in_stock"
		}
		return "out_of_stock"
	}
	if manualStockTotal < 0 || manualStockTotal > 0 {
		return "in_stock"
	}
	return "out_of_stock"
}

// applyUpstreamMappings ????? upstream ???????:
//  1. ?????????????????(auto / manual);
//  2. ???? SKU ??,???????? Product/SKU ???????,
//     ??? computeStockStatus / computeStockCount ???? upstream ???????
//
// ?? productID -> displayFulfillmentType ??,?????????????????
// ??? ApplyAutoStockCounts ?????
//
// ????(? web ? decorateUpstreamStock ??):
//   - ?? ProductMapping:????(?????),???????????;
//   - ? mapping ??? SKU ??????:????(??? 0)?
func (h *Handler) applyUpstreamMappings(products []models.Product) map[uint]string {
	ftMap := make(map[uint]string)

	var mappedIDs []uint
	for _, p := range products {
		if p.IsMapped && p.FulfillmentType == constants.FulfillmentTypeUpstream {
			mappedIDs = append(mappedIDs, p.ID)
		}
	}
	if len(mappedIDs) == 0 {
		return ftMap
	}

	mappings, err := h.ProductMappingRepo.ListByLocalProductIDs(mappedIDs)
	if err != nil {
		logger.Warnw("channel_catalog_list_product_mappings", "error", err)
		// ????????????:????????????????
		for _, id := range mappedIDs {
			ftMap[id] = constants.FulfillmentTypeManual
		}
		setProductsUnlimited(products, ftMap)
		return ftMap
	}

	mappingByProduct := make(map[uint]*models.ProductMapping, len(mappings))
	mappingIDs := make([]uint, 0, len(mappings))
	for i := range mappings {
		m := &mappings[i]
		mappingByProduct[m.LocalProductID] = m
		mappingIDs = append(mappingIDs, m.ID)
		ft := m.UpstreamFulfillmentType
		if ft != constants.FulfillmentTypeAuto {
			ft = constants.FulfillmentTypeManual
		}
		ftMap[m.LocalProductID] = ft
	}

	// ?? mapping ??? mapped ??:?????
	for _, id := range mappedIDs {
		if _, ok := mappingByProduct[id]; !ok {
			ftMap[id] = constants.FulfillmentTypeManual
		}
	}

	skuMappings, err := h.SKUMappingRepo.ListByProductMappingIDs(mappingIDs)
	if err != nil {
		logger.Warnw("channel_catalog_list_sku_mappings", "error", err)
		setProductsUnlimited(products, ftMap)
		return ftMap
	}

	// ? productMappingID ??
	skusByMapping := make(map[uint][]*models.SKUMapping, len(mappingIDs))
	for i := range skuMappings {
		sm := &skuMappings[i]
		skusByMapping[sm.ProductMappingID] = append(skusByMapping[sm.ProductMappingID], sm)
	}

	for i := range products {
		p := &products[i]
		displayType, ok := ftMap[p.ID]
		if !ok {
			continue
		}

		mapping := mappingByProduct[p.ID]
		if mapping == nil {
			// ? mapping:?????
			writeProductStock(p, displayType, -1)
			continue
		}

		smByLocal := make(map[uint]*models.SKUMapping)
		for _, sm := range skusByMapping[mapping.ID] {
			smByLocal[sm.LocalSKUID] = sm
		}

		hasUnlimited := false
		hasActiveMapping := false
		totalStock := 0

		for j := range p.SKUs {
			sku := &p.SKUs[j]
			sm, ok := smByLocal[sku.ID]
			if !ok || !sm.UpstreamIsActive {
				writeSKUStock(sku, displayType, 0)
				continue
			}
			hasActiveMapping = true
			writeSKUStock(sku, displayType, sm.UpstreamStock)

			if sm.UpstreamStock < 0 {
				hasUnlimited = true
			} else {
				totalStock += sm.UpstreamStock
			}
		}

		switch {
		case !hasActiveMapping:
			writeProductStock(p, displayType, 0)
		case hasUnlimited:
			writeProductStock(p, displayType, -1)
		default:
			writeProductStock(p, displayType, totalStock)
		}
	}

	return ftMap
}

// writeProductStock ? displayType ?????? Product ??????stock<0 ?????
func writeProductStock(p *models.Product, displayType string, stock int) {
	if displayType == constants.FulfillmentTypeAuto {
		if stock < 0 {
			p.AutoStockAvailable = -1
		} else {
			p.AutoStockAvailable = int64(stock)
		}
		return
	}
	if stock < 0 {
		p.ManualStockTotal = constants.ManualStockUnlimited
	} else {
		p.ManualStockTotal = stock
	}
}

// writeSKUStock ? writeProductStock,??? SKU?
func writeSKUStock(sku *models.ProductSKU, displayType string, stock int) {
	if displayType == constants.FulfillmentTypeAuto {
		if stock < 0 {
			sku.AutoStockAvailable = -1
		} else {
			sku.AutoStockAvailable = int64(stock)
		}
		return
	}
	if stock < 0 {
		sku.ManualStockTotal = constants.ManualStockUnlimited
	} else {
		sku.ManualStockTotal = stock
	}
}

// setProductsUnlimited ? ftMap ????????(?? SKU)?????????????????
func setProductsUnlimited(products []models.Product, ftMap map[uint]string) {
	for i := range products {
		p := &products[i]
		dt, ok := ftMap[p.ID]
		if !ok {
			continue
		}
		writeProductStock(p, dt, -1)
		for j := range p.SKUs {
			writeSKUStock(&p.SKUs[j], dt, -1)
		}
	}
}
