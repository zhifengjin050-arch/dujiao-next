package public

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/dto"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/i18n"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/service"
	"github.com/dujiao-next/internal/version"

	"github.com/gin-gonic/gin"
)

const (
	publicConfigCacheKey = "public:config"
	publicConfigCacheTTL = 60 * time.Second
	publicLowStockLimit  = 5
)

// publicSKUView ?? SKU ????,??????
type publicSKUView struct {
	models.ProductSKU
	PromotionPriceAmount *models.Money
	MemberPriceAmount    *models.Money
}

// publicProductView ????????,???????? dto.ProductResp
type publicProductView struct {
	models.Product
	PromotionID          *uint
	PromotionName        string
	PromotionType        string
	PromotionPriceAmount *models.Money
	PromotionRules       []dto.PromotionRuleResp
	MemberPrices         []dto.MemberLevelPrice
	PublicSKUs           []publicSKUView
	ManualStockAvailable int
	AutoStockAvailable   int64
	StockStatus          string
	IsSoldOut            bool
}

// toProductResp ???????????? DTO
func (v *publicProductView) toProductResp() dto.ProductResp {
	skus := make([]dto.SKUResp, 0, len(v.PublicSKUs))
	for _, sv := range v.PublicSKUs {
		skus = append(skus, dto.SKUResp{
			ID:                   sv.ID,
			SKUCode:              sv.SKUCode,
			SpecValues:           sv.SpecValuesJSON,
			PriceAmount:          sv.PriceAmount,
			ManualStockTotal:     sv.ManualStockTotal,
			ManualStockSold:      sv.ManualStockSold,
			AutoStockAvailable:   sv.AutoStockAvailable,
			UpstreamStock:        sv.UpstreamStock,
			IsActive:             sv.IsActive,
			PromotionPriceAmount: sv.PromotionPriceAmount,
			MemberPriceAmount:    sv.MemberPriceAmount,
		})
	}

	resp := dto.ProductResp{
		ID:                   v.Product.ID,
		CategoryID:           v.Product.CategoryID,
		Slug:                 v.Product.Slug,
		Title:                v.Product.TitleJSON,
		Description:          v.Product.DescriptionJSON,
		Content:              v.Product.ContentJSON,
		PriceAmount:          v.Product.PriceAmount,
		Images:               v.Product.Images,
		Tags:                 v.Product.Tags,
		PurchaseType:         v.Product.PurchaseType,
		MaxPurchaseQuantity:  v.Product.MaxPurchaseQuantity,
		FulfillmentType:      v.Product.FulfillmentType,
		ManualFormSchema:     v.Product.ManualFormSchemaJSON,
		ManualStockAvailable: v.ManualStockAvailable,
		AutoStockAvailable:   v.AutoStockAvailable,
		StockStatus:          v.StockStatus,
		IsSoldOut:            v.IsSoldOut,
		PaymentChannelIDs:    service.DecodeChannelIDs(v.Product.PaymentChannelIDs),
		Category:             dto.NewCategoryResp(&v.Product.Category),
		SKUs:                 skus,
		PromotionID:          v.PromotionID,
		PromotionName:        v.PromotionName,
		PromotionType:        v.PromotionType,
		PromotionPriceAmount: v.PromotionPriceAmount,
		PromotionRules:       v.PromotionRules,
		MemberPrices:         v.MemberPrices,
	}
	return resp
}

