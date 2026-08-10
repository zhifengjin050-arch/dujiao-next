package service

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var reSlugSafe = regexp.MustCompile(`[^a-zA-Z0-9\x{4e00}-\x{9fff}\-]+`)

// SanitizeSlug ?? slug ? URL ????
func SanitizeSlug(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(raw) + 8)
	for _, r := range raw {
		if r == '/' || r == '\\' || r == ' ' || r == '_' || r == '.' || r == ',' {
			b.WriteByte('-')
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		if r >= 0x4e00 && r <= 0x9fff {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	result := strings.Trim(b.String(), "-")
	result = reSlugSafe.ReplaceAllString(result, "")
	result = strings.Trim(result, "-")
	repeated := regexp.MustCompile(`-{2,}`)
	result = repeated.ReplaceAllString(result, "-")
	if result == "" {
		return "product"
	}
	if len(result) > 128 {
		result = result[:128]
		result = strings.Trim(result, "-")
		repeated2 := regexp.MustCompile(`-{2,}`)
		result = repeated2.ReplaceAllString(result, "-")
	}
	return strings.ToLower(result)
}

// ProductService ??????
type ProductService struct {
	repo                 repository.ProductRepository
	productSKURepo       repository.ProductSKURepository
	cardSecretRepo       repository.CardSecretRepository
	cardSecretBatchRepo  repository.CardSecretBatchRepository
	categoryRepo         repository.CategoryRepository
	memberLevelPriceRepo repository.MemberLevelPriceRepository
	cartRepo             repository.CartRepository
	productMappingRepo   repository.ProductMappingRepository
	orderRepo            repository.OrderRepository
}

// NewProductService ??????
func NewProductService(
	repo repository.ProductRepository,
	productSKURepo repository.ProductSKURepository,
	cardSecretRepo repository.CardSecretRepository,
	cardSecretBatchRepo repository.CardSecretBatchRepository,
	categoryRepo repository.CategoryRepository,
	memberLevelPriceRepo repository.MemberLevelPriceRepository,
	cartRepo repository.CartRepository,
	productMappingRepo repository.ProductMappingRepository,
	orderRepo repository.OrderRepository,
) *ProductService {
	return &ProductService{
		repo:                 repo,
		productSKURepo:       productSKURepo,
		cardSecretRepo:       cardSecretRepo,
		cardSecretBatchRepo:  cardSecretBatchRepo,
		categoryRepo:         categoryRepo,
		memberLevelPriceRepo: memberLevelPriceRepo,
		cartRepo:             cartRepo,
		productMappingRepo:   productMappingRepo,
		orderRepo:            orderRepo,
	}
}

// CreateProductInput ??/??????
type CreateProductInput struct {
	CategoryID           uint
	Slug                 string
	SeoMetaJSON          map[string]interface{}
	TitleJSON            map[string]interface{}
	DescriptionJSON      map[string]interface{}
	ContentJSON          map[string]interface{}
	InstructionsJSON     map[string]interface{}
	ManualFormSchemaJSON map[string]interface{}
	PriceAmount          decimal.Decimal
	CostPriceAmount      decimal.Decimal
	Images               []string
	Tags                 []string
	PurchaseType         string
	MaxPurchaseQuantity  *int
	FulfillmentType      string
	ManualStockTotal     *int
	SKUs                 []ProductSKUInput
	PaymentChannelIDs    []uint
	IsAffiliateEnabled   *bool
	IsActive             *bool
	SortOrder            int
}

type ProductSKUInput struct {
	ID               uint
	SKUCode          string
	SpecValuesJSON   map[string]interface{}
	PriceAmount      decimal.Decimal
	CostPriceAmount  decimal.Decimal
	ManualStockTotal int
	IsActive         *bool
	SortOrder        int
}

// ListPublic ????????
func (s *ProductService) ListPublic(categoryID, search string, page, pageSize int) ([]models.Product, int64, error) {
	categoryIDs, err := expandPublicCategoryIDs(s.categoryRepo, categoryID)
	if err != nil {
		return nil, 0, err
	}

	filter := repository.ProductListFilter{
		Page:         page,
		PageSize:     pageSize,
		CategoryID:   categoryID,
		CategoryIDs:  categoryIDs,
		Search:       search,
		OnlyActive:   true,
		WithCategory: true,
	}
	return s.repo.List(filter)
}

// ListPublicUpdatedAfter ????????????????(?????????)
func (s *ProductService) ListPublicUpdatedAfter(updatedAfter *time.Time, page, pageSize int) ([]models.Product, int64, error) {
	filter := repository.ProductListFilter{
		Page:         page,
		PageSize:     pageSize,
		OnlyActive:   true,
		WithCategory: true,
		UpdatedAfter: updatedAfter,
	}
	return s.repo.List(filter)
}

// ListPublicExact ????????(??????,??????)
func (s *ProductService) ListPublicExact(categoryID string, page, pageSize int) ([]models.Product, int64, error) {
	filter := repository.ProductListFilter{
		Page:         page,
		PageSize:     pageSize,
		CategoryID:   categoryID,
		OnlyActive:   true,
		WithCategory: true,
	}
	return s.repo.List(filter)
}

// GetPublicBySlug ????????
func (s *ProductService) GetPublicBySlug(slug string) (*models.Product, error) {
	product, err := s.repo.GetBySlug(slug, true)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrNotFound
	}
	return product, nil
}

// ListAdmin ????????
func (s *ProductService) ListAdmin(categoryID, search, fulfillmentType, stockStatus string, lowStockThreshold int, page, pageSize int) ([]models.Product, int64, error) {
	filter := repository.ProductListFilter{
		Page:              page,
		PageSize:          pageSize,
		CategoryID:        categoryID,
		Search:            search,
		FulfillmentType:   strings.TrimSpace(fulfillmentType),
		StockStatus:       normalizeStockStatus(stockStatus),
		LowStockThreshold: lowStockThreshold,
		OnlyActive:        false,
		WithCategory:      true,
	}
	return s.repo.List(filter)
}

// GetAdminByID ????????
func (s *ProductService) GetAdminByID(id string) (*models.Product, error) {
	product, err := s.repo.GetAdminByID(id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrNotFound
	}
	return product, nil
}

// Create ????
func (s *ProductService) Create(input CreateProductInput) (*models.Product, error) {
	if err := validateProductCategoryAssignment(s.categoryRepo, input.CategoryID, 0); err != nil {
		return nil, err
	}

	input.Slug = SanitizeSlug(input.Slug)

	count, err := s.repo.CountBySlug(input.Slug, nil)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrSlugExists
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	isAffiliateEnabled := false
	if input.IsAffiliateEnabled != nil {
		isAffiliateEnabled = *input.IsAffiliateEnabled
	}
	purchaseType := normalizePurchaseType(input.PurchaseType)
	if purchaseType == "" {
		return nil, ErrProductPurchaseInvalid
	}
	fulfillmentType := normalizeFulfillmentType(input.FulfillmentType)
	if fulfillmentType == "" {
		return nil, ErrFulfillmentInvalid
	}

	priceAmount := input.PriceAmount.Round(2)
	if len(input.SKUs) == 0 && priceAmount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrProductPriceInvalid
	}

	manualStockTotal := 0
	if input.ManualStockTotal != nil {
		manualStockTotal = *input.ManualStockTotal
	}
	if manualStockTotal < constants.ManualStockUnlimited {
		return nil, ErrManualStockInvalid
	}
	maxPurchaseQuantity := 0
	if input.MaxPurchaseQuantity != nil {
		maxPurchaseQuantity = normalizeMaxPurchaseQuantity(*input.MaxPurchaseQuantity)
	}

	costPriceAmount := input.CostPriceAmount.Round(2)

	var normalizedSKUs []normalizedProductSKU
	if len(input.SKUs) > 0 {
		if s.productSKURepo == nil {
			return nil, ErrProductSKUInvalid
		}
		var normalizeErr error
		normalizedSKUs, priceAmount, manualStockTotal, normalizeErr = normalizeProductSKUInputs(input.SKUs, fulfillmentType, nil)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		costPriceAmount = minActiveCostPrice(normalizedSKUs)
	}

	product := models.Product{
		CategoryID:           input.CategoryID,
		Slug:                 input.Slug,
		SeoMetaJSON:          models.JSON(input.SeoMetaJSON),
		TitleJSON:            models.JSON(input.TitleJSON),
		DescriptionJSON:      models.JSON(input.DescriptionJSON),
		ContentJSON:          models.JSON(input.ContentJSON),
		InstructionsJSON:     models.JSON(input.InstructionsJSON),
		ManualFormSchemaJSON: models.JSON{},
		PriceAmount:          models.NewMoneyFromDecimal(priceAmount),
		CostPriceAmount:      models.NewMoneyFromDecimal(costPriceAmount),
		Images:               models.StringArray(input.Images),
		Tags:                 models.StringArray(input.Tags),
		PurchaseType:         purchaseType,
		MaxPurchaseQuantity:  maxPurchaseQuantity,
		FulfillmentType:      fulfillmentType,
		ManualStockTotal:     manualStockTotal,
		ManualStockLocked:    0,
		ManualStockSold:      0,
		PaymentChannelIDs:    EncodeChannelIDs(input.PaymentChannelIDs),
		IsAffiliateEnabled:   isAffiliateEnabled,
		IsActive:             isActive,
		SortOrder:            input.SortOrder,
	}
	if fulfillmentType == constants.FulfillmentTypeManual {
		_, normalizedSchemaJSON, err := parseManualFormSchema(models.JSON(input.ManualFormSchemaJSON))
		if err != nil {
			return nil, err
		}
		product.ManualFormSchemaJSON = normalizedSchemaJSON
	}

	if err := s.repo.Transaction(func(tx *gorm.DB) error {
		productRepo := s.repo.WithTx(tx)
		var skuRepo repository.ProductSKURepository
		var cardSecretRepo repository.CardSecretRepository
		if s.productSKURepo != nil {
			skuRepo = s.productSKURepo.WithTx(tx)
		}
		if s.cardSecretRepo != nil {
			cardSecretRepo = s.cardSecretRepo.WithTx(tx)
		}
		if err := productRepo.Create(&product); err != nil {
			return err
		}
		if len(normalizedSKUs) > 0 {
			return applyProductSKUsWithStockGuard(skuRepo, cardSecretRepo, product.ID, fulfillmentType, normalizedSKUs)
		}
		return syncSingleProductSKU(skuRepo, product.ID, priceAmount, costPriceAmount, manualStockTotal, true)
	}); err != nil {
		return nil, err
	}
	return s.repo.GetByID(strconv.FormatUint(uint64(product.ID), 10))
}

