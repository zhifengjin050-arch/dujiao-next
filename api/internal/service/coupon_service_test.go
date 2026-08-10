package service

import (
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

func newCouponServiceForTest(t *testing.T) (*CouponService, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:coupon_service_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Coupon{}, &models.CouponUsage{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	couponRepo := repository.NewCouponRepository(db)
	usageRepo := repository.NewCouponUsageRepository(db)
	return NewCouponService(couponRepo, usageRepo), db
}

func createCouponFixture(t *testing.T, db *gorm.DB, coupon models.Coupon) models.Coupon {
	t.Helper()
	if err := db.Create(&coupon).Error; err != nil {
		t.Fatalf("create coupon fixture failed: %v", err)
	}
	return coupon
}

func TestCouponServiceApplyCoupon_RespectsPaymentRoleAndMemberLevel(t *testing.T) {
	svc, db := newCouponServiceForTest(t)
	now := time.Now()
	items := []models.OrderItem{
		{
			ProductID:  100,
			Quantity:   1,
			TotalPrice: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		},
	}
	subtotal := models.NewMoneyFromDecimal(decimal.NewFromInt(100))

	testCases := []struct {
		name          string
		code          string
		roles         models.StringArray
		memberLevels  models.UintArray
		isGuest       bool
		memberLevelID uint
		expectErr     error
	}{
		{
			name:          "no restrictions allows guest",
			code:          "NO_LIMIT",
			isGuest:       true,
			memberLevelID: 0,
		},
		{
			name:          "member-only coupon blocks guest",
			code:          "MEMBER_ONLY",
			roles:         models.StringArray{constants.PaymentRoleMember},
			isGuest:       true,
			memberLevelID: 0,
			expectErr:     ErrCouponPaymentRoleMemberOnly,
		},
		{
			name:          "guest-only coupon blocks member",
			code:          "GUEST_ONLY",
			roles:         models.StringArray{constants.PaymentRoleGuest},
			isGuest:       false,
			memberLevelID: 1,
			expectErr:     ErrCouponPaymentRoleGuestOnly,
		},
		{
			name:          "member-level limited coupon blocks other levels",
			code:          "VIP2_ONLY",
			memberLevels:  models.UintArray{2},
			isGuest:       false,
			memberLevelID: 1,
			expectErr:     ErrCouponMemberLevelNotAllowed,
		},
		{
			name:          "member-level limited coupon allows matching level",
			code:          "VIP3_OK",
			memberLevels:  models.UintArray{3},
			isGuest:       false,
			memberLevelID: 3,
		},
		{
			name:          "combined restrictions allow matching member",
			code:          "MEMBER_VIP5",
			roles:         models.StringArray{constants.PaymentRoleMember},
			memberLevels:  models.UintArray{5},
			isGuest:       false,
			memberLevelID: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_ = createCouponFixture(t, db, models.Coupon{
				Code:         tc.code,
				Type:         constants.CouponTypeFixed,
				Value:        models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
				MinAmount:    models.NewMoneyFromDecimal(decimal.Zero),
				MaxDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
				ScopeType:    constants.ScopeTypeProduct,
				ScopeRefIDs:  "[100]",
				IsActive:     true,
				PaymentRoles: tc.roles,
				MemberLevels: tc.memberLevels,
				StartsAt:     &now,
			})

			_, _, err := svc.ApplyCoupon(subtotal, tc.code, 0, items, tc.isGuest, tc.memberLevelID)
			if tc.expectErr == nil && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if tc.expectErr != nil && err != tc.expectErr {
				t.Fatalf("expected %v, got %v", tc.expectErr, err)
			}
		})
	}
}
