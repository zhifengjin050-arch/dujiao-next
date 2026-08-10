package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/queue"
	"github.com/dujiao-next/internal/repository"

	"github.com/hibiken/asynq"
)

// NotificationEnqueueInput ????????
type NotificationEnqueueInput struct {
	EventType string
	BizType   string
	BizID     uint
	Locale    string
	Force     bool
	Data      models.JSON
}

// NotificationTestSendInput ????????
type NotificationTestSendInput struct {
	Channel   string
	Target    string
	Scene     string
	Locale    string
	Variables map[string]interface{}
}

// NotificationService ??????
type NotificationService struct {
	settingService *SettingService
	emailService   *EmailService
	queueClient    *queue.Client
	dashboardSvc   *DashboardService
	logService     *NotificationLogService
	telegramSender *TelegramNotifyService
}

// NewNotificationService ????????
func NewNotificationService(
	settingService *SettingService,
	emailService *EmailService,
	queueClient *queue.Client,
	dashboardSvc *DashboardService,
	logService *NotificationLogService,
	defaultTelegramCfg config.TelegramAuthConfig,
) *NotificationService {
	return &NotificationService{
		settingService: settingService,
		emailService:   emailService,
		queueClient:    queueClient,
		dashboardSvc:   dashboardSvc,
		logService:     logService,
		telegramSender: NewTelegramNotifyService(settingService, defaultTelegramCfg),
	}
}

// Enqueue ??????
func (s *NotificationService) Enqueue(input NotificationEnqueueInput) error {
	eventType := strings.ToLower(strings.TrimSpace(input.EventType))
	if !isNotificationEventSupported(eventType) {
		return ErrNotificationEventInvalid
	}
	if s == nil || s.queueClient == nil {
		return nil
	}

	payload := queue.NotificationDispatchPayload{
		EventType: eventType,
		BizType:   strings.TrimSpace(input.BizType),
		BizID:     input.BizID,
		Locale:    strings.TrimSpace(input.Locale),
		Force:     input.Force,
		Data:      notificationJSONToMap(input.Data),
	}
	return s.queueClient.EnqueueNotificationDispatch(payload, asynq.MaxRetry(5))
}

// Dispatch ????????
func (s *NotificationService) Dispatch(ctx context.Context, payload queue.NotificationDispatchPayload) error {
	if s == nil {
		return nil
	}
	eventType := strings.ToLower(strings.TrimSpace(payload.EventType))
	if !isNotificationEventSupported(eventType) {
		return ErrNotificationEventInvalid
	}

	setting, err := s.settingService.GetNotificationCenterSetting()
	if err != nil {
		return err
	}
	if !setting.Scenes.IsSceneEnabled(eventType) {
		return nil
	}

	if eventType == constants.NotificationEventExceptionAlertCheck {
		return s.dispatchExceptionAlertCheck(ctx, setting, payload)
	}
	return s.dispatchSingleEvent(ctx, setting, payload)
}

