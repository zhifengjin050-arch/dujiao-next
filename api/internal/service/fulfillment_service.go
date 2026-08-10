package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/queue"
	"github.com/dujiao-next/internal/repository"

	"gorm.io/gorm"
)

// FulfillmentService ????
type FulfillmentService struct {
	orderRepo             repository.OrderRepository
	fulfillmentRepo       repository.FulfillmentRepository
	secretRepo            repository.CardSecretRepository
	queueClient           *queue.Client
	settingService        *SettingService
	defaultEmailConfig    config.EmailConfig
	downstreamCallbackSvc *DownstreamCallbackService
	userOAuthIdentityRepo repository.UserOAuthIdentityRepository
}

// SetDownstreamCallbackService ????????(??????)
func (s *FulfillmentService) SetDownstreamCallbackService(svc *DownstreamCallbackService) {
	s.downstreamCallbackSvc = svc
}

// NewFulfillmentService ??????
func NewFulfillmentService(
	orderRepo repository.OrderRepository,
	fulfillmentRepo repository.FulfillmentRepository,
	secretRepo repository.CardSecretRepository,
	queueClient *queue.Client,
	settingService *SettingService,
	defaultEmailConfig config.EmailConfig,
	userOAuthIdentityRepo repository.UserOAuthIdentityRepository,
) *FulfillmentService {
	return &FulfillmentService{
		orderRepo:             orderRepo,
		fulfillmentRepo:       fulfillmentRepo,
		secretRepo:            secretRepo,
		queueClient:           queueClient,
		settingService:        settingService,
		defaultEmailConfig:    defaultEmailConfig,
		userOAuthIdentityRepo: userOAuthIdentityRepo,
	}
}

// CreateManualInput ????????
type CreateManualInput struct {
	OrderID      uint
	AdminID      uint
	Payload      string
	DeliveryData models.JSON
	DeliveredAt  *time.Time
}