// Update ????
func (s *ProductService) Update(id string, input CreateProductInput) (*models.Product, error) {
	priceAmount := input.PriceAmount.Round(2)
	if len(input.SKUs) == 0 && priceAmount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrProductPriceInvalid
	}
	product, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrNotFound
	}
	if err := validateProductCategoryAssignment(s.categoryRepo, input.CategoryID, product.CategoryID); err != nil {
		return nil, err
	}

	input.Slug = SanitizeSlug(input.Slug)

	count, err := s.repo.CountBySlug(input.Slug, &id)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrSlugExists
	}

	product.CategoryID = input.CategoryID
	product.Category = models.Category{}
	product.Slug = input.Slug
	product.SeoMetaJSON = models.JSON(input.SeoMetaJSON)
	product.TitleJSON = models.JSON(input.TitleJSON)
	product.DescriptionJSON = models.JSON(input.DescriptionJSON)
	product.ContentJSON = models.JSON(input.ContentJSON)
	product.InstructionsJSON = models.JSON(input.InstructionsJSON)
	product.ManualFormSchemaJSON = models.JSON{}
	product.PriceAmount = models.NewMoneyFromDecimal(priceAmount)
	product.SortOrder = input.SortOrder
	product.Images = models.StringArray(input.Images)
	product.Tags = models.StringArray(input.Tags)
	product.PaymentChannelIDs = EncodeChannelIDs(input.PaymentChannelIDs)
	if input.IsActive != nil {
		product.IsActive = *input.IsActive
	}
	if input.IsAffiliateEnabled != nil {
		product.IsAffiliateEnabled = *input.IsAffiliateEnabled
	}
	rawPurchaseType := strings.TrimSpace(input.PurchaseType)
	if rawPurchaseType == "" {
		rawPurchaseType = product.PurchaseType
	}
	purchaseType := normalizePurchaseType(rawPurchaseType)
	if purchaseType == "" {
		return nil, ErrProductPurchaseInvalid
	}
	product.PurchaseType = purchaseType
	if input.MaxPurchaseQuantity != nil {
		product.MaxPurchaseQuantity = normalizeMaxPurchaseQuantity(*input.MaxPurchaseQuantity)
	}
	rawFulfillmentType := strings.TrimSpace(input.FulfillmentType)
	if rawFulfillmentType == "" {
		rawFulfillmentType = product.FulfillmentType
	}
	fulfillmentType := normalizeFulfillmentType(rawFulfillmentType)
	if fulfillmentType == "" {
		return nil, ErrFulfillmentInvalid
	}
	// ??????????????? upstream,????? auto/manual ??????
	if product.IsMapped {
		fulfillmentType = constants.FulfillmentTypeUpstream
	}
	product.FulfillmentType = fulfillmentType
	if fulfillmentType == constants.FulfillmentTypeManual {
		_, normalizedSchemaJSON, err := parseManualFormSchema(models.JSON(input.ManualFormSchemaJSON))
		if err != nil {
			return nil, err
		}
		product.ManualFormSchemaJSON = normalizedSchemaJSON
	}

	manualStockTotal := product.ManualStockTotal
	if input.ManualStockTotal != nil {
		manualStockTotal = *input.ManualStockTotal
	}
	if manualStockTotal < constants.ManualStockUnlimited {
		return nil, ErrManualStockInvalid
	}

	var normalizedSKUs []normalizedProductSKU
	if len(input.SKUs) > 0 {
		if s.productSKURepo == nil {
			return nil, ErrProductSKUInvalid
		}
		existingSKUs, listErr := s.productSKURepo.ListByProduct(product.ID, false)
		if listErr != nil {
			return nil, listErr
		}
		existingSKUMap := make(map[uint]models.ProductSKU, len(existingSKUs))
		for _, sku := range existingSKUs {
			existingSKUMap[sku.ID] = sku
		}
		var normalizeErr error
		normalizedSKUs, priceAmount, manualStockTotal, normalizeErr = normalizeProductSKUInputs(input.SKUs, fulfillmentType, existingSKUMap)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
	}

	product.PriceAmount = models.NewMoneyFromDecimal(priceAmount)
	if len(normalizedSKUs) > 0 {
		product.CostPriceAmount = models.NewMoneyFromDecimal(minActiveCostPrice(normalizedSKUs))
	} else {
		product.CostPriceAmount = models.NewMoneyFromDecimal(input.CostPriceAmount.Round(2))
	}
	product.ManualStockTotal = manualStockTotal

	if err := s.repo.Transaction(func(tx *gorm.DB) error {
		productRepo := s.repo.WithTx(tx)
		var skuRepo repository.ProductSKURepository
		var cardSecretRepo repository.CardSecretRepository
		if s.productSKURepo != nil {
			skuRepo = s.productSKURepo.WithTx(tx)
		}
		if s.cardSecretRepo != nil {
			cardSecretRepo = s.cardSecretRepo.WithTx(tx)
		}
		if len(normalizedSKUs) > 0 {
			if err := applyProductSKUsWithStockGuard(skuRepo, cardSecretRepo, product.ID, fulfillmentType, normalizedSKUs); err != nil {
				return err
			}
		}
		if err := productRepo.Update(product); err != nil {
			return err
		}
		if len(normalizedSKUs) > 0 {
			return nil
		}
		return syncSingleProductSKU(skuRepo, product.ID, priceAmount, product.CostPriceAmount.Decimal, product.ManualStockTotal, true)
	}); err != nil {
		return nil, err
	}
	return s.repo.GetByID(id)
}

