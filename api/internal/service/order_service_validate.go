package service

import (
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"github.com/shopspring/decimal"
)

func (s *OrderService) buildOrderResult(input orderCreateParams) (*orderBuildResult, error) {
	if len(input.Items) == 0 {
		return nil, ErrInvalidOrderItem
	}
	if input.IsGuest && input.GuestEmail == "" {
		return nil, ErrGuestEmailRequired
	}
	if input.IsGuest && input.GuestPassword == "" {
		return nil, ErrGuestPasswordRequired
	}

	mergedItems, err := mergeCreateOrderItems(input.Items)
	if err != nil {
		return nil, err
	}
	if len(mergedItems) == 0 {
		return nil, ErrInvalidOrderItem
	}

	var plans []childOrderPlan
	var orderItems []models.OrderItem
	originalAmount := decimal.Zero
	memberDiscountAmount := decimal.Zero
	promotionDiscountAmount := decimal.Zero
	currency := resolveServiceSiteCurrency(s.settingService)
	now := time.Now()
	var promotionIDValue uint
	var promotionSeen bool
	promotionSame := true
	var noPromotionSeen bool

	// ????????
	var userMemberLevelID uint
	var memberLevelIDSnapshot *uint
	if input.UserID > 0 && s.userRepo != nil {
		user, _ := s.userRepo.GetByID(input.UserID)
		if user != nil && user.MemberLevelID > 0 {
			userMemberLevelID = user.MemberLevelID
			lid := user.MemberLevelID
			memberLevelIDSnapshot = &lid
		}
	}

	promotionService := NewPromotionService(s.promotionRepo)
	manualFormData := input.ManualFormData
	if manualFormData == nil {
		manualFormData = map[string]models.JSON{}
	}
	for _, item := range mergedItems {
		if item.ProductID == 0 || item.Quantity <= 0 {
			return nil, ErrInvalidOrderItem
		}
		product, err := s.productRepo.GetByID(strconv.FormatUint(uint64(item.ProductID), 10))
		if err != nil {
			return nil, err
		}
		if product == nil || !product.IsActive {
			return nil, ErrProductNotAvailable
		}
		if err := validateProductPurchaseQuantity(product, item.Quantity); err != nil {
			return nil, err
		}
		purchaseType := strings.TrimSpace(product.PurchaseType)
		if purchaseType == "" {
			purchaseType = constants.ProductPurchaseMember
		}
		if input.IsGuest && purchaseType == constants.ProductPurchaseMember {
			return nil, ErrProductPurchaseNotAllowed
		}
		sku, err := resolveProductOrderSKU(s.productSKURepo, product, item.SKUID)
		if err != nil {
			return nil, err
		}

		productCurrency := currency
		basePrice := sku.PriceAmount.Decimal.Round(2)

		// 1. ?????
		priceCarrier := *product
		priceCarrier.PriceAmount = sku.PriceAmount
		promotion, promoUnitPrice, err := promotionService.ApplyPromotion(&priceCarrier, item.Quantity)
		if err != nil {
			return nil, err
		}
		promoUnitPriceAmount := promoUnitPrice.Decimal.Round(2)

		// 2. ?????
		memberUnitPrice := basePrice
		itemMemberDiscount := decimal.Zero
		if userMemberLevelID > 0 && s.memberLevelService != nil {
			memberUnitPrice, _ = s.memberLevelService.ResolveMemberPrice(userMemberLevelID, product.ID, sku.ID, basePrice)
		}

		// 3. ????(??? vs ???)
		unitPriceAmount := promoUnitPriceAmount
		promotionDiscount := decimal.Zero
		if memberUnitPrice.LessThan(promoUnitPriceAmount) {
			// ?????,?????
			unitPriceAmount = memberUnitPrice
			itemMemberDiscount = basePrice.Sub(memberUnitPrice).
				Mul(decimal.NewFromInt(int64(item.Quantity))).
				Round(2)
			memberDiscountAmount = memberDiscountAmount.Add(itemMemberDiscount).Round(2)
			promotion = nil // ??????
		} else if promotion != nil && basePrice.GreaterThan(promoUnitPriceAmount) {
			// ????????,?????
			promotionDiscount = basePrice.Sub(promoUnitPriceAmount).
				Mul(decimal.NewFromInt(int64(item.Quantity))).
				Round(2)
			promotionDiscountAmount = promotionDiscountAmount.Add(promotionDiscount).Round(2)
		}

		if unitPriceAmount.LessThanOrEqual(decimal.Zero) || productCurrency == "" {
			return nil, ErrProductPriceInvalid
		}

		baseTotal := basePrice.Mul(decimal.NewFromInt(int64(item.Quantity))).Round(2)
		total := unitPriceAmount.Mul(decimal.NewFromInt(int64(item.Quantity))).Round(2)
		originalAmount = originalAmount.Add(baseTotal).Round(2)
		fulfillmentType := strings.TrimSpace(product.FulfillmentType)
		if fulfillmentType == "" {
			fulfillmentType = constants.FulfillmentTypeManual
		}
		if fulfillmentType != constants.FulfillmentTypeManual && fulfillmentType != constants.FulfillmentTypeAuto && fulfillmentType != constants.FulfillmentTypeUpstream {
			return nil, ErrFulfillmentInvalid
		}
		if fulfillmentType == constants.FulfillmentTypeManual &&
			shouldEnforceManualSKUStock(product, sku) &&
			manualSKUAvailable(sku) < item.Quantity {
			return nil, ErrManualStockInsufficient
		}

		manualSchemaSnapshot := models.JSON{}
		manualSubmission := models.JSON{}
		if fulfillmentType == constants.FulfillmentTypeManual ||
			(fulfillmentType == constants.FulfillmentTypeUpstream && len(product.ManualFormSchemaJSON) > 0) {
			submission := resolveManualFormSubmission(manualFormData, product.ID, sku.ID)
			normalizedSchema, normalizedSubmission, err := validateAndNormalizeManualForm(product.ManualFormSchemaJSON, submission)
			if err != nil {
				return nil, err
			}
			manualSchemaSnapshot = normalizedSchema
			manualSubmission = normalizedSubmission
		}

		var promotionID *uint
		if promotion != nil {
			pid := promotion.ID
			promotionID = &pid
			if !promotionSeen {
				promotionSeen = true
				promotionIDValue = pid
			} else if promotionIDValue != pid {
				promotionSame = false
			}
		} else {
			noPromotionSeen = true
		}

		orderItem := models.OrderItem{
			ProductID: product.ID,
			SKUID:     sku.ID,
			TitleJSON: product.TitleJSON,
			SKUSnapshotJSON: models.JSON{
				"sku_id":      sku.ID,
				"sku_code":    sku.SKUCode,
				"spec_values": sku.SpecValuesJSON,
				"image":       firstProductImage(product.Images),
			},
			Tags:                         product.Tags,
			UnitPrice:                    models.NewMoneyFromDecimal(unitPriceAmount),
			CostPrice:                    sku.CostPriceAmount, // ?????
			Quantity:                     item.Quantity,
			TotalPrice:                   models.NewMoneyFromDecimal(total),
			MemberDiscount:               models.NewMoneyFromDecimal(itemMemberDiscount),
			CouponDiscount:               models.NewMoneyFromDecimal(decimal.Zero),
			PromotionDiscount:            models.NewMoneyFromDecimal(promotionDiscount),
			PromotionID:                  promotionID,
			FulfillmentType:              fulfillmentType,
			ManualFormSchemaSnapshotJSON: manualSchemaSnapshot,
			ManualFormSubmissionJSON:     manualSubmission,
			InstructionsJSON:             product.InstructionsJSON,
			CreatedAt:                    now,
			UpdatedAt:                    now,
		}
		orderItems = append(orderItems, orderItem)
		plans = append(plans, childOrderPlan{
			Product:           product,
			SKU:               sku,
			Item:              orderItem,
			TotalAmount:       total,
			MemberDiscount:    itemMemberDiscount,
			PromotionDiscount: promotionDiscount,
			Currency:          productCurrency,
		})
	}
	if currency == "" {
		return nil, ErrInvalidOrderAmount
	}

	var orderPromotionID *uint
	if promotionSeen && promotionSame && !noPromotionSeen {
		orderPromotionID = &promotionIDValue
	}

	discountAmount := decimal.Zero
	var appliedCoupon *models.Coupon
	couponCode := strings.TrimSpace(input.CouponCode)
	if couponCode != "" {
		couponService := NewCouponService(s.couponRepo, s.couponUsageRepo)
		discount, coupon, err := couponService.ApplyCoupon(
			models.NewMoneyFromDecimal(originalAmount),
			couponCode,
			input.UserID,
			orderItems,
			input.IsGuest,
			userMemberLevelID,
		)
		if err != nil {
			return nil, err
		}
		discountAmount = discount.Decimal.Round(2)
		appliedCoupon = coupon
	}

	if appliedCoupon != nil && discountAmount.GreaterThan(decimal.Zero) {
		if err := applyCouponDiscountToItems(plans, appliedCoupon, discountAmount); err != nil {
			return nil, err
		}
		discountAmount = decimal.Zero
		for i := range plans {
			discountAmount = discountAmount.Add(plans[i].CouponDiscount).Round(2)
		}
	}

	totalAmount := decimal.Zero
	for i := range plans {
		plan := &plans[i]
		plan.Item.MemberDiscount = models.NewMoneyFromDecimal(plan.MemberDiscount)
		plan.Item.CouponDiscount = models.NewMoneyFromDecimal(plan.CouponDiscount)
		plan.Item.PromotionDiscount = models.NewMoneyFromDecimal(plan.PromotionDiscount)
		plan.Item.TotalPrice = models.NewMoneyFromDecimal(plan.TotalAmount)
		planTotal := plan.TotalAmount.Sub(plan.CouponDiscount).Round(2)
		if planTotal.LessThan(decimal.Zero) {
			planTotal = decimal.Zero
		}
		totalAmount = totalAmount.Add(planTotal).Round(2)
	}
	if totalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidOrderAmount
	}

	return &orderBuildResult{
		Plans:                   plans,
		OrderItems:              orderItems,
		OriginalAmount:          originalAmount,
		MemberDiscountAmount:    memberDiscountAmount,
		PromotionDiscountAmount: promotionDiscountAmount,
		DiscountAmount:          discountAmount,
		TotalAmount:             totalAmount,
		Currency:                currency,
		OrderPromotionID:        orderPromotionID,
		MemberLevelID:           memberLevelIDSnapshot,
		AppliedCoupon:           appliedCoupon,
	}, nil
}