// GetConfig ??????
func (h *Handler) GetConfig(c *gin.Context) {
	// ????
	defaults := map[string]interface{}{
		"languages":                        append([]string(nil), constants.SupportedLocales...),
		constants.SettingFieldSiteCurrency: constants.SiteCurrencyDefault,
		"contact": map[string]interface{}{
			"telegram": "https://t.me/dujiaoka",
			"whatsapp": "https://wa.me/1234567890",
		},
		"scripts": make([]interface{}, 0),
	}

	var cached map[string]interface{}
	if hit, err := cache.GetJSON(c.Request.Context(), publicConfigCacheKey, &cached); err == nil && hit {
		cached["server_time"] = time.Now().UnixMilli()
		cached["app_version"] = version.Version
		response.Success(c, cached)
		return
	}

	data, err := h.SettingService.GetConfig(defaults)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.config_fetch_failed", err)
		return
	}

	publicChannels, err := h.PaymentService.GetAvailableChannels(service.AvailablePaymentChannelFilter{
		PaymentType: constants.PaymentTypeOrder,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.config_fetch_failed", err)
		return
	}
	data["payment_channels"] = publicChannels

	// ??????
	if h.SettingService != nil {
		walletRechargeChannelIDs := h.SettingService.GetWalletRechargeChannelIDs()
		if len(walletRechargeChannelIDs) > 0 {
			data["wallet_recharge_channel_ids"] = walletRechargeChannelIDs
		}
		if h.SettingService.GetWalletOnlyPayment() {
			data["wallet_only_payment"] = true
		}
	}

	if h.CaptchaService != nil {
		publicCaptcha, captchaErr := h.CaptchaService.GetPublicSetting()
		if captchaErr != nil {
			shared.RespondError(c, response.CodeInternal, "error.config_fetch_failed", captchaErr)
			return
		}
		data["captcha"] = publicCaptcha
	}
	telegramAuthConfig := map[string]interface{}{
		"enabled":      false,
		"bot_username": "",
		"mini_app_url": "",
	}
	if h.TelegramAuthService != nil {
		telegramAuthConfig = h.TelegramAuthService.PublicConfig()
	} else if h.Config != nil {
		telegramAuthConfig["enabled"] = h.Config.TelegramAuth.Enabled
		telegramAuthConfig["bot_username"] = strings.TrimSpace(h.Config.TelegramAuth.BotUsername)
		telegramAuthConfig["mini_app_url"] = strings.TrimSpace(h.Config.TelegramAuth.MiniAppURL)
	}
	data["telegram_auth"] = telegramAuthConfig

	affiliateSetting, err := h.SettingService.GetAffiliateSetting()
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.config_fetch_failed", err)
		return
	}
	data["affiliate"] = service.AffiliateSettingToMap(affiliateSetting)

	// ???????
	smtpSetting, _ := h.SettingService.GetSMTPSetting(h.Config.Email)
	data["smtp_enabled"] = smtpSetting.Enabled
	registrationEnabled, _ := h.SettingService.GetRegistrationEnabled(true)
	emailVerificationEnabled, _ := h.SettingService.GetEmailVerificationEnabled(true)
	data["registration_enabled"] = registrationEnabled
	data["email_verification_enabled"] = emailVerificationEnabled

	// ????
	navConfigVal, _ := h.SettingService.GetByKey(constants.SettingKeyNavConfig)
	if navConfigVal != nil {
		data["nav_config"] = navConfigVal
	} else {
		data["nav_config"] = map[string]interface{}{
			"builtin":      map[string]interface{}{"blog": true, "notice": true, "about": true},
			"custom_items": make([]interface{}, 0),
		}
	}

	_ = cache.SetJSON(c.Request.Context(), publicConfigCacheKey, data, publicConfigCacheTTL)
	data["server_time"] = time.Now().UnixMilli()
	data["app_version"] = version.Version
	response.Success(c, data)
}

// GetPublicMemberLevels ??????????
func (h *Handler) GetPublicMemberLevels(c *gin.Context) {
	levels, err := h.MemberLevelService.ListActiveLevels()
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.member_level_fetch_failed", err)
		return
	}

	views := make([]dto.MemberLevelResp, 0, len(levels))
	for _, l := range levels {
		views = append(views, dto.MemberLevelResp{
			ID:                l.ID,
			Name:              l.NameJSON,
			Slug:              l.Slug,
			Icon:              l.Icon,
			DiscountRate:      l.DiscountRate.Decimal.InexactFloat64(),
			RechargeThreshold: l.RechargeThreshold.Decimal.InexactFloat64(),
			SpendThreshold:    l.SpendThreshold.Decimal.InexactFloat64(),
			IsDefault:         l.IsDefault,
			SortOrder:         l.SortOrder,
		})
	}
	response.Success(c, views)
}

// GetProducts ??????
func (h *Handler) GetProducts(c *gin.Context) {
	// ??????
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	// ??????
	categoryID := c.Query("category_id")
	search := strings.TrimSpace(c.Query("search"))

	products, total, err := h.ProductService.ListPublic(categoryID, search, page, pageSize)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.product_fetch_failed", err)
		return
	}

	var promotionService *service.PromotionService
	if h.PromotionRepo != nil {
		promotionService = service.NewPromotionService(h.PromotionRepo)
	}

	if err := h.ProductService.ApplyAutoStockCounts(products); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.product_fetch_failed", err)
		return
	}

	decorated := make([]dto.ProductResp, 0, len(products))
	for i := range products {
		item, derr := h.decoratePublicProduct(&products[i], promotionService)
		if derr != nil {
			shared.RespondError(c, response.CodeInternal, "error.product_fetch_failed", derr)
			return
		}
		decorated = append(decorated, item)
	}

	// ??????
	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, decorated, pagination)
}