// SendTest ??????
func (s *NotificationService) SendTest(ctx context.Context, input NotificationTestSendInput) error {
	if s == nil {
		return ErrNotificationSendFailed
	}
	channel := strings.ToLower(strings.TrimSpace(input.Channel))
	target := strings.TrimSpace(input.Target)
	if channel == "" || target == "" {
		return ErrNotificationConfigInvalid
	}

	setting, err := s.settingService.GetNotificationCenterSetting()
	if err != nil {
		return err
	}
	scene := strings.ToLower(strings.TrimSpace(input.Scene))
	if scene == "" {
		scene = constants.NotificationEventExceptionAlert
	}
	template := setting.Templates.TemplateByEvent(scene).ResolveLocaleTemplate(resolveNotificationLocale(input.Locale, setting.DefaultLocale))
	variables := cloneNotificationVariables(input.Variables)
	if variables == nil {
		variables = map[string]interface{}{}
	}
	locale := resolveNotificationLocale(input.Locale, setting.DefaultLocale)
	applyNotificationTestVariables(variables, buildNotificationTestVariables(scene, locale))
	variables["event_type"] = scene
	variables["message"] = pickNotificationMessage(variables["message"], "test message")
	title := renderNotificationTemplate(template.Title, variables)
	body := renderNotificationTemplate(template.Body, variables)
	if strings.TrimSpace(body) == "" {
		body = title
	}
	if strings.TrimSpace(title) == "" {
		title = "Notification Test"
	}

	switch channel {
	case "email":
		sendErr := ErrNotificationSendFailed
		if s.emailService != nil {
			sendErr = s.emailService.SendCustomEmail(target, title, body)
		}
		s.recordSendAttempt(notificationSendAttempt{
			eventType: scene,
			channel:   channel,
			recipient: target,
			locale:    locale,
			title:     title,
			body:      body,
			variables: variables,
			isTest:    true,
			sendErr:   sendErr,
		})
		return sendErr
	case "telegram":
		sendErr := ErrNotificationSendFailed
		gatewayCtx, cancel := detachOutboundRequestContext(ctx)
		defer cancel()
		if s.telegramSender != nil {
			sendErr = s.telegramSender.SendMessage(gatewayCtx, target, composeTelegramMessage(title, body))
		}
		s.recordSendAttempt(notificationSendAttempt{
			eventType: scene,
			channel:   channel,
			recipient: target,
			locale:    locale,
			title:     title,
			body:      body,
			variables: variables,
			isTest:    true,
			sendErr:   sendErr,
		})
		return sendErr
	default:
		return ErrNotificationConfigInvalid
	}
}