func syncSingleProductSKU(skuRepo repository.ProductSKURepository, productID uint, priceAmount decimal.Decimal, costPriceAmount decimal.Decimal, manualStockTotal int, createWhenMissing bool) error {
	if skuRepo == nil || productID == 0 {
		return nil
	}
	skus, err := skuRepo.ListByProduct(productID, false)
	if err != nil {
		return err
	}
	if len(skus) == 0 {
		if !createWhenMissing {
			return nil
		}
		return skuRepo.Create(&models.ProductSKU{
			ProductID:         productID,
			SKUCode:           models.DefaultSKUCode,
			SpecValuesJSON:    models.JSON{},
			PriceAmount:       models.NewMoneyFromDecimal(priceAmount),
			CostPriceAmount:   models.NewMoneyFromDecimal(costPriceAmount),
			ManualStockTotal:  manualStockTotal,
			ManualStockLocked: 0,
			ManualStockSold:   0,
			IsActive:          true,
			SortOrder:         0,
		})
	}
	targetIndex := pickSingleModeTargetSKUIndex(skus)
	if targetIndex < 0 || targetIndex >= len(skus) {
		return ErrProductSKUInvalid
	}

	target := skus[targetIndex]
	target.PriceAmount = models.NewMoneyFromDecimal(priceAmount)
	target.CostPriceAmount = models.NewMoneyFromDecimal(costPriceAmount)
	target.ManualStockTotal = manualStockTotal
	target.IsActive = true
	if strings.TrimSpace(target.SKUCode) == "" {
		target.SKUCode = models.DefaultSKUCode
	}
	if err := skuRepo.Update(&target); err != nil {
		return err
	}

	for i := range skus {
		if i == targetIndex {
			continue
		}
		if err := skuRepo.Delete(skus[i].ID); err != nil {
			return err
		}
	}
	return nil
}

