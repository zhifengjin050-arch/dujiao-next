package queue

import (
	"encoding/json"

	"github.com/dujiao-next/internal/constants"

	"github.com/hibiken/asynq"
)

const (
	// TaskOrderStatusEmail ??????????
	TaskOrderStatusEmail = constants.TaskOrderStatusEmail
	// TaskOrderAutoFulfill ??????
	TaskOrderAutoFulfill = constants.TaskOrderAutoFulfill
	// TaskOrderTimeoutCancel ??????
	TaskOrderTimeoutCancel = constants.TaskOrderTimeoutCancel
	// TaskWalletRechargeExpire ??????????
	TaskWalletRechargeExpire = constants.TaskWalletRechargeExpire
	// TaskNotificationDispatch ????????
	TaskNotificationDispatch = constants.TaskNotificationDispatch
	// TaskAffiliateConfirmCommissions ????????
	TaskAffiliateConfirmCommissions = constants.TaskAffiliateConfirmCommissions
	// TaskUpstreamSyncStock ????????
	TaskUpstreamSyncStock = constants.TaskUpstreamSyncStock
	// TaskProcurementSubmit ??????
	TaskProcurementSubmit = constants.TaskProcurementSubmit
	// TaskProcurementPollStatus ????????
	TaskProcurementPollStatus = constants.TaskProcurementPollStatus
	// TaskProcurementSyncAccepted ?????????
	TaskProcurementSyncAccepted = constants.TaskProcurementSyncAccepted
	// TaskDownstreamCallback ????????
	TaskDownstreamCallback = constants.TaskDownstreamCallback
	// TaskReconciliationRun ??????
	TaskReconciliationRun = constants.TaskReconciliationRun
	// TaskBotNotify Bot ??????
	TaskBotNotify = constants.TaskBotNotify
	// TaskTelegramBroadcast Telegram ????
	TaskTelegramBroadcast = constants.TaskTelegramBroadcast
)

// OrderStatusEmailPayload ??????????
type OrderStatusEmailPayload struct {
	OrderID        uint   `json:"order_id"`
	RefundRecordID uint   `json:"refund_record_id,omitempty"`
	Status         string `json:"status"`
}

// OrderAutoFulfillPayload ????????
type OrderAutoFulfillPayload struct {
	OrderID uint `json:"order_id"`
}

// OrderTimeoutCancelPayload ????????
type OrderTimeoutCancelPayload struct {
	OrderID uint `json:"order_id"`
}

// WalletRechargeExpirePayload ????????????
type WalletRechargeExpirePayload struct {
	PaymentID uint `json:"payment_id"`
}

// NotificationDispatchPayload ??????????
type NotificationDispatchPayload struct {
	EventType string                 `json:"event_type"`
	BizType   string                 `json:"biz_type"`
	BizID     uint                   `json:"biz_id"`
	Locale    string                 `json:"locale"`
	Force     bool                   `json:"force"`
	Data      map[string]interface{} `json:"data"`
}

// NewOrderStatusEmailTask ??????????
func NewOrderStatusEmailTask(payload OrderStatusEmailPayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskOrderStatusEmail, body), nil
}

// NewOrderAutoFulfillTask ????????
func NewOrderAutoFulfillTask(payload OrderAutoFulfillPayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskOrderAutoFulfill, body), nil
}

// NewOrderTimeoutCancelTask ????????
func NewOrderTimeoutCancelTask(payload OrderTimeoutCancelPayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskOrderTimeoutCancel, body), nil
}

// NewWalletRechargeExpireTask ????????????
func NewWalletRechargeExpireTask(payload WalletRechargeExpirePayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskWalletRechargeExpire, body), nil
}

// NewNotificationDispatchTask ??????????
func NewNotificationDispatchTask(payload NotificationDispatchPayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskNotificationDispatch, body), nil
}

// NewNotificationInventoryAlertCheckTask ??????????
func NewNotificationInventoryAlertCheckTask() (*asynq.Task, error) {
	return NewNotificationDispatchTask(NotificationDispatchPayload{
		EventType: constants.NotificationEventExceptionAlertCheck,
		BizType:   constants.NotificationBizTypeDashboardAlert,
		BizID:     0,
		Data: map[string]interface{}{
			"message": "scheduled_inventory_alert_check",
		},
	})
}

// NewAffiliateConfirmCommissionsTask ??????????
func NewAffiliateConfirmCommissionsTask() *asynq.Task {
	return asynq.NewTask(TaskAffiliateConfirmCommissions, nil)
}

// NewUpstreamSyncStockTask ??????????
func NewUpstreamSyncStockTask() *asynq.Task {
	return asynq.NewTask(TaskUpstreamSyncStock, nil)
}

// NewProcurementSyncAcceptedTask ???????????
func NewProcurementSyncAcceptedTask() *asynq.Task {
	return asynq.NewTask(TaskProcurementSyncAccepted, nil)
}

// ProcurementSubmitPayload ????????
type ProcurementSubmitPayload struct {
	ProcurementOrderID uint `json:"procurement_order_id"`
}

// NewProcurementSubmitTask ????????
func NewProcurementSubmitTask(payload ProcurementSubmitPayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskProcurementSubmit, body), nil
}

// ProcurementPollStatusPayload ??????????
type ProcurementPollStatusPayload struct {
	ProcurementOrderID uint `json:"procurement_order_id"`
}

// NewProcurementPollStatusTask ??????????
func NewProcurementPollStatusTask(payload ProcurementPollStatusPayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskProcurementPollStatus, body), nil
}

// ReconciliationRunPayload ????????
type ReconciliationRunPayload struct {
	JobID uint `json:"job_id"`
}

// NewReconciliationRunTask ????????
func NewReconciliationRunTask(payload ReconciliationRunPayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskReconciliationRun, body), nil
}

// DownstreamCallbackPayload ??????????
type DownstreamCallbackPayload struct {
	DownstreamOrderRefID uint `json:"downstream_order_ref_id"`
}

// NewDownstreamCallbackTask ??????????
func NewDownstreamCallbackTask(payload DownstreamCallbackPayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskDownstreamCallback, body), nil
}

// BotNotifyPayload Bot ????????
type BotNotifyPayload struct {
	EventType      string `json:"event_type,omitempty"`
	OrderID        uint   `json:"order_id"`
	TelegramUserID string `json:"telegram_user_id"`
	RechargeNo     string `json:"recharge_no,omitempty"`
	Amount         string `json:"amount,omitempty"`
	Currency       string `json:"currency,omitempty"`
}

const (
	// BotNotifyEventOrderPaid ???????????
	BotNotifyEventOrderPaid = "order_paid"
	// BotNotifyEventOrderFulfilled ?????????
	BotNotifyEventOrderFulfilled = "order_fulfilled"
	// BotNotifyEventWalletRechargeSucceeded ???????????
	BotNotifyEventWalletRechargeSucceeded = "wallet_recharge_succeeded"
)

// NewBotNotifyTask ?? Bot ??????
func NewBotNotifyTask(payload BotNotifyPayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskBotNotify, body), nil
}

// TelegramBroadcastPayload Telegram ???????
type TelegramBroadcastPayload struct {
	BroadcastID uint `json:"broadcast_id"`
}

// NewTelegramBroadcastTask ?? Telegram ?????
func NewTelegramBroadcastTask(payload TelegramBroadcastPayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskTelegramBroadcast, body), nil
}