func (s *NotificationService) dispatchExceptionAlertCheck(ctx context.Context, setting NotificationCenterSetting, payload queue.NotificationDispatchPayload) error {
	if s.dashboardSvc == nil || s.settingService == nil {
		return nil
	}

	dashboardSetting, err := s.settingService.GetDashboardSetting()
	if err != nil {
		return err
	}

	var firstErr error
	inventoryAlerts, err := s.dashboardSvc.GetInventoryAlertItems(ctx, dashboardSetting.Alert.LowStockThreshold)
	if err != nil {
		return err
	}
	for _, itemPayload := range buildInventoryAlertDispatchPayloads(setting, dashboardSetting, payload, inventoryAlerts) {
		allowed, intervalErr := acquireInventoryAlertInterval(ctx, setting.InventoryAlertIntervalSeconds, itemPayload)
		if intervalErr != nil {
			logger.Warnw("notification_inventory_alert_interval_failed", "error", intervalErr)
		}
		if intervalErr == nil && !allowed {
			continue
		}
		if err := s.dispatchSingleEvent(ctx, setting, itemPayload); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	overview, err := s.dashboardSvc.GetOverview(ctx, DashboardQueryInput{
		Range:        "today",
		Timezone:     time.Local.String(),
		ForceRefresh: true,
	})
	if err != nil {
		return err
	}
	if overview == nil || len(overview.Alerts) == 0 {
		return firstErr
	}

	alertLocale := resolveNotificationLocale(payload.Locale, setting.DefaultLocale)
	for _, alert := range overview.Alerts {
		if isInventoryAlertType(alert.Type) {
			continue
		}
		data := cloneNotificationVariables(payload.Data)
		if data == nil {
			data = map[string]interface{}{}
		}
		alertType := strings.TrimSpace(alert.Type)
		data["alert_type"] = alertTypeLabelByType(alertLocale, alertType)
		data["alert_type_label"] = data["alert_type"]
		data["alert_type_key"] = alertType
		data["alert_level"] = strings.TrimSpace(alert.Level)
		data["alert_value"] = fmt.Sprintf("%d", alert.Value)
		data["alert_threshold"] = fmt.Sprintf("%d", thresholdValueByAlertType(dashboardSetting.Alert, alertType))
		data["message"] = thresholdMessageByAlertType(alertType)

		itemPayload := payload
		itemPayload.EventType = constants.NotificationEventExceptionAlert
		itemPayload.BizType = constants.NotificationBizTypeDashboardAlert
		itemPayload.Data = data
		if err := s.dispatchSingleEvent(ctx, setting, itemPayload); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *NotificationService) dispatchSingleEvent(ctx context.Context, setting NotificationCenterSetting, payload queue.NotificationDispatchPayload) error {
	if !payload.Force {
		ok, err := acquireNotificationDedupe(ctx, setting.DedupeTTLSeconds, payload)
		if err != nil {
			logger.Warnw("notification_dedupe_failed", "event_type", payload.EventType, "error", err)
		}
		if err == nil && !ok {
			return nil
		}
	}

	locale := resolveNotificationLocale(payload.Locale, setting.DefaultLocale)
	template := setting.Templates.TemplateByEvent(payload.EventType).ResolveLocaleTemplate(locale)
	variables := buildNotificationTemplateVariables(payload)
	title := renderNotificationTemplate(template.Title, variables)
	body := renderNotificationTemplate(template.Body, variables)
	if strings.TrimSpace(body) == "" {
		body = title
	}
	if strings.TrimSpace(title) == "" {
		title = "Notification"
	}

	var firstErr error
	if setting.Channels.Email.Enabled && len(setting.Channels.Email.Recipients) > 0 {
		for _, recipient := range setting.Channels.Email.Recipients {
			var sendErr error
			if s.emailService == nil {
				sendErr = ErrNotificationSendFailed
			} else {
				sendErr = s.emailService.SendCustomEmail(recipient, title, body)
			}
			s.recordSendAttempt(notificationSendAttempt{
				eventType: payload.EventType,
				bizType:   payload.BizType,
				bizID:     payload.BizID,
				channel:   "email",
				recipient: recipient,
				locale:    locale,
				title:     title,
				body:      body,
				variables: variables,
				sendErr:   sendErr,
			})
			if sendErr != nil {
				logger.Warnw("notification_email_send_failed",
					"event_type", payload.EventType,
					"biz_type", payload.BizType,
					"biz_id", payload.BizID,
					"recipient", recipient,
					"error", sendErr,
				)
				if firstErr == nil {
					firstErr = sendErr
				}
			}
		}
	}
	if setting.Channels.Telegram.Enabled && len(setting.Channels.Telegram.Recipients) > 0 {
		message := composeTelegramMessage(title, body)
		for _, recipient := range setting.Channels.Telegram.Recipients {
			var sendErr error
			if s.telegramSender == nil {
				sendErr = ErrNotificationSendFailed
			} else {
				sendErr = s.telegramSender.SendMessage(ctx, recipient, message)
			}
			s.recordSendAttempt(notificationSendAttempt{
				eventType: payload.EventType,
				bizType:   payload.BizType,
				bizID:     payload.BizID,
				channel:   "telegram",
				recipient: recipient,
				locale:    locale,
				title:     title,
				body:      body,
				variables: variables,
				sendErr:   sendErr,
			})
			if sendErr != nil {
				logger.Warnw("notification_telegram_send_failed",
					"event_type", payload.EventType,
					"biz_type", payload.BizType,
					"biz_id", payload.BizID,
					"recipient", recipient,
					"error", sendErr,
				)
				if firstErr == nil {
					firstErr = sendErr
				}
			}
		}
	}
	if firstErr != nil {
		return fmt.Errorf("%w: %v", ErrNotificationSendFailed, firstErr)
	}
	return nil
}

type notificationSendAttempt struct {
	eventType string
	bizType   string
	bizID     uint
	channel   string
	recipient string
	locale    string
	title     string
	body      string
	variables map[string]interface{}
	isTest    bool
	sendErr   error
}

func (s *NotificationService) recordSendAttempt(attempt notificationSendAttempt) {
	if s == nil || s.logService == nil {
		return
	}
	status := notificationLogStatusSuccess
	errMessage := ""
	if attempt.sendErr != nil {
		status = notificationLogStatusFailed
		errMessage = attempt.sendErr.Error()
	}
	if err := s.logService.Record(NotificationLogRecordInput{
		EventType:    attempt.eventType,
		BizType:      attempt.bizType,
		BizID:        attempt.bizID,
		Channel:      attempt.channel,
		Recipient:    attempt.recipient,
		Locale:       attempt.locale,
		Title:        attempt.title,
		Body:         attempt.body,
		Status:       status,
		ErrorMessage: errMessage,
		IsTest:       attempt.isTest,
		Variables:    notificationVariablesToJSON(attempt.variables),
	}); err != nil {
		logger.Warnw("notification_log_record_failed",
			"event_type", attempt.eventType,
			"biz_type", attempt.bizType,
			"biz_id", attempt.bizID,
			"channel", attempt.channel,
			"recipient", attempt.recipient,
			"error", err,
		)
	}
}

func notificationVariablesToJSON(data map[string]interface{}) models.JSON {
	if len(data) == 0 {
		return models.JSON{}
	}
	result := make(models.JSON, len(data))
	for key, value := range data {
		result[key] = value
	}
	return result
}

func isNotificationEventSupported(eventType string) bool {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case constants.NotificationEventWalletRechargeSuccess,
		constants.NotificationEventOrderPaidSuccess,
		constants.NotificationEventManualFulfillmentPending,
		constants.NotificationEventExceptionAlert,
		constants.NotificationEventExceptionAlertCheck:
		return true
	default:
		return false
	}
}

func acquireNotificationDedupe(ctx context.Context, ttlSeconds int, payload queue.NotificationDispatchPayload) (bool, error) {
	if ttlSeconds <= 0 {
		ttlSeconds = 300
	}
	key := buildNotificationDedupeKey(payload)
	return cache.SetNX(ctx, key, "1", time.Duration(ttlSeconds)*time.Second)
}

func buildNotificationDedupeKey(payload queue.NotificationDispatchPayload) string {
	signature := strings.Builder{}
	signature.WriteString(strings.ToLower(strings.TrimSpace(payload.EventType)))
	signature.WriteString("|")
	signature.WriteString(strings.ToLower(strings.TrimSpace(payload.BizType)))
	signature.WriteString("|")
	signature.WriteString(fmt.Sprintf("%d", payload.BizID))
	signature.WriteString("|")

	keys := make([]string, 0, len(payload.Data))
	for key := range payload.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == "occurred_at" {
			continue
		}
		signature.WriteString(key)
		signature.WriteString("=")
		signature.WriteString(strings.TrimSpace(fmt.Sprintf("%v", payload.Data[key])))
		signature.WriteString(";")
	}
	hash := sha1.Sum([]byte(signature.String()))
	return "notification:dedupe:" + hex.EncodeToString(hash[:])
}

