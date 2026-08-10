package admin

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/provider"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type adminPaymentFixture struct {
	User1ID              uint
	User2ID              uint
	OrderPaymentID       uint
	RechargePaymentUser1 uint
	RechargePaymentUser2 uint
	RechargeNoUser1      string
	RechargeNoUser2      string
}

func setupAdminPaymentHandlerTest(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:admin_payment_handler_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Order{},
		&models.PaymentChannel{},
		&models.Payment{},
		&models.WalletRechargeOrder{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	paymentRepo := repository.NewPaymentRepository(db)
	paymentChannelRepo := repository.NewPaymentChannelRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	paymentService := service.NewPaymentService(service.PaymentServiceOptions{
		PaymentRepo:   paymentRepo,
		ChannelRepo:   paymentChannelRepo,
		WalletRepo:    walletRepo,
		ExpireMinutes: 15,
	})

	h := &Handler{Container: &provider.Container{
		PaymentService:     paymentService,
		PaymentChannelRepo: paymentChannelRepo,
		WalletRepo:         walletRepo,
	}}
	return h, db
}

func seedAdminPaymentData(t *testing.T, db *gorm.DB) adminPaymentFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)

	user1 := models.User{
		Email:        "admin_handler_user1@example.com",
		PasswordHash: "hash",
		Status:       constants.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	user2 := models.User{
		Email:        "admin_handler_user2@example.com",
		PasswordHash: "hash",
		Status:       constants.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(&user1).Error; err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}
	if err := db.Create(&user2).Error; err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}

	channel1 := models.PaymentChannel{
		Name:            "Alipay Official",
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeAlipay,
		InteractionMode: constants.PaymentInteractionRedirect,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		ConfigJSON:      models.JSON{},
		IsActive:        true,
		SortOrder:       100,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	channel2 := models.PaymentChannel{
		Name:            "Wechat Official",
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeWechat,
		InteractionMode: constants.PaymentInteractionQR,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		ConfigJSON:      models.JSON{},
		IsActive:        true,
		SortOrder:       90,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&channel1).Error; err != nil {
		t.Fatalf("create channel1 failed: %v", err)
	}
	if err := db.Create(&channel2).Error; err != nil {
		t.Fatalf("create channel2 failed: %v", err)
	}

	order := models.Order{
		OrderNo:                 "DJADMINHANDLER001",
		UserID:                  user1.ID,
		Status:                  constants.OrderStatusPendingPayment,
		Currency:                "CNY",
		OriginalAmount:          models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		DiscountAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		PromotionDiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:             models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		WalletPaidAmount:        models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount:        models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		RefundedAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order failed: %v", err)
	}

	orderPayment := models.Payment{
		OrderID:         order.ID,
		ChannelID:       channel1.ID,
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeAlipay,
		InteractionMode: constants.PaymentInteractionRedirect,
		Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.PaymentStatusSuccess,
		ProviderRef:     "order_payment_ref_1",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&orderPayment).Error; err != nil {
		t.Fatalf("create order payment failed: %v", err)
	}

	rechargePaymentUser1 := models.Payment{
		OrderID:         0,
		ChannelID:       channel2.ID,
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeWechat,
		InteractionMode: constants.PaymentInteractionQR,
		Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.PaymentStatusPending,
		ProviderRef:     "recharge_payment_ref_user1",
		CreatedAt:       now.Add(time.Second),
		UpdatedAt:       now.Add(time.Second),
	}
	if err := db.Create(&rechargePaymentUser1).Error; err != nil {
		t.Fatalf("create user1 recharge payment failed: %v", err)
	}
	rechargeNoUser1 := "DJADMINRECHARGE001"
	if err := db.Create(&models.WalletRechargeOrder{
		RechargeNo:      rechargeNoUser1,
		UserID:          user1.ID,
		PaymentID:       rechargePaymentUser1.ID,
		ChannelID:       channel2.ID,
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeWechat,
		InteractionMode: constants.PaymentInteractionQR,
		Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		PayableAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.WalletRechargeStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create user1 recharge order failed: %v", err)
	}

	rechargePaymentUser2 := models.Payment{
		OrderID:         0,
		ChannelID:       channel2.ID,
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeWechat,
		InteractionMode: constants.PaymentInteractionQR,
		Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.PaymentStatusPending,
		ProviderRef:     "recharge_payment_ref_user2",
		CreatedAt:       now.Add(2 * time.Second),
		UpdatedAt:       now.Add(2 * time.Second),
	}
	if err := db.Create(&rechargePaymentUser2).Error; err != nil {
		t.Fatalf("create user2 recharge payment failed: %v", err)
	}
	rechargeNoUser2 := "DJADMINRECHARGE002"
	if err := db.Create(&models.WalletRechargeOrder{
		RechargeNo:      rechargeNoUser2,
		UserID:          user2.ID,
		PaymentID:       rechargePaymentUser2.ID,
		ChannelID:       channel2.ID,
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeWechat,
		InteractionMode: constants.PaymentInteractionQR,
		Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		PayableAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.WalletRechargeStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create user2 recharge order failed: %v", err)
	}

	return adminPaymentFixture{
		User1ID:              user1.ID,
		User2ID:              user2.ID,
		OrderPaymentID:       orderPayment.ID,
		RechargePaymentUser1: rechargePaymentUser1.ID,
		RechargePaymentUser2: rechargePaymentUser2.ID,
		RechargeNoUser1:      rechargeNoUser1,
		RechargeNoUser2:      rechargeNoUser2,
	}
}

func TestBuildAdminPaymentFilterInvalidOrderID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/payments?order_id=bad", nil)

	_, err := buildAdminPaymentFilter(c, 1, 20)
	if err == nil {
		t.Fatalf("expected invalid order_id error")
	}
}

func TestGetAdminPaymentsFiltersByUserID(t *testing.T) {
	h, db := setupAdminPaymentHandlerTest(t)
	fixture := seedAdminPaymentData(t, db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	url := fmt.Sprintf("/admin/payments?user_id=%d&page=1&page_size=20", fixture.User1ID)
	c.Request = httptest.NewRequest(http.MethodGet, url, nil)

	h.GetAdminPayments(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status want 200 got %d", w.Code)
	}

	var resp struct {
		StatusCode int                      `json:"status_code"`
		Pagination responsePaginationAssert `json:"pagination"`
		Data       []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.StatusCode != 0 {
		t.Fatalf("status_code want 0 got %d", resp.StatusCode)
	}
	if resp.Pagination.Total != 2 {
		t.Fatalf("pagination total want 2 got %d", resp.Pagination.Total)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("data len want 2 got %d", len(resp.Data))
	}

	gotIDs := map[uint]struct{}{}
	for _, row := range resp.Data {
		idRaw, ok := row["id"].(float64)
		if !ok {
			t.Fatalf("row id missing or invalid: %+v", row)
		}
		gotIDs[uint(idRaw)] = struct{}{}
	}
	if _, ok := gotIDs[fixture.OrderPaymentID]; !ok {
		t.Fatalf("missing order payment id %d", fixture.OrderPaymentID)
	}
	if _, ok := gotIDs[fixture.RechargePaymentUser1]; !ok {
		t.Fatalf("missing user1 recharge payment id %d", fixture.RechargePaymentUser1)
	}
	if _, ok := gotIDs[fixture.RechargePaymentUser2]; ok {
		t.Fatalf("unexpected user2 recharge payment id %d", fixture.RechargePaymentUser2)
	}
}

func TestExportAdminPaymentsByUserID(t *testing.T) {
	h, db := setupAdminPaymentHandlerTest(t)
	fixture := seedAdminPaymentData(t, db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	url := fmt.Sprintf("/admin/payments/export?user_id=%d", fixture.User1ID)
	c.Request = httptest.NewRequest(http.MethodGet, url, nil)

	h.ExportAdminPayments(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status want 200 got %d", w.Code)
	}
	if contentType := strings.TrimSpace(w.Header().Get("Content-Type")); !strings.HasPrefix(contentType, "text/csv") {
		t.Fatalf("content-type should be csv, got %s", contentType)
	}

	records, err := csv.NewReader(strings.NewReader(w.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("read csv failed: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("csv rows want 3 got %d", len(records))
	}
	header := strings.Join(records[0], ",")
	if header != "id,order_id,recharge_no,recharge_status,recharge_user_id,channel_id,provider_type,channel_type,status,amount,currency,created_at,paid_at,expired_at,provider_ref" {
		t.Fatalf("csv header mismatch, got %s", header)
	}

	foundRechargeNo := false
	for _, row := range records[1:] {
		if len(row) < 3 {
			t.Fatalf("csv row columns too short: %+v", row)
		}
		if row[2] == fixture.RechargeNoUser1 {
			foundRechargeNo = true
		}
		if row[2] == fixture.RechargeNoUser2 {
			t.Fatalf("csv should not include user2 recharge data")
		}
	}
	if !foundRechargeNo {
		t.Fatalf("csv missing user1 recharge row")
	}
}

type responsePaginationAssert struct {
	Page      int   `json:"page"`
	PageSize  int   `json:"page_size"`
	Total     int64 `json:"total"`
	TotalPage int64 `json:"total_page"`
}

func TestParseAdminPaymentQueryUint(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/payments?user_id=12", nil)

	parsed, err := shared.ParseQueryUint(c.Query("user_id"), true)
	if err != nil {
		t.Fatalf("parse user_id failed: %v", err)
	}
	if parsed != 12 {
		t.Fatalf("parsed user_id want 12 got %d", parsed)
	}

	w, c = httptest.NewRecorder(), nil
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/payments?user_id=0", nil)
	_, err = shared.ParseQueryUint(c.Query("user_id"), true)
	if err == nil {
		t.Fatalf("expected parse error for user_id=0")
	}

	w, c = httptest.NewRecorder(), nil
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/payments", nil)
	parsed, err = shared.ParseQueryUint(c.Query("user_id"), true)
	if err != nil {
		t.Fatalf("unexpected error for empty query: %v", err)
	}
	if parsed != 0 {
		t.Fatalf("parsed empty user_id want 0 got %d", parsed)
	}
}

func TestWriteAdminPaymentCSVRows(t *testing.T) {
	h, db := setupAdminPaymentHandlerTest(t)
	fixture := seedAdminPaymentData(t, db)

	payments, _, err := h.PaymentService.ListPayments(repository.PaymentListFilter{
		Page:     1,
		PageSize: 50,
		UserID:   fixture.User1ID,
	})
	if err != nil {
		t.Fatalf("list payments failed: %v", err)
	}

	builder := &strings.Builder{}
	writer := csv.NewWriter(builder)
	if err := h.writeAdminPaymentCSVRows(writer, payments); err != nil {
		t.Fatalf("write csv rows failed: %v", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatalf("flush csv rows failed: %v", err)
	}

	rows, err := csv.NewReader(strings.NewReader(builder.String())).ReadAll()
	if err != nil {
		t.Fatalf("read csv rows failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("csv row count want 2 got %d", len(rows))
	}

	foundRecharge := false
	for _, row := range rows {
		if len(row) < 3 {
			t.Fatalf("row columns too short: %+v", row)
		}
		if row[2] == fixture.RechargeNoUser1 {
			foundRecharge = true
		}
	}
	if !foundRecharge {
		t.Fatalf("csv rows should include user1 recharge info")
	}
}

func TestGetAdminPaymentsBadQueryReturnsBadRequestCode(t *testing.T) {
	h, _ := setupAdminPaymentHandlerTest(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/payments?channel_id=abc", nil)

	h.GetAdminPayments(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status want 200 got %d", w.Code)
	}
	var resp struct {
		StatusCode int `json:"status_code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("status_code want 400 got %d", resp.StatusCode)
	}
}
