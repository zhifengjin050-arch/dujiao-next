package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/shopspring/decimal"
)

// CouponService ?????
type CouponService struct {
	couponRepo repository.CouponRepository
	usageRepo  repository.CouponUsageRepository
}

// NewCouponService ???????
func NewCouponService(couponRepo repository.CouponRepository, usageRepo repository.CouponUsageRepository) *CouponService {
	return &CouponService{
		couponRepo: couponRepo,
		usageRepo:  usageRepo,
	}
}

// ApplyCoupon ?????????
func (s *CouponService) ApplyCoupon(subtotal models.Money, code string, userID uint, items []models.OrderItem, isGuest bool, memberLevelID uint) (models.Money, *models.Coupon, error) {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return models.Money{}, nil, ErrCouponInvalid
	}

	coupon, err := s.couponRepo.GetByCode(trimmed)
	if err != nil {
		return models.Money{}, nil, err
	}
	if coupon == nil {
		return models.Money{}, nil, ErrCouponNotFound
	}
	if !coupon.IsActive {
		return models.Money{}, coupon, ErrCouponInactive
	}

	now := time.Now()
	if coupon.StartsAt != nil && now.Before(*coupon.StartsAt) {
		return models.Money{}, coupon, ErrCouponNotStarted
	}
	if coupon.EndsAt != nil && now.After(*coupon.EndsAt) {
		return models.Money{}, coupon, ErrCouponExpired
	}

	if coupon.UsageLimit > 0 && coupon.UsedCount >= coupon.UsageLimit {
		return models.Money{}, coupon, ErrCouponUsageLimit
	}
	if roleErr := resolveCouponPaymentRoleError(coupon, isGuest); roleErr != nil {
		return models.Money{}, coupon, roleErr
	}
	if !matchesCouponMemberLevel(coupon, memberLevelID) {
		return models.Money{}, coupon, ErrCouponMemberLevelNotAllowed
	}

	if coupon.PerUserLimit > 0 && userID != 0 {
		count, err := s.usageRepo.CountByUser(coupon.ID, userID)
		if err != nil {
			return models.Money{}, coupon, err
		}
		if int(count) >= coupon.PerUserLimit {
			return models.Money{}, coupon, ErrCouponPerUserLimit
		}
	}

	eligibleSubtotal, err := s.resolveEligibleSubtotal(coupon, items)
	if err != nil {
		return models.Money{}, coupon, err
	}

	if eligibleSubtotal.Decimal.Cmp(coupon.MinAmount.Decimal) < 0 {
		return models.Money{}, coupon, ErrCouponMinAmount
	}

	discount, err := s.calculateDiscount(coupon, eligibleSubtotal)
	if err != nil {
		return models.Money{}, coupon, err
	}

	if coupon.MaxDiscount.Decimal.GreaterThan(decimal.Zero) && discount.Decimal.GreaterThan(coupon.MaxDiscount.Decimal) {
		discount = models.NewMoneyFromDecimal(coupon.MaxDiscount.Decimal)
	}

	if discount.Decimal.GreaterThan(eligibleSubtotal.Decimal) {
		discount = models.NewMoneyFromDecimal(eligibleSubtotal.Decimal)
	}

	return discount, coupon, nil
}

// matchesCouponRole ?????????????????????;???????????
func matchesCouponRole(coupon *models.Coupon, isGuest bool) bool {
	if coupon == nil || len(coupon.PaymentRoles) == 0 {
		return true
	}
	targetRole := constants.PaymentRoleMember
	if isGuest {
		targetRole = constants.PaymentRoleGuest
	}
	for _, role := range coupon.PaymentRoles {
		if strings.EqualFold(strings.TrimSpace(role), targetRole) {
			return true
		}
	}
	return false
}

// resolveCouponPaymentRoleError ??????????????????
// ?????????????????????;??????????????
func resolveCouponPaymentRoleError(coupon *models.Coupon, isGuest bool) error {
	if matchesCouponRole(coupon, isGuest) {
		return nil
	}
	if coupon == nil || len(coupon.PaymentRoles) == 0 {
		return ErrCouponPaymentRoleNotAllowed
	}

	roles := make(map[string]struct{}, len(coupon.PaymentRoles))
	for _, role := range coupon.PaymentRoles {
		normalized := strings.ToLower(strings.TrimSpace(role))
		if normalized != constants.PaymentRoleGuest && normalized != constants.PaymentRoleMember {
			continue
		}
		roles[normalized] = struct{}{}
	}

	if len(roles) == 1 {
		if _, ok := roles[constants.PaymentRoleGuest]; ok {
			return ErrCouponPaymentRoleGuestOnly
		}
		if _, ok := roles[constants.PaymentRoleMember]; ok {
			return ErrCouponPaymentRoleMemberOnly
		}
	}
	return ErrCouponPaymentRoleNotAllowed
}

// matchesCouponMemberLevel ?????????????????????;???????????
func matchesCouponMemberLevel(coupon *models.Coupon, memberLevelID uint) bool {
	if coupon == nil || len(coupon.MemberLevels) == 0 {
		return true
	}
	if memberLevelID == 0 {
		return false
	}
	for _, levelID := range coupon.MemberLevels {
		if levelID == memberLevelID {
			return true
		}
	}
	return false
}

func (s *CouponService) resolveEligibleSubtotal(coupon *models.Coupon, items []models.OrderItem) (models.Money, error) {
	if strings.ToLower(strings.TrimSpace(coupon.ScopeType)) != constants.ScopeTypeProduct {
		return models.Money{}, ErrCouponScopeInvalid
	}

	ids, err := decodeScopeIDs(coupon.ScopeRefIDs)
	if err != nil {
		return models.Money{}, ErrCouponScopeInvalid
	}
	if len(ids) == 0 {
		return models.Money{}, ErrCouponScopeInvalid
	}

	eligible := decimal.Zero
	for _, item := range items {
		if _, ok := ids[item.ProductID]; ok {
			eligible = eligible.Add(item.TotalPrice.Decimal)
		}
	}

	if eligible.IsZero() {
		return models.Money{}, ErrCouponScopeInvalid
	}
	return models.NewMoneyFromDecimal(eligible), nil
}

func (s *CouponService) calculateDiscount(coupon *models.Coupon, eligibleSubtotal models.Money) (models.Money, error) {
	switch strings.ToLower(strings.TrimSpace(coupon.Type)) {
	case constants.CouponTypeFixed:
		if coupon.Value.Decimal.LessThanOrEqual(decimal.Zero) {
			return models.Money{}, ErrCouponInvalid
		}
		return models.NewMoneyFromDecimal(coupon.Value.Decimal), nil
	case constants.CouponTypePercent:
		if coupon.Value.Decimal.LessThanOrEqual(decimal.Zero) {
			return models.Money{}, ErrCouponInvalid
		}
		percent := coupon.Value.Decimal.Div(decimal.NewFromInt(100))
		discount := eligibleSubtotal.Decimal.Mul(percent)
		return models.NewMoneyFromDecimal(discount), nil
	default:
		return models.Money{}, ErrCouponInvalid
	}
}

func decodeScopeIDs(raw string) (map[uint]struct{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[uint]struct{}{}, nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(trimmed), &ids); err != nil {
		return nil, err
	}
	result := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		result[id] = struct{}{}
	}
	return result, nil
}
