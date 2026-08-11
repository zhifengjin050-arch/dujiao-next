package queue

import (
	"encoding/json"
	"testing"
)

func TestNewOrderStatusEmailTask(t *testing.T) {
	payload := OrderStatusEmailPayload{
		OrderID: 100,
		Status:  "completed",
	}
	task, err := NewOrderStatusEmailTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskOrderStatusEmail {
		t.Fatalf("expected type %s, got %s", TaskOrderStatusEmail, task.Type())
	}

	// 验证 payload 可以正确反序列化
	var decoded OrderStatusEmailPayload
	if err := json.Unmarshal(task.Payload(), &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded.OrderID != 100 {
		t.Fatalf("expected OrderID=100, got %d", decoded.OrderID)
	}
	if decoded.Status != "completed" {
		t.Fatalf("expected Status=completed, got %s", decoded.Status)
	}
}

func TestNewOrderStatusEmailTaskWithRefund(t *testing.T) {
	payload := OrderStatusEmailPayload{
		OrderID:        200,
		RefundRecordID: 50,
		Status:         "refunded",
	}
	task, err := NewOrderStatusEmailTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded OrderStatusEmailPayload
	if err := json.Unmarshal(task.Payload(), &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded.RefundRecordID != 50 {
		t.Fatalf("expected RefundRecordID=50, got %d", decoded.RefundRecordID)
	}
}

func TestNewOrderAutoFulfillTask(t *testing.T) {
	payload := OrderAutoFulfillPayload{
		OrderID: 300,
	}
	task, err := NewOrderAutoFulfillTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskOrderAutoFulfill {
		t.Fatalf("expected type %s, got %s", TaskOrderAutoFulfill, task.Type())
	}

	var decoded OrderAutoFulfillPayload
	if err := json.Unmarshal(task.Payload(), &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded.OrderID != 300 {
		t.Fatalf("expected OrderID=300, got %d", decoded.OrderID)
	}
}

func TestNewOrderTimeoutCancelTask(t *testing.T) {
	payload := OrderTimeoutCancelPayload{
		OrderID: 400,
	}
	task, err := NewOrderTimeoutCancelTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskOrderTimeoutCancel {
		t.Fatalf("expected type %s, got %s", TaskOrderTimeoutCancel, task.Type())
	}
}

func TestNewWalletRechargeExpireTask(t *testing.T) {
	payload := WalletRechargeExpirePayload{
		PaymentID: 500,
	}
	task, err := NewWalletRechargeExpireTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskWalletRechargeExpire {
		t.Fatalf("expected type %s, got %s", TaskWalletRechargeExpire, task.Type())
	}

	var decoded WalletRechargeExpirePayload
	if err := json.Unmarshal(task.Payload(), &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded.PaymentID != 500 {
		t.Fatalf("expected PaymentID=500, got %d", decoded.PaymentID)
	}
}

func TestNewNotificationDispatchTask(t *testing.T) {
	payload := NotificationDispatchPayload{
		EventType: "order.created",
		BizType:   "order",
		BizID:     600,
		Locale:    "zh-CN",
		Force:     true,
		Data: map[string]interface{}{
			"amount": "99.00",
		},
	}
	task, err := NewNotificationDispatchTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskNotificationDispatch {
		t.Fatalf("expected type %s, got %s", TaskNotificationDispatch, task.Type())
	}

	var decoded NotificationDispatchPayload
	if err := json.Unmarshal(task.Payload(), &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded.Locale != "zh-CN" {
		t.Fatalf("expected Locale=zh-CN, got %s", decoded.Locale)
	}
	if !decoded.Force {
		t.Fatal("expected Force=true")
	}
}

func TestNewNotificationInventoryAlertCheckTask(t *testing.T) {
	task, err := NewNotificationInventoryAlertCheckTask()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskNotificationDispatch {
		t.Fatalf("expected type %s, got %s", TaskNotificationDispatch, task.Type())
	}

	var decoded NotificationDispatchPayload
	if err := json.Unmarshal(task.Payload(), &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded.BizType != "dashboard_alert" {
		t.Fatalf("expected BizType=dashboard_alert, got %s", decoded.BizType)
	}
}

func TestNewAffiliateConfirmCommissionsTask(t *testing.T) {
	task := NewAffiliateConfirmCommissionsTask()
	if task.Type() != TaskAffiliateConfirmCommissions {
		t.Fatalf("expected type %s, got %s", TaskAffiliateConfirmCommissions, task.Type())
	}
}

func TestNewUpstreamSyncStockTask(t *testing.T) {
	task := NewUpstreamSyncStockTask()
	if task.Type() != TaskUpstreamSyncStock {
		t.Fatalf("expected type %s, got %s", TaskUpstreamSyncStock, task.Type())
	}
}

func TestNewProcurementSyncAcceptedTask(t *testing.T) {
	task := NewProcurementSyncAcceptedTask()
	if task.Type() != TaskProcurementSyncAccepted {
		t.Fatalf("expected type %s, got %s", TaskProcurementSyncAccepted, task.Type())
	}
}

func TestNewProcurementSubmitTask(t *testing.T) {
	payload := ProcurementSubmitPayload{
		ProcurementOrderID: 700,
	}
	task, err := NewProcurementSubmitTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskProcurementSubmit {
		t.Fatalf("expected type %s, got %s", TaskProcurementSubmit, task.Type())
	}

	var decoded ProcurementSubmitPayload
	if err := json.Unmarshal(task.Payload(), &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded.ProcurementOrderID != 700 {
		t.Fatalf("expected ProcurementOrderID=700, got %d", decoded.ProcurementOrderID)
	}
}

func TestNewProcurementPollStatusTask(t *testing.T) {
	payload := ProcurementPollStatusPayload{
		ProcurementOrderID: 800,
	}
	task, err := NewProcurementPollStatusTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskProcurementPollStatus {
		t.Fatalf("expected type %s, got %s", TaskProcurementPollStatus, task.Type())
	}
}

func TestNewReconciliationRunTask(t *testing.T) {
	payload := ReconciliationRunPayload{
		JobID: 900,
	}
	task, err := NewReconciliationRunTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskReconciliationRun {
		t.Fatalf("expected type %s, got %s", TaskReconciliationRun, task.Type())
	}

	var decoded ReconciliationRunPayload
	if err := json.Unmarshal(task.Payload(), &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded.JobID != 900 {
		t.Fatalf("expected JobID=900, got %d", decoded.JobID)
	}
}

func TestNewDownstreamCallbackTask(t *testing.T) {
	payload := DownstreamCallbackPayload{
		DownstreamOrderRefID: 1000,
	}
	task, err := NewDownstreamCallbackTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskDownstreamCallback {
		t.Fatalf("expected type %s, got %s", TaskDownstreamCallback, task.Type())
	}
}

func TestNewBotNotifyTask(t *testing.T) {
	payload := BotNotifyPayload{
		EventType:      BotNotifyEventOrderPaid,
		OrderID:        1100,
		TelegramUserID: "123456789",
		Amount:         "99.00",
		Currency:       "CNY",
	}
	task, err := NewBotNotifyTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskBotNotify {
		t.Fatalf("expected type %s, got %s", TaskBotNotify, task.Type())
	}

	var decoded BotNotifyPayload
	if err := json.Unmarshal(task.Payload(), &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded.EventType != BotNotifyEventOrderPaid {
		t.Fatalf("expected EventType=%s, got %s", BotNotifyEventOrderPaid, decoded.EventType)
	}
	if decoded.TelegramUserID != "123456789" {
		t.Fatalf("expected TelegramUserID=123456789, got %s", decoded.TelegramUserID)
	}
}

func TestNewBotNotifyTaskFulfilled(t *testing.T) {
	payload := BotNotifyPayload{
		EventType:      BotNotifyEventOrderFulfilled,
		OrderID:        1200,
		TelegramUserID: "987654321",
	}
	task, err := NewBotNotifyTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskBotNotify {
		t.Fatalf("expected type %s, got %s", TaskBotNotify, task.Type())
	}
}

func TestNewBotNotifyTaskWalletRecharge(t *testing.T) {
	payload := BotNotifyPayload{
		EventType:      BotNotifyEventWalletRechargeSucceeded,
		OrderID:        1300,
		TelegramUserID: "111222333",
		RechargeNo:     "R20240101001",
		Amount:         "50.00",
		Currency:       "USD",
	}
	task, err := NewBotNotifyTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskBotNotify {
		t.Fatalf("expected type %s, got %s", TaskBotNotify, task.Type())
	}

	var decoded BotNotifyPayload
	if err := json.Unmarshal(task.Payload(), &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded.RechargeNo != "R20240101001" {
		t.Fatalf("expected RechargeNo=R20240101001, got %s", decoded.RechargeNo)
	}
	if decoded.Currency != "USD" {
		t.Fatalf("expected Currency=USD, got %s", decoded.Currency)
	}
}

func TestNewTelegramBroadcastTask(t *testing.T) {
	payload := TelegramBroadcastPayload{
		BroadcastID: 1400,
	}
	task, err := NewTelegramBroadcastTask(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TaskTelegramBroadcast {
		t.Fatalf("expected type %s, got %s", TaskTelegramBroadcast, task.Type())
	}
}

func TestAllTaskTypeConstants(t *testing.T) {
	// 验证所有任务类型常量非空
	taskTypes := []string{
		TaskOrderStatusEmail,
		TaskOrderAutoFulfill,
		TaskOrderTimeoutCancel,
		TaskWalletRechargeExpire,
		TaskNotificationDispatch,
		TaskAffiliateConfirmCommissions,
		TaskUpstreamSyncStock,
		TaskProcurementSubmit,
		TaskProcurementPollStatus,
		TaskProcurementSyncAccepted,
		TaskDownstreamCallback,
		TaskReconciliationRun,
		TaskBotNotify,
		TaskTelegramBroadcast,
	}

	for _, tt := range taskTypes {
		if tt == "" {
			t.Fatal("task type should not be empty")
		}
	}
}

func TestBotNotifyEventConstants(t *testing.T) {
	if BotNotifyEventOrderPaid != "order_paid" {
		t.Fatalf("expected 'order_paid', got %q", BotNotifyEventOrderPaid)
	}
	if BotNotifyEventOrderFulfilled != "order_fulfilled" {
		t.Fatalf("expected 'order_fulfilled', got %q", BotNotifyEventOrderFulfilled)
	}
	if BotNotifyEventWalletRechargeSucceeded != "wallet_recharge_succeeded" {
		t.Fatalf("expected 'wallet_recharge_succeeded', got %q", BotNotifyEventWalletRechargeSucceeded)
	}
}
