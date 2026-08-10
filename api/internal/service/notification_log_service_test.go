package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/queue"
	"github.com/dujiao-next/internal/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupNotificationLogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:notification_log_service_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.NotificationLog{}); err != nil {
		t.Fatalf("auto migrate notification log failed: %v", err)
	}
	return db
}

func setupNotificationLogService(t *testing.T) (*NotificationService, *NotificationLogService) {
	t.Helper()

	db := setupNotificationLogTestDB(t)
	logSvc := NewNotificationLogService(repository.NewNotificationLogRepository(db))
	settingRepo := newMockSettingRepo()
	settingRepo.store[constants.SettingKeyNotificationCenterConfig] = NotificationCenterSettingToMap(NotificationCenterSetting{
		DefaultLocale: constants.LocaleEnUS,
		Scenes: NotificationSceneSetting{
			OrderPaidSuccess: true,
			ExceptionAlert:   true,
		},
		Channels: NotificationChannelsSetting{
			Email: NotificationChannelSetting{
				Enabled:    true,
				Recipients: []string{"telegram_success@login.local", "failure@example.com"},
			},
		},
		Templates: NotificationTemplatesSetting{
			OrderPaidSuccess: NotificationSceneTemplate{
				ENUS: NotificationLocalizedTemplate{
					Title: "Order {{order_no}}",
					Body:  "Customer {{customer_email}}",
				},
			},
			ExceptionAlert: NotificationSceneTemplate{
				ENUS: NotificationLocalizedTemplate{
					Title: "Alert {{message}}",
					Body:  "{{message}}",
				},
			},
		},
	})

	return &NotificationService{
		settingService: NewSettingService(settingRepo),
		emailService:   NewEmailService(&config.EmailConfig{Enabled: false}),
		logService:     logSvc,
	}, logSvc
}

func TestNotificationServiceSendTestRecordsSuccessLog(t *testing.T) {
	svc, logSvc := setupNotificationLogService(t)

	err := svc.SendTest(context.Background(), NotificationTestSendInput{
		Channel: "email",
		Target:  "telegram_test_success@login.local",
		Scene:   constants.NotificationEventOrderPaidSuccess,
		Locale:  constants.LocaleEnUS,
	})
	if err != nil {
		t.Fatalf("SendTest failed: %v", err)
	}

	isTest := true
	items, total, err := logSvc.ListForAdmin(repository.NotificationLogListFilter{
		Page:      1,
		PageSize:  10,
		EventType: constants.NotificationEventOrderPaidSuccess,
		IsTest:    &isTest,
	})
	if err != nil {
		t.Fatalf("list notification logs failed: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected 1 test log, total=%d len=%d", total, len(items))
	}

	item := items[0]
	if item.Status != notificationLogStatusSuccess {
		t.Fatalf("status want success got %s", item.Status)
	}
	if !item.IsTest {
		t.Fatalf("expected is_test true")
	}
	if item.Recipient != "telegram_test_success@login.local" {
		t.Fatalf("unexpected recipient: %s", item.Recipient)
	}
	if !strings.Contains(item.Title, "DJ202603230001") {
		t.Fatalf("title should include sample order no, got %s", item.Title)
	}
}

func TestNotificationServiceDispatchSingleEventRecordsPerRecipientResult(t *testing.T) {
	svc, logSvc := setupNotificationLogService(t)

	setting, err := svc.settingService.GetNotificationCenterSetting()
	if err != nil {
		t.Fatalf("get notification center setting failed: %v", err)
	}

	dispatchErr := svc.dispatchSingleEvent(context.Background(), setting, queue.NotificationDispatchPayload{
		EventType: constants.NotificationEventOrderPaidSuccess,
		BizType:   constants.NotificationBizTypeOrder,
		BizID:     88,
		Locale:    constants.LocaleEnUS,
		Force:     true,
		Data: map[string]interface{}{
			"order_no":       "DJ-LOG-88",
			"customer_email": "member@example.com",
		},
	})
	if dispatchErr == nil {
		t.Fatalf("dispatchSingleEvent should return error when one recipient fails")
	}
	if !errors.Is(dispatchErr, ErrNotificationSendFailed) {
		t.Fatalf("dispatchSingleEvent should wrap ErrNotificationSendFailed, got %v", dispatchErr)
	}

	isTest := false
	items, total, err := logSvc.ListForAdmin(repository.NotificationLogListFilter{
		Page:      1,
		PageSize:  10,
		EventType: constants.NotificationEventOrderPaidSuccess,
		IsTest:    &isTest,
	})
	if err != nil {
		t.Fatalf("list notification logs failed: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("expected 2 live logs, total=%d len=%d", total, len(items))
	}

	var successCount int
	var failedCount int
	for _, item := range items {
		if item.BizType != constants.NotificationBizTypeOrder {
			t.Fatalf("unexpected biz_type: %s", item.BizType)
		}
		if item.BizID != 88 {
			t.Fatalf("unexpected biz_id: %d", item.BizID)
		}
		switch item.Status {
		case notificationLogStatusSuccess:
			successCount++
		case notificationLogStatusFailed:
			failedCount++
			if item.ErrorMessage == "" {
				t.Fatalf("failed log should contain error message")
			}
		default:
			t.Fatalf("unexpected status: %s", item.Status)
		}
	}
	if successCount != 1 || failedCount != 1 {
		t.Fatalf("unexpected success/failed distribution: success=%d failed=%d", successCount, failedCount)
	}
}