// CreateManual ??????
func (s *FulfillmentService) CreateManual(input CreateManualInput) (*models.Fulfillment, error) {
	if input.OrderID == 0 || input.AdminID == 0 {
		return nil, ErrFulfillmentInvalid
	}
	payload := strings.TrimSpace(input.Payload)
	deliveryData := normalizeManualDeliveryData(input.DeliveryData)
	if payload == "" && len(deliveryData) == 0 {
		return nil, ErrFulfillmentInvalid
	}
	if payload == "" {
		payload = buildManualDeliveryPayload(deliveryData)
	}
	ftype := constants.FulfillmentTypeManual

	order, err := s.orderRepo.GetByID(input.OrderID)
	if err != nil {
		return nil, ErrOrderFetchFailed
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if order.ParentID == nil && len(order.Children) > 0 {
		return nil, ErrFulfillmentInvalid
	}
	if order.Status != constants.OrderStatusPaid && order.Status != constants.OrderStatusFulfilling {
		return nil, ErrOrderStatusInvalid
	}

	now := time.Now()
	deliveredAt := input.DeliveredAt
	if deliveredAt == nil {
		deliveredAt = &now
	}

	var created *models.Fulfillment
	err = s.orderRepo.Transaction(func(tx *gorm.DB) error {
		var existing models.Fulfillment
		if err := tx.Where("order_id = ?", input.OrderID).First(&existing).Error; err == nil {
			return ErrFulfillmentExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		fulfillment := &models.Fulfillment{
			OrderID:       input.OrderID,
			Type:          ftype,
			Status:        constants.FulfillmentStatusDelivered,
			Payload:       payload,
			LogisticsJSON: deliveryData,
			DeliveredBy:   &input.AdminID,
			DeliveredAt:   deliveredAt,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := tx.Create(fulfillment).Error; err != nil {
			return ErrFulfillmentCreateFailed
		}
		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
			"status":     constants.OrderStatusDelivered,
			"updated_at": now,
		}).Error; err != nil {
			return ErrOrderUpdateFailed
		}
		created = fulfillment
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrFulfillmentExists) {
			return nil, ErrFulfillmentExists
		}
		if errors.Is(err, ErrOrderUpdateFailed) {
			return nil, ErrOrderUpdateFailed
		}
		return nil, ErrFulfillmentCreateFailed
	}
	if s.queueClient != nil {
		if order.ParentID != nil {
			status, syncErr := syncParentStatus(s.orderRepo, *order.ParentID, now)
			if syncErr != nil {
				logger.Warnw("fulfillment_sync_parent_status_failed",
					"order_id", order.ID,
					"parent_order_id", *order.ParentID,
					"target_status", constants.OrderStatusDelivered,
					"error", syncErr,
				)
			} else {
				if status == "" {
					status = constants.OrderStatusDelivered
				}
				if _, err := enqueueOrderStatusEmailTaskIfEligible(s.orderRepo, s.queueClient, s.settingService, s.defaultEmailConfig, *order.ParentID, status); err != nil {
					logger.Warnw("fulfillment_enqueue_status_email_failed",
						"order_id", order.ID,
						"target_order_id", *order.ParentID,
						"status", status,
						"error", err,
					)
				}
			}
		} else {
			if _, err := enqueueOrderStatusEmailTaskIfEligible(s.orderRepo, s.queueClient, s.settingService, s.defaultEmailConfig, input.OrderID, constants.OrderStatusDelivered); err != nil {
				logger.Warnw("fulfillment_enqueue_status_email_failed",
					"order_id", order.ID,
					"target_order_id", input.OrderID,
					"status", constants.OrderStatusDelivered,
					"error", err,
				)
			}
		}
	}
	// Telegram ??:??????????
	notifyOrderID := input.OrderID
	if order.ParentID != nil {
		notifyOrderID = *order.ParentID
	}
	go s.NotifyBotOrderFulfilled(order.UserID, notifyOrderID)
	// B ?:?????????????
	if s.downstreamCallbackSvc != nil {
		s.downstreamCallbackSvc.EnqueueCallback(input.OrderID)
	}
	return created, nil
}

// CreateAuto ????
func (s *FulfillmentService) CreateAuto(orderID uint) (*models.Fulfillment, error) {
	if orderID == 0 {
		return nil, ErrFulfillmentInvalid
	}

	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, ErrOrderFetchFailed
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if order.ParentID == nil && len(order.Children) > 0 {
		return nil, ErrFulfillmentInvalid
	}
	if order.Status != constants.OrderStatusPaid {
		return nil, ErrOrderStatusInvalid
	}
	if len(order.Items) == 0 {
		return nil, ErrFulfillmentInvalid
	}

	for _, item := range order.Items {
		if strings.TrimSpace(item.FulfillmentType) != constants.FulfillmentTypeAuto {
			return nil, ErrFulfillmentNotAuto
		}
	}

	now := time.Now()
	var fulfillment *models.Fulfillment
	err = s.orderRepo.Transaction(func(tx *gorm.DB) error {
		var existing models.Fulfillment
		if err := tx.Where("order_id = ?", orderID).First(&existing).Error; err == nil {
			return ErrFulfillmentExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if s.secretRepo == nil {
			return ErrFulfillmentCreateFailed
		}
		secretRepo := s.secretRepo.WithTx(tx)
		reservedRows, err := secretRepo.ListByOrderAndStatus(orderID, models.CardSecretStatusReserved)
		if err != nil {
			return err
		}
		reservedByKey := make(map[string][]models.CardSecret)
		for _, reserved := range reservedRows {
			key := buildOrderItemKey(reserved.ProductID, reserved.SKUID)
			reservedByKey[key] = append(reservedByKey[key], reserved)
		}
		var secrets []models.CardSecret
		for _, item := range order.Items {
			if item.ProductID == 0 || item.Quantity <= 0 {
				return ErrFulfillmentInvalid
			}
			key := buildOrderItemKey(item.ProductID, item.SKUID)
			cachedReserved := reservedByKey[key]
			selected := make([]models.CardSecret, 0, item.Quantity)
			if len(cachedReserved) > 0 {
				take := item.Quantity
				if len(cachedReserved) < take {
					take = len(cachedReserved)
				}
				selected = append(selected, cachedReserved[:take]...)
				reservedByKey[key] = cachedReserved[take:]
			}

			if len(selected) < item.Quantity {
				need := item.Quantity - len(selected)
				var availableRows []models.CardSecret
				query := tx.Where("product_id = ? AND status = ?", item.ProductID, models.CardSecretStatusAvailable)
				if item.SKUID > 0 {
					query = query.Where("sku_id = ?", item.SKUID)
				}
				if err := query.Order("id asc").Limit(need).Find(&availableRows).Error; err != nil {
					return err
				}
				selected = append(selected, availableRows...)
			}
			if len(selected) < item.Quantity {
				return ErrCardSecretInsufficient
			}
			secrets = append(secrets, selected...)
		}

		ids := make([]uint, 0, len(secrets))
		secretLines := make([]string, 0, len(secrets))
		for _, secret := range secrets {
			ids = append(ids, secret.ID)
			secretLines = append(secretLines, secret.Secret)
		}

		affected, err := secretRepo.MarkUsed(ids, orderID, now)
		if err != nil {
			return err
		}
		if int(affected) != len(ids) {
			logger.Warnw("stock_deduct_partial",
				"order_id", orderID,
				"expected", len(ids),
				"affected", affected,
			)
			return ErrCardSecretInsufficient
		}
		logger.Infow("stock_deducted",
			"order_id", orderID,
			"count", affected,
			"card_secret_ids", ids,
		)

		payload := strings.Join(secretLines, "\n")
		fulfillment = &models.Fulfillment{
			OrderID:     orderID,
			Type:        constants.FulfillmentTypeAuto,
			Status:      constants.FulfillmentStatusDelivered,
			Payload:     payload,
			DeliveredAt: &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Create(fulfillment).Error; err != nil {
			return ErrFulfillmentCreateFailed
		}
		if err := tx.Model(&models.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
			"status":     constants.OrderStatusCompleted,
			"updated_at": now,
		}).Error; err != nil {
			return ErrOrderUpdateFailed
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrFulfillmentExists):
			return nil, ErrFulfillmentExists
		case errors.Is(err, ErrCardSecretInsufficient):
			return nil, ErrCardSecretInsufficient
		case errors.Is(err, ErrOrderUpdateFailed):
			return nil, ErrOrderUpdateFailed
		case errors.Is(err, ErrFulfillmentNotAuto):
			return nil, ErrFulfillmentNotAuto
		default:
			return nil, ErrFulfillmentCreateFailed
		}
	}
	if s.queueClient != nil {
		if order.ParentID != nil {
			status, syncErr := syncParentStatus(s.orderRepo, *order.ParentID, now)
			if syncErr != nil {
				logger.Warnw("fulfillment_sync_parent_status_failed",
					"order_id", order.ID,
					"parent_order_id", *order.ParentID,
					"target_status", constants.OrderStatusCompleted,
					"error", syncErr,
				)
			} else {
				if status == "" {
					status = constants.OrderStatusCompleted
				}
				if _, err := enqueueOrderStatusEmailTaskIfEligible(s.orderRepo, s.queueClient, s.settingService, s.defaultEmailConfig, *order.ParentID, status); err != nil {
					logger.Warnw("fulfillment_enqueue_status_email_failed",
						"order_id", order.ID,
						"target_order_id", *order.ParentID,
						"status", status,
						"error", err,
					)
				}
			}
		} else {
			if _, err := enqueueOrderStatusEmailTaskIfEligible(s.orderRepo, s.queueClient, s.settingService, s.defaultEmailConfig, orderID, constants.OrderStatusCompleted); err != nil {
				logger.Warnw("fulfillment_enqueue_status_email_failed",
					"order_id", order.ID,
					"target_order_id", orderID,
					"status", constants.OrderStatusCompleted,
					"error", err,
				)
			}
		}
	}
	// Telegram ??:??????????
	notifyOrderID := orderID
	if order.ParentID != nil {
		notifyOrderID = *order.ParentID
	}
	go s.NotifyBotOrderFulfilled(order.UserID, notifyOrderID)
	// B ?:?????????????
	if s.downstreamCallbackSvc != nil {
		s.downstreamCallbackSvc.EnqueueCallback(orderID)
	}
	return fulfillment, nil
}

// NotifyBotOrderFulfilled ???? Telegram ????? asynq ???? Bot
func (s *FulfillmentService) NotifyBotOrderFulfilled(userID, orderID uint) {
	if s.queueClient == nil || userID == 0 || s.userOAuthIdentityRepo == nil {
		return
	}

	identity, err := s.userOAuthIdentityRepo.GetByUserProvider(userID, "telegram")
	if err != nil {
		logger.Warnw("fulfillment_notify_bot_fetch_identity_failed",
			"order_id", orderID, "user_id", userID, "error", err)
		return
	}
	if identity == nil || strings.TrimSpace(identity.ProviderUserID) == "" {
		return
	}

	if err := s.queueClient.EnqueueBotNotify(queue.BotNotifyPayload{
		EventType:      queue.BotNotifyEventOrderFulfilled,
		OrderID:        orderID,
		TelegramUserID: strings.TrimSpace(identity.ProviderUserID),
	}); err != nil {
		logger.Warnw("fulfillment_notify_bot_enqueue_failed",
			"order_id", orderID, "user_id", userID, "error", err)
	}
}

func normalizeManualDeliveryData(raw models.JSON) models.JSON {
	if len(raw) == 0 {
		return models.JSON{}
	}
	normalized := models.JSON{}
	note := strings.TrimSpace(toStringValue(raw["note"]))
	if note != "" {
		normalized["note"] = note
	}
	if entriesRaw, ok := raw["entries"]; ok {
		entries := normalizeManualDeliveryEntries(entriesRaw)
		if len(entries) > 0 {
			normalized["entries"] = entries
		}
	}

	keys := make([]string, 0, len(raw))
	for key := range raw {
		if key == "note" || key == "entries" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := normalizeManualDeliveryPrimitive(raw[key])
		if value == nil {
			continue
		}
		normalized[key] = value
	}

	return normalized
}

func normalizeManualDeliveryEntries(raw interface{}) []models.JSON {
	appendEntry := func(entries []models.JSON, row map[string]interface{}) []models.JSON {
		key := strings.TrimSpace(toStringValue(row["key"]))
		value := strings.TrimSpace(toStringValue(row["value"]))
		if key == "" && value == "" {
			return entries
		}
		entry := models.JSON{}
		if key != "" {
			entry["key"] = key
		}
		if value != "" {
			entry["value"] = value
		}
		return append(entries, entry)
	}

	entries := make([]models.JSON, 0)
	switch value := raw.(type) {
	case []models.JSON:
		for _, item := range value {
			entries = appendEntry(entries, item)
		}
		return entries
	case []map[string]interface{}:
		for _, item := range value {
			entries = appendEntry(entries, item)
		}
		return entries
	case []interface{}:
		for _, item := range value {
			row, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			entries = appendEntry(entries, row)
		}
		return entries
	default:
		return nil
	}
}

func normalizeManualDeliveryPrimitive(raw interface{}) interface{} {
	switch value := raw.(type) {
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return nil
		}
		return text
	case bool:
		return value
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return value
	default:
		return nil
	}
}

func buildManualDeliveryPayload(data models.JSON) string {
	if len(data) == 0 {
		return ""
	}
	lines := make([]string, 0, 8)
	if note, ok := data["note"].(string); ok && strings.TrimSpace(note) != "" {
		lines = append(lines, note)
	}
	if entries, ok := data["entries"].([]models.JSON); ok {
		for _, item := range entries {
			key := strings.TrimSpace(toStringValue(item["key"]))
			value := strings.TrimSpace(toStringValue(item["value"]))
			if key == "" && value == "" {
				continue
			}
			if key == "" {
				lines = append(lines, value)
				continue
			}
			if value == "" {
				lines = append(lines, key)
				continue
			}
			lines = append(lines, fmt.Sprintf("%s: %s", key, value))
		}
	}

	extraKeys := make([]string, 0, len(data))
	for key := range data {
		if key == "note" || key == "entries" {
			continue
		}
		extraKeys = append(extraKeys, key)
	}
	sort.Strings(extraKeys)
	for _, key := range extraKeys {
		valueText := strings.TrimSpace(toStringValue(data[key]))
		if valueText == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", key, valueText))
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func toStringValue(raw interface{}) string {
	switch value := raw.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, bool:
		return fmt.Sprintf("%v", value)
	default:
		return ""
	}
}