// GetProductBySlug ?? slug ??????
func (h *Handler) GetProductBySlug(c *gin.Context) {
	slug := c.Param("slug")

	product, err := h.ProductService.GetPublicBySlug(slug)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.product_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.product_fetch_failed", err)
		return
	}

	var promotionService *service.PromotionService
	if h.PromotionRepo != nil {
		promotionService = service.NewPromotionService(h.PromotionRepo)
	}

	temp := []models.Product{*product}
	if err := h.ProductService.ApplyAutoStockCounts(temp); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.product_fetch_failed", err)
		return
	}
	*product = temp[0]

	decorated, derr := h.decoratePublicProduct(product, promotionService)
	if derr != nil {
		shared.RespondError(c, response.CodeInternal, "error.product_fetch_failed", derr)
		return
	}

	response.Success(c, decorated)
}

func (h *Handler) decoratePublicProduct(product *models.Product, promotionService *service.PromotionService, userMemberLevelID ...uint) (dto.ProductResp, error) {
	if product == nil {
		return dto.ProductResp{}, nil
	}

	item := publicProductView{Product: *product}
	displayPrice := resolvePublicDisplayPrice(product)
	displaySKUID := resolvePublicDisplaySKUID(product)
	item.Product.PriceAmount = displayPrice
	h.decorateProductStock(product, &item)

	// ??????????????
	if promotionService != nil {
		allRules, err := promotionService.GetProductPromotions(product.ID)
		if err == nil && len(allRules) > 0 {
			rules := make([]dto.PromotionRuleResp, 0, len(allRules))
			for _, r := range allRules {
				rules = append(rules, dto.PromotionRuleResp{
					ID:        r.ID,
					Name:      strings.TrimSpace(r.Name),
					Type:      strings.TrimSpace(r.Type),
					Value:     r.Value,
					MinAmount: r.MinAmount,
				})
			}
			item.PromotionRules = rules
		}
	}

	// ????????
	var memberLevelID uint
	if len(userMemberLevelID) > 0 {
		memberLevelID = userMemberLevelID[0]
	}
	if h.Container != nil && h.MemberLevelService != nil {
		levelPrices, _ := h.MemberLevelService.GetLevelPricesByProduct(product.ID)
		if len(levelPrices) > 0 {
			views := make([]dto.MemberLevelPrice, 0, len(levelPrices))
			for _, lp := range levelPrices {
				views = append(views, dto.MemberLevelPrice{
					MemberLevelID: lp.MemberLevelID,
					SKUID:         lp.SKUID,
					PriceAmount:   lp.PriceAmount,
				})
			}
			item.MemberPrices = views
		}
	}

	// ?? SKU ?????? active SKU ?????
	skuViews := make([]publicSKUView, 0, len(item.Product.SKUs))
	var displayPromotion *models.Promotion
	var displayPromotionPrice *models.Money

	for _, sku := range item.Product.SKUs {
		sv := publicSKUView{ProductSKU: sku}

		// ??????????
		if memberLevelID > 0 && h.Container != nil && h.MemberLevelService != nil && sku.IsActive {
			memberPrice, _ := h.MemberLevelService.ResolveMemberPrice(memberLevelID, product.ID, sku.ID, sku.PriceAmount.Decimal)
			if memberPrice.LessThan(sku.PriceAmount.Decimal) {
				mp := models.NewMoneyFromDecimal(memberPrice)
				sv.MemberPriceAmount = &mp
			}
		}

		if promotionService != nil && sku.IsActive {
			priceCarrier := *product
			priceCarrier.PriceAmount = sku.PriceAmount
			promotion, discountedPrice, err := promotionService.ApplyPromotion(&priceCarrier, 1)
			if err != nil && !errors.Is(err, service.ErrPromotionInvalid) {
				return dto.ProductResp{}, err
			}
			if promotion != nil && discountedPrice.Decimal.LessThan(sku.PriceAmount.Decimal) {
				sv.PromotionPriceAmount = &discountedPrice
				if displaySKUID != 0 && sku.ID == displaySKUID {
					displayPromotion = promotion
					cp := discountedPrice
					displayPromotionPrice = &cp
				}
			}
		}
		skuViews = append(skuViews, sv)
	}

	item.PublicSKUs = skuViews

	// ?????????????????,????????????? SKU?
	if displayPromotion != nil && displayPromotionPrice != nil {
		promotionID := displayPromotion.ID
		item.PromotionID = &promotionID
		item.PromotionName = strings.TrimSpace(displayPromotion.Name)
		item.PromotionType = strings.TrimSpace(displayPromotion.Type)
		item.PromotionPriceAmount = displayPromotionPrice
	}

	return item.toProductResp(), nil
}

func resolvePublicDisplayPrice(product *models.Product) models.Money {
	if product == nil {
		return models.Money{}
	}
	for _, sku := range product.SKUs {
		if !sku.IsActive {
			continue
		}
		return sku.PriceAmount
	}
	return product.PriceAmount
}

