package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func setupOrderRefundServiceTest(t *testing.T) (*OrderRefundService, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:order_service_refund_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Order{},
		&models.OrderItem{},
		&models.Fulfillment{},
		&models.SiteConnection{},
		&models.ProcurementOrder{},
		&models.AffiliateProfile{},
		&models.AffiliateCommission{},
		&models.AffiliateWithdrawRequest{},
		&models.WalletAccount{},
		&models.WalletTransaction{},
		&models.OrderRefundRecord{},
		&models.Setting{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	models.DB = db

	orderRepo := repository.NewOrderRepository(db)
	orderRefundRecordRepo := repository.NewOrderRefundRecordRepository(db)
	affiliateSvc := NewAffiliateService(repository.NewAffiliateRepository(db), nil, nil, nil, nil)
	userRepo := repository.NewUserRepository(db)
	settingSvc := NewSettingService(repository.NewSettingRepository(db))
	return NewOrderRefundService(orderRepo, userRepo, orderRefundRecordRepo, affiliateSvc, settingSvc), db
}

func createOrderRefundTestSiteConnection(t *testing.T, db *gorm.DB, id uint) *models.SiteConnection {
	t.Helper()
	conn := &models.SiteConnection{
		ID:        id,
		Name:      fmt.Sprintf("refund-conn-%d", id),
		BaseURL:   "https://upstream.example.com",
		ApiKey:    fmt.Sprintf("refund-key-%d", id),
		ApiSecret: "secret",
		Protocol:  constants.ConnectionProtocolDujiaoNext,
		Status:    constants.ConnectionStatusActive,
		RetryMax:  3,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(conn).Error; err != nil {
		t.Fatalf("create site connection failed: %v", err)
	}
	return conn
}

func TestOrderRefundServiceAdminManualRefundGuestCreatesRecord(t *testing.T) {
	svc, db := setupOrderRefundServiceTest(t)
	now := time.Now()
	order := &models.Order{
		OrderNo:          "REFUND-MANUAL-GUEST-001",
		UserID:           0,
		GuestEmail:       "guest-refund-record@example.com",
		Status:           constants.OrderStatusCompleted,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		PaidAt:           &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create guest order failed: %v", err)
	}
	conn := createOrderRefundTestSiteConnection(t, db, 1)
	proc := &models.ProcurementOrder{
		ConnectionID:    conn.ID,
		LocalOrderID:    order.ID,
		LocalOrderNo:    order.OrderNo,
		Status:          constants.ProcurementStatusFulfilled,
		LocalSellAmount: models.NewMoneyFromDecimal(order.TotalAmount.Decimal),
		Currency:        order.Currency,
		TraceID:         "manual-refund-proc-sync",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(proc).Error; err != nil {
		t.Fatalf("create procurement order failed: %v", err)
	}

	updatedOrder, createdRecord, err := svc.AdminManualRefund(AdminManualRefundInput{
		OrderID: order.ID,
		Amount:  models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		Remark:  "manual partial refund",
	})
	if err != nil {
		t.Fatalf("admin manual refund failed: %v", err)
	}
	if updatedOrder == nil || updatedOrder.Status != constants.OrderStatusPartiallyRefunded {
		t.Fatalf("expected partially_refunded order, got %+v", updatedOrder)
	}
	if createdRecord == nil || createdRecord.ID == 0 {
		t.Fatalf("expected created refund record, got %+v", createdRecord)
	}

	var record models.OrderRefundRecord
	if err := db.Order("id DESC").First(&record).Error; err != nil {
		t.Fatalf("query refund record failed: %v", err)
	}
	if record.OrderID != order.ID {
		t.Fatalf("unexpected refund record order id: %d", record.OrderID)
	}
	if record.UserID != 0 {
		t.Fatalf("unexpected refund record user id: %d", record.UserID)
	}
	if record.GuestEmail != "guest-refund-record@example.com" {
		t.Fatalf("unexpected refund record guest email: %s", record.GuestEmail)
	}
	if record.Type != constants.OrderRefundTypeManual {
		t.Fatalf("unexpected refund record type: %s", record.Type)
	}
	if !record.Amount.Decimal.Equal(decimal.NewFromInt(20)) {
		t.Fatalf("unexpected refund record amount: %s", record.Amount.String())
	}
	if record.Remark != "manual partial refund" {
		t.Fatalf("unexpected refund record remark: %s", record.Remark)
	}
	var refreshedProc models.ProcurementOrder
	if err := db.First(&refreshedProc, proc.ID).Error; err != nil {
		t.Fatalf("reload procurement order failed: %v", err)
	}
	if refreshedProc.Status != constants.ProcurementStatusFulfilled {
		t.Fatalf("expected procurement status fulfilled, got: %s", refreshedProc.Status)
	}
}

func TestOrderRefundServiceAdminManualRefundExpiredWindow(t *testing.T) {
	svc, db := setupOrderRefundServiceTest(t)
	paidAt := time.Now().AddDate(0, 0, -31)
	order := &models.Order{
		OrderNo:          "REFUND-MANUAL-EXPIRED-001",
		UserID:           0,
		GuestEmail:       "guest-refund-expired@example.com",
		Status:           constants.OrderStatusCompleted,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		PaidAt:           &paidAt,
		CreatedAt:        paidAt,
		UpdatedAt:        paidAt,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create guest order failed: %v", err)
	}

	_, _, err := svc.AdminManualRefund(AdminManualRefundInput{
		OrderID: order.ID,
		Amount:  models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		Remark:  "manual refund expired",
	})
	if !errors.Is(err, ErrOrderRefundExpired) {
		t.Fatalf("expected refund expired, got: %v", err)
	}
}

func TestOrderRefundServiceAdminManualRefundNoLimitWhenZero(t *testing.T) {
	svc, db := setupOrderRefundServiceTest(t)
	paidAt := time.Now().AddDate(0, 0, -90)
	order := &models.Order{
		OrderNo:          "REFUND-MANUAL-NOLIMIT-001",
		UserID:           0,
		GuestEmail:       "guest-refund-nolimit@example.com",
		Status:           constants.OrderStatusCompleted,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		PaidAt:           &paidAt,
		CreatedAt:        paidAt,
		UpdatedAt:        paidAt,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create guest order failed: %v", err)
	}

	if _, err := svc.settingService.Update(constants.SettingKeyOrderConfig, map[string]interface{}{
		orderConfigFieldMaxRefundDays: 0,
	}); err != nil {
		t.Fatalf("update order refund config failed: %v", err)
	}

	updatedOrder, _, err := svc.AdminManualRefund(AdminManualRefundInput{
		OrderID: order.ID,
		Amount:  models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		Remark:  "manual refund no limit",
	})
	if err != nil {
		t.Fatalf("expected no-limit refund success, got: %v", err)
	}
	if updatedOrder == nil || updatedOrder.Status != constants.OrderStatusPartiallyRefunded {
		t.Fatalf("expected partially_refunded order, got %+v", updatedOrder)
	}
}

func createOrderRefundTestChildWithFulfillmentType(
	t *testing.T,
	db *gorm.DB,
	parent *models.Order,
	orderNo string,
	status string,
	total decimal.Decimal,
	fulfillmentType string,
	withFulfillment bool,
) *models.Order {
	t.Helper()
	now := time.Now()
	child := &models.Order{
		OrderNo:          orderNo,
		ParentID:         &parent.ID,
		UserID:           parent.UserID,
		GuestEmail:       parent.GuestEmail,
		GuestPassword:    parent.GuestPassword,
		GuestLocale:      parent.GuestLocale,
		Status:           status,
		Currency:         parent.Currency,
		OriginalAmount:   models.NewMoneyFromDecimal(total),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(total),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(total),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		PaidAt:           parent.PaidAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("create child order failed: %v", err)
	}
	item := &models.OrderItem{
		OrderID:         child.ID,
		ProductID:       child.ID + 2000,
		SKUID:           1,
		TitleJSON:       models.JSON{"zh-CN": orderNo},
		UnitPrice:       models.NewMoneyFromDecimal(total),
		CostPrice:       models.NewMoneyFromDecimal(decimal.Zero),
		Quantity:        1,
		TotalPrice:      models.NewMoneyFromDecimal(total),
		FulfillmentType: fulfillmentType,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("create child order item failed: %v", err)
	}
	if withFulfillment {
		fulfillment := &models.Fulfillment{
			OrderID:     child.ID,
			Type:        fulfillmentType,
			Status:      constants.FulfillmentStatusDelivered,
			Payload:     "DELIVERED-CONTENT",
			DeliveredAt: &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := db.Create(fulfillment).Error; err != nil {
			t.Fatalf("create child fulfillment failed: %v", err)
		}
	}
	return child
}

func TestOrderRefundServiceAdminManualRefundParentPartialMixedChildrenStatus(t *testing.T) {
	svc, db := setupOrderRefundServiceTest(t)
	now := time.Now()
	parent := &models.Order{
		OrderNo:          "REFUND-MANUAL-MIXED-PARTIAL",
		UserID:           0,
		GuestEmail:       "guest-mixed-partial@example.com",
		Status:           constants.OrderStatusCompleted,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		PaidAt:           &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.Create(parent).Error; err != nil {
		t.Fatalf("create parent order failed: %v", err)
	}

	manualChild := createOrderRefundTestChildWithFulfillmentType(
		t, db, parent, "REFUND-MANUAL-MIXED-PARTIAL-1", constants.OrderStatusPaid,
		decimal.NewFromInt(60), constants.FulfillmentTypeManual, false,
	)
	autoChild := createOrderRefundTestChildWithFulfillmentType(
		t, db, parent, "REFUND-MANUAL-MIXED-PARTIAL-2", constants.OrderStatusCompleted,
		decimal.NewFromInt(40), constants.FulfillmentTypeAuto, true,
	)

	updatedOrder, _, err := svc.AdminManualRefund(AdminManualRefundInput{
		OrderID: parent.ID,
		Amount:  models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		Remark:  "manual mixed partial refund",
	})
	if err != nil {
		t.Fatalf("admin manual refund failed: %v", err)
	}
	if updatedOrder == nil || updatedOrder.Status != constants.OrderStatusPartiallyRefunded {
		t.Fatalf("expected parent partially_refunded, got %+v", updatedOrder)
	}

	var refreshedManual models.Order
	if err := db.First(&refreshedManual, manualChild.ID).Error; err != nil {
		t.Fatalf("reload manual child failed: %v", err)
	}
	if refreshedManual.Status != constants.OrderStatusPartiallyRefunded {
		t.Fatalf("expected manual child partially_refunded, got: %s", refreshedManual.Status)
	}

	var refreshedAuto models.Order
	if err := db.First(&refreshedAuto, autoChild.ID).Error; err != nil {
		t.Fatalf("reload auto child failed: %v", err)
	}
	if refreshedAuto.Status != constants.OrderStatusPartiallyRefunded {
		t.Fatalf("expected auto child partially_refunded, got: %s", refreshedAuto.Status)
	}
}

func TestOrderRefundServiceAdminManualRefundParentFullMixedChildrenStatus(t *testing.T) {
	svc, db := setupOrderRefundServiceTest(t)
	now := time.Now()
	parent := &models.Order{
		OrderNo:          "REFUND-MANUAL-MIXED-FULL",
		UserID:           0,
		GuestEmail:       "guest-mixed-full@example.com",
		Status:           constants.OrderStatusCompleted,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		PaidAt:           &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.Create(parent).Error; err != nil {
		t.Fatalf("create parent order failed: %v", err)
	}

	manualChild := createOrderRefundTestChildWithFulfillmentType(
		t, db, parent, "REFUND-MANUAL-MIXED-FULL-1", constants.OrderStatusPaid,
		decimal.NewFromInt(60), constants.FulfillmentTypeManual, false,
	)
	autoChild := createOrderRefundTestChildWithFulfillmentType(
		t, db, parent, "REFUND-MANUAL-MIXED-FULL-2", constants.OrderStatusCompleted,
		decimal.NewFromInt(40), constants.FulfillmentTypeAuto, true,
	)

	updatedOrder, _, err := svc.AdminManualRefund(AdminManualRefundInput{
		OrderID: parent.ID,
		Amount:  models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		Remark:  "manual mixed full refund",
	})
	if err != nil {
		t.Fatalf("admin manual refund failed: %v", err)
	}
	if updatedOrder == nil || updatedOrder.Status != constants.OrderStatusRefunded {
		t.Fatalf("expected parent refunded, got %+v", updatedOrder)
	}

	var refreshedManual models.Order
	if err := db.First(&refreshedManual, manualChild.ID).Error; err != nil {
		t.Fatalf("reload manual child failed: %v", err)
	}
	if refreshedManual.Status != constants.OrderStatusRefunded {
		t.Fatalf("expected manual child refunded, got: %s", refreshedManual.Status)
	}

	var refreshedAuto models.Order
	if err := db.First(&refreshedAuto, autoChild.ID).Error; err != nil {
		t.Fatalf("reload auto child failed: %v", err)
	}
	if refreshedAuto.Status != constants.OrderStatusRefunded {
		t.Fatalf("expected auto child refunded, got: %s", refreshedAuto.Status)
	}
}

func TestOrderRefundServiceResolveStatusEmailRefundDetails(t *testing.T) {
	svc, db := setupOrderRefundServiceTest(t)
	now := time.Now()

	parent := &models.Order{
		OrderNo:          "REFUND-DETAILS-PARENT",
		UserID:           0,
		GuestEmail:       "guest-details-parent@example.com",
		Status:           constants.OrderStatusPartiallyRefunded,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		PaidAt:           &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.Create(parent).Error; err != nil {
		t.Fatalf("create parent order failed: %v", err)
	}

	child := &models.Order{
		OrderNo:          "REFUND-DETAILS-PARENT-1",
		ParentID:         &parent.ID,
		UserID:           0,
		GuestEmail:       parent.GuestEmail,
		Status:           constants.OrderStatusPartiallyRefunded,
		Currency:         parent.Currency,
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		PaidAt:           &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("create child order failed: %v", err)
	}

	unrelated := &models.Order{
		OrderNo:          "REFUND-DETAILS-UNRELATED",
		UserID:           0,
		GuestEmail:       "guest-unrelated@example.com",
		Status:           constants.OrderStatusPaid,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		PaidAt:           &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.Create(unrelated).Error; err != nil {
		t.Fatalf("create unrelated order failed: %v", err)
	}

	parentRecord := &models.OrderRefundRecord{
		OrderID:    parent.ID,
		Type:       constants.OrderRefundTypeManual,
		Amount:     models.NewMoneyFromDecimal(decimal.NewFromInt(12)),
		Currency:   "CNY",
		Remark:     "parent refund record",
		CreatedAt:  now,
		UpdatedAt:  now,
		GuestEmail: parent.GuestEmail,
	}
	if err := db.Create(parentRecord).Error; err != nil {
		t.Fatalf("create parent refund record failed: %v", err)
	}
	childRecord := &models.OrderRefundRecord{
		OrderID:    child.ID,
		Type:       constants.OrderRefundTypeManual,
		Amount:     models.NewMoneyFromDecimal(decimal.NewFromInt(8)),
		Currency:   "CNY",
		Remark:     "child refund record",
		CreatedAt:  now,
		UpdatedAt:  now,
		GuestEmail: child.GuestEmail,
	}
	if err := db.Create(childRecord).Error; err != nil {
		t.Fatalf("create child refund record failed: %v", err)
	}

	details, ok, err := svc.ResolveStatusEmailRefundDetails(parent.ID, parentRecord.ID)
	if err != nil {
		t.Fatalf("resolve same order refund details failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected same order refund record to match")
	}
	if !details.Amount.Decimal.Equal(decimal.NewFromInt(12)) || details.Reason != "parent refund record" {
		t.Fatalf("unexpected same order details: %+v", details)
	}

	details, ok, err = svc.ResolveStatusEmailRefundDetails(parent.ID, childRecord.ID)
	if err != nil {
		t.Fatalf("resolve child refund details failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected child order refund record to match parent order")
	}
	if !details.Amount.Decimal.Equal(decimal.NewFromInt(8)) || details.Reason != "child refund record" {
		t.Fatalf("unexpected child order details: %+v", details)
	}

	_, ok, err = svc.ResolveStatusEmailRefundDetails(unrelated.ID, childRecord.ID)
	if err != nil {
		t.Fatalf("resolve unrelated order refund details failed: %v", err)
	}
	if ok {
		t.Fatalf("expected unrelated order to not match refund record")
	}

	parsed, err := svc.ResolveOrderStatusEmailRefundDetails(parent, parentRecord.ID)
	if err != nil {
		t.Fatalf("resolve order status email refund details by record failed: %v", err)
	}
	if !parsed.Amount.Decimal.Equal(decimal.NewFromInt(12)) || parsed.Reason != "parent refund record" {
		t.Fatalf("unexpected parsed details by record: %+v", parsed)
	}

	fallback, err := svc.ResolveOrderStatusEmailRefundDetails(parent, 999999)
	if err != nil {
		t.Fatalf("resolve order status email refund details fallback failed: %v", err)
	}
	if !fallback.Amount.Decimal.Equal(decimal.NewFromInt(20)) || fallback.Reason != "" {
		t.Fatalf("unexpected fallback details: %+v", fallback)
	}

	var nilSvc *OrderRefundService
	fallbackFromNilSvc, err := nilSvc.ResolveOrderStatusEmailRefundDetails(parent, parentRecord.ID)
	if err != nil {
		t.Fatalf("resolve order status email refund details with nil service failed: %v", err)
	}
	if !fallbackFromNilSvc.Amount.Decimal.Equal(decimal.NewFromInt(20)) || fallbackFromNilSvc.Reason != "" {
		t.Fatalf("unexpected fallback from nil service: %+v", fallbackFromNilSvc)
	}
}