func buildNotificationTemplateVariables(payload queue.NotificationDispatchPayload) map[string]interface{} {
	data := cloneNotificationVariables(payload.Data)
	if data == nil {
		data = map[string]interface{}{}
	}
	data["event_type"] = strings.ToLower(strings.TrimSpace(payload.EventType))
	data["biz_type"] = strings.TrimSpace(payload.BizType)
	data["biz_id"] = fmt.Sprintf("%d", payload.BizID)
	data["occurred_at"] = time.Now().Format("2006-01-02 15:04:05")
	return data
}

func renderNotificationTemplate(tmpl string, variables map[string]interface{}) string {
	return renderTemplate(tmpl, variables)
}

func resolveNotificationLocale(locale, fallback string) string {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = strings.TrimSpace(fallback)
	}
	if _, ok := notificationSupportedLocales[locale]; ok {
		return locale
	}
	return constants.LocaleZhCN
}

func composeTelegramMessage(title, body string) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return body
	}
	if body == "" {
		return title
	}
	return title + "\n\n" + body
}

func notificationJSONToMap(data models.JSON) map[string]interface{} {
	if data == nil {
		return map[string]interface{}{}
	}
	result := make(map[string]interface{}, len(data))
	for key, value := range data {
		result[key] = value
	}
	return result
}

func cloneNotificationVariables(data map[string]interface{}) map[string]interface{} {
	if len(data) == 0 {
		return map[string]interface{}{}
	}
	result := make(map[string]interface{}, len(data))
	for key, value := range data {
		result[key] = value
	}
	return result
}

func thresholdValueByAlertType(setting DashboardAlertSetting, alertType string) int64 {
	switch alertType {
	case constants.NotificationAlertTypeOutOfStockProducts:
		return setting.OutOfStockProductsThreshold
	case constants.NotificationAlertTypeLowStockProducts:
		return setting.LowStockThreshold
	case constants.NotificationAlertTypePendingOrders:
		return setting.PendingPaymentOrdersThreshold
	case constants.NotificationAlertTypePaymentsFailed:
		return setting.PaymentsFailedThreshold
	default:
		return 0
	}
}