func normalizeGuestEmail(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", ErrGuestEmailRequired
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

func (s *OrderService) resolveExpireMinutes() int {
	return resolveOrderPaymentExpireMinutes(s.settingService, s.expireMinutes)
}

func resolveManualFormSubmission(manualFormData map[string]models.JSON, productID, skuID uint) models.JSON {
	if len(manualFormData) == 0 || productID == 0 {
		return models.JSON{}
	}

	itemKey := buildOrderItemKey(productID, skuID)
	if submission, ok := manualFormData[itemKey]; ok {
		if submission == nil {
			return models.JSON{}
		}
		return submission
	}

	legacyKey := strconv.FormatUint(uint64(productID), 10)
	if submission, ok := manualFormData[legacyKey]; ok {
		if submission == nil {
			return models.JSON{}
		}
		return submission
	}

	return models.JSON{}
}

func firstProductImage(images models.StringArray) string {
	for _, raw := range images {
		image := strings.TrimSpace(raw)
		if image != "" {
			return image
		}
	}
	return ""
}

// mergeCreateOrderItems ??????????
func mergeCreateOrderItems(items []CreateOrderItem) ([]CreateOrderItem, error) {
	if len(items) == 0 {
		return nil, nil
	}
	merged := make([]CreateOrderItem, 0, len(items))
	indexMap := make(map[string]int)
	for _, item := range items {
		if item.ProductID == 0 || item.Quantity <= 0 {
			return nil, ErrInvalidOrderItem
		}
		key := buildOrderItemKey(item.ProductID, item.SKUID)
		if idx, ok := indexMap[key]; ok {
			merged[idx].Quantity += item.Quantity
			continue
		}
		indexMap[key] = len(merged)
		merged = append(merged, CreateOrderItem{
			ProductID: item.ProductID,
			SKUID:     item.SKUID,
			Quantity:  item.Quantity,
		})
	}
	return merged, nil
}

// applyCouponDiscountToItems ???????????
func applyCouponDiscountToItems(plans []childOrderPlan, coupon *models.Coupon, discountAmount decimal.Decimal) error {
	if coupon == nil || discountAmount.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	scopeType := strings.ToLower(strings.TrimSpace(coupon.ScopeType))
	if scopeType != constants.ScopeTypeProduct {
		return ErrCouponScopeInvalid
	}
	ids, err := decodeScopeIDs(coupon.ScopeRefIDs)
	if err != nil {
		return ErrCouponScopeInvalid
	}
	eligibleIndexes := make([]int, 0, len(plans))
	eligibleTotal := decimal.Zero
	for i := range plans {
		if _, ok := ids[plans[i].Item.ProductID]; !ok {
			continue
		}
		eligibleIndexes = append(eligibleIndexes, i)
		eligibleTotal = eligibleTotal.Add(plans[i].TotalAmount)
	}
	if len(eligibleIndexes) == 0 || eligibleTotal.LessThanOrEqual(decimal.Zero) {
		return ErrCouponScopeInvalid
	}

	remaining := discountAmount
	for i, idx := range eligibleIndexes {
		if i == len(eligibleIndexes)-1 {
			alloc := remaining.Round(2)
			if alloc.LessThan(decimal.Zero) {
				alloc = decimal.Zero
			}
			if alloc.GreaterThan(plans[idx].TotalAmount) {
				alloc = plans[idx].TotalAmount
			}
			plans[idx].CouponDiscount = alloc
			break
		}
		ratio := plans[idx].TotalAmount.Div(eligibleTotal)
		alloc := discountAmount.Mul(ratio).Round(2)
		if alloc.GreaterThan(remaining) {
			alloc = remaining
		}
		if alloc.LessThan(decimal.Zero) {
			alloc = decimal.Zero
		}
		if alloc.GreaterThan(plans[idx].TotalAmount) {
			alloc = plans[idx].TotalAmount
		}
		plans[idx].CouponDiscount = alloc
		remaining = remaining.Sub(alloc).Round(2)
	}
	return nil
}

// buildChildOrderNo ??????
func buildChildOrderNo(parentOrderNo string, seq int) string {
	if seq <= 0 {
		return parentOrderNo
	}
	return fmt.Sprintf("%s-%02d", parentOrderNo, seq)
}

// fillOrderItemsFromChildren ?????????(??????)
func fillOrderItemsFromChildren(order *models.Order) {
	if order == nil || len(order.Items) > 0 || len(order.Children) == 0 {
		return
	}
	items := make([]models.OrderItem, 0)
	for _, child := range order.Children {
		for _, item := range child.Items {
			copied := item
			copied.OrderID = order.ID
			items = append(items, copied)
		}
	}
	order.Items = items
}

// fillOrdersItemsFromChildren ?????????
func fillOrdersItemsFromChildren(orders []models.Order) {
	for i := range orders {
		fillOrderItemsFromChildren(&orders[i])
	}
}
