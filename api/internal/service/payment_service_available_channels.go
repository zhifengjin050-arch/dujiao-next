package service

import (
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
)

// AvailablePaymentChannelFilter ??????????
type AvailablePaymentChannelFilter struct {
	TargetAmount *models.Money
	User         *models.User
	PaymentType  string
}

// GetAvailableChannels ????????????
func (s *PaymentService) GetAvailableChannels(filter AvailablePaymentChannelFilter) ([]map[string]interface{}, error) {
	channels, _, err := s.ListChannels(repository.PaymentChannelListFilter{
		Page:       1,
		PageSize:   200,
		ActiveOnly: true,
	})
	if err != nil {
		return nil, err
	}

	availableChannels := make([]map[string]interface{}, 0, len(channels))
	for _, channel := range channels {
		if !matchesChannelAmount(channel, filter.TargetAmount) {
			continue
		}
		if !matchesChannelRole(channel, filter.User) {
			continue
		}
		if !matchesChannelMemberLevel(channel, filter.User) {
			continue
		}
		if !matchesChannelPaymentType(channel, filter.PaymentType) {
			continue
		}

		ch := map[string]interface{}{
			"id":                    channel.ID,
			"name":                  channel.Name,
			"provider_type":         channel.ProviderType,
			"channel_type":          channel.ChannelType,
			"interaction_mode":      channel.InteractionMode,
			"fee_rate":              channel.FeeRate,
			"fixed_fee":             channel.FixedFee,
			"min_amount":            channel.MinAmount,
			"max_amount":            channel.MaxAmount,
			"hide_amount_out_range": channel.HideAmountOutRange,
		}
		if channel.Icon != "" {
			ch["icon"] = channel.Icon
		}
		availableChannels = append(availableChannels, ch)
	}

	return availableChannels, nil
}

func matchesChannelAmount(channel models.PaymentChannel, targetAmount *models.Money) bool {
	if targetAmount == nil || !channel.HideAmountOutRange {
		return true
	}
	minAmt := channel.MinAmount.Decimal
	maxAmt := channel.MaxAmount.Decimal
	amtDec := targetAmount.Decimal
	if minAmt.IsPositive() && amtDec.LessThan(minAmt) {
		return false
	}
	if maxAmt.IsPositive() && amtDec.GreaterThan(maxAmt) {
		return false
	}
	return true
}

func matchesChannelRole(channel models.PaymentChannel, user *models.User) bool {
	if len(channel.PaymentRoles) == 0 {
		return true
	}
	targetRole := constants.PaymentRoleGuest
	if user != nil {
		targetRole = constants.PaymentRoleMember
	}
	for _, role := range channel.PaymentRoles {
		if role == targetRole {
			return true
		}
	}
	return false
}

func matchesChannelMemberLevel(channel models.PaymentChannel, user *models.User) bool {
	if len(channel.MemberLevels) == 0 {
		return true
	}
	if user == nil || user.MemberLevelID == 0 {
		return false
	}
	for _, levelID := range channel.MemberLevels {
		if levelID == user.MemberLevelID {
			return true
		}
	}
	return false
}

func matchesChannelPaymentType(channel models.PaymentChannel, paymentType string) bool {
	if len(channel.PaymentTypes) == 0 || paymentType == "" {
		return true
	}
	for _, pt := range channel.PaymentTypes {
		if pt == paymentType {
			return true
		}
	}
	return false
}