func alertTypeLabelByType(locale, alertType string) string {
	type labels struct{ zhCN, zhTW, enUS string }
	m := map[string]labels{
		constants.NotificationAlertTypeOutOfStockProducts: {"????", "????", "Out of Stock"},
		constants.NotificationAlertTypeLowStockProducts:   {"?????", "?????", "Low Stock"},
		constants.NotificationAlertTypePendingOrders:      {"?????", "?????", "Pending Payment"},
		constants.NotificationAlertTypePaymentsFailed:     {"????", "????", "Payment Failed"},
	}
	l, ok := m[alertType]
	if !ok {
		return alertType
	}
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "zh-tw":
		return l.zhTW
	case "en-us", "en":
		return l.enUS
	default:
		return l.zhCN
	}
}

func thresholdMessageByAlertType(alertType string) string {
	switch alertType {
	case constants.NotificationAlertTypeOutOfStockProducts:
		return "????????????"
	case constants.NotificationAlertTypeLowStockProducts:
		return "???????????,?????"
	case constants.NotificationAlertTypePendingOrders:
		return "?????????????"
	case constants.NotificationAlertTypePaymentsFailed:
		return "????????????"
	default:
		return "????????"
	}
}

func isInventoryAlertType(alertType string) bool {
	switch strings.ToLower(strings.TrimSpace(alertType)) {
	case constants.NotificationAlertTypeOutOfStockProducts, constants.NotificationAlertTypeLowStockProducts:
		return true
	default:
		return false
	}
}

func acquireInventoryAlertInterval(ctx context.Context, intervalSeconds int, payload queue.NotificationDispatchPayload) (bool, error) {
	alertType := resolveInventoryAlertTypeKey(payload.Data)
	if !isInventoryAlertType(alertType) {
		return true, nil
	}
	intervalSeconds = normalizeNotificationInventoryAlertInterval(intervalSeconds)
	key := "notification:inventory_interval:" + alertType
	return cache.SetNX(ctx, key, "1", time.Duration(intervalSeconds)*time.Second)
}

func resolveInventoryAlertTypeKey(data map[string]interface{}) string {
	if len(data) == 0 {
		return ""
	}
	if key := normalizeInventoryAlertTypeKey(fmt.Sprintf("%v", data["alert_type_key"])); key != "" {
		return key
	}
	return normalizeInventoryAlertTypeKey(fmt.Sprintf("%v", data["alert_type"]))
}

func normalizeInventoryAlertTypeKey(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	inventoryAlertTypes := []string{
		constants.NotificationAlertTypeOutOfStockProducts,
		constants.NotificationAlertTypeLowStockProducts,
	}
	locales := []string{
		constants.LocaleZhCN,
		constants.LocaleZhTW,
		constants.LocaleEnUS,
	}

	for _, alertType := range inventoryAlertTypes {
		if normalized == strings.ToLower(strings.TrimSpace(alertType)) {
			return alertType
		}
		for _, locale := range locales {
			label := strings.ToLower(strings.TrimSpace(alertTypeLabelByType(locale, alertType)))
			if normalized == label {
				return alertType
			}
		}
	}
	return ""
}