func pickSingleModeTargetSKUIndex(skus []models.ProductSKU) int {
	if len(skus) == 0 {
		return -1
	}
	defaultCode := strings.ToUpper(strings.TrimSpace(models.DefaultSKUCode))

	for i := range skus {
		if !skus[i].IsActive {
			continue
		}
		if strings.ToUpper(strings.TrimSpace(skus[i].SKUCode)) == defaultCode {
			return i
		}
	}
	for i := range skus {
		if skus[i].IsActive {
			return i
		}
	}
	for i := range skus {
		if strings.ToUpper(strings.TrimSpace(skus[i].SKUCode)) == defaultCode {
			return i
		}
	}
	return 0
}

type normalizedProductSKU struct {
	ID               uint
	SKUCode          string
	SpecValuesJSON   models.JSON
	PriceAmount      models.Money
	CostPriceAmount  models.Money
	ManualStockTotal int
	IsActive         bool
	SortOrder        int
}

func normalizeProductSKUInputs(inputs []ProductSKUInput, fulfillmentType string, existingSKUMap map[uint]models.ProductSKU) ([]normalizedProductSKU, decimal.Decimal, int, error) {
	if len(inputs) == 0 {
		return nil, decimal.Zero, 0, ErrProductSKUInvalid
	}
	seenCode := make(map[string]struct{}, len(inputs))
	normalized := make([]normalizedProductSKU, 0, len(inputs))
	hasActive := false
	minActivePrice := decimal.Zero
	manualStockTotal := 0
	hasUnlimitedStock := false

	for _, input := range inputs {
		skuCode := strings.TrimSpace(input.SKUCode)
		if skuCode == "" {
			return nil, decimal.Zero, 0, ErrProductSKUInvalid
		}
		codeKey := strings.ToLower(skuCode)
		if _, exists := seenCode[codeKey]; exists {
			return nil, decimal.Zero, 0, ErrProductSKUInvalid
		}
		seenCode[codeKey] = struct{}{}

		priceAmount := input.PriceAmount.Round(2)
		if priceAmount.LessThanOrEqual(decimal.Zero) {
			return nil, decimal.Zero, 0, ErrProductPriceInvalid
		}
		costPriceAmount := input.CostPriceAmount.Round(2)
		if costPriceAmount.LessThan(decimal.Zero) {
			return nil, decimal.Zero, 0, ErrProductPriceInvalid
		}

		manualTotal := input.ManualStockTotal
		if manualTotal < constants.ManualStockUnlimited {
			return nil, decimal.Zero, 0, ErrManualStockInvalid
		}
		if fulfillmentType != constants.FulfillmentTypeManual {
			manualTotal = 0
		}
		if existingSKUMap != nil && input.ID > 0 {
			_, ok := existingSKUMap[input.ID]
			if !ok {
				return nil, decimal.Zero, 0, ErrProductSKUInvalid
			}
		}

		isActive := true
		if input.IsActive != nil {
			isActive = *input.IsActive
		}
		specValues := models.JSON{}
		if input.SpecValuesJSON != nil {
			specValues = models.JSON(input.SpecValuesJSON)
		}

		normalized = append(normalized, normalizedProductSKU{
			ID:               input.ID,
			SKUCode:          skuCode,
			SpecValuesJSON:   specValues,
			PriceAmount:      models.NewMoneyFromDecimal(priceAmount),
			CostPriceAmount:  models.NewMoneyFromDecimal(costPriceAmount),
			ManualStockTotal: manualTotal,
			IsActive:         isActive,
			SortOrder:        input.SortOrder,
		})

		if isActive {
			if !hasActive || priceAmount.LessThan(minActivePrice) {
				minActivePrice = priceAmount
			}
			hasActive = true
			if fulfillmentType == constants.FulfillmentTypeManual {
				if manualTotal == constants.ManualStockUnlimited {
					hasUnlimitedStock = true
				} else {
					manualStockTotal += manualTotal
				}
			}
		}
	}

	if !hasActive {
		return nil, decimal.Zero, 0, ErrProductSKUInvalid
	}
	if fulfillmentType != constants.FulfillmentTypeManual {
		manualStockTotal = 0
	} else if hasUnlimitedStock {
		manualStockTotal = constants.ManualStockUnlimited
	}
	return normalized, minActivePrice, manualStockTotal, nil
}

