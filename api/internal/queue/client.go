package queue

import (
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"

	"github.com/hibiken/asynq"
)

const (
	// DefaultQueue ??????
	DefaultQueue = constants.QueueDefault
)

// Client ???????
type Client struct {
	client       *asynq.Client
	enabled      bool
	defaultQueue string
}

// NewClient ???????
func NewClient(cfg *config.QueueConfig) (*Client, error) {
	if cfg == nil || !cfg.Enabled {
		return &Client{enabled: false, defaultQueue: DefaultQueue}, nil
	}
	opt := buildRedisOpt(cfg)
	client := asynq.NewClient(opt)
	return &Client{
		client:       client,
		enabled:      true,
		defaultQueue: DefaultQueue,
	}, nil
}

// Enabled ??????
func (c *Client) Enabled() bool {
	return c != nil && c.enabled && c.client != nil
}

// Close ?????
func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

// EnqueueOrderStatusEmail ??????????
func (c *Client) EnqueueOrderStatusEmail(payload OrderStatusEmailPayload, opts ...asynq.Option) error {
	if !c.Enabled() {
		return nil
	}
	task, err := NewOrderStatusEmailTask(payload)
	if err != nil {
		return err
	}
	options := append([]asynq.Option{asynq.Queue(c.defaultQueue)}, opts...)
	_, err = c.client.Enqueue(task, options...)
	return err
}

// EnqueueOrderAutoFulfill ????????
func (c *Client) EnqueueOrderAutoFulfill(payload OrderAutoFulfillPayload, opts ...asynq.Option) error {
	if !c.Enabled() {
		return nil
	}
	task, err := NewOrderAutoFulfillTask(payload)
	if err != nil {
		return err
	}
	options := append([]asynq.Option{asynq.Queue(c.defaultQueue)}, opts...)
	_, err = c.client.Enqueue(task, options...)
	return err
}

// EnqueueOrderTimeoutCancel ??????????
func (c *Client) EnqueueOrderTimeoutCancel(payload OrderTimeoutCancelPayload, delay time.Duration) error {
	if !c.Enabled() {
		return nil
	}
	if delay < 0 {
		delay = 0
	}
	task, err := NewOrderTimeoutCancelTask(payload)
	if err != nil {
		return err
	}
	options := []asynq.Option{asynq.Queue(c.defaultQueue), asynq.ProcessIn(delay)}
	_, err = c.client.Enqueue(task, options...)
	return err
}

// EnqueueWalletRechargeExpire ????????????
func (c *Client) EnqueueWalletRechargeExpire(payload WalletRechargeExpirePayload, delay time.Duration) error {
	if !c.Enabled() {
		return nil
	}
	if delay < 0 {
		delay = 0
	}
	task, err := NewWalletRechargeExpireTask(payload)
	if err != nil {
		return err
	}
	options := []asynq.Option{asynq.Queue(c.defaultQueue), asynq.ProcessIn(delay)}
	_, err = c.client.Enqueue(task, options...)
	return err
}

// EnqueueNotificationDispatch ??????????
func (c *Client) EnqueueNotificationDispatch(payload NotificationDispatchPayload, opts ...asynq.Option) error {
	if !c.Enabled() {
		return nil
	}
	task, err := NewNotificationDispatchTask(payload)
	if err != nil {
		return err
	}
	options := append([]asynq.Option{asynq.Queue(c.defaultQueue)}, opts...)
	_, err = c.client.Enqueue(task, options...)
	return err
}

// EnqueueProcurementSubmit ????????
func (c *Client) EnqueueProcurementSubmit(payload ProcurementSubmitPayload, opts ...asynq.Option) error {
	if !c.Enabled() {
		return nil
	}
	task, err := NewProcurementSubmitTask(payload)
	if err != nil {
		return err
	}
	// ?????????????,asynq ???????(DB/Redis ????)
	options := append([]asynq.Option{asynq.Queue(c.defaultQueue), asynq.MaxRetry(3)}, opts...)
	_, err = c.client.Enqueue(task, options...)
	return err
}

// EnqueueProcurementPollStatus ??????????
func (c *Client) EnqueueProcurementPollStatus(payload ProcurementPollStatusPayload, delay time.Duration) error {
	if !c.Enabled() {
		return nil
	}
	if delay < 0 {
		delay = 0
	}
	task, err := NewProcurementPollStatusTask(payload)
	if err != nil {
		return err
	}
	options := []asynq.Option{asynq.Queue(c.defaultQueue), asynq.ProcessIn(delay)}
	_, err = c.client.Enqueue(task, options...)
	return err
}

// EnqueueDownstreamCallback ??????????
func (c *Client) EnqueueDownstreamCallback(payload DownstreamCallbackPayload, opts ...asynq.Option) error {
	if !c.Enabled() {
		return nil
	}
	task, err := NewDownstreamCallbackTask(payload)
	if err != nil {
		return err
	}
	options := append([]asynq.Option{asynq.Queue(c.defaultQueue)}, opts...)
	_, err = c.client.Enqueue(task, options...)
	return err
}

// EnqueueReconciliationRun ????????
func (c *Client) EnqueueReconciliationRun(payload ReconciliationRunPayload, opts ...asynq.Option) error {
	if !c.Enabled() {
		return nil
	}
	task, err := NewReconciliationRunTask(payload)
	if err != nil {
		return err
	}
	options := append([]asynq.Option{asynq.Queue(c.defaultQueue)}, opts...)
	_, err = c.client.Enqueue(task, options...)
	return err
}

// EnqueueBotNotify ?? Bot ??????
func (c *Client) EnqueueBotNotify(payload BotNotifyPayload, opts ...asynq.Option) error {
	if !c.Enabled() {
		return nil
	}
	task, err := NewBotNotifyTask(payload)
	if err != nil {
		return err
	}
	options := append([]asynq.Option{asynq.Queue(c.defaultQueue), asynq.MaxRetry(5)}, opts...)
	_, err = c.client.Enqueue(task, options...)
	return err
}

// EnqueueTelegramBroadcast ?? Telegram ?????
func (c *Client) EnqueueTelegramBroadcast(payload TelegramBroadcastPayload, opts ...asynq.Option) error {
	if !c.Enabled() {
		return nil
	}
	task, err := NewTelegramBroadcastTask(payload)
	if err != nil {
		return err
	}
	options := append([]asynq.Option{asynq.Queue(c.defaultQueue), asynq.MaxRetry(3)}, opts...)
	_, err = c.client.Enqueue(task, options...)
	return err
}

// BuildServerConfig ????????
func BuildServerConfig(cfg *config.QueueConfig) (asynq.RedisClientOpt, asynq.Config) {
	opt := buildRedisOpt(cfg)
	concurrency := 10
	if cfg != nil && cfg.Concurrency > 0 {
		concurrency = cfg.Concurrency
	}
	queues := map[string]int{DefaultQueue: 1}
	if cfg != nil && len(cfg.Queues) > 0 {
		queues = cfg.Queues
	}
	return opt, asynq.Config{
		Concurrency: concurrency,
		Queues:      queues,
	}
}

func buildRedisOpt(cfg *config.QueueConfig) asynq.RedisClientOpt {
	host := "127.0.0.1"
	port := 6379
	password := ""
	db := 0
	if cfg != nil {
		if strings.TrimSpace(cfg.Host) != "" {
			host = strings.TrimSpace(cfg.Host)
		}
		if cfg.Port > 0 {
			port = cfg.Port
		}
		password = cfg.Password
		db = cfg.DB
	}
	return asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	}
}