func buildInventoryAlertDispatchPayloads(
	setting NotificationCenterSetting,
	dashboardSetting DashboardSetting,
	payload queue.NotificationDispatchPayload,
	rows []repository.DashboardInventoryAlertRow,
) []queue.NotificationDispatchPayload {
	filtered := filterInventoryAlertRows(rows, setting.IgnoredProductIDs)
	if len(filtered) == 0 {
		return nil
	}

	locale := resolveNotificationLocale(payload.Locale, setting.DefaultLocale)
	groups := map[string][]repository.DashboardInventoryAlertRow{
		constants.NotificationAlertTypeOutOfStockProducts: {},
		constants.NotificationAlertTypeLowStockProducts:   {},
	}
	for _, row := range filtered {
		alertType := strings.ToLower(strings.TrimSpace(row.AlertType))
		if !isInventoryAlertType(alertType) {
			continue
		}
		groups[alertType] = append(groups[alertType], row)
	}

	result := make([]queue.NotificationDispatchPayload, 0, 2)
	for _, alertType := range []string{constants.NotificationAlertTypeOutOfStockProducts, constants.NotificationAlertTypeLowStockProducts} {
		group := groups[alertType]
		if len(group) == 0 {
			continue
		}

		productCount := inventoryAlertUniqueProductCount(group)
		if alertType == constants.NotificationAlertTypeOutOfStockProducts && int64(productCount) < thresholdValueByAlertType(dashboardSetting.Alert, alertType) {
			continue
		}

		data := cloneNotificationVariables(payload.Data)
		if data == nil {
			data = map[string]interface{}{}
		}
		data["alert_type"] = alertTypeLabelByType(locale, alertType)
		data["alert_type_label"] = data["alert_type"]
		data["alert_type_key"] = alertType
		data["alert_level"] = inventoryAlertLevel(alertType)
		data["alert_value"] = fmt.Sprintf("%d", len(group))
		data["alert_threshold"] = fmt.Sprintf("%d", thresholdValueByAlertType(dashboardSetting.Alert, alertType))
		data["affected_items_count"] = fmt.Sprintf("%d", len(group))
		data["affected_product_count"] = fmt.Sprintf("%d", productCount)
		data["affected_items_summary"] = buildInventoryAlertSummary(locale, group)
		data["inventory_alert_scope"] = "inventory"
		data["message"] = buildInventoryAlertMessage(locale, alertType, productCount, len(group), setting.InventoryAlertIntervalSeconds)

		itemPayload := payload
		itemPayload.EventType = constants.NotificationEventExceptionAlert
		itemPayload.BizType = constants.NotificationBizTypeDashboardAlert
		itemPayload.Data = data
		result = append(result, itemPayload)
	}
	return result
}

func filterInventoryAlertRows(rows []repository.DashboardInventoryAlertRow, ignoredProductIDs []uint) []repository.DashboardInventoryAlertRow {
	if len(rows) == 0 {
		return nil
	}
	if len(ignoredProductIDs) == 0 {
		return append([]repository.DashboardInventoryAlertRow(nil), rows...)
	}
	ignored := make(map[uint]struct{}, len(ignoredProductIDs))
	for _, id := range ignoredProductIDs {
		if id == 0 {
			continue
		}
		ignored[id] = struct{}{}
	}
	result := make([]repository.DashboardInventoryAlertRow, 0, len(rows))
	for _, row := range rows {
		if _, skip := ignored[row.ProductID]; skip {
			continue
		}
		result = append(result, row)
	}
	return result
}

func inventoryAlertUniqueProductCount(rows []repository.DashboardInventoryAlertRow) int {
	seen := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		if row.ProductID == 0 {
			continue
		}
		seen[row.ProductID] = struct{}{}
	}
	return len(seen)
}

func inventoryAlertLevel(alertType string) string {
	switch strings.ToLower(strings.TrimSpace(alertType)) {
	case constants.NotificationAlertTypeOutOfStockProducts:
		return "error"
	default:
		return "warning"
	}
}