// minActiveCostPrice ?????? SKU ???????? SKU ????
func minActiveCostPrice(skus []normalizedProductSKU) decimal.Decimal {
	first := true
	min := decimal.Zero
	for _, s := range skus {
		if !s.IsActive {
			continue
		}
		d := s.CostPriceAmount.Decimal
		if first || d.LessThan(min) {
			min = d
			first = false
		}
	}
	return min
}

func applyProductSKUsWithStockGuard(
	skuRepo repository.ProductSKURepository,
	cardSecretRepo repository.CardSecretRepository,
	productID uint,
	fulfillmentType string,
	rows []normalizedProductSKU,
) error {
	if skuRepo == nil || productID == 0 || len(rows) == 0 {
		return ErrProductSKUInvalid
	}
	existingRows, err := skuRepo.ListByProduct(productID, false)
	if err != nil {
		return err
	}
	existingByID := make(map[uint]models.ProductSKU, len(existingRows))
	existingByCode := make(map[string]models.ProductSKU, len(existingRows))
	for _, row := range existingRows {
		existingByID[row.ID] = row
		existingByCode[strings.ToLower(strings.TrimSpace(row.SKUCode))] = row
	}
	if err := ensureAutoSKUCardSecretStockSafe(cardSecretRepo, productID, fulfillmentType, existingRows, rows, existingByID, existingByCode); err != nil {
		return err
	}

	kept := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		if row.ID > 0 {
			existing, ok := existingByID[row.ID]
			if !ok {
				return ErrProductSKUInvalid
			}
			existing.SKUCode = row.SKUCode
			existing.SpecValuesJSON = row.SpecValuesJSON
			existing.PriceAmount = row.PriceAmount
			existing.CostPriceAmount = row.CostPriceAmount
			existing.ManualStockTotal = row.ManualStockTotal
			existing.IsActive = row.IsActive
			existing.SortOrder = row.SortOrder
			if err := skuRepo.Update(&existing); err != nil {
				return err
			}
			kept[existing.ID] = struct{}{}
			existingByCode[strings.ToLower(strings.TrimSpace(existing.SKUCode))] = existing
			continue
		}

		codeKey := strings.ToLower(strings.TrimSpace(row.SKUCode))
		if existing, ok := existingByCode[codeKey]; ok {
			existing.SpecValuesJSON = row.SpecValuesJSON
			existing.PriceAmount = row.PriceAmount
			existing.CostPriceAmount = row.CostPriceAmount
			existing.ManualStockTotal = row.ManualStockTotal
			existing.IsActive = row.IsActive
			existing.SortOrder = row.SortOrder
			if err := skuRepo.Update(&existing); err != nil {
				return err
			}
			kept[existing.ID] = struct{}{}
			continue
		}

		// ??? sku_code ??????,????????
		if err := skuRepo.PurgeSoftDeletedByProductAndCode(productID, row.SKUCode); err != nil {
			return err
		}
		item := models.ProductSKU{
			ProductID:         productID,
			SKUCode:           row.SKUCode,
			SpecValuesJSON:    row.SpecValuesJSON,
			PriceAmount:       row.PriceAmount,
			CostPriceAmount:   row.CostPriceAmount,
			ManualStockTotal:  row.ManualStockTotal,
			ManualStockLocked: 0,
			ManualStockSold:   0,
			IsActive:          row.IsActive,
			SortOrder:         row.SortOrder,
		}
		if err := skuRepo.Create(&item); err != nil {
			return err
		}
		kept[item.ID] = struct{}{}
	}

	for _, existing := range existingRows {
		if _, ok := kept[existing.ID]; ok {
			continue
		}
		if err := skuRepo.Delete(existing.ID); err != nil {
			return err
		}
	}
	return nil
}