func resolvePublicDisplaySKUID(product *models.Product) uint {
	if product == nil {
		return 0
	}
	for _, sku := range product.SKUs {
		if !sku.IsActive {
			continue
		}
		return sku.ID
	}
	return 0
}

func (h *Handler) decorateProductStock(product *models.Product, item *publicProductView) {
	if product == nil || item == nil {
		return
	}

	stockStatus := constants.ProductStockStatusInStock
	manualAvailable := 0

	item.ManualStockAvailable = manualAvailable
	item.AutoStockTotal = 0
	item.AutoStockLocked = 0
	item.AutoStockSold = 0
	item.AutoStockAvailable = 0
	item.StockStatus = stockStatus
	item.IsSoldOut = false

	fulfillmentType := strings.TrimSpace(product.FulfillmentType)
	if fulfillmentType == "" {
		fulfillmentType = constants.FulfillmentTypeManual
	}

	// upstream ??:?? SKU ??????????
	if fulfillmentType == constants.FulfillmentTypeUpstream {
		h.decorateUpstreamStock(product, item)
		return
	}

	if fulfillmentType == constants.FulfillmentTypeManual {
		hasActiveSKU := false
		hasUnlimitedSKU := false
		skuRemaining := 0
		for _, sku := range product.SKUs {
			if !sku.IsActive {
				continue
			}
			hasActiveSKU = true
			if sku.ManualStockTotal == constants.ManualStockUnlimited {
				hasUnlimitedSKU = true
				continue
			}
			if sku.ManualStockTotal > 0 {
				skuRemaining += sku.ManualStockTotal
			}
		}
		if hasActiveSKU {
			if hasUnlimitedSKU {
				item.ManualStockAvailable = constants.ManualStockUnlimited
				item.StockStatus = constants.ProductStockStatusUnlimited
				item.IsSoldOut = false
				return
			}
			manualAvailable = skuRemaining
		} else if product.ManualStockTotal == constants.ManualStockUnlimited {
			item.ManualStockAvailable = constants.ManualStockUnlimited
			item.StockStatus = constants.ProductStockStatusUnlimited
			item.IsSoldOut = false
			return
		} else {
			manualAvailable = product.ManualStockTotal
			if manualAvailable < 0 {
				manualAvailable = 0
			}
		}
		item.ManualStockAvailable = manualAvailable

		switch {
		case manualAvailable <= 0:
			item.StockStatus = constants.ProductStockStatusOutOfStock
			item.IsSoldOut = true
		case manualAvailable <= publicLowStockLimit:
			item.StockStatus = constants.ProductStockStatusLowStock
		default:
			item.StockStatus = constants.ProductStockStatusInStock
		}
		return
	}

	autoAvailable := int64(0)
	autoTotal := int64(0)
	autoLocked := int64(0)
	autoSold := int64(0)
	for _, sku := range product.SKUs {
		if !sku.IsActive {
			continue
		}
		autoAvailable += sku.AutoStockAvailable
		autoTotal += sku.AutoStockTotal
		autoLocked += sku.AutoStockLocked
		autoSold += sku.AutoStockSold
	}
	item.AutoStockAvailable = autoAvailable
	item.AutoStockTotal = autoTotal
	item.AutoStockLocked = autoLocked
	item.AutoStockSold = autoSold

	switch {
	case autoAvailable <= 0:
		item.StockStatus = constants.ProductStockStatusOutOfStock
		item.IsSoldOut = true
	case autoAvailable <= int64(publicLowStockLimit):
		item.StockStatus = constants.ProductStockStatusLowStock
	default:
		item.StockStatus = constants.ProductStockStatusInStock
	}
}

