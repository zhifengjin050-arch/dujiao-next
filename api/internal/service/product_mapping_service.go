package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/upstream"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrMappingNotFound         = errors.New("product mapping not found")
	ErrMappingAlreadyExists    = errors.New("product mapping already exists for this upstream product")
	ErrUpstreamProductNotFound = errors.New("upstream product not found")
	ErrMappingInactive         = errors.New("product mapping is inactive")
)

// ProductMappingService 商品映射业务服务
type ProductMappingService struct {
	mappingRepo     repository.ProductMappingRepository
	skuMappingRepo  repository.SKUMappingRepository
	productRepo     repository.ProductRepository
	productSKURepo  repository.ProductSKURepository
	categoryRepo    repository.CategoryRepository
	connService     *SiteConnectionService
	categoryService *CategoryService
	mediaService    *MediaService
}

// NewProductMappingService 创建商品映射服务
func NewProductMappingService(
	mappingRepo repository.ProductMappingRepository,
	skuMappingRepo repository.SKUMappingRepository,
	productRepo repository.ProductRepository,
	productSKURepo repository.ProductSKURepository,
	categoryRepo repository.CategoryRepository,
	connService *SiteConnectionService,
) *ProductMappingService {
	return &ProductMappingService{
		mappingRepo:    mappingRepo,
		skuMappingRepo: skuMappingRepo,
		productRepo:    productRepo,
		productSKURepo: productSKURepo,
		categoryRepo:   categoryRepo,
		connService:    connService,
	}
}

// SetCategoryService 设置分类服务（避免循环依赖）
func (s *ProductMappingService) SetCategoryService(cs *CategoryService) {
	s.categoryService = cs
}

// SetMediaService 设置素材服务（避免循环依赖）
func (s *ProductMappingService) SetMediaService(ms *MediaService) {
	s.mediaService = ms
}