func ensureAutoSKUCardSecretStockSafe(
	cardSecretRepo repository.CardSecretRepository,
	productID uint,
	fulfillmentType string,
	existingRows []models.ProductSKU,
	rows []normalizedProductSKU,
	existingByID map[uint]models.ProductSKU,
	existingByCode map[string]models.ProductSKU,
) error {
	if cardSecretRepo == nil || productID == 0 || strings.TrimSpace(fulfillmentType) != constants.FulfillmentTypeAuto {
		return nil
	}

	nextActive := make(map[uint]bool, len(existingRows))
	kept := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		if row.ID > 0 {
			existing, ok := existingByID[row.ID]
			if !ok {
				return ErrProductSKUInvalid
			}
			nextActive[existing.ID] = row.IsActive
			kept[existing.ID] = struct{}{}
			continue
		}

		codeKey := strings.ToLower(strings.TrimSpace(row.SKUCode))
		if existing, ok := existingByCode[codeKey]; ok {
			nextActive[existing.ID] = row.IsActive
			kept[existing.ID] = struct{}{}
		}
	}

	for _, existing := range existingRows {
		if _, ok := nextActive[existing.ID]; !ok {
			nextActive[existing.ID] = false
		}
		if _, ok := kept[existing.ID]; !ok {
			nextActive[existing.ID] = false
		}
		if !existing.IsActive || nextActive[existing.ID] {
			continue
		}
		total, available, used, err := cardSecretRepo.CountByProduct(productID, existing.ID)
		if err != nil {
			return err
		}
		outstanding := total - used
		if available > 0 || outstanding > 0 {
			return ErrProductSKUHasCardSecretStock
		}
	}
	return nil
}

