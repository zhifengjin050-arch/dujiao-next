package service

import (
	"strings"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/queue"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/telegramidentity"
)

// enqueueOrderStatusEmailTaskIfEligible ???????????????????????
// ??? skipped ?????????(?? Telegram ????)?
func enqueueOrderStatusEmailTaskIfEligible(
	orderRepo repository.OrderRepository,
	queueClient *queue.Client,
	settingService *SettingService,
	defaultEmailConfig config.EmailConfig,
	orderID uint,
	status string,
) (skipped bool, err error) {
	if queueClient == nil || orderID == 0 {
		return true, nil
	}
	if settingService != nil {
		smtpSetting, smtpErr := settingService.GetSMTPSetting(defaultEmailConfig)
		if smtpErr != nil {
			return false, smtpErr
		}
		if !smtpSetting.Enabled {
			return true, nil
		}
		if !smtpSetting.OrderNotificationEnabled {
			return true, nil
		}
	}
	if orderRepo == nil {
		if err := queueClient.EnqueueOrderStatusEmail(queue.OrderStatusEmailPayload{
			OrderID: orderID,
			Status:  strings.TrimSpace(status),
		}); err != nil {
			return false, err
		}
		return false, nil
	}

	receiverEmail, lookupErr := orderRepo.ResolveReceiverEmailByOrderID(orderID)
	if lookupErr == nil {
		receiverEmail = strings.TrimSpace(receiverEmail)
		if receiverEmail == "" {
			return true, nil
		}
		if telegramidentity.IsPlaceholderEmail(receiverEmail) {
			return true, nil
		}
	}

	if err := queueClient.EnqueueOrderStatusEmail(queue.OrderStatusEmailPayload{
		OrderID: orderID,
		Status:  strings.TrimSpace(status),
	}); err != nil {
		return false, err
	}
	return false, nil
}
