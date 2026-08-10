package admin

import (
	"errors"
	"strconv"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// CreatePaymentChannelRequest ????????
type CreatePaymentChannelRequest struct {
	Name               string                 `json:"name" binding:"required"`
	Icon               *string                `json:"icon"`
	ProviderType       string                 `json:"provider_type" binding:"required"`
	ChannelType        string                 `json:"channel_type" binding:"required"`
	InteractionMode    string                 `json:"interaction_mode" binding:"required"`
	FeeRate            *models.Money          `json:"fee_rate"`
	FixedFee           *models.Money          `json:"fixed_fee"`
	MinAmount          *models.Money          `json:"min_amount"`
	MaxAmount          *models.Money          `json:"max_amount"`
	HideAmountOutRange *bool                  `json:"hide_amount_out_range"`
	PaymentRoles       []string               `json:"payment_roles"`
	MemberLevels       []uint                 `json:"member_levels"`
	PaymentTypes       []string               `json:"payment_types"`
	ConfigJSON         map[string]interface{} `json:"config_json"`
	IsActive           *bool                  `json:"is_active"`
	SortOrder          int                    `json:"sort_order"`
}

// CreatePaymentChannel ??????
func (h *Handler) CreatePaymentChannel(c *gin.Context) {
	var req CreatePaymentChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	channel := &models.PaymentChannel{
		Name:            req.Name,
		ProviderType:    req.ProviderType,
		ChannelType:     req.ChannelType,
		InteractionMode: req.InteractionMode,
		ConfigJSON:      models.JSON(req.ConfigJSON),
		PaymentRoles:    req.PaymentRoles,
		MemberLevels:    req.MemberLevels,
		PaymentTypes:    req.PaymentTypes,
		SortOrder:       req.SortOrder,
		IsActive:        true,
	}
	if req.Icon != nil {
		channel.Icon = *req.Icon
	}
	if req.IsActive != nil {
		channel.IsActive = *req.IsActive
	}
	if req.HideAmountOutRange != nil {
		channel.HideAmountOutRange = *req.HideAmountOutRange
	}
	if req.FeeRate != nil {
		channel.FeeRate = *req.FeeRate
	}
	if req.FixedFee != nil {
		channel.FixedFee = *req.FixedFee
	}
	if req.MinAmount != nil {
		channel.MinAmount = *req.MinAmount
	}
	if req.MaxAmount != nil {
		channel.MaxAmount = *req.MaxAmount
	}

	if err := h.PaymentService.ValidateChannel(channel); err != nil {
		switch {
		case errors.Is(err, service.ErrPaymentProviderNotSupported):
			shared.RespondError(c, response.CodeBadRequest, "error.payment_provider_not_supported", nil)
		case errors.Is(err, service.ErrPaymentChannelConfigInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.payment_channel_config_invalid", nil)
		default:
			shared.RespondError(c, response.CodeBadRequest, "error.payment_channel_invalid", nil)
		}
		return
	}

	if err := h.PaymentChannelRepo.Create(channel); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_channel_create_failed", err)
		return
	}
	_ = cache.Del(c.Request.Context(), publicConfigCacheKey)

	response.Success(c, channel)
}

// UpdatePaymentChannelRequest ????????
type UpdatePaymentChannelRequest struct {
	Name               string                 `json:"name"`
	Icon               *string                `json:"icon"`
	ProviderType       string                 `json:"provider_type"`
	ChannelType        string                 `json:"channel_type"`
	InteractionMode    string                 `json:"interaction_mode"`
	FeeRate            *models.Money          `json:"fee_rate"`
	FixedFee           *models.Money          `json:"fixed_fee"`
	MinAmount          *models.Money          `json:"min_amount"`
	MaxAmount          *models.Money          `json:"max_amount"`
	HideAmountOutRange *bool                  `json:"hide_amount_out_range"`
	PaymentRoles       []string               `json:"payment_roles"`
	MemberLevels       []uint                 `json:"member_levels"`
	PaymentTypes       []string               `json:"payment_types"`
	ConfigJSON         map[string]interface{} `json:"config_json"`
	IsActive           *bool                  `json:"is_active"`
	SortOrder          *int                   `json:"sort_order"`
}

// UpdatePaymentChannel ??????
func (h *Handler) UpdatePaymentChannel(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.payment_channel_invalid", nil)
		return
	}

	var req UpdatePaymentChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	channel, err := h.PaymentService.GetChannel(id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPaymentChannelNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.payment_channel_not_found", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.payment_channel_update_failed", err)
		}
		return
	}

	if req.Name != "" {
		channel.Name = req.Name
	}
	if req.Icon != nil {
		channel.Icon = *req.Icon
	}
	if req.ProviderType != "" {
		channel.ProviderType = req.ProviderType
	}
	if req.ChannelType != "" {
		channel.ChannelType = req.ChannelType
	}
	if req.InteractionMode != "" {
		channel.InteractionMode = req.InteractionMode
	}
	if req.FeeRate != nil {
		channel.FeeRate = *req.FeeRate
	}
	if req.FixedFee != nil {
		channel.FixedFee = *req.FixedFee
	}
	if req.MinAmount != nil {
		channel.MinAmount = *req.MinAmount
	}
	if req.MaxAmount != nil {
		channel.MaxAmount = *req.MaxAmount
	}
	if req.PaymentRoles != nil {
		channel.PaymentRoles = req.PaymentRoles
	}
	if req.MemberLevels != nil {
		channel.MemberLevels = req.MemberLevels
	}
	if req.PaymentTypes != nil {
		channel.PaymentTypes = req.PaymentTypes
	}
	if req.ConfigJSON != nil {
		channel.ConfigJSON = models.JSON(req.ConfigJSON)
	}
	if req.IsActive != nil {
		channel.IsActive = *req.IsActive
	}
	if req.HideAmountOutRange != nil {
		channel.HideAmountOutRange = *req.HideAmountOutRange
	}
	if req.SortOrder != nil {
		channel.SortOrder = *req.SortOrder
	}

	if err := h.PaymentService.ValidateChannel(channel); err != nil {
		switch {
		case errors.Is(err, service.ErrPaymentProviderNotSupported):
			shared.RespondError(c, response.CodeBadRequest, "error.payment_provider_not_supported", nil)
		case errors.Is(err, service.ErrPaymentChannelConfigInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.payment_channel_config_invalid", nil)
		default:
			shared.RespondError(c, response.CodeBadRequest, "error.payment_channel_invalid", nil)
		}
		return
	}

	if err := h.PaymentChannelRepo.Update(channel); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_channel_update_failed", err)
		return
	}
	_ = cache.Del(c.Request.Context(), publicConfigCacheKey)

	response.Success(c, channel)
}

