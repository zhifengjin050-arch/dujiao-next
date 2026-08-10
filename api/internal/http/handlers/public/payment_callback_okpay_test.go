package public

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

type okpayCallbackFixture struct {
	orderRepo   repository.OrderRepository
	paymentRepo repository.PaymentRepository
	handler     *Handler
	order       *models.Order
	payment     *models.Payment
}

func newOkpayCallbackFixture(t *testing.T, exchangeRate string) *okpayCallbackFixture {
	t.Helper()

	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:payment_callback_okpay_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Product{},
		&models.ProductSKU{},
		&models.Order{},
		&models.OrderItem{},
		&models.Fulfillment{},
		&models.PaymentChannel{},
		&models.Payment{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	user := &models.User{
		Email:        "okpay-callback@example.com",
		PasswordHash: "hash",
		Status:       constants.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	order := &models.Order{
		OrderNo:                 "DJOKPAYCALLBACK001",
		UserID:                  user.ID,
		Status:                  constants.OrderStatusPendingPayment,
		Currency:                "CNY",
		OriginalAmount:          models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		DiscountAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		PromotionDiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:             models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		WalletPaidAmount:        models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount:        models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		RefundedAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create order failed: %v", err)
	}
	channel := &models.PaymentChannel{
		Name:            "OKPAY",
		ProviderType:    constants.PaymentProviderOkpay,
		ChannelType:     constants.PaymentChannelTypeUsdt,
		InteractionMode: constants.PaymentInteractionQR,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		ConfigJSON: models.JSON{
			"merchant_id":    "shop-1",
			"merchant_token": "token-1",
			"return_url":     "https://example.com/pay",
			"callback_url":   "https://api.example.com/api/v1/payments/callback",
			"exchange_rate":  exchangeRate,
		},
		IsActive:  true,
		SortOrder: 10,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(channel).Error; err != nil {
		t.Fatalf("create channel failed: %v", err)
	}
	payment := &models.Payment{
		OrderID:         order.ID,
		ChannelID:       channel.ID,
		ProviderType:    channel.ProviderType,
		ChannelType:     channel.ChannelType,
		InteractionMode: channel.InteractionMode,
		Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.PaymentStatusPending,
		ProviderRef:     "OKPAY-ORDER-1",
		GatewayOrderNo:  "DJP9001",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("create payment failed: %v", err)
	}

	orderRepo := repository.NewOrderRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	channelRepo := repository.NewPaymentChannelRepository(db)
	productRepo := repository.NewProductRepository(db)
	productSKURepo := repository.NewProductSKURepository(db)
	paymentService := service.NewPaymentService(service.PaymentServiceOptions{
		OrderRepo:      orderRepo,
		ProductRepo:    productRepo,
		ProductSKURepo: productSKURepo,
		PaymentRepo:    paymentRepo,
		ChannelRepo:    channelRepo,
		ExpireMinutes:  15,
	})

	return &okpayCallbackFixture{
		orderRepo:   orderRepo,
		paymentRepo: paymentRepo,
		handler: &Handler{Container: &provider.Container{
			OrderRepo:          orderRepo,
			PaymentRepo:        paymentRepo,
			PaymentChannelRepo: channelRepo,
			PaymentService:     paymentService,
		}},
		order:   order,
		payment: payment,
	}
}

func TestPaymentCallbackHandlesOkpay(t *testing.T) {
	fixture := newOkpayCallbackFixture(t, "7")
	bodyWithoutSign := "code=200&data[order_id]=OKPAY-ORDER-1&data[unique_id]=DJP9001&data[pay_user_id]=7238234930&data[amount]=616.00000000&data[coin]=USDT&data[status]=1&data[type]=deposit&id=shop-1&status=success"
	sign := md5HexUpper(bodyWithoutSign + "&token=token-1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/callback", strings.NewReader(bodyWithoutSign+"&sign="+sign))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	fixture.handler.PaymentCallback(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if strings.TrimSpace(w.Body.String()) != constants.OkpayCallbackSuccess {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}

	updatedPayment, err := fixture.paymentRepo.GetByID(fixture.payment.ID)
	if err != nil {
		t.Fatalf("reload payment failed: %v", err)
	}
	if updatedPayment == nil || updatedPayment.Status != constants.PaymentStatusSuccess {
		t.Fatalf("payment status not updated: %+v", updatedPayment)
	}
	updatedOrder, err := fixture.orderRepo.GetByID(fixture.order.ID)
	if err != nil {
		t.Fatalf("reload order failed: %v", err)
	}
	if updatedOrder == nil || updatedOrder.Status != constants.OrderStatusPaid {
		t.Fatalf("order status not updated: %+v", updatedOrder)
	}
}

func TestPaymentCallbackRejectsOkpayAmountMismatch(t *testing.T) {
	fixture := newOkpayCallbackFixture(t, "7")
	bodyWithoutSign := "code=200&data[order_id]=OKPAY-ORDER-1&data[unique_id]=DJP9001&data[pay_user_id]=7238234930&data[amount]=615.00000000&data[coin]=USDT&data[status]=1&data[type]=deposit&id=shop-1&status=success"
	sign := md5HexUpper(bodyWithoutSign + "&token=token-1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/callback", strings.NewReader(bodyWithoutSign+"&sign="+sign))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	fixture.handler.PaymentCallback(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if strings.TrimSpace(w.Body.String()) != constants.OkpayCallbackFail {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}

	updatedPayment, err := fixture.paymentRepo.GetByID(fixture.payment.ID)
	if err != nil {
		t.Fatalf("reload payment failed: %v", err)
	}
	if updatedPayment == nil || updatedPayment.Status != constants.PaymentStatusPending {
		t.Fatalf("payment status should stay pending: %+v", updatedPayment)
	}

	updatedOrder, err := fixture.orderRepo.GetByID(fixture.order.ID)
	if err != nil {
		t.Fatalf("reload order failed: %v", err)
	}
	if updatedOrder == nil || updatedOrder.Status != constants.OrderStatusPendingPayment {
		t.Fatalf("order status should stay pending payment: %+v", updatedOrder)
	}
}

func md5HexUpper(raw string) string {
	sum := md5.Sum([]byte(raw))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}