// decorateUpstreamStock ?? SKU ?????????????? SKU ?????
func (h *Handler) decorateUpstreamStock(product *models.Product, item *publicProductView) {
	// ?????? ID ?? product mapping
	mapping, err := h.ProductMappingRepo.GetByLocalProductID(product.ID)
	if err != nil || mapping == nil {
		// ??????,????????(?????)
		item.Product.FulfillmentType = constants.FulfillmentTypeManual
		item.StockStatus = constants.ProductStockStatusInStock
		item.IsSoldOut = false
		return
	}

	// ????????????????:auto ?? manual
	displayType := mapping.UpstreamFulfillmentType
	if displayType != constants.FulfillmentTypeAuto {
		displayType = constants.FulfillmentTypeManual
	}
	item.Product.FulfillmentType = displayType

	// ????????? SKU ??
	skuMappings, err := h.SKUMappingRepo.ListByProductMapping(mapping.ID)
	if err != nil || len(skuMappings) == 0 {
		item.StockStatus = constants.ProductStockStatusInStock
		item.IsSoldOut = false
		return
	}

	// ??? SKU ID ????
	skuMappingByLocal := make(map[uint]*models.SKUMapping, len(skuMappings))
	for i := range skuMappings {
		skuMappingByLocal[skuMappings[i].LocalSKUID] = &skuMappings[i]
	}

	// ???? SKU ?????,?????????
	hasUnlimited := false
	totalStock := 0
	hasActiveMapping := false

	for i := range item.Product.SKUs {
		sku := &item.Product.SKUs[i]
		sm, ok := skuMappingByLocal[sku.ID]
		if !ok || !sm.UpstreamIsActive {
			sku.UpstreamStock = 0
			continue
		}
		hasActiveMapping = true
		sku.UpstreamStock = sm.UpstreamStock

		// ???????????????,?????????????????
		if displayType == constants.FulfillmentTypeAuto {
			if sm.UpstreamStock == -1 {
				sku.AutoStockAvailable = -1 // ??? auto ?? -1 ??????,?????????
			} else {
				sku.AutoStockAvailable = int64(sm.UpstreamStock)
			}
		} else {
			if sm.UpstreamStock == -1 {
				sku.ManualStockTotal = constants.ManualStockUnlimited
			} else {
				sku.ManualStockTotal = sm.UpstreamStock
			}
		}

		if sm.UpstreamStock == -1 {
			hasUnlimited = true
		} else {
			totalStock += sm.UpstreamStock
		}
	}

	if !hasActiveMapping {
		item.StockStatus = constants.ProductStockStatusOutOfStock
		item.IsSoldOut = true
		return
	}

	if hasUnlimited {
		if displayType == constants.FulfillmentTypeAuto {
			item.AutoStockAvailable = -1
		} else {
			item.ManualStockAvailable = constants.ManualStockUnlimited
		}
		item.StockStatus = constants.ProductStockStatusUnlimited
		item.IsSoldOut = false
		return
	}

	if displayType == constants.FulfillmentTypeAuto {
		item.AutoStockAvailable = int64(totalStock)
	} else {
		item.ManualStockAvailable = totalStock
	}

	switch {
	case totalStock <= 0:
		item.StockStatus = constants.ProductStockStatusOutOfStock
		item.IsSoldOut = true
	case totalStock <= publicLowStockLimit:
		item.StockStatus = constants.ProductStockStatusLowStock
	default:
		item.StockStatus = constants.ProductStockStatusInStock
	}
}

// GetPosts ????/????
func (h *Handler) GetPosts(c *gin.Context) {
	// ??????
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	// ??????
	postType := c.Query("type") // blog ? notice

	posts, total, err := h.PostService.ListPublic(postType, page, pageSize)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.post_fetch_failed", err)
		return
	}

	// ??????
	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, dto.NewPostRespList(posts), pagination)
}

// GetPostBySlug ?? slug ??????
func (h *Handler) GetPostBySlug(c *gin.Context) {
	slug := c.Param("slug")

	post, err := h.PostService.GetPublicBySlug(slug)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.post_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.post_fetch_failed", err)
		return
	}

	response.Success(c, dto.NewPostResp(post))
}

// GetCategories ??????
func (h *Handler) GetCategories(c *gin.Context) {
	categories, err := h.CategoryService.List()
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.category_fetch_failed", err)
		return
	}

	response.Success(c, dto.NewCategoryRespList(categories))
}

// CreateGuestOrderRequest ??????
type CreateGuestOrderRequest struct {
	Email               string                       `json:"email" binding:"required"`
	OrderPassword       string                       `json:"order_password" binding:"required"`
	Items               []OrderItemRequest           `json:"items" binding:"required"`
	CouponCode          string                       `json:"coupon_code"`
	AffiliateCode       string                       `json:"affiliate_code"`
	AffiliateVisitorKey string                       `json:"affiliate_visitor_key"`
	ManualFormData      map[string]models.JSON       `json:"manual_form_data"`
	CaptchaPayload      shared.CaptchaPayloadRequest `json:"captcha_payload"`
}