// DeletePaymentChannel ??????
func (h *Handler) DeletePaymentChannel(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.payment_channel_invalid", nil)
		return
	}

	if err := h.PaymentChannelRepo.Delete(id); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_channel_delete_failed", err)
		return
	}
	_ = cache.Del(c.Request.Context(), publicConfigCacheKey)

	response.Success(c, gin.H{"deleted": true})
}

// GetPaymentChannel ????????
func (h *Handler) GetPaymentChannel(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.payment_channel_invalid", nil)
		return
	}

	channel, err := h.PaymentService.GetChannel(id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPaymentChannelNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.payment_channel_not_found", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.payment_channel_fetch_failed", err)
		}
		return
	}

	response.Success(c, channel)
}

// GetPaymentChannels ????????
func (h *Handler) GetPaymentChannels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	providerType := c.Query("provider_type")
	channelType := c.Query("channel_type")
	activeOnly := c.DefaultQuery("active_only", "")
	activeOnlyBool := false
	if activeOnly != "" {
		parsed, err := strconv.ParseBool(activeOnly)
		if err != nil {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
			return
		}
		activeOnlyBool = parsed
	}

	channels, total, err := h.PaymentService.ListChannels(repository.PaymentChannelListFilter{
		Page:         page,
		PageSize:     pageSize,
		ProviderType: providerType,
		ChannelType:  channelType,
		ActiveOnly:   activeOnlyBool,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.payment_channel_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, channels, pagination)
}