func normalizePurchaseType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", constants.ProductPurchaseMember:
		return constants.ProductPurchaseMember
	case constants.ProductPurchaseGuest:
		return constants.ProductPurchaseGuest
	default:
		return ""
	}
}

func normalizeFulfillmentType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", constants.FulfillmentTypeManual:
		return constants.FulfillmentTypeManual
	case constants.FulfillmentTypeAuto:
		return constants.FulfillmentTypeAuto
	case constants.FulfillmentTypeUpstream:
		return constants.FulfillmentTypeUpstream
	default:
		return ""
	}
}

func normalizeStockStatus(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", "all":
		return ""
	case "low", "normal", "unlimited":
		return value
	default:
		return ""
	}
}

func expandPublicCategoryIDs(categoryRepo repository.CategoryRepository, categoryID string) ([]uint, error) {
	normalizedCategoryID := strings.TrimSpace(categoryID)
	if normalizedCategoryID == "" {
		return nil, nil
	}

	parsedCategoryID, err := strconv.ParseUint(normalizedCategoryID, 10, 64)
	if err != nil || parsedCategoryID == 0 {
		return nil, nil
	}
	if categoryRepo == nil {
		return []uint{uint(parsedCategoryID)}, nil
	}

	category, err := categoryRepo.GetByID(normalizedCategoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return []uint{uint(parsedCategoryID)}, nil
	}
	if category.ParentID > 0 {
		return []uint{category.ID}, nil
	}

	categories, err := categoryRepo.List()
	if err != nil {
		return nil, err
	}

	categoryIDs := []uint{category.ID}
	for _, item := range categories {
		if item.ParentID == category.ID {
			categoryIDs = append(categoryIDs, item.ID)
		}
	}
	return categoryIDs, nil
}

func validateProductCategoryAssignment(categoryRepo repository.CategoryRepository, categoryID uint, currentCategoryID uint) error {
	if categoryID == 0 || categoryRepo == nil {
		return nil
	}

	categoryIDText := strconv.FormatUint(uint64(categoryID), 10)
	category, err := categoryRepo.GetByID(categoryIDText)
	if err != nil {
		return err
	}
	if category == nil {
		return ErrProductCategoryInvalid
	}

	childCount, err := categoryRepo.CountChildren(categoryIDText)
	if err != nil {
		return err
	}
	if childCount > 0 && categoryID != currentCategoryID {
		return ErrProductCategoryInvalid
	}

	return nil
}

