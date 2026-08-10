package service

import (
	"errors"
	"strings"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/queue"
	"github.com/dujiao-next/internal/repository"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrderService ????
type OrderService struct {
	orderRepo             repository.OrderRepository
	orderRefundRecordRepo repository.OrderRefundRecordRepository
	userRepo              repository.UserRepository
	productRepo           repository.ProductRepository
	productSKURepo        repository.ProductSKURepository
	cardSecretRepo        repository.CardSecretRepository
	paymentRepo           repository.PaymentRepository
	couponRepo            repository.CouponRepository
	couponUsageRepo       repository.CouponUsageRepository
	promotionRepo         repository.PromotionRepository
	queueClient           *queue.Client
	settingService        *SettingService
	defaultEmailConfig    config.EmailConfig
	walletService         *WalletService
	affiliateSvc          *AffiliateService
	memberLevelService    *MemberLevelService
	riskControlSvc        *OrderRiskControlService
	expireMinutes         int
}

// OrderServiceOptions ????????
type OrderServiceOptions struct {
	OrderRepo             repository.OrderRepository
	OrderRefundRecordRepo repository.OrderRefundRecordRepository
	UserRepo              repository.UserRepository
	ProductRepo           repository.ProductRepository
	ProductSKURepo        repository.ProductSKURepository
	CardSecretRepo        repository.CardSecretRepository
	PaymentRepo           repository.PaymentRepository
	CouponRepo            repository.CouponRepository
	CouponUsageRepo       repository.CouponUsageRepository
	PromotionRepo         repository.PromotionRepository
	QueueClient           *queue.Client
	SettingService        *SettingService
	DefaultEmailConfig    config.EmailConfig
	WalletService         *WalletService
	AffiliateService      *AffiliateService
	MemberLevelService    *MemberLevelService
	RiskControlService    *OrderRiskControlService
	ExpireMinutes         int
}

// NewOrderService ??????
func NewOrderService(opts OrderServiceOptions) *OrderService {
	return &OrderService{
		orderRepo:             opts.OrderRepo,
		orderRefundRecordRepo: opts.OrderRefundRecordRepo,
		userRepo:              opts.UserRepo,
		productRepo:           opts.ProductRepo,
		productSKURepo:        opts.ProductSKURepo,
		cardSecretRepo:        opts.CardSecretRepo,
		paymentRepo:           opts.PaymentRepo,
		couponRepo:            opts.CouponRepo,
		couponUsageRepo:       opts.CouponUsageRepo,
		promotionRepo:         opts.PromotionRepo,
		queueClient:           opts.QueueClient,
		settingService:        opts.SettingService,
		defaultEmailConfig:    opts.DefaultEmailConfig,
		walletService:         opts.WalletService,
		affiliateSvc:          opts.AffiliateService,
		memberLevelService:    opts.MemberLevelService,
		riskControlSvc:        opts.RiskControlService,
		expireMinutes:         opts.ExpireMinutes,
	}
}

// CreateOrderInput ??????
type CreateOrderInput struct {
	UserID              uint
	Items               []CreateOrderItem
	CouponCode          string
	AffiliateCode       string
	AffiliateVisitorKey string
	ClientIP            string
	ManualFormData      map[string]models.JSON
	SkipRiskControl     bool // ??????(????)
	SkipIPRiskControl   bool // ?? IP ????(??/Bot ??)
}

// CreateGuestOrderInput ????????
type CreateGuestOrderInput struct {
	Email               string
	OrderPassword       string
	Locale              string
	Items               []CreateOrderItem
	CouponCode          string
	AffiliateCode       string
	AffiliateVisitorKey string
	ClientIP            string
	ManualFormData      map[string]models.JSON
}

// CreateOrderItem ???????
type CreateOrderItem struct {
	ProductID       uint
	SKUID           uint
	Quantity        int
	FulfillmentType string
}

// childOrderPlan ???????
type childOrderPlan struct {
	Product           *models.Product
	SKU               *models.ProductSKU
	Item              models.OrderItem
	TotalAmount       decimal.Decimal
	MemberDiscount    decimal.Decimal
	PromotionDiscount decimal.Decimal
	CouponDiscount    decimal.Decimal
	Currency          string
}

var allowedTransitions = map[string]map[string]bool{
	constants.OrderStatusPendingPayment: {
		constants.OrderStatusPaid:     true,
		constants.OrderStatusCanceled: true,
	},
	constants.OrderStatusPaid: {
		constants.OrderStatusFulfilling:         true,
		constants.OrderStatusPartiallyDelivered: true,
		constants.OrderStatusDelivered:          true,
		constants.OrderStatusPartiallyRefunded:  true,
		constants.OrderStatusRefunded:           true,
	},
	constants.OrderStatusFulfilling: {
		constants.OrderStatusPartiallyDelivered: true,
		constants.OrderStatusDelivered:          true,
		constants.OrderStatusPartiallyRefunded:  true,
		constants.OrderStatusRefunded:           true,
	},
	constants.OrderStatusPartiallyDelivered: {
		constants.OrderStatusDelivered:         true,
		constants.OrderStatusCompleted:         true,
		constants.OrderStatusPartiallyRefunded: true,
		constants.OrderStatusRefunded:          true,
	},
	constants.OrderStatusDelivered: {
		constants.OrderStatusCompleted:         true,
		constants.OrderStatusPartiallyRefunded: true,
		constants.OrderStatusRefunded:          true,
	},
	constants.OrderStatusCompleted: {
		constants.OrderStatusPartiallyRefunded: true,
		constants.OrderStatusRefunded:          true,
	},
	constants.OrderStatusPartiallyRefunded: {
		constants.OrderStatusRefunded: true,
	},
}

// CreateOrder ????
func (s *OrderService) CreateOrder(input CreateOrderInput) (*models.Order, error) {
	if input.UserID == 0 {
		return nil, ErrInvalidOrderItem
	}
	return s.createOrder(orderCreateParams{
		UserID:              input.UserID,
		Items:               input.Items,
		CouponCode:          input.CouponCode,
		AffiliateCode:       input.AffiliateCode,
		AffiliateVisitorKey: input.AffiliateVisitorKey,
		ClientIP:            input.ClientIP,
		ManualFormData:      input.ManualFormData,
		SkipRiskControl:     input.SkipRiskControl,
		SkipIPRiskControl:   input.SkipIPRiskControl,
	})
}

// CreateGuestOrder ??????
func (s *OrderService) CreateGuestOrder(input CreateGuestOrderInput) (*models.Order, error) {
	email, err := normalizeGuestEmail(input.Email)
	if err != nil {
		return nil, err
	}
	password := strings.TrimSpace(input.OrderPassword)
	if password == "" {
		return nil, ErrGuestPasswordRequired
	}
	locale := strings.TrimSpace(input.Locale)
	return s.createOrder(orderCreateParams{
		UserID:              0,
		GuestEmail:          email,
		GuestPassword:       password,
		GuestLocale:         locale,
		Items:               input.Items,
		CouponCode:          input.CouponCode,
		AffiliateCode:       input.AffiliateCode,
		AffiliateVisitorKey: input.AffiliateVisitorKey,
		ClientIP:            input.ClientIP,
		IsGuest:             true,
		ManualFormData:      input.ManualFormData,
	})
}

type orderCreateParams struct {
	UserID              uint
	GuestEmail          string
	GuestPassword       string
	GuestLocale         string
	Items               []CreateOrderItem
	CouponCode          string
	AffiliateCode       string
	AffiliateVisitorKey string
	ClientIP            string
	IsGuest             bool
	ManualFormData      map[string]models.JSON
	SkipRiskControl     bool
	SkipIPRiskControl   bool
}

// OrderPreview ??????
type OrderPreview struct {
	Currency                string             `json:"currency"`
	OriginalAmount          models.Money       `json:"original_amount"`
	MemberDiscountAmount    models.Money       `json:"member_discount_amount"`
	DiscountAmount          models.Money       `json:"discount_amount"`
	PromotionDiscountAmount models.Money       `json:"promotion_discount_amount"`
	TotalAmount             models.Money       `json:"total_amount"`
	Items                   []OrderPreviewItem `json:"items"`
}

// OrderPreviewItem ???????
type OrderPreviewItem struct {
	ProductID         uint               `json:"product_id"`
	SKUID             uint               `json:"sku_id"`
	TitleJSON         models.JSON        `json:"title"`
	SKUSnapshotJSON   models.JSON        `json:"sku_snapshot"`
	Tags              models.StringArray `json:"tags"`
	UnitPrice         models.Money       `json:"unit_price"`
	Quantity          int                `json:"quantity"`
	TotalPrice        models.Money       `json:"total_price"`
	MemberDiscount    models.Money       `json:"member_discount_amount"`
	CouponDiscount    models.Money       `json:"coupon_discount_amount"`
	PromotionDiscount models.Money       `json:"promotion_discount_amount"`
	FulfillmentType   string             `json:"fulfillment_type"`
}

type orderBuildResult struct {
	Plans                   []childOrderPlan
	OrderItems              []models.OrderItem
	OriginalAmount          decimal.Decimal
	MemberDiscountAmount    decimal.Decimal
	PromotionDiscountAmount decimal.Decimal
	DiscountAmount          decimal.Decimal
	TotalAmount             decimal.Decimal
	Currency                string
	OrderPromotionID        *uint
	MemberLevelID           *uint
	AppliedCoupon           *models.Coupon
}

// PreviewOrder ????????
func (s *OrderService) PreviewOrder(input CreateOrderInput) (*OrderPreview, error) {
	if input.UserID == 0 {
		return nil, ErrInvalidOrderItem
	}
	return s.previewOrder(orderCreateParams{
		UserID:              input.UserID,
		Items:               input.Items,
		CouponCode:          input.CouponCode,
		AffiliateCode:       input.AffiliateCode,
		AffiliateVisitorKey: input.AffiliateVisitorKey,
		ClientIP:            input.ClientIP,
		ManualFormData:      input.ManualFormData,
	})
}

// PreviewGuestOrder ????????
func (s *OrderService) PreviewGuestOrder(input CreateGuestOrderInput) (*OrderPreview, error) {
	return s.previewOrder(orderCreateParams{
		GuestEmail:          input.Email,
		GuestPassword:       input.OrderPassword,
		GuestLocale:         input.Locale,
		Items:               input.Items,
		CouponCode:          input.CouponCode,
		AffiliateCode:       input.AffiliateCode,
		AffiliateVisitorKey: input.AffiliateVisitorKey,
		ClientIP:            input.ClientIP,
		IsGuest:             true,
		ManualFormData:      input.ManualFormData,
	})
}

func (s *OrderService) previewOrder(input orderCreateParams) (*OrderPreview, error) {
	result, err := s.buildOrderResult(input)
	if err != nil {
		return nil, err
	}
	items := make([]OrderPreviewItem, 0, len(result.Plans))
	for _, plan := range result.Plans {
		item := plan.Item
		items = append(items, OrderPreviewItem{
			ProductID:         item.ProductID,
			SKUID:             item.SKUID,
			TitleJSON:         item.TitleJSON,
			SKUSnapshotJSON:   item.SKUSnapshotJSON,
			Tags:              item.Tags,
			UnitPrice:         item.UnitPrice,
			Quantity:          item.Quantity,
			TotalPrice:        item.TotalPrice,
			MemberDiscount:    item.MemberDiscount,
			CouponDiscount:    item.CouponDiscount,
			PromotionDiscount: item.PromotionDiscount,
			FulfillmentType:   item.FulfillmentType,
		})
	}
	return &OrderPreview{
		Currency:                result.Currency,
		OriginalAmount:          models.NewMoneyFromDecimal(result.OriginalAmount),
		MemberDiscountAmount:    models.NewMoneyFromDecimal(result.MemberDiscountAmount),
		DiscountAmount:          models.NewMoneyFromDecimal(result.DiscountAmount),
		PromotionDiscountAmount: models.NewMoneyFromDecimal(result.PromotionDiscountAmount),
		TotalAmount:             models.NewMoneyFromDecimal(result.TotalAmount),
		Items:                   items,
	}, nil
}

func (s *OrderService) createOrder(input orderCreateParams) (*models.Order, error) {
	// ????(??????)
	if s.riskControlSvc != nil && !input.SkipRiskControl {
		if err := s.riskControlSvc.CheckOrderAllowed(RiskCheckInput{
			UserID:      input.UserID,
			GuestEmail:  input.GuestEmail,
			ClientIP:    input.ClientIP,
			IsGuest:     input.IsGuest,
			SkipIPCheck: input.SkipIPRiskControl,
		}); err != nil {
			return nil, err
		}
	}

	result, err := s.buildOrderResult(input)
	if err != nil {
		return nil, err
	}

	// ??????????,?????(???)??????????
	if s.settingService != nil && s.settingService.GetWalletOnlyPayment() {
		if input.UserID == 0 {
			// ?????,wallet-only ????????
			return nil, ErrWalletOnlyPaymentRequired
		}
		if s.walletService == nil {
			return nil, ErrWalletOnlyPaymentRequired
		}
		account, accErr := s.walletService.GetAccount(input.UserID)
		if accErr != nil {
			return nil, ErrWalletOnlyPaymentRequired
		}
		if account.Balance.Decimal.LessThan(result.TotalAmount) {
			return nil, ErrWalletInsufficientBalance
		}
	}

	affiliateCode := normalizeAffiliateCode(input.AffiliateCode)
	affiliateVisitorKey := strings.TrimSpace(input.AffiliateVisitorKey)
	var affiliateProfileID *uint
	if s.affiliateSvc != nil {
		resolvedID, resolvedCode, resolveErr := s.affiliateSvc.ResolveOrderAffiliateSnapshot(input.UserID, affiliateCode, affiliateVisitorKey)
		if resolveErr != nil {
			return nil, resolveErr
		}
		affiliateProfileID = resolvedID
		affiliateCode = resolvedCode
	}

	if len(input.Items) == 0 {
		return nil, ErrInvalidOrderItem
	}
	if s.productSKURepo == nil {
		return nil, ErrProductSKUInvalid
	}
	if input.IsGuest && input.GuestEmail == "" {
		return nil, ErrGuestEmailRequired
	}
	if input.IsGuest && input.GuestPassword == "" {
		return nil, ErrGuestPasswordRequired
	}

	expireMinutes := s.resolveExpireMinutes()
	now := time.Now()
	expiresAt := now.Add(time.Duration(expireMinutes) * time.Minute)
	order := &models.Order{
		OrderNo:                 generateOrderNo(),
		UserID:                  input.UserID,
		GuestEmail:              input.GuestEmail,
		GuestPassword:           input.GuestPassword,
		GuestLocale:             input.GuestLocale,
		Status:                  constants.OrderStatusPendingPayment,
		Currency:                result.Currency,
		OriginalAmount:          models.NewMoneyFromDecimal(result.OriginalAmount),
		MemberDiscountAmount:    models.NewMoneyFromDecimal(result.MemberDiscountAmount),
		DiscountAmount:          models.NewMoneyFromDecimal(result.DiscountAmount),
		PromotionDiscountAmount: models.NewMoneyFromDecimal(result.PromotionDiscountAmount),
		TotalAmount:             models.NewMoneyFromDecimal(result.TotalAmount),
		WalletPaidAmount:        models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount:        models.NewMoneyFromDecimal(result.TotalAmount),
		RefundedAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		MemberLevelID:           result.MemberLevelID,
		CouponID:                nil,
		PromotionID:             result.OrderPromotionID,
		AffiliateProfileID:      affiliateProfileID,
		AffiliateCode:           affiliateCode,
		ExpiresAt:               &expiresAt,
		ClientIP:                strings.TrimSpace(input.ClientIP),
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	if result.AppliedCoupon != nil {
		order.CouponID = &result.AppliedCoupon.ID
	}

	err = s.orderRepo.Transaction(func(tx *gorm.DB) error {
		orderRepo := s.orderRepo.WithTx(tx)
		var productSKURepo repository.ProductSKURepository
		if s.productSKURepo != nil {
			productSKURepo = s.productSKURepo.WithTx(tx)
		}
		if err := orderRepo.Create(order, nil); err != nil {
			return err
		}

		for idx := range result.Plans {
			plan := result.Plans[idx]
			childOrder := &models.Order{
				OrderNo:                 buildChildOrderNo(order.OrderNo, idx+1),
				ParentID:                &order.ID,
				UserID:                  order.UserID,
				GuestEmail:              order.GuestEmail,
				GuestPassword:           order.GuestPassword,
				GuestLocale:             order.GuestLocale,
				Status:                  constants.OrderStatusPendingPayment,
				Currency:                plan.Currency,
				OriginalAmount:          models.NewMoneyFromDecimal(plan.TotalAmount),
				MemberDiscountAmount:    models.NewMoneyFromDecimal(plan.MemberDiscount),
				DiscountAmount:          models.NewMoneyFromDecimal(plan.CouponDiscount),
				PromotionDiscountAmount: models.NewMoneyFromDecimal(plan.PromotionDiscount),
				TotalAmount:             models.NewMoneyFromDecimal(normalizeOrderAmount(plan.TotalAmount.Sub(plan.CouponDiscount))),
				WalletPaidAmount:        models.NewMoneyFromDecimal(decimal.Zero),
				OnlinePaidAmount:        models.NewMoneyFromDecimal(normalizeOrderAmount(plan.TotalAmount.Sub(plan.CouponDiscount))),
				RefundedAmount:          models.NewMoneyFromDecimal(decimal.Zero),
				CouponID:                nil,
				PromotionID:             plan.Item.PromotionID,
				AffiliateProfileID:      affiliateProfileID,
				AffiliateCode:           affiliateCode,
				ExpiresAt:               &expiresAt,
				ClientIP:                order.ClientIP,
				CreatedAt:               now,
				UpdatedAt:               now,
			}
			if result.AppliedCoupon != nil && plan.CouponDiscount.GreaterThan(decimal.Zero) {
				childOrder.CouponID = &result.AppliedCoupon.ID
			}
			if err := orderRepo.Create(childOrder, []models.OrderItem{plan.Item}); err != nil {
				return err
			}

			if strings.TrimSpace(plan.Item.FulfillmentType) == constants.FulfillmentTypeAuto {
				if s.cardSecretRepo == nil {
					return ErrCardSecretInsufficient
				}
				secretRepo := s.cardSecretRepo.WithTx(tx)
				var rows []models.CardSecret
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("product_id = ? AND sku_id = ? AND status = ?", plan.Item.ProductID, plan.Item.SKUID, models.CardSecretStatusAvailable).
					Order("id asc").Limit(plan.Item.Quantity).Find(&rows).Error; err != nil {
					return err
				}
				if len(rows) < plan.Item.Quantity {
					logger.Infow("stock_reserve_insufficient",
						"product_id", plan.Item.ProductID,
						"sku_id", plan.Item.SKUID,
						"required", plan.Item.Quantity,
						"available", len(rows),
					)
					return ErrCardSecretInsufficient
				}
				ids := make([]uint, 0, len(rows))
				for _, row := range rows {
					ids = append(ids, row.ID)
				}
				affected, err := secretRepo.Reserve(ids, childOrder.ID, now)
				if err != nil {
					return err
				}
				if int(affected) != len(ids) {
					logger.Warnw("stock_reserve_partial",
						"order_id", childOrder.ID,
						"expected", len(ids),
						"affected", affected,
					)
					return ErrCardSecretInsufficient
				}
				logger.Infow("stock_reserved",
					"order_id", childOrder.ID,
					"product_id", plan.Item.ProductID,
					"sku_id", plan.Item.SKUID,
					"count", affected,
					"card_secret_ids", ids,
				)
			}
			if strings.TrimSpace(plan.Item.FulfillmentType) == constants.FulfillmentTypeManual &&
				plan.SKU != nil &&
				shouldEnforceManualSKUStock(plan.Product, plan.SKU) {
				affected, err := productSKURepo.ReserveManualStock(plan.Item.SKUID, plan.Item.Quantity)
				if err != nil {
					return err
				}
				if affected == 0 {
					return ErrManualStockInsufficient
				}
			}
		}

		if result.AppliedCoupon != nil {
			couponRepo := s.couponRepo.WithTx(tx)
			usageRepo := s.couponUsageRepo.WithTx(tx)
			usage := &models.CouponUsage{
				CouponID:       result.AppliedCoupon.ID,
				UserID:         input.UserID,
				OrderID:        order.ID,
				DiscountAmount: models.NewMoneyFromDecimal(result.DiscountAmount),
				CreatedAt:      now,
			}
			if err := usageRepo.Create(usage); err != nil {
				return err
			}
			if err := couponRepo.IncrementUsedCount(result.AppliedCoupon.ID, 1); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrCardSecretInsufficient) {
			return nil, ErrCardSecretInsufficient
		}
		if errors.Is(err, ErrManualStockInsufficient) {
			return nil, ErrManualStockInsufficient
		}
		return nil, ErrOrderCreateFailed
	}

	if s.queueClient != nil && s.queueClient.Enabled() {
		if err := s.queueClient.EnqueueOrderTimeoutCancel(queue.OrderTimeoutCancelPayload{
			OrderID: order.ID,
		}, time.Duration(expireMinutes)*time.Minute); err != nil {
			logger.Errorw("order_enqueue_timeout_cancel_failed",
				"order_id", order.ID,
				"order_no", order.OrderNo,
				"error", err,
			)
			// ????????,??????,???????????
			logger.Warnw("order_timeout_cancel_not_enabled",
				"order_id", order.ID,
				"order_no", order.OrderNo,
			)
		}
	} else {
		logger.Warnw("order_timeout_cancel_not_enabled",
			"order_id", order.ID,
			"order_no", order.OrderNo,
		)
	}

	full, err := s.orderRepo.GetByID(order.ID)
	if err == nil && full != nil {
		fillOrderItemsFromChildren(full)
		return full, nil
	}
	fillOrderItemsFromChildren(order)
	return order, nil
}

func generateOrderNo() string {
	return generateSerialNo("DJ")
}