// CreateGuestOrder ??????
func (h *Handler) CreateGuestOrder(c *gin.Context) {
	var req CreateGuestOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	if h.CaptchaService != nil {
		if captchaErr := h.CaptchaService.Verify(constants.CaptchaSceneGuestCreateOrder, req.CaptchaPayload.ToServicePayload(), c.ClientIP()); captchaErr != nil {
			switch {
			case errors.Is(captchaErr, service.ErrCaptchaRequired):
				shared.RespondError(c, response.CodeBadRequest, "error.captcha_required", nil)
				return
			case errors.Is(captchaErr, service.ErrCaptchaInvalid):
				shared.RespondError(c, response.CodeBadRequest, "error.captcha_invalid", nil)
				return
			case errors.Is(captchaErr, service.ErrCaptchaConfigInvalid):
				shared.RespondError(c, response.CodeInternal, "error.captcha_config_invalid", captchaErr)
				return
			default:
				shared.RespondError(c, response.CodeInternal, "error.captcha_verify_failed", captchaErr)
				return
			}
		}
	}
	var items []service.CreateOrderItem
	for _, item := range req.Items {
		items = append(items, service.CreateOrderItem{
			ProductID:       item.ProductID,
			SKUID:           item.SKUID,
			Quantity:        item.Quantity,
			FulfillmentType: item.FulfillmentType,
		})
	}
	order, err := h.OrderService.CreateGuestOrder(service.CreateGuestOrderInput{
		Email:               req.Email,
		OrderPassword:       req.OrderPassword,
		Locale:              i18n.ResolveLocale(c),
		Items:               items,
		CouponCode:          req.CouponCode,
		AffiliateCode:       req.AffiliateCode,
		AffiliateVisitorKey: req.AffiliateVisitorKey,
		ClientIP:            c.ClientIP(),
		ManualFormData:      req.ManualFormData,
	})
	if err != nil {
		respondGuestOrderCreateError(c, err)
		return
	}
	orderDetail := dto.NewOrderDetail(order)
	h.enrichOrderWithAllowedChannels(order, &orderDetail)
	response.Success(c, orderDetail)
}

// CreateGuestOrderAndPayRequest ?????????????
type CreateGuestOrderAndPayRequest struct {
	Email               string                       `json:"email" binding:"required"`
	OrderPassword       string                       `json:"order_password" binding:"required"`
	Items               []OrderItemRequest           `json:"items" binding:"required"`
	CouponCode          string                       `json:"coupon_code"`
	AffiliateCode       string                       `json:"affiliate_code"`
	AffiliateVisitorKey string                       `json:"affiliate_visitor_key"`
	ManualFormData      map[string]models.JSON       `json:"manual_form_data"`
	CaptchaPayload      shared.CaptchaPayloadRequest `json:"captcha_payload"`
	ChannelID           uint                         `json:"channel_id"`
}

// CreateGuestOrderAndPay ???????????(????)
func (h *Handler) CreateGuestOrderAndPay(c *gin.Context) {
	var req CreateGuestOrderAndPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	if h.CaptchaService != nil {
		if captchaErr := h.CaptchaService.Verify(constants.CaptchaSceneGuestCreateOrder, req.CaptchaPayload.ToServicePayload(), c.ClientIP()); captchaErr != nil {
			switch {
			case errors.Is(captchaErr, service.ErrCaptchaRequired):
				shared.RespondError(c, response.CodeBadRequest, "error.captcha_required", nil)
				return
			case errors.Is(captchaErr, service.ErrCaptchaInvalid):
				shared.RespondError(c, response.CodeBadRequest, "error.captcha_invalid", nil)
				return
			case errors.Is(captchaErr, service.ErrCaptchaConfigInvalid):
				shared.RespondError(c, response.CodeInternal, "error.captcha_config_invalid", captchaErr)
				return
			default:
				shared.RespondError(c, response.CodeInternal, "error.captcha_verify_failed", captchaErr)
				return
			}
		}
	}
	var items []service.CreateOrderItem
	for _, item := range req.Items {
		items = append(items, service.CreateOrderItem{
			ProductID:       item.ProductID,
			SKUID:           item.SKUID,
			Quantity:        item.Quantity,
			FulfillmentType: item.FulfillmentType,
		})
	}
	order, err := h.OrderService.CreateGuestOrder(service.CreateGuestOrderInput{
		Email:               req.Email,
		OrderPassword:       req.OrderPassword,
		Locale:              i18n.ResolveLocale(c),
		Items:               items,
		CouponCode:          req.CouponCode,
		AffiliateCode:       req.AffiliateCode,
		AffiliateVisitorKey: req.AffiliateVisitorKey,
		ClientIP:            c.ClientIP(),
		ManualFormData:      req.ManualFormData,
	})
	if err != nil {
		respondGuestOrderCreateError(c, err)
		return
	}
	orderResp := dto.NewOrderDetail(order)
	h.enrichOrderWithAllowedChannels(order, &orderResp)

	// ?????????,?????
	if req.ChannelID == 0 {
		response.Success(c, gin.H{
			"order":    orderResp,
			"order_no": order.OrderNo,
		})
		return
	}

	result, err := h.PaymentService.CreatePayment(service.CreatePaymentInput{
		OrderID:    order.ID,
		ChannelID:  req.ChannelID,
		UseBalance: false,
		ClientIP:   c.ClientIP(),
		Context:    c.Request.Context(),
	})
	if err != nil {
		resp := gin.H{
			"order":         orderResp,
			"order_no":      order.OrderNo,
			"payment_error": err.Error(),
		}
		response.Success(c, resp)
		return
	}

	resp := gin.H{
		"order":              orderResp,
		"order_no":           order.OrderNo,
		"order_paid":         result.OrderPaid,
		"wallet_paid_amount": result.WalletPaidAmount,
		"online_pay_amount":  result.OnlinePayAmount,
	}
	if result.Payment != nil {
		resp["payment_id"] = result.Payment.ID
		resp["provider_type"] = result.Payment.ProviderType
		resp["channel_type"] = result.Payment.ChannelType
		resp["interaction_mode"] = result.Payment.InteractionMode
		resp["pay_url"] = result.Payment.PayURL
		resp["qr_code"] = result.Payment.QRCode
		resp["expires_at"] = result.Payment.ExpiredAt
	}
	response.Success(c, resp)
}