// Delete ????(??????????)
func (s *ProductService) Delete(id string) error {
	product, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if product == nil {
		return ErrNotFound
	}

	// ?????(?? SQLite ??)

	// 1. ???????????????
	available, err := s.cardSecretRepo.CountAvailable(product.ID, 0)
	if err != nil {
		return err
	}
	if available > 0 {
		return ErrProductHasStock
	}
	reserved, err := s.cardSecretRepo.CountReserved(product.ID, 0)
	if err != nil {
		return err
	}
	if reserved > 0 {
		return ErrProductHasStock
	}

	// 2. ?????????
	orderCount, err := s.orderRepo.CountOrderItemsByProduct(product.ID)
	if err != nil {
		return err
	}
	if orderCount > 0 {
		return ErrProductHasOrderRecord
	}

	// ???????
	return s.repo.Transaction(func(tx *gorm.DB) error {
		if err := s.cardSecretRepo.WithTx(tx).DeleteByProduct(product.ID); err != nil {
			return err
		}
		if err := s.cardSecretBatchRepo.WithTx(tx).DeleteByProduct(product.ID); err != nil {
			return err
		}
		if err := s.productSKURepo.WithTx(tx).DeleteByProduct(product.ID); err != nil {
			return err
		}
		if err := s.memberLevelPriceRepo.WithTx(tx).DeleteByProduct(product.ID); err != nil {
			return err
		}
		if err := s.cartRepo.WithTx(tx).DeleteByProduct(product.ID); err != nil {
			return err
		}
		if err := s.productMappingRepo.WithTx(tx).DeleteByLocalProduct(product.ID); err != nil {
			return err
		}
		return s.repo.WithTx(tx).Delete(id)
	})
}

// QuickUpdate ??????????(? is_active?sort_order)
func (s *ProductService) QuickUpdate(id string, fields map[string]interface{}) (*models.Product, error) {
	product, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrNotFound
	}
	if err := s.repo.QuickUpdate(id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetByID(id)
}

// ApplyAutoStockCounts ???????????????????
func (s *ProductService) ApplyAutoStockCounts(products []models.Product) error {
	var productIDs []uint
	for _, p := range products {
		if p.FulfillmentType == constants.FulfillmentTypeAuto {
			productIDs = append(productIDs, p.ID)
		}
	}
	if len(productIDs) == 0 {
		return nil
	}

	counts, err := s.cardSecretRepo.CountStockByProductIDs(productIDs)
	if err != nil {
		return err
	}

	// map[product_id]map[sku_id]map[status]total
	stockMap := make(map[uint]map[uint]map[string]int64)
	for _, count := range counts {
		if stockMap[count.ProductID] == nil {
			stockMap[count.ProductID] = make(map[uint]map[string]int64)
		}
		if stockMap[count.ProductID][count.SKUID] == nil {
			stockMap[count.ProductID][count.SKUID] = make(map[string]int64)
		}
		stockMap[count.ProductID][count.SKUID][count.Status] = count.Total
	}

	for i := range products {
		if products[i].FulfillmentType != constants.FulfillmentTypeAuto {
			continue
		}
		pMap := stockMap[products[i].ID]
		if pMap == nil {
			continue
		}

		var pAvailable, pLocked, pUsed int64
		for _, statusMap := range pMap {
			pAvailable += statusMap[models.CardSecretStatusAvailable]
			pLocked += statusMap[models.CardSecretStatusReserved]
			pUsed += statusMap[models.CardSecretStatusUsed]
		}
		products[i].AutoStockAvailable = pAvailable
		products[i].AutoStockTotal = pAvailable + pLocked
		products[i].AutoStockLocked = pLocked
		products[i].AutoStockSold = pUsed

		legacyTargetIdx := resolveLegacyStockTargetSKUIndex(products[i].SKUs)
		for j := range products[i].SKUs {
			skuID := products[i].SKUs[j].ID
			statusMap := pMap[skuID]

			available := statusMap[models.CardSecretStatusAvailable]
			locked := statusMap[models.CardSecretStatusReserved]
			used := statusMap[models.CardSecretStatusUsed]

			// ?? skuID ? 0 ???????,????? DEFAULT SKU?
			// ???? DEFAULT SKU,???????? SKU,????????? SKU?
			if j == legacyTargetIdx && pMap[0] != nil {
				available += pMap[0][models.CardSecretStatusAvailable]
				locked += pMap[0][models.CardSecretStatusReserved]
				used += pMap[0][models.CardSecretStatusUsed]
			}

			products[i].SKUs[j].AutoStockAvailable = available
			products[i].SKUs[j].AutoStockTotal = available + locked
			products[i].SKUs[j].AutoStockLocked = locked
			products[i].SKUs[j].AutoStockSold = used
		}
	}
	return nil
}

func resolveLegacyStockTargetSKUIndex(skus []models.ProductSKU) int {
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
	if firstActiveIdx >= 0 {
		return firstActiveIdx
	}

	// ?????:???? SKU ?,?????? DEFAULT SKU?
	for idx := range skus {
		if strings.ToUpper(strings.TrimSpace(skus[idx].SKUCode)) == defaultCode {
			return idx
		}
	}
	return 0
}