// ImportUpstreamProduct 从上游导入商品（克隆为本地商品 + 建立映射）
func (s *ProductMappingService) ImportUpstreamProduct(connectionID uint, upstreamProductID uint, categoryID uint, slug string) (*models.ProductMapping, error) {
	if err := validateProductCategoryAssignment(s.categoryRepo, categoryID, 0); err != nil {
		return nil, err
	}

	// 检查是否已存在映射
	existing, err := s.mappingRepo.GetByConnectionAndUpstreamID(connectionID, upstreamProductID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrMappingAlreadyExists
	}

	// 获取连接
	conn, err := s.connService.GetByID(connectionID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, ErrConnectionNotFound
	}

	// 获取适配器
	adapter, err := s.connService.GetAdapter(conn)
	if err != nil {
		return nil, err
	}

	// 拉取上游商品
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	upProduct, err := adapter.GetProduct(ctx, upstreamProductID)
	if err != nil {
		return nil, fmt.Errorf("fetch upstream product: %w", err)
	}
	if upProduct == nil {
		return nil, ErrUpstreamProductNotFound
	}

	// 下载图片到本地
	localImages := s.downloadImages(ctx, adapter, upProduct.Images)

	// 下载 Content 中引用的图片
	localContent := s.downloadContentImages(ctx, adapter, upProduct.Content)

	// 确定交付类型：上游商品映射后统一使用 upstream 类型
	fulfillmentType := constants.FulfillmentTypeUpstream

	// 解析价格（先汇率转换，再应用加价比例）
	exchangeRate := conn.ExchangeRate
	markupPercent := conn.PriceMarkupPercent
	roundingMode := conn.PriceRoundingMode

	priceAmount, priceErr := decimal.NewFromString(upProduct.PriceAmount)
	if priceErr != nil {
		logger.Warnw("import_product_price_parse_error",
			"upstream_product_id", upstreamProductID,
			"price_amount", upProduct.PriceAmount,
			"error", priceErr,
		)
		priceAmount = decimal.Zero
	}
	costPriceAmount := convertCurrency(priceAmount, exchangeRate) // 成本价 = 上游价格 × 汇率（本地币种，不含加价）
	priceAmount = CalculateLocalPrice(priceAmount, exchangeRate, markupPercent, roundingMode)
	if priceAmount.LessThanOrEqual(decimal.Zero) && len(upProduct.SKUs) > 0 {
		// 取转换加价后 SKU 最低价
		for _, sku := range upProduct.SKUs {
			skuPrice, _ := decimal.NewFromString(sku.PriceAmount)
			localPrice := CalculateLocalPrice(skuPrice, exchangeRate, markupPercent, roundingMode)
			if localPrice.GreaterThan(decimal.Zero) && (priceAmount.IsZero() || localPrice.LessThan(priceAmount)) {
				priceAmount = localPrice
				costPriceAmount = convertCurrency(skuPrice, exchangeRate)
			}
		}
	}

	// 自动生成 slug（如果未提供）
	if slug == "" {
		slug = fmt.Sprintf("upstream-%d-%d-%d", connectionID, upstreamProductID, time.Now().UnixMilli())
	}

	// 创建本地商品
	product := models.Product{
		CategoryID:           categoryID,
		Slug:                 slug,
		SeoMetaJSON:          upProduct.SeoMeta,
		TitleJSON:            upProduct.Title,
		DescriptionJSON:      upProduct.Description,
		ContentJSON:          localContent,
		ManualFormSchemaJSON: upProduct.ManualFormSchema,
		PriceAmount:          models.NewMoneyFromDecimal(priceAmount.Round(2)),
		CostPriceAmount:      models.NewMoneyFromDecimal(costPriceAmount.Round(2)),
		Images:               models.StringArray(localImages),
		Tags:                 models.StringArray(upProduct.Tags),
		PurchaseType:         constants.ProductPurchaseMember,
		FulfillmentType:      fulfillmentType,
		ManualStockTotal:     0,
		IsMapped:             true,
		IsActive:             false, // 默认下架，管理员手动上架
		SortOrder:            0,
	}

	var mapping *models.ProductMapping

	// 使用事务一次性创建本地商品、SKU、映射与 SKU 映射，避免留下半成功数据。
	if err := s.productRepo.Transaction(func(tx *gorm.DB) error {
		productRepo := s.productRepo.WithTx(tx)
		mappingRepo := s.mappingRepo.WithTx(tx)
		skuMappingRepo := s.skuMappingRepo.WithTx(tx)
		if err := productRepo.Create(&product); err != nil {
			return fmt.Errorf("create local product: %w", err)
		}

		// 创建 SKU
		skuRepo := s.productSKURepo.WithTx(tx)
		localSKUs := make([]models.ProductSKU, 0, len(upProduct.SKUs))
		for _, upSKU := range upProduct.SKUs {
			skuPrice, skuPriceErr := decimal.NewFromString(upSKU.PriceAmount)
			if skuPriceErr != nil {
				logger.Warnw("import_sku_price_parse_error",
					"upstream_sku_id", upSKU.ID,
					"price_amount", upSKU.PriceAmount,
					"error", skuPriceErr,
				)
				skuPrice = decimal.Zero
			}
			localPrice := CalculateLocalPrice(skuPrice, exchangeRate, markupPercent, roundingMode)
			localSKU := models.ProductSKU{
				ProductID:       product.ID,
				SKUCode:         upSKU.SKUCode,
				SpecValuesJSON:  upSKU.SpecValues,
				PriceAmount:     models.NewMoneyFromDecimal(localPrice.Round(2)),
				CostPriceAmount: models.NewMoneyFromDecimal(convertCurrency(skuPrice, exchangeRate).Round(2)), // 成本价 = 上游价格 × 汇率（本地币种）
				IsActive:        upSKU.IsActive,
				SortOrder:       0,
			}
			if err := skuRepo.Create(&localSKU); err != nil {
				return fmt.Errorf("create local sku: %w", err)
			}
			localSKUs = append(localSKUs, localSKU)
		}

		// 如果没有 SKU，创建默认 SKU
		if len(upProduct.SKUs) == 0 {
			defaultSKU := models.ProductSKU{
				ProductID:      product.ID,
				SKUCode:        models.DefaultSKUCode,
				SpecValuesJSON: models.JSON{},
				PriceAmount:    models.NewMoneyFromDecimal(priceAmount.Round(2)),
				IsActive:       true,
				SortOrder:      0,
			}
			if err := skuRepo.Create(&defaultSKU); err != nil {
				return fmt.Errorf("create default sku: %w", err)
			}
			localSKUs = append(localSKUs, defaultSKU)
		}

		// 确定上游原始交付类型（auto/manual）
		upstreamFulfillmentType := upProduct.FulfillmentType
		if upstreamFulfillmentType != constants.FulfillmentTypeAuto {
			upstreamFulfillmentType = constants.FulfillmentTypeManual
		}

		now := time.Now()
		mapping = &models.ProductMapping{
			ConnectionID:            connectionID,
			LocalProductID:          product.ID,
			UpstreamProductID:       upstreamProductID,
			UpstreamFulfillmentType: upstreamFulfillmentType,
			IsActive:                true,
			LastSyncedAt:            &now,
		}
		if err := mappingRepo.Create(mapping); err != nil {
			return fmt.Errorf("create product mapping: %w", err)
		}
		if err := createSKUMappingsWithRepo(skuMappingRepo, mapping.ID, localSKUs, upProduct.SKUs); err != nil {
			return fmt.Errorf("create sku mappings: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return mapping, nil
}

func createSKUMappingsWithRepo(
	skuMappingRepo repository.SKUMappingRepository,
	mappingID uint,
	localSKUs []models.ProductSKU,
	upstreamSKUs []upstream.UpstreamSKU,
) error {
	if skuMappingRepo == nil {
		return nil
	}
	if len(localSKUs) == 0 || len(upstreamSKUs) == 0 {
		return nil
	}

	// 按 SKUCode 匹配
	upstreamByCode := make(map[string]upstream.UpstreamSKU, len(upstreamSKUs))
	for _, us := range upstreamSKUs {
		upstreamByCode[strings.ToLower(strings.TrimSpace(us.SKUCode))] = us
	}

	for _, localSKU := range localSKUs {
		code := strings.ToLower(strings.TrimSpace(localSKU.SKUCode))
		upSKU, ok := upstreamByCode[code]
		if !ok {
			// 如果只有一个 SKU（DEFAULT），匹配第一个上游 SKU
			if len(localSKUs) == 1 && len(upstreamSKUs) == 1 {
				upSKU = upstreamSKUs[0]
			} else {
				continue
			}
		}

		upPrice, _ := decimal.NewFromString(upSKU.PriceAmount)
		now := time.Now()
		skuMapping := &models.SKUMapping{
			ProductMappingID: mappingID,
			LocalSKUID:       localSKU.ID,
			UpstreamSKUID:    upSKU.ID,
			UpstreamPrice:    models.NewMoneyFromDecimal(upPrice.Round(2)),
			UpstreamIsActive: upSKU.IsActive,
			UpstreamStock:    upSKU.StockQuantity,
			StockSyncedAt:    &now,
		}
		if err := skuMappingRepo.Create(skuMapping); err != nil {
			return err
		}
	}

	return nil
}

// downloadImages 下载上游图片到本地
func (s *ProductMappingService) downloadImages(ctx context.Context, adapter upstream.Adapter, images []string) []string {
	var localImages []string
	for _, img := range images {
		if strings.TrimSpace(img) == "" {
			continue
		}
		localPath, err := adapter.DownloadImage(ctx, img)
		if err != nil {
			// 下载失败保留原始 URL
			localImages = append(localImages, img)
			continue
		}
		if s.mediaService != nil {
			s.mediaService.RecordLocalFile(localPath, "upstream")
		}
		localImages = append(localImages, localPath)
	}
	return localImages
}

// downloadContentImages 下载多语言 Content 中的图片并替换 URL
func (s *ProductMappingService) downloadContentImages(ctx context.Context, adapter upstream.Adapter, content models.JSON) models.JSON {
	if len(content) == 0 {
		return content
	}

	// models.JSON 是 map[string]interface{}，值为各语言的 Markdown 文本
	imgRegex := regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)|<img[^>]+src=["']([^"']+)["']`)
	downloaded := make(map[string]string) // originalURL -> localPath

	// 第一遍：收集所有唯一图片 URL
	for _, val := range content {
		text, ok := val.(string)
		if !ok || text == "" {
			continue
		}
		matches := imgRegex.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			url := m[1]
			if url == "" {
				url = m[2]
			}
			if url == "" || strings.HasPrefix(url, "/uploads/") {
				continue
			}
			downloaded[url] = "" // 占位
		}
	}

	if len(downloaded) == 0 {
		return content
	}

	// 下载图片
	for url := range downloaded {
		localPath, err := adapter.DownloadImage(ctx, url)
		if err != nil {
			downloaded[url] = url // 失败保留原始
		} else {
			if s.mediaService != nil {
				s.mediaService.RecordLocalFile(localPath, "upstream")
			}
			downloaded[url] = localPath
		}
	}

	// 第二遍：替换所有语言文本中的 URL
	result := make(models.JSON, len(content))
	for lang, val := range content {
		text, ok := val.(string)
		if !ok {
			result[lang] = val
			continue
		}
		for original, local := range downloaded {
			if original != local {
				text = strings.ReplaceAll(text, original, local)
			}
		}
		result[lang] = text
	}

	return result
}

// SyncProduct 同步单个映射商品的上游数据（全量同步）
func (s *ProductMappingService) SyncProduct(mappingID uint) error {
	mapping, err := s.mappingRepo.GetByID(mappingID)
	if err != nil {
		return err
	}
	if mapping == nil {
		return ErrMappingNotFound
	}

	conn, err := s.connService.GetByID(mapping.ConnectionID)
	if err != nil {
		return err
	}
	if conn == nil {
		return ErrConnectionNotFound
	}

	adapter, err := s.connService.GetAdapter(conn)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	upProduct, err := adapter.GetProduct(ctx, mapping.UpstreamProductID)
	if err != nil {
		return fmt.Errorf("fetch upstream product: %w", err)
	}

	now := time.Now()

	// ── 1. 同步本地商品字段（表单配置、上下架状态） ──
	localProduct, err := s.productRepo.GetByID(strconv.FormatUint(uint64(mapping.LocalProductID), 10))
	if err != nil {
		return fmt.Errorf("get local product: %w", err)
	}
	if localProduct != nil {
		changed := false

		// 同步人工交付表单配置
		if upProduct.ManualFormSchema != nil {
			localProduct.ManualFormSchemaJSON = upProduct.ManualFormSchema
			changed = true
		}

		// 如果上游商品已下架，本地也自动下架（但上游上架不自动上架，留给管理员决定）
		if !upProduct.IsActive && localProduct.IsActive {
			localProduct.IsActive = false
			changed = true
		}

		if changed {
			_ = s.productRepo.Update(localProduct)
		}
	}

	// ── 2. 同步 SKU：新增 / 更新 / 停用 ──
	skuMappings, err := s.skuMappingRepo.ListByProductMapping(mappingID)
	if err != nil {
		return err
	}

	// 构建上游 SKU 查找表
	upstreamSKUMap := make(map[uint]upstream.UpstreamSKU, len(upProduct.SKUs))
	for _, us := range upProduct.SKUs {
		upstreamSKUMap[us.ID] = us
	}

	// 构建已有映射查找表（按上游 SKU ID）
	existingByUpstreamID := make(map[uint]*models.SKUMapping, len(skuMappings))
	for i := range skuMappings {
		existingByUpstreamID[skuMappings[i].UpstreamSKUID] = &skuMappings[i]
	}

	// 2a. 更新已有映射 + 同步本地 SKU
	for i := range skuMappings {
		upSKU, ok := upstreamSKUMap[skuMappings[i].UpstreamSKUID]
		if !ok {
			// 上游 SKU 已删除 → 停用本地 SKU 和映射
			skuMappings[i].UpstreamIsActive = false
			skuMappings[i].UpstreamStock = 0
			skuMappings[i].StockSyncedAt = &now
			_ = s.skuMappingRepo.Update(&skuMappings[i])

			// 停用本地 SKU
			localSKU, _ := s.productSKURepo.GetByID(skuMappings[i].LocalSKUID)
			if localSKU != nil && localSKU.IsActive {
				localSKU.IsActive = false
				_ = s.productSKURepo.Update(localSKU)
			}
			continue
		}

		upPrice, _ := decimal.NewFromString(upSKU.PriceAmount)

		// 更新 SKU 映射记录
		skuMappings[i].UpstreamPrice = models.NewMoneyFromDecimal(upPrice.Round(2))
		skuMappings[i].UpstreamIsActive = upSKU.IsActive
		skuMappings[i].StockSyncedAt = &now
		skuMappings[i].UpstreamStock = upSKU.StockQuantity
		_ = s.skuMappingRepo.Update(&skuMappings[i])

		// 同步本地 SKU 字段
		localSKU, _ := s.productSKURepo.GetByID(skuMappings[i].LocalSKUID)
		if localSKU != nil {
			localSKU.SpecValuesJSON = upSKU.SpecValues
			localSKU.IsActive = upSKU.IsActive
			// 如果启用了自动同步价格，按加价比例更新本地售价和成本价
			if conn.AutoSyncPrice {
				newLocalPrice := CalculateLocalPrice(upPrice, conn.ExchangeRate, conn.PriceMarkupPercent, conn.PriceRoundingMode)
				localSKU.PriceAmount = models.NewMoneyFromDecimal(newLocalPrice.Round(2))
				localSKU.CostPriceAmount = models.NewMoneyFromDecimal(convertCurrency(upPrice, conn.ExchangeRate).Round(2))
			}
			_ = s.productSKURepo.Update(localSKU)
		}
	}

	// 2b. 上游新增的 SKU → 创建本地 SKU + 映射
	for _, upSKU := range upProduct.SKUs {
		if _, exists := existingByUpstreamID[upSKU.ID]; exists {
			continue
		}

		skuPrice, _ := decimal.NewFromString(upSKU.PriceAmount)
		localPrice := CalculateLocalPrice(skuPrice, conn.ExchangeRate, conn.PriceMarkupPercent, conn.PriceRoundingMode)
		newLocalSKU := models.ProductSKU{
			ProductID:       mapping.LocalProductID,
			SKUCode:         upSKU.SKUCode,
			SpecValuesJSON:  upSKU.SpecValues,
			PriceAmount:     models.NewMoneyFromDecimal(localPrice.Round(2)),
			CostPriceAmount: models.NewMoneyFromDecimal(convertCurrency(skuPrice, conn.ExchangeRate).Round(2)), // 成本价 = 上游价格 × 汇率（本地币种）
			IsActive:        upSKU.IsActive,
			SortOrder:       0,
		}
		if err := s.productSKURepo.Create(&newLocalSKU); err != nil {
			continue
		}

		newMapping := &models.SKUMapping{
			ProductMappingID: mappingID,
			LocalSKUID:       newLocalSKU.ID,
			UpstreamSKUID:    upSKU.ID,
			UpstreamPrice:    models.NewMoneyFromDecimal(skuPrice.Round(2)),
			UpstreamIsActive: upSKU.IsActive,
			UpstreamStock:    upSKU.StockQuantity,
			StockSyncedAt:    &now,
		}
		_ = s.skuMappingRepo.Create(newMapping)
	}

	// ── 2c. 如果启用了自动同步价格，更新 Product.PriceAmount 为最低 SKU 价格 ──
	if conn.AutoSyncPrice && localProduct != nil {
		s.recalcProductPrice(localProduct)
	}

	// ── 3. 更新同步时间 + 上游交付类型 ──
	upFulfillment := upProduct.FulfillmentType
	if upFulfillment != constants.FulfillmentTypeAuto {
		upFulfillment = constants.FulfillmentTypeManual
	}
	mapping.UpstreamFulfillmentType = upFulfillment
	mapping.LastSyncedAt = &now
	return s.mappingRepo.Update(mapping)
}

// SyncAllStock 同步所有活跃映射的库存（供定时任务调用）
// 使用 Redis 锁防止任务重叠执行，并发调用上游 API 提升吞吐量
func (s *ProductMappingService) SyncAllStock() error {
	ctx := context.Background()
	const lockKey = "upstream:sync_stock_running"

	locked, err := cache.SetNX(ctx, lockKey, "1", 30*time.Minute)
	if err != nil {
		logger.Warnw("sync_stock_lock_error", "error", err)
		// Redis 不可用时降级为直接执行
	} else if !locked {
		logger.Debugw("sync_stock_skip_already_running")
		return nil
	}
	defer cache.Del(ctx, lockKey)

	mappings, err := s.mappingRepo.ListAllActive()
	if err != nil {
		return err
	}
	if len(mappings) == 0 {
		return nil
	}

	// ── 按连接分组 ──
	byConn := make(map[uint][]models.ProductMapping)
	for _, m := range mappings {
		byConn[m.ConnectionID] = append(byConn[m.ConnectionID], m)
	}

	var mu sync.Mutex
	var errs []error
	var wg sync.WaitGroup

	// 每个连接并发处理
	const connConcurrency = 3
	sem := make(chan struct{}, connConcurrency)

	for connID, connMappings := range byConn {
		wg.Add(1)
		sem <- struct{}{}
		go func(connID uint, connMappings []models.ProductMapping) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.syncConnectionStock(connID, connMappings); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				logger.Warnw("sync_connection_stock_failed", "connection_id", connID, "error", err)
			}
		}(connID, connMappings)
	}
	wg.Wait()
	return errors.Join(errs...)
}

// syncConnectionStock 按连接批量同步：一次 ListProducts 拉取所有商品，内存匹配映射
func (s *ProductMappingService) syncConnectionStock(connectionID uint, connMappings []models.ProductMapping) error {
	conn, err := s.connService.GetByID(connectionID)
	if err != nil || conn == nil {
		return fmt.Errorf("get connection %d: %w", connectionID, err)
	}

	adapter, err := s.connService.GetAdapter(conn)
	if err != nil {
		return fmt.Errorf("get adapter for connection %d: %w", connectionID, err)
	}

	// 读取上次同步时间用于增量同步
	syncCtx := context.Background()
	lastSyncKey := fmt.Sprintf("upstream:last_sync:%d", connectionID)
	var updatedAfter *time.Time
	if lastSyncStr, err := cache.GetString(syncCtx, lastSyncKey); err == nil && lastSyncStr != "" {
		if t, err := time.Parse(time.RFC3339, lastSyncStr); err == nil {
			// 往前推 1 分钟作为安全窗口
			safeTime := t.Add(-1 * time.Minute)
			updatedAfter = &safeTime
		}
	}
	syncStartTime := time.Now()

	// 批量拉取上游商品（分页）
	upstreamProducts := make(map[uint]upstream.UpstreamProduct)
	page := 1
	const pageSize = 50
	for {
		ctx, cancel := context.WithTimeout(syncCtx, 30*time.Second)
		result, err := adapter.ListProducts(ctx, upstream.ListProductsOpts{
			Page:         page,
			PageSize:     pageSize,
			UpdatedAfter: updatedAfter,
		})
		cancel()
		if err != nil {
			// 增量拉取失败时回退到全量
			if updatedAfter != nil {
				logger.Warnw("sync_incremental_failed_fallback_full", "connection_id", connectionID, "error", err)
				updatedAfter = nil
				page = 1
				upstreamProducts = make(map[uint]upstream.UpstreamProduct)
				continue
			}
			return fmt.Errorf("list upstream products page %d: %w", page, err)
		}

		for _, p := range result.Items {
			upstreamProducts[p.ID] = p
		}

		if len(upstreamProducts) >= result.Total || len(result.Items) == 0 {
			break
		}
		page++
		if page > 200 { // 安全限制
			break
		}
	}

	// 如果是增量同步且无更新，跳过
	if updatedAfter != nil && len(upstreamProducts) == 0 {
		logger.Debugw("sync_skip_no_updates", "connection_id", connectionID)
		// 仍然更新时间戳
		_ = cache.SetString(syncCtx, lastSyncKey, syncStartTime.Format(time.RFC3339), 48*time.Hour)
		return nil
	}

	// 对每个映射执行同步
	now := time.Now()
	for i := range connMappings {
		mapping := &connMappings[i]
		upProduct, ok := upstreamProducts[mapping.UpstreamProductID]
		if !ok {
			if updatedAfter != nil {
				// 增量模式下未返回说明没有变化，跳过
				continue
			}
			// 全量模式下未找到说明上游已删除/下架
			logger.Warnw("sync_upstream_product_missing",
				"connection_id", connectionID,
				"upstream_product_id", mapping.UpstreamProductID,
				"local_product_id", mapping.LocalProductID,
			)
			continue
		}
		s.syncProductFromData(mapping, conn, &upProduct, &now)
	}

	// 记录本次同步时间
	_ = cache.SetString(syncCtx, lastSyncKey, syncStartTime.Format(time.RFC3339), 48*time.Hour)

	logger.Infow("sync_connection_stock_done",
		"connection_id", connectionID,
		"mappings", len(connMappings),
		"upstream_fetched", len(upstreamProducts),
		"incremental", updatedAfter != nil,
	)
	return nil
}

// syncProductFromData 使用已拉取的上游数据同步单个映射（不再发 HTTP 请求）
func (s *ProductMappingService) syncProductFromData(mapping *models.ProductMapping, conn *models.SiteConnection, upProduct *upstream.UpstreamProduct, now *time.Time) {
	// ── 1. 同步本地商品字段 ──
	localProduct, err := s.productRepo.GetByID(strconv.FormatUint(uint64(mapping.LocalProductID), 10))
	if err != nil || localProduct == nil {
		return
	}

	changed := false
	if upProduct.ManualFormSchema != nil {
		localProduct.ManualFormSchemaJSON = upProduct.ManualFormSchema
		changed = true
	}
	if !upProduct.IsActive && localProduct.IsActive {
		localProduct.IsActive = false
		changed = true
	}
	if changed {
		_ = s.productRepo.Update(localProduct)
	}

	// ── 2. 同步 SKU ──
	skuMappings, err := s.skuMappingRepo.ListByProductMapping(mapping.ID)
	if err != nil {
		return
	}

	upstreamSKUMap := make(map[uint]upstream.UpstreamSKU, len(upProduct.SKUs))
	for _, us := range upProduct.SKUs {
		upstreamSKUMap[us.ID] = us
	}

	existingByUpstreamID := make(map[uint]*models.SKUMapping, len(skuMappings))
	for i := range skuMappings {
		existingByUpstreamID[skuMappings[i].UpstreamSKUID] = &skuMappings[i]
	}

	// 更新已有映射
	for i := range skuMappings {
		upSKU, ok := upstreamSKUMap[skuMappings[i].UpstreamSKUID]
		if !ok {
			skuMappings[i].UpstreamIsActive = false
			skuMappings[i].UpstreamStock = 0
			skuMappings[i].StockSyncedAt = now
			_ = s.skuMappingRepo.Update(&skuMappings[i])
			localSKU, _ := s.productSKURepo.GetByID(skuMappings[i].LocalSKUID)
			if localSKU != nil && localSKU.IsActive {
				localSKU.IsActive = false
				_ = s.productSKURepo.Update(localSKU)
			}
			continue
		}

		upPrice, priceErr := decimal.NewFromString(upSKU.PriceAmount)
		if priceErr != nil {
			logger.Warnw("sync_sku_price_parse_error",
				"upstream_sku_id", upSKU.ID,
				"price_amount", upSKU.PriceAmount,
				"error", priceErr,
			)
			// 仅同步库存状态，跳过价格更新
			skuMappings[i].UpstreamIsActive = upSKU.IsActive
			skuMappings[i].StockSyncedAt = now
			skuMappings[i].UpstreamStock = upSKU.StockQuantity
			_ = s.skuMappingRepo.Update(&skuMappings[i])
			continue
		}
		skuMappings[i].UpstreamPrice = models.NewMoneyFromDecimal(upPrice.Round(2))
		skuMappings[i].UpstreamIsActive = upSKU.IsActive
		skuMappings[i].StockSyncedAt = now
		skuMappings[i].UpstreamStock = upSKU.StockQuantity
		_ = s.skuMappingRepo.Update(&skuMappings[i])

		localSKU, _ := s.productSKURepo.GetByID(skuMappings[i].LocalSKUID)
		if localSKU != nil {
			localSKU.SpecValuesJSON = upSKU.SpecValues
			localSKU.IsActive = upSKU.IsActive
			if conn.AutoSyncPrice {
				newLocalPrice := CalculateLocalPrice(upPrice, conn.ExchangeRate, conn.PriceMarkupPercent, conn.PriceRoundingMode)
				localSKU.PriceAmount = models.NewMoneyFromDecimal(newLocalPrice.Round(2))
				localSKU.CostPriceAmount = models.NewMoneyFromDecimal(convertCurrency(upPrice, conn.ExchangeRate).Round(2))
			}
			_ = s.productSKURepo.Update(localSKU)
		}
	}

	// 上游新增 SKU
	for _, upSKU := range upProduct.SKUs {
		if _, exists := existingByUpstreamID[upSKU.ID]; exists {
			continue
		}
		skuPrice, priceErr := decimal.NewFromString(upSKU.PriceAmount)
		if priceErr != nil {
			logger.Warnw("sync_new_sku_price_parse_error",
				"upstream_sku_id", upSKU.ID,
				"price_amount", upSKU.PriceAmount,
				"error", priceErr,
			)
			continue
		}
		localPrice := CalculateLocalPrice(skuPrice, conn.ExchangeRate, conn.PriceMarkupPercent, conn.PriceRoundingMode)
		newLocalSKU := models.ProductSKU{
			ProductID:       mapping.LocalProductID,
			SKUCode:         upSKU.SKUCode,
			SpecValuesJSON:  upSKU.SpecValues,
			PriceAmount:     models.NewMoneyFromDecimal(localPrice.Round(2)),
			CostPriceAmount: models.NewMoneyFromDecimal(convertCurrency(skuPrice, conn.ExchangeRate).Round(2)),
			IsActive:        upSKU.IsActive,
			SortOrder:       0,
		}
		if err := s.productSKURepo.Create(&newLocalSKU); err != nil {
			continue
		}
		newSKUMapping := &models.SKUMapping{
			ProductMappingID: mapping.ID,
			LocalSKUID:       newLocalSKU.ID,
			UpstreamSKUID:    upSKU.ID,
			UpstreamPrice:    models.NewMoneyFromDecimal(skuPrice.Round(2)),
			UpstreamIsActive: upSKU.IsActive,
			UpstreamStock:    upSKU.StockQuantity,
			StockSyncedAt:    now,
		}
		_ = s.skuMappingRepo.Create(newSKUMapping)
	}

	// 同步价格
	if conn.AutoSyncPrice && localProduct != nil {
		s.recalcProductPrice(localProduct)
	}

	// ── 3. 更新映射记录 ──
	upFulfillment := upProduct.FulfillmentType
	if upFulfillment != constants.FulfillmentTypeAuto {
		upFulfillment = constants.FulfillmentTypeManual
	}
	mapping.UpstreamFulfillmentType = upFulfillment
	mapping.LastSyncedAt = now
	_ = s.mappingRepo.Update(mapping)
}

// GetByID 获取映射详情
func (s *ProductMappingService) GetByID(id uint) (*models.ProductMapping, error) {
	return s.mappingRepo.GetByID(id)
}

// List 列表查询映射
func (s *ProductMappingService) List(filter repository.ProductMappingListFilter) ([]models.ProductMapping, int64, error) {
	return s.mappingRepo.List(filter)
}

// SetActive 启用/禁用映射
func (s *ProductMappingService) SetActive(id uint, active bool) error {
	mapping, err := s.mappingRepo.GetByID(id)
	if err != nil {
		return err
	}
	if mapping == nil {
		return ErrMappingNotFound
	}
	mapping.IsActive = active
	return s.mappingRepo.Update(mapping)
}

// Delete 删除映射（不删除本地商品）
func (s *ProductMappingService) Delete(id uint) error {
	mapping, err := s.mappingRepo.GetByID(id)
	if err != nil {
		return err
	}
	if mapping == nil {
		return ErrMappingNotFound
	}

	// 删除 SKU 映射
	if err := s.skuMappingRepo.DeleteByProductMapping(id); err != nil {
		return err
	}

	// 还原本地商品状态：取消映射标记、交付类型改回 manual、自动下架
	if mapping.LocalProductID > 0 {
		localProduct, err := s.productRepo.GetByID(strconv.FormatUint(uint64(mapping.LocalProductID), 10))
		if err == nil && localProduct != nil {
			localProduct.IsMapped = false
			if localProduct.FulfillmentType == constants.FulfillmentTypeUpstream {
				localProduct.FulfillmentType = constants.FulfillmentTypeManual
				localProduct.IsActive = false // 下架，防止用户下单后无法交付
			}
			_ = s.productRepo.Update(localProduct)
		}
	}

	return s.mappingRepo.Delete(id)
}

// GetSKUMappings 获取映射的 SKU 映射列表
func (s *ProductMappingService) GetSKUMappings(mappingID uint) ([]models.SKUMapping, error) {
	return s.skuMappingRepo.ListByProductMapping(mappingID)
}

// ReapplyMarkup 对指定连接的所有映射商品重新应用加价规则
func (s *ProductMappingService) ReapplyMarkup(connectionID uint) (int, error) {
	conn, err := s.connService.GetByID(connectionID)
	if err != nil {
		return 0, err
	}
	if conn == nil {
		return 0, ErrConnectionNotFound
	}

	mappings, err := s.mappingRepo.ListActiveByConnection(connectionID)
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, mapping := range mappings {
		skuMappings, err := s.skuMappingRepo.ListByProductMapping(mapping.ID)
		if err != nil {
			continue
		}

		for _, sm := range skuMappings {
			newLocalPrice := CalculateLocalPrice(sm.UpstreamPrice.Decimal, conn.ExchangeRate, conn.PriceMarkupPercent, conn.PriceRoundingMode)
			localSKU, err := s.productSKURepo.GetByID(sm.LocalSKUID)
			if err != nil || localSKU == nil {
				continue
			}
			localSKU.PriceAmount = models.NewMoneyFromDecimal(newLocalPrice.Round(2))
			localSKU.CostPriceAmount = models.NewMoneyFromDecimal(convertCurrency(sm.UpstreamPrice.Decimal, conn.ExchangeRate).Round(2)) // 成本价 = 上游价格 × 汇率（本地币种）
			_ = s.productSKURepo.Update(localSKU)
		}

		// 更新 Product.PriceAmount
		localProduct, err := s.productRepo.GetByID(strconv.FormatUint(uint64(mapping.LocalProductID), 10))
		if err == nil && localProduct != nil {
			s.recalcProductPrice(localProduct)
			updated++
		}
	}

	return updated, nil
}

// recalcProductPrice 重新计算商品基准价格和成本价为最低活跃 SKU 价格
func (s *ProductMappingService) recalcProductPrice(product *models.Product) {
	allSKUs, err := s.productSKURepo.ListByProduct(product.ID, true)
	if err != nil || len(allSKUs) == 0 {
		return
	}
	minPrice := allSKUs[0].PriceAmount.Decimal
	minCostPrice := allSKUs[0].CostPriceAmount.Decimal
	for _, sku := range allSKUs[1:] {
		if sku.PriceAmount.Decimal.LessThan(minPrice) {
			minPrice = sku.PriceAmount.Decimal
		}
		if sku.CostPriceAmount.Decimal.LessThan(minCostPrice) {
			minCostPrice = sku.CostPriceAmount.Decimal
		}
	}
	product.PriceAmount = models.NewMoneyFromDecimal(minPrice.Round(2))
	product.CostPriceAmount = models.NewMoneyFromDecimal(minCostPrice.Round(2))
	_ = s.productRepo.Update(product)
}

// GetMappedUpstreamIDs 获取指定连接下所有已映射的上游商品 ID
func (s *ProductMappingService) GetMappedUpstreamIDs(connectionID uint) ([]uint, error) {
	return s.mappingRepo.ListUpstreamIDsByConnection(connectionID)
}

// ListUpstreamProducts 通过连接代理拉取上游商品列表（分页）
func (s *ProductMappingService) ListUpstreamProducts(connectionID uint, page, pageSize int) (*upstream.ProductListResult, error) {
	conn, err := s.connService.GetByID(connectionID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, ErrConnectionNotFound
	}

	adapter, err := s.connService.GetAdapter(conn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return adapter.ListProducts(ctx, upstream.ListProductsOpts{
		Page:     page,
		PageSize: pageSize,
	})
}

// ListUpstreamCategories 通过连接代理拉取上游分类列表
// 返回 (categories, supported, error)，supported 为 false 表示上游不支持分类 API
func (s *ProductMappingService) ListUpstreamCategories(connectionID uint) ([]upstream.UpstreamCategory, bool, error) {
	conn, err := s.connService.GetByID(connectionID)
	if err != nil {
		return nil, false, err
	}
	if conn == nil {
		return nil, false, ErrConnectionNotFound
	}

	adapter, err := s.connService.GetAdapter(conn)
	if err != nil {
		return nil, false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := adapter.ListCategories(ctx)
	if err != nil {
		return nil, false, err
	}

	return result.Categories, result.Supported, nil
}

// BatchImportByCategoryResult 按分类批量导入结果
type BatchImportByCategoryResult struct {
	Total        int    `json:"total"`
	SuccessCount int    `json:"success_count"`
	CategoryID   uint   `json:"category_id"`
	CategoryName string `json:"category_name,omitempty"`
	Errors       []struct {
		UpstreamProductID uint   `json:"upstream_product_id"`
		Error             string `json:"error"`
	} `json:"errors,omitempty"`
}

// BatchImportByCategory 按上游分类批量导入商品
func (s *ProductMappingService) BatchImportByCategory(
	connectionID uint,
	upstreamCategoryID uint,
	autoCreateCategory bool,
	localCategoryID uint,
) (*BatchImportByCategoryResult, error) {
	conn, err := s.connService.GetByID(connectionID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, ErrConnectionNotFound
	}

	adapter, err := s.connService.GetAdapter(conn)
	if err != nil {
		return nil, err
	}

	// 分页拉取上游所有商品，筛选属于目标分类的
	var targetProducts []upstream.UpstreamProduct
	page := 1
	pageSize := 50
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		result, fetchErr := adapter.ListProducts(ctx, upstream.ListProductsOpts{
			Page:     page,
			PageSize: pageSize,
		})
		cancel()
		if fetchErr != nil {
			return nil, fmt.Errorf("fetch upstream products page %d: %w", page, fetchErr)
		}
		for _, p := range result.Items {
			if p.CategoryID == upstreamCategoryID {
				targetProducts = append(targetProducts, p)
			}
		}
		if len(result.Items) < pageSize || page*pageSize >= result.Total {
			break
		}
		page++
	}

	if len(targetProducts) == 0 {
		return &BatchImportByCategoryResult{Total: 0, SuccessCount: 0}, nil
	}

	// 确定本地分类 ID
	categoryID := localCategoryID
	categoryName := ""

	if autoCreateCategory && categoryID == 0 {
		// 拉取上游分类列表用于自动创建
		catCtx, catCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer catCancel()
		catResult, catErr := adapter.ListCategories(catCtx)
		if catErr != nil {
			return nil, fmt.Errorf("fetch upstream categories: %w", catErr)
		}
		catMap := make(map[uint]upstream.UpstreamCategory)
		for _, c := range catResult.Categories {
			catMap[c.ID] = c
		}
		cat, createErr := s.findOrCreateCategoryFromUpstream(upstreamCategoryID, catMap)
		if createErr != nil {
			return nil, fmt.Errorf("auto create category: %w", createErr)
		}
		categoryID = cat.ID
		if nameMap, ok := cat.NameJSON["zh-CN"]; ok {
			if n, ok := nameMap.(string); ok {
				categoryName = n
			}
		}
	}

	// 逐个导入
	result := &BatchImportByCategoryResult{
		Total:        len(targetProducts),
		CategoryID:   categoryID,
		CategoryName: categoryName,
	}
	for _, p := range targetProducts {
		_, importErr := s.ImportUpstreamProduct(connectionID, p.ID, categoryID, "")
		if importErr != nil {
			if errors.Is(importErr, ErrMappingAlreadyExists) {
				result.SuccessCount++ // 已映射的算成功
				continue
			}
			result.Errors = append(result.Errors, struct {
				UpstreamProductID uint   `json:"upstream_product_id"`
				Error             string `json:"error"`
			}{
				UpstreamProductID: p.ID,
				Error:             importErr.Error(),
			})
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

// findOrCreateCategoryFromUpstream 根据上游分类信息查找或创建本地分类
func (s *ProductMappingService) findOrCreateCategoryFromUpstream(
	upstreamCategoryID uint, catMap map[uint]upstream.UpstreamCategory,
) (*models.Category, error) {
	target, ok := catMap[upstreamCategoryID]
	if !ok {
		return nil, fmt.Errorf("upstream category %d not found", upstreamCategoryID)
	}

	// 如果上游分类有父分类，先确保父分类存在
	var localParentID uint
	if target.ParentID > 0 {
		if parent, parentOK := catMap[target.ParentID]; parentOK {
			parentCat, parentErr := s.findOrCreateLocalCategory(parent.Slug, parent.Name, 0)
			if parentErr != nil {
				return nil, fmt.Errorf("create parent category: %w", parentErr)
			}
			localParentID = parentCat.ID
		}
	}

	return s.findOrCreateLocalCategory(target.Slug, target.Name, localParentID)
}

// findOrCreateLocalCategory 按 slug 查找或创建本地分类
func (s *ProductMappingService) findOrCreateLocalCategory(slug string, nameJSON models.JSON, parentID uint) (*models.Category, error) {
	// 先查找是否已存在同 slug 分类
	existing, err := s.categoryRepo.GetBySlug(slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// 不存在，创建新分类
	if s.categoryService == nil {
		return nil, fmt.Errorf("category service not available")
	}

	cat, err := s.categoryService.Create(CreateCategoryInput{
		ParentID: parentID,
		Slug:     slug,
		NameJSON: map[string]interface{}(nameJSON),
	})
	if err != nil {
		// slug 冲突，追加后缀重试
		if errors.Is(err, ErrSlugExists) {
			for i := 2; i <= 10; i++ {
				suffixedSlug := fmt.Sprintf("%s-%d", slug, i)
				cat, err = s.categoryService.Create(CreateCategoryInput{
					ParentID: parentID,
					Slug:     suffixedSlug,
					NameJSON: map[string]interface{}(nameJSON),
				})
				if err == nil {
					return cat, nil
				}
				if !errors.Is(err, ErrSlugExists) {
					return nil, err
				}
			}
			return nil, fmt.Errorf("slug conflict after retries: %s", slug)
		}
		return nil, err
	}
	return cat, nil
}