func buildInventoryAlertSummary(locale string, rows []repository.DashboardInventoryAlertRow) string {
	if len(rows) == 0 {
		return ""
	}
	sortedRows := append([]repository.DashboardInventoryAlertRow(nil), rows...)
	sort.SliceStable(sortedRows, func(i, j int) bool {
		if sortedRows[i].ProductID != sortedRows[j].ProductID {
			return sortedRows[i].ProductID < sortedRows[j].ProductID
		}
		if sortedRows[i].SKUID != sortedRows[j].SKUID {
			return sortedRows[i].SKUID < sortedRows[j].SKUID
		}
		return strings.Compare(sortedRows[i].AlertType, sortedRows[j].AlertType) < 0
	})

	lines := make([]string, 0, len(sortedRows))
	for idx, row := range sortedRows {
		title := resolveNotificationLocalizedJSON(row.ProductTitleJSON, locale, constants.LocaleZhCN)
		if title == "" {
			title = localizedNotificationText(locale, "?????", "?????", "Unnamed item")
		}
		skuText := buildInventoryAlertSKUSummary(row, locale)
		fulfillmentLabel := notificationFulfillmentLabel(locale, row.FulfillmentType)
		statusLabel := inventoryAlertStatusLabel(locale, row.AlertType)

		line := fmt.Sprintf("%d. %s", idx+1, title)
		if skuText != "" {
			line += " / " + skuText
		}
		if fulfillmentLabel != "" {
			line += " [" + fulfillmentLabel + "]"
		}
		line += localizedNotificationText(
			locale,
			fmt.Sprintf(" ?? %d(%s)", row.AvailableStock, statusLabel),
			fmt.Sprintf(" ?? %d(%s)", row.AvailableStock, statusLabel),
			fmt.Sprintf(" | Remaining %d (%s)", row.AvailableStock, statusLabel),
		)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func buildInventoryAlertSKUSummary(row repository.DashboardInventoryAlertRow, locale string) string {
	specText := notificationInterfaceText(row.SKUSpecValuesJSON, locale, constants.LocaleZhCN)
	if specText != "" {
		return specText
	}
	code := strings.TrimSpace(row.SKUCode)
	if code == "" || strings.EqualFold(code, models.DefaultSKUCode) {
		return ""
	}
	return code
}

func inventoryAlertStatusLabel(locale, alertType string) string {
	switch strings.ToLower(strings.TrimSpace(alertType)) {
	case constants.NotificationAlertTypeOutOfStockProducts:
		return localizedNotificationText(locale, "??", "??", "Out of stock")
	default:
		return localizedNotificationText(locale, "???", "???", "Low stock")
	}
}

func buildInventoryAlertMessage(locale, alertType string, productCount, itemCount, intervalSeconds int) string {
	intervalText := formatInventoryAlertInterval(locale, intervalSeconds)
	switch strings.ToLower(strings.TrimSpace(alertType)) {
	case constants.NotificationAlertTypeOutOfStockProducts:
		return localizedNotificationText(
			locale,
			fmt.Sprintf("??? %d ? SKU ??(?? %d ???);?????? %s ?????", itemCount, productCount, intervalText),
			fmt.Sprintf("??? %d ? SKU ??(?? %d ???);?????? %s ?????", itemCount, productCount, intervalText),
			fmt.Sprintf("Detected %d out-of-stock SKUs across %d products; this alert is sent at most once every %s.", itemCount, productCount, intervalText),
		)
	default:
		return localizedNotificationText(
			locale,
			fmt.Sprintf("??? %d ? SKU ???(?? %d ???);?????? %s ?????", itemCount, productCount, intervalText),
			fmt.Sprintf("??? %d ? SKU ???(?? %d ???);?????? %s ?????", itemCount, productCount, intervalText),
			fmt.Sprintf("Detected %d low-stock SKUs across %d products; this alert is sent at most once every %s.", itemCount, productCount, intervalText),
		)
	}
}

func formatInventoryAlertInterval(locale string, seconds int) string {
	seconds = normalizeNotificationInventoryAlertInterval(seconds)
	switch {
	case seconds%3600 == 0:
		hours := seconds / 3600
		return localizedNotificationText(locale, fmt.Sprintf("%d ??", hours), fmt.Sprintf("%d ??", hours), fmt.Sprintf("%d hours", hours))
	case seconds%60 == 0:
		minutes := seconds / 60
		return localizedNotificationText(locale, fmt.Sprintf("%d ??", minutes), fmt.Sprintf("%d ??", minutes), fmt.Sprintf("%d minutes", minutes))
	default:
		return localizedNotificationText(locale, fmt.Sprintf("%d ?", seconds), fmt.Sprintf("%d ?", seconds), fmt.Sprintf("%d seconds", seconds))
	}
}

func pickNotificationMessage(value interface{}, fallback string) string {
	normalized := strings.TrimSpace(fmt.Sprintf("%v", value))
	if normalized == "" || normalized == "<nil>" {
		return fallback
	}
	return normalized
}

func applyNotificationTestVariables(target map[string]interface{}, defaults map[string]interface{}) {
	if target == nil || len(defaults) == 0 {
		return
	}
	for key, value := range defaults {
		if _, exists := target[key]; exists {
			continue
		}
		target[key] = value
	}
}

func buildNotificationTestVariables(scene, locale string) map[string]interface{} {
	locale = resolveNotificationLocale(locale, constants.LocaleZhCN)
	switch strings.ToLower(strings.TrimSpace(scene)) {
	case constants.NotificationEventWalletRechargeSuccess:
		return map[string]interface{}{
			"customer_label":  localizedNotificationText(locale, "?? <zhangsan@example.com>", "?? <zhangsan@example.com>", "Alex Zhang <zhangsan@example.com>"),
			"customer_email":  "zhangsan@example.com",
			"recharge_no":     "RC202603230001",
			"amount":          "100.00",
			"currency":        "USD",
			"payment_channel": "epay/alipay",
		}
	case constants.NotificationEventOrderPaidSuccess:
		return map[string]interface{}{
			"customer_label":            localizedNotificationText(locale, "?? <zhangsan@example.com>", "?? <zhangsan@example.com>", "Alex Zhang <zhangsan@example.com>"),
			"customer_email":            "zhangsan@example.com",
			"order_no":                  "DJ202603230001",
			"amount":                    "299.00",
			"currency":                  "USD",
			"payment_channel":           "epay/alipay",
			"items_summary":             buildNotificationTestOrderItems(locale),
			"fulfillment_items_summary": buildNotificationTestFulfillmentItems(locale),
			"delivery_summary":          buildNotificationDeliverySummary(locale, notificationOrderItemCounts{Total: 2, Auto: 1, Manual: 1}),
		}
	case constants.NotificationEventManualFulfillmentPending:
		return map[string]interface{}{
			"customer_label":            localizedNotificationText(locale, "?? <zhangsan@example.com>", "?? <zhangsan@example.com>", "Alex Zhang <zhangsan@example.com>"),
			"customer_email":            "zhangsan@example.com",
			"order_no":                  "DJ202603230001",
			"order_status":              constants.OrderStatusPaid,
			"fulfillment_items_summary": buildNotificationTestFulfillmentItems(locale),
			"delivery_summary":          buildNotificationDeliverySummary(locale, notificationOrderItemCounts{Total: 2, Auto: 1, Manual: 1}),
		}
	default:
		return map[string]interface{}{
			"alert_type":             alertTypeLabelByType(locale, constants.NotificationAlertTypeLowStockProducts),
			"alert_type_label":       alertTypeLabelByType(locale, constants.NotificationAlertTypeLowStockProducts),
			"alert_level":            "warning",
			"alert_value":            "2",
			"alert_threshold":        "5",
			"affected_items_count":   "2",
			"affected_product_count": "2",
			"affected_items_summary": buildNotificationTestInventoryItems(locale),
			"message": localizedNotificationText(
				locale,
				"??? 2 ??????,?? 2 ??????;?????? 30 ???????",
				"??? 2 ??????,?? 2 ??????;?????? 30 ???????",
				"Detected 2 low-stock products across 2 inventory items; this alert is sent at most once every 30 minutes.",
			),
		}
	}
}

func buildNotificationTestOrderItems(locale string) string {
	return strings.Join([]string{
		localizedNotificationText(locale, "1. Netflix ?? / ??: HK x1 [????]", "1. Netflix ?? / ??: HK x1 [????]", "1. Netflix Annual / Region: HK x1 [Auto]"),
		localizedNotificationText(locale, "2. ChatGPT Plus ?? / ??: 1?? x1 [????]", "2. ChatGPT Plus ?? / ??: 1?? x1 [????]", "2. ChatGPT Plus Recharge / Cycle: 1 month x1 [Manual]"),
	}, "\n")
}

func buildNotificationTestFulfillmentItems(locale string) string {
	return localizedNotificationText(
		locale,
		"1. ChatGPT Plus ?? / ??: 1?? x1 [????]",
		"1. ChatGPT Plus ?? / ??: 1?? x1 [????]",
		"1. ChatGPT Plus Recharge / Cycle: 1 month x1 [Manual]",
	)
}

func buildNotificationTestInventoryItems(locale string) string {
	return strings.Join([]string{
		localizedNotificationText(locale, "1. Netflix ?? / ??: HK [????] ?? 1(???)", "1. Netflix ?? / ??: HK [????] ?? 1(???)", "1. Netflix Annual / Region: HK [Auto] | Remaining 1 (Low stock)"),
		localizedNotificationText(locale, "2. ChatGPT Plus ?? / ??: 1?? [????] ?? 0(??)", "2. ChatGPT Plus ?? / ??: 1?? [????] ?? 0(??)", "2. ChatGPT Plus Recharge / Cycle: 1 month [Manual] | Remaining 0 (Out of stock)"),
	}, "\n")
}
