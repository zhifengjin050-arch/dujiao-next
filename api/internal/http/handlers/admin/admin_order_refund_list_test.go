package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/provider"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type adminOrderRefundFixture struct {
	MemberUserID   uint
	MemberOrderID  uint
	GuestOrderID   uint
	ManualRefundID uint
	WalletRefundID uint
}

func setupAdminOrderRefundHandlerTest(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:admin_order_refund_handler_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserOAuthIdentity{},
		&models.Order{},
		&models.OrderItem{},
		&models.Fulfillment{},
		&models.OrderRefundRecord{},
		&models.AffiliateProfile{},
		&models.AffiliateCommission{},
		&models.AffiliateWithdrawRequest{},
		&models.WalletAccount{},
		&models.WalletTransaction{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	orderRepo := repository.NewOrderRepository(db)
	orderRefundRecordRepo := repository.NewOrderRefundRecordRepository(db)
	userRepo := repository.NewUserRepository(db)
	affiliateSvc := service.NewAffiliateService(repository.NewAffiliateRepository(db), nil, nil, nil, nil)
	orderRefundService := service.NewOrderRefundService(orderRepo, userRepo, orderRefundRecordRepo, affiliateSvc, nil)

	h := &Handler{Container: &provider.Container{
		OrderRepo:             orderRepo,
		UserRepo:              userRepo,
		OrderRefundRecordRepo: orderRefundRecordRepo,
		OrderRefundService:    orderRefundService,
	}}
	return h, db
}

func seedAdminOrderRefundData(t *testing.T, db *gorm.DB) adminOrderRefundFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)

	member := &models.User{
		Email:        "refund-member@example.com",
		DisplayName:  "Refund Member",
		PasswordHash: "hash",
		Status:       constants.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(member).Error; err != nil {
		t.Fatalf("create member user failed: %v", err)
	}
	if err := db.Create(&models.UserOAuthIdentity{
		UserID:         member.ID,
		Provider:       "telegram",
		ProviderUserID: "700001",
		Username:       "refund_member_tg",
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error; err != nil {
		t.Fatalf("create oauth identity failed: %v", err)
	}

	memberOrder := &models.Order{
		OrderNo:          "DJ-ADMIN-REFUND-ORDER-1",
		UserID:           member.ID,
		Status:           constants.OrderStatusPartiallyRefunded,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		CreatedAt:        now.Add(-2 * time.Hour),
		UpdatedAt:        now.Add(-2 * time.Hour),
	}
	guestOrder := &models.Order{
		OrderNo:          "DJ-ADMIN-REFUND-ORDER-2",
		UserID:           0,
		GuestEmail:       "refund-guest@example.com",
		GuestLocale:      constants.LocaleZhCN,
		Status:           constants.OrderStatusPartiallyRefunded,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		CreatedAt:        now.Add(-90 * time.Minute),
		UpdatedAt:        now.Add(-90 * time.Minute),
	}
	for _, order := range []*models.Order{memberOrder, guestOrder} {
		if err := db.Create(order).Error; err != nil {
			t.Fatalf("create order failed: %v", err)
		}
	}

	items := []models.OrderItem{
		{
			OrderID:         memberOrder.ID,
			ProductID:       1,
			TitleJSON:       models.JSON{"zh-CN": "????????"},
			UnitPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
			CostPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
			Quantity:        1,
			TotalPrice:      models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
			CouponDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
			FulfillmentType: constants.FulfillmentTypeManual,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			OrderID:         guestOrder.ID,
			ProductID:       2,
			TitleJSON:       models.JSON{"zh-CN": "????????"},
			UnitPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
			CostPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(40)),
			Quantity:        1,
			TotalPrice:      models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
			CouponDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
			FulfillmentType: constants.FulfillmentTypeManual,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}
	for idx := range items {
		if err := db.Create(&items[idx]).Error; err != nil {
			t.Fatalf("create order item failed: %v", err)
		}
	}

	manualRefund := &models.OrderRefundRecord{
		UserID:     member.ID,
		OrderID:    memberOrder.ID,
		Type:       constants.OrderRefundTypeManual,
		Amount:     models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		Currency:   "CNY",
		Remark:     "manual refund reason",
		CreatedAt:  now.Add(-40 * time.Minute),
		UpdatedAt:  now.Add(-40 * time.Minute),
		GuestEmail: "",
	}
	walletRefund := &models.OrderRefundRecord{
		UserID:     0,
		GuestEmail: "refund-guest@example.com",
		OrderID:    guestOrder.ID,
		Type:       constants.OrderRefundTypeWallet,
		Amount:     models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		Currency:   "CNY",
		Remark:     "wallet refund reason",
		CreatedAt:  now.Add(-20 * time.Minute),
		UpdatedAt:  now.Add(-20 * time.Minute),
	}
	for _, record := range []*models.OrderRefundRecord{manualRefund, walletRefund} {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("create refund record failed: %v", err)
		}
	}

	return adminOrderRefundFixture{
		MemberUserID:   member.ID,
		MemberOrderID:  memberOrder.ID,
		GuestOrderID:   guestOrder.ID,
		ManualRefundID: manualRefund.ID,
		WalletRefundID: walletRefund.ID,
	}
}

func TestGetAdminOrderRefundsFiltersByUserKeywordAndProduct(t *testing.T) {
	h, db := setupAdminOrderRefundHandlerTest(t)
	fixture := seedAdminOrderRefundData(t, db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/admin/order-refunds?page=1&page_size=20&user_keyword=refund_member_tg&product_keyword=????????",
		nil,
	)

	h.GetAdminOrderRefunds(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status want 200 got %d", w.Code)
	}

	var resp struct {
		StatusCode int `json:"status_code"`
		Pagination struct {
			Total int64 `json:"total"`
		} `json:"pagination"`
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.StatusCode != 0 {
		t.Fatalf("status_code want 0 got %d", resp.StatusCode)
	}
	if resp.Pagination.Total != 1 || len(resp.Data) != 1 {
		t.Fatalf("unexpected list result: total=%d len=%d", resp.Pagination.Total, len(resp.Data))
	}

	row := resp.Data[0]
	if uint(row["id"].(float64)) != fixture.ManualRefundID {
		t.Fatalf("unexpected refund id: %+v", row["id"])
	}
	if row["refund_type_label"] != "manual" {
		t.Fatalf("unexpected refund_type_label: %+v", row["refund_type_label"])
	}
	if row["order_no"] != "DJ-ADMIN-REFUND-ORDER-1" {
		t.Fatalf("unexpected order_no: %+v", row["order_no"])
	}
	if row["user_email"] != "refund-member@example.com" {
		t.Fatalf("unexpected user_email: %+v", row["user_email"])
	}
	items, ok := row["items"].([]interface{})
	if !ok || len(items) == 0 {
		t.Fatalf("items should not be empty: %+v", row["items"])
	}
}

func TestGetAdminOrderRefundsGuestRowUsesGuestEmail(t *testing.T) {
	h, db := setupAdminOrderRefundHandlerTest(t)
	fixture := seedAdminOrderRefundData(t, db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/admin/order-refunds?page=1&page_size=20&guest_email=refund-guest@example.com",
		nil,
	)

	h.GetAdminOrderRefunds(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status want 200 got %d", w.Code)
	}

	var resp struct {
		StatusCode int                      `json:"status_code"`
		Data       []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.StatusCode != 0 {
		t.Fatalf("status_code want 0 got %d", resp.StatusCode)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("unexpected data size: %d", len(resp.Data))
	}

	row := resp.Data[0]
	if uint(row["id"].(float64)) != fixture.WalletRefundID {
		t.Fatalf("unexpected refund id: %+v", row["id"])
	}
	if row["guest_email"] != "refund-guest@example.com" {
		t.Fatalf("unexpected guest_email: %+v", row["guest_email"])
	}
	if row["guest_locale"] != constants.LocaleZhCN {
		t.Fatalf("unexpected guest_locale: %+v", row["guest_locale"])
	}
	if row["user_email"] != nil {
		t.Fatalf("guest refund should not include user_email: %+v", row["user_email"])
	}
	if row["user_display_name"] != nil {
		t.Fatalf("guest refund should not include user_display_name: %+v", row["user_display_name"])
	}
}

func TestGetAdminOrderRefundDetailIncludesOrderIDAndGuestEmail(t *testing.T) {
	h, db := setupAdminOrderRefundHandlerTest(t)
	fixture := seedAdminOrderRefundData(t, db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", fixture.WalletRefundID)}}
	c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/order-refunds/%d", fixture.WalletRefundID), nil)

	h.GetAdminOrderRefund(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status want 200 got %d", w.Code)
	}

	var resp struct {
		StatusCode int                    `json:"status_code"`
		Data       map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.StatusCode != 0 {
		t.Fatalf("status_code want 0 got %d", resp.StatusCode)
	}
	if uint(resp.Data["id"].(float64)) != fixture.WalletRefundID {
		t.Fatalf("unexpected refund id: %+v", resp.Data["id"])
	}
	if resp.Data["refund_type_label"] != "wallet" {
		t.Fatalf("unexpected refund_type_label: %+v", resp.Data["refund_type_label"])
	}
	if uint(resp.Data["order_id"].(float64)) != fixture.GuestOrderID {
		t.Fatalf("unexpected order_id: %+v", resp.Data["order_id"])
	}
	if resp.Data["guest_email"] != "refund-guest@example.com" {
		t.Fatalf("unexpected guest_email: %+v", resp.Data["guest_email"])
	}
	if resp.Data["guest_locale"] != constants.LocaleZhCN {
		t.Fatalf("unexpected guest_locale: %+v", resp.Data["guest_locale"])
	}
	if resp.Data["user_email"] != nil {
		t.Fatalf("guest refund should not include user_email: %+v", resp.Data["user_email"])
	}
	if resp.Data["user_display_name"] != nil {
		t.Fatalf("guest refund should not include user_display_name: %+v", resp.Data["user_display_name"])
	}
	if resp.Data["remark"] != "wallet refund reason" {
		t.Fatalf("unexpected remark: %+v", resp.Data["remark"])
	}
	detailItems, ok := resp.Data["items"].([]interface{})
	if !ok || len(detailItems) == 0 {
		t.Fatalf("items should not be empty: %+v", resp.Data["items"])
	}
}