// PreviewGuestOrder ????????
func (h *Handler) PreviewGuestOrder(c *gin.Context) {
	var req CreateGuestOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	var items []service.CreateOrderItem
	for _, item := range req.Items {
		items = append(items, service.CreateOrderItem{
			ProductID:       item.ProductID,
			SKUID:           item.SKUID,
			Quantity:        item.Quantity,
			FulfillmentType: item.FulfillmentType,
		})
	}
	preview, err := h.OrderService.PreviewGuestOrder(service.CreateGuestOrderInput{
		Email:               req.Email,
		OrderPassword:       req.OrderPassword,
		Locale:              i18n.ResolveLocale(c),
		Items:               items,
		CouponCode:          req.CouponCode,
		AffiliateCode:       req.AffiliateCode,
		AffiliateVisitorKey: req.AffiliateVisitorKey,
		ClientIP:            c.ClientIP(),
		ManualFormData:      req.ManualFormData,
	})
	if err != nil {
		respondGuestOrderPreviewError(c, err)
		return
	}
	response.Success(c, preview)
}

// ListGuestOrders ????????
func (h *Handler) ListGuestOrders(c *gin.Context) {
	email := strings.TrimSpace(c.Query("email"))
	password := strings.TrimSpace(c.Query("order_password"))
	orderNo := strings.TrimSpace(c.Query("order_no"))
	if email == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.guest_email_required", nil)
		return
	}
	if password == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.guest_password_required", nil)
		return
	}

	if orderNo != "" {
		order, err := h.OrderService.GetOrderByGuestOrderNo(orderNo, email, password)
		if err != nil {
			if errors.Is(err, service.ErrGuestOrderNotFound) {
				pagination := response.Pagination{
					Page:      1,
					PageSize:  1,
					Total:     0,
					TotalPage: 1,
				}
				response.SuccessWithPage(c, []models.Order{}, pagination)
				return
			}
			shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
			return
		}
		pagination := response.Pagination{
			Page:      1,
			PageSize:  1,
			Total:     1,
			TotalPage: 1,
		}
		response.SuccessWithPage(c, dto.NewOrderSummaryList([]models.Order{*order}), pagination)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	orders, total, err := h.OrderService.ListOrdersByGuest(email, password, page, pageSize)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, dto.NewOrderSummaryList(orders), pagination)
}

// GetGuestOrderByOrderNo ????????????
func (h *Handler) GetGuestOrderByOrderNo(c *gin.Context) {
	orderNo := strings.TrimSpace(c.Param("order_no"))
	if orderNo == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	email := strings.TrimSpace(c.Query("email"))
	password := strings.TrimSpace(c.Query("order_password"))

	var order *models.Order
	var err error

	if email != "" && password != "" {
		order, err = h.OrderService.GetOrderByGuestOrderNo(orderNo, email, password)
	} else {
		order, err = h.OrderService.GetOrderByOrderNoPublic(orderNo)
	}

	if err != nil {
		if errors.Is(err, service.ErrGuestOrderNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	orderDetail := dto.NewOrderDetailTruncated(order)
	h.enrichOrderWithAllowedChannels(order, &orderDetail)
	h.enrichOrderWithRefundRecords(order, &orderDetail)
	response.Success(c, orderDetail)
}

// DownloadGuestFulfillment ????????(??)
// ?????????? order_no
func (h *Handler) DownloadGuestFulfillment(c *gin.Context) {
	email := strings.TrimSpace(c.Query("email"))
	password := strings.TrimSpace(c.Query("order_password"))
	if email == "" || password == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.guest_email_required", nil)
		return
	}
	orderNo := strings.TrimSpace(c.Param("order_no"))
	if orderNo == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	order, err := h.OrderRepo.GetAnyByOrderNoAndGuest(orderNo, email, password)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	if order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
		return
	}
	respondFulfillmentDownload(c, order)
}

func (h *Handler) LookupGuestOrder(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.keyword_required", nil)
		return
	}
	order, err := h.OrderService.LookupGuestOrder(keyword)
	if err != nil {
		if errors.Is(err, service.ErrGuestOrderNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	orderDetail := dto.NewOrderDetailTruncated(order)
	h.enrichOrderWithAllowedChannels(order, &orderDetail)
	h.enrichOrderWithRefundRecords(order, &orderDetail)
	response.Success(c, orderDetail)
}

// CreateGuestPaymentRequest ????????
type CreateGuestPaymentRequest struct {
	Email         string `json:"email"`
	OrderPassword string `json:"order_password"`
	OrderNo       string `json:"order_no" binding:"required"`
	ChannelID     uint   `json:"channel_id" binding:"required"`
}

type LatestGuestPaymentQuery struct {
	Email         string `form:"email"`
	OrderPassword string `form:"order_password"`
	OrderNo       string `form:"order_no" binding:"required"`
}

// CreateGuestPayment ??????
func (h *Handler) CreateGuestPayment(c *gin.Context) {
	var req CreateGuestPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.OrderPassword)

	var guestOrder *models.Order
	var err error

	if email != "" && password != "" {
		guestOrder, err = h.OrderService.GetOrderByGuestOrderNo(req.OrderNo, email, password)
	} else {
		guestOrder, err = h.OrderService.GetOrderByOrderNoPublic(req.OrderNo)
	}

	if err != nil {
		if errors.Is(err, service.ErrGuestOrderNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	result, err := h.PaymentService.CreatePayment(service.CreatePaymentInput{
		OrderID:    guestOrder.ID,
		ChannelID:  req.ChannelID,
		UseBalance: false,
		ClientIP:   c.ClientIP(),
		Context:    c.Request.Context(),
	})
	if err != nil {
		respondPaymentCreateError(c, err)
		return
	}
	response.Success(c, dto.NewCreatePaymentResp(result))
}

// CaptureGuestPaymentRequest ?????????
type CaptureGuestPaymentRequest struct {
	Email         string `json:"email"`
	OrderPassword string `json:"order_password"`
}

// CaptureGuestPayment ???????
func (h *Handler) CaptureGuestPayment(c *gin.Context) {
	paymentID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.payment_invalid", nil)
		return
	}
	var req CaptureGuestPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.OrderPassword)

	payment, err := h.PaymentService.GetPayment(paymentID)
	if err != nil {
		if errors.Is(err, service.ErrPaymentNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.payment_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", err)
		return
	}
	if email != "" && password != "" {
		if _, err := h.OrderService.GetOrderByGuest(payment.OrderID, email, password); err != nil {
			if errors.Is(err, service.ErrGuestOrderNotFound) {
				shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
				return
			}
			shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
			return
		}
	}

	updated, err := h.PaymentService.CapturePayment(service.CapturePaymentInput{
		PaymentID: paymentID,
		Context:   c.Request.Context(),
	})
	if err != nil {
		respondPaymentCaptureError(c, err)
		return
	}
	response.Success(c, gin.H{
		"payment_id": updated.ID,
		"status":     updated.Status,
	})
}

// GetGuestLatestPayment ???????????
func (h *Handler) GetGuestLatestPayment(c *gin.Context) {
	var query LatestGuestPaymentQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	email := strings.TrimSpace(query.Email)
	password := strings.TrimSpace(query.OrderPassword)

	var order *models.Order
	var err error

	if email != "" && password != "" {
		order, err = h.OrderService.GetOrderByGuestOrderNo(query.OrderNo, email, password)
	} else {
		order, err = h.OrderService.GetOrderByOrderNoPublic(query.OrderNo)
	}

	if err != nil {
		if errors.Is(err, service.ErrGuestOrderNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	if order.ParentID != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.payment_invalid", nil)
		return
	}
	if order.Status != constants.OrderStatusPendingPayment {
		shared.RespondError(c, response.CodeBadRequest, "error.order_status_invalid", nil)
		return
	}
	if order.ExpiresAt != nil && !order.ExpiresAt.After(time.Now()) {
		shared.RespondError(c, response.CodeBadRequest, "error.order_status_invalid", nil)
		return
	}

	payment, err := h.PaymentRepo.GetLatestPendingByOrder(order.ID, time.Now())
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_fetch_failed", err)
		return
	}
	if payment == nil {
		shared.RespondError(c, response.CodeNotFound, "error.payment_not_found", nil)
		return
	}

	response.Success(c, dto.NewLatestPaymentResp(payment, order.OrderNo))
}
