package service

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
)

var notificationSupportedLocales = map[string]struct{}{
	constants.LocaleZhCN: {},
	constants.LocaleZhTW: {},
	constants.LocaleEnUS: {},
}

var telegramChatIDPattern = regexp.MustCompile(`^-?\d{5,20}$`)

const (
	notificationInventoryAlertIntervalDefaultSeconds = 1800
	notificationInventoryAlertIntervalMinSeconds     = 60
	notificationInventoryAlertIntervalMaxSeconds     = 604800
)

// NotificationChannelSetting ??????
type NotificationChannelSetting struct {
	Enabled    bool     `json:"enabled"`
	Recipients []string `json:"recipients"`
}

// NotificationChannelsSetting ??????
type NotificationChannelsSetting struct {
	Email    NotificationChannelSetting `json:"email"`
	Telegram NotificationChannelSetting `json:"telegram"`
}

// NotificationSceneSetting ??????
type NotificationSceneSetting struct {
	WalletRechargeSuccess    bool `json:"wallet_recharge_success"`
	OrderPaidSuccess         bool `json:"order_paid_success"`
	ManualFulfillmentPending bool `json:"manual_fulfillment_pending"`
	ExceptionAlert           bool `json:"exception_alert"`
}

// NotificationLocalizedTemplate ???????
type NotificationLocalizedTemplate struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// NotificationSceneTemplate ????????
type NotificationSceneTemplate struct {
	ZHCN NotificationLocalizedTemplate `json:"zh-CN"`
	ZHTW NotificationLocalizedTemplate `json:"zh-TW"`
	ENUS NotificationLocalizedTemplate `json:"en-US"`
}

// NotificationTemplatesSetting ??????
type NotificationTemplatesSetting struct {
	WalletRechargeSuccess    NotificationSceneTemplate `json:"wallet_recharge_success"`
	OrderPaidSuccess         NotificationSceneTemplate `json:"order_paid_success"`
	ManualFulfillmentPending NotificationSceneTemplate `json:"manual_fulfillment_pending"`
	ExceptionAlert           NotificationSceneTemplate `json:"exception_alert"`
}

// NotificationCenterSetting ??????
type NotificationCenterSetting struct {
	DefaultLocale                 string                       `json:"default_locale"`
	Channels                      NotificationChannelsSetting  `json:"channels"`
	Scenes                        NotificationSceneSetting     `json:"scenes"`
	Templates                     NotificationTemplatesSetting `json:"templates"`
	DedupeTTLSeconds              int                          `json:"dedupe_ttl_seconds"`
	InventoryAlertIntervalSeconds int                          `json:"inventory_alert_interval_seconds"`
	IgnoredProductIDs             []uint                       `json:"ignored_product_ids"`
}

// NotificationCenterSettingPatch ????????
type NotificationCenterSettingPatch struct {
	DefaultLocale                 *string                     `json:"default_locale"`
	Channels                      *NotificationChannelsPatch  `json:"channels"`
	Scenes                        *NotificationScenePatch     `json:"scenes"`
	Templates                     *NotificationTemplatesPatch `json:"templates"`
	DedupeTTLSeconds              *int                        `json:"dedupe_ttl_seconds"`
	InventoryAlertIntervalSeconds *int                        `json:"inventory_alert_interval_seconds"`
	IgnoredProductIDs             *[]uint                     `json:"ignored_product_ids"`
}

// NotificationChannelsPatch ??????
type NotificationChannelsPatch struct {
	Email    *NotificationChannelPatch `json:"email"`
	Telegram *NotificationChannelPatch `json:"telegram"`
}

// NotificationChannelPatch ??????
type NotificationChannelPatch struct {
	Enabled    *bool     `json:"enabled"`
	Recipients *[]string `json:"recipients"`
}

// NotificationScenePatch ??????
type NotificationScenePatch struct {
	WalletRechargeSuccess    *bool `json:"wallet_recharge_success"`
	OrderPaidSuccess         *bool `json:"order_paid_success"`
	ManualFulfillmentPending *bool `json:"manual_fulfillment_pending"`
	ExceptionAlert           *bool `json:"exception_alert"`
}

// NotificationTemplatesPatch ??????
type NotificationTemplatesPatch struct {
	WalletRechargeSuccess    *NotificationSceneTemplatePatch `json:"wallet_recharge_success"`
	OrderPaidSuccess         *NotificationSceneTemplatePatch `json:"order_paid_success"`
	ManualFulfillmentPending *NotificationSceneTemplatePatch `json:"manual_fulfillment_pending"`
	ExceptionAlert           *NotificationSceneTemplatePatch `json:"exception_alert"`
}

// NotificationSceneTemplatePatch ???????
type NotificationSceneTemplatePatch struct {
	ZHCN *NotificationLocalizedTemplatePatch `json:"zh-CN"`
	ZHTW *NotificationLocalizedTemplatePatch `json:"zh-TW"`
	ENUS *NotificationLocalizedTemplatePatch `json:"en-US"`
}

// NotificationLocalizedTemplatePatch ???????
type NotificationLocalizedTemplatePatch struct {
	Title *string `json:"title"`
	Body  *string `json:"body"`
}

// NotificationCenterDefaultSetting ????????
func NotificationCenterDefaultSetting() NotificationCenterSetting {
	return NormalizeNotificationCenterSetting(NotificationCenterSetting{
		DefaultLocale: constants.LocaleZhCN,
		Channels: NotificationChannelsSetting{
			Email: NotificationChannelSetting{
				Enabled:    false,
				Recipients: []string{},
			},
			Telegram: NotificationChannelSetting{
				Enabled:    false,
				Recipients: []string{},
			},
		},
		Scenes: NotificationSceneSetting{
			WalletRechargeSuccess:    true,
			OrderPaidSuccess:         true,
			ManualFulfillmentPending: true,
			ExceptionAlert:           true,
		},
		Templates: NotificationTemplatesSetting{
			WalletRechargeSuccess: NotificationSceneTemplate{
				ZHCN: NotificationLocalizedTemplate{
					Title: "????????",
					Body:  "??:{{customer_label}}\n??:{{customer_email}}\n????:{{recharge_no}}\n????:{{amount}} {{currency}}\n????:{{payment_channel}}",
				},
				ZHTW: NotificationLocalizedTemplate{
					Title: "????????",
					Body:  "??:{{customer_label}}\n??:{{customer_email}}\n????:{{recharge_no}}\n????:{{amount}} {{currency}}\n????:{{payment_channel}}",
				},
				ENUS: NotificationLocalizedTemplate{
					Title: "Wallet Recharge Succeeded",
					Body:  "Customer: {{customer_label}}\nEmail: {{customer_email}}\nRecharge No: {{recharge_no}}\nAmount: {{amount}} {{currency}}\nChannel: {{payment_channel}}",
				},
			},
			OrderPaidSuccess: NotificationSceneTemplate{
				ZHCN: NotificationLocalizedTemplate{
					Title: "????????",
					Body:  "???:{{customer_label}}\n??:{{customer_email}}\n???:{{order_no}}\n????:{{amount}} {{currency}}\n????:{{payment_channel}}\n????:\n{{items_summary}}\n????:{{delivery_summary}}",
				},
				ZHTW: NotificationLocalizedTemplate{
					Title: "????????",
					Body:  "???:{{customer_label}}\n??:{{customer_email}}\n???:{{order_no}}\n????:{{amount}} {{currency}}\n????:{{payment_channel}}\n????:\n{{items_summary}}\n????:{{delivery_summary}}",
				},
				ENUS: NotificationLocalizedTemplate{
					Title: "Order Payment Succeeded",
					Body:  "Customer: {{customer_label}}\nEmail: {{customer_email}}\nOrder No: {{order_no}}\nAmount: {{amount}} {{currency}}\nChannel: {{payment_channel}}\nItems:\n{{items_summary}}\nDelivery Summary: {{delivery_summary}}",
				},
			},
			ManualFulfillmentPending: NotificationSceneTemplate{
				ZHCN: NotificationLocalizedTemplate{
					Title: "?????????",
					Body:  "???:{{customer_label}}\n??:{{customer_email}}\n???:{{order_no}}\n????:{{order_status}}\n?????:\n{{fulfillment_items_summary}}\n????:{{delivery_summary}}",
				},
				ZHTW: NotificationLocalizedTemplate{
					Title: "?????????",
					Body:  "???:{{customer_label}}\n??:{{customer_email}}\n???:{{order_no}}\n????:{{order_status}}\n?????:\n{{fulfillment_items_summary}}\n????:{{delivery_summary}}",
				},
				ENUS: NotificationLocalizedTemplate{
					Title: "Manual Fulfillment Required",
					Body:  "Customer: {{customer_label}}\nEmail: {{customer_email}}\nOrder No: {{order_no}}\nOrder Status: {{order_status}}\nPending Items:\n{{fulfillment_items_summary}}\nDelivery Summary: {{delivery_summary}}",
				},
			},
			ExceptionAlert: NotificationSceneTemplate{
				ZHCN: NotificationLocalizedTemplate{
					Title: "??????",
					Body:  "????:{{alert_type}}\n????:{{alert_level}}\n???:{{alert_value}}\n??:{{alert_threshold}}\n??:{{message}}\n{{affected_items_summary}}",
				},
				ZHTW: NotificationLocalizedTemplate{
					Title: "??????",
					Body:  "????:{{alert_type}}\n????:{{alert_level}}\n???:{{alert_value}}\n??:{{alert_threshold}}\n??:{{message}}\n{{affected_items_summary}}",
				},
				ENUS: NotificationLocalizedTemplate{
					Title: "System Exception Alert",
					Body:  "Type: {{alert_type}}\nLevel: {{alert_level}}\nCurrent: {{alert_value}}\nThreshold: {{alert_threshold}}\nDetails: {{message}}\n{{affected_items_summary}}",
				},
			},
		},
		DedupeTTLSeconds:              300,
		InventoryAlertIntervalSeconds: notificationInventoryAlertIntervalDefaultSeconds,
		IgnoredProductIDs:             []uint{},
	})
}

// NormalizeNotificationCenterSetting ?????????
func NormalizeNotificationCenterSetting(setting NotificationCenterSetting) NotificationCenterSetting {
	setting.DefaultLocale = normalizeNotificationLocale(setting.DefaultLocale)
	setting.Channels.Email.Recipients = normalizeEmailRecipients(setting.Channels.Email.Recipients)
	setting.Channels.Telegram.Recipients = normalizeTelegramRecipients(setting.Channels.Telegram.Recipients)
	setting.DedupeTTLSeconds = normalizeNotificationDedupeTTL(setting.DedupeTTLSeconds)
	setting.InventoryAlertIntervalSeconds = normalizeNotificationInventoryAlertInterval(setting.InventoryAlertIntervalSeconds)
	setting.IgnoredProductIDs = normalizeNotificationIgnoredProductIDs(setting.IgnoredProductIDs)
	setting.Templates = normalizeNotificationTemplates(setting.Templates)
	return setting
}

// ValidateNotificationCenterSetting ????????
func ValidateNotificationCenterSetting(setting NotificationCenterSetting) error {
	normalized := NormalizeNotificationCenterSetting(setting)
	if _, ok := notificationSupportedLocales[normalized.DefaultLocale]; !ok {
		return fmt.Errorf("%w: ???????", ErrNotificationConfigInvalid)
	}

	if normalized.Channels.Email.Enabled && len(normalized.Channels.Email.Recipients) == 0 {
		return fmt.Errorf("%w: ???????????????", ErrNotificationConfigInvalid)
	}
	if normalized.Channels.Telegram.Enabled && len(normalized.Channels.Telegram.Recipients) == 0 {
		return fmt.Errorf("%w: Telegram ????????????ID", ErrNotificationConfigInvalid)
	}
	if normalized.Channels.Email.Enabled {
		for _, recipient := range normalized.Channels.Email.Recipients {
			if _, err := mail.ParseAddress(recipient); err != nil {
				return fmt.Errorf("%w: ??????????", ErrNotificationConfigInvalid)
			}
		}
	}
	if normalized.Channels.Telegram.Enabled {
		for _, recipient := range normalized.Channels.Telegram.Recipients {
			if !telegramChatIDPattern.MatchString(recipient) {
				return fmt.Errorf("%w: Telegram ???ID?????", ErrNotificationConfigInvalid)
			}
		}
	}

	if normalized.DedupeTTLSeconds < 30 || normalized.DedupeTTLSeconds > 86400 {
		return fmt.Errorf("%w: ?????? 30-86400 ???", ErrNotificationConfigInvalid)
	}
	if normalized.InventoryAlertIntervalSeconds < notificationInventoryAlertIntervalMinSeconds || normalized.InventoryAlertIntervalSeconds > notificationInventoryAlertIntervalMaxSeconds {
		return fmt.Errorf("%w: ???????? 60-604800 ???", ErrNotificationConfigInvalid)
	}
	return nil
}

// NotificationCenterSettingToMap ????????? settings ????
func NotificationCenterSettingToMap(setting NotificationCenterSetting) map[string]interface{} {
	normalized := NormalizeNotificationCenterSetting(setting)
	return map[string]interface{}{
		"default_locale": normalized.DefaultLocale,
		"channels": map[string]interface{}{
			"email": map[string]interface{}{
				"enabled":    normalized.Channels.Email.Enabled,
				"recipients": cloneStringSlice(normalized.Channels.Email.Recipients),
			},
			"telegram": map[string]interface{}{
				"enabled":    normalized.Channels.Telegram.Enabled,
				"recipients": cloneStringSlice(normalized.Channels.Telegram.Recipients),
			},
		},
		"scenes": map[string]interface{}{
			"wallet_recharge_success":    normalized.Scenes.WalletRechargeSuccess,
			"order_paid_success":         normalized.Scenes.OrderPaidSuccess,
			"manual_fulfillment_pending": normalized.Scenes.ManualFulfillmentPending,
			"exception_alert":            normalized.Scenes.ExceptionAlert,
		},
		"templates": map[string]interface{}{
			"wallet_recharge_success":    notificationSceneTemplateToMap(normalized.Templates.WalletRechargeSuccess),
			"order_paid_success":         notificationSceneTemplateToMap(normalized.Templates.OrderPaidSuccess),
			"manual_fulfillment_pending": notificationSceneTemplateToMap(normalized.Templates.ManualFulfillmentPending),
			"exception_alert":            notificationSceneTemplateToMap(normalized.Templates.ExceptionAlert),
		},
		"dedupe_ttl_seconds":               normalized.DedupeTTLSeconds,
		"inventory_alert_interval_seconds": normalized.InventoryAlertIntervalSeconds,
		"ignored_product_ids":              cloneUintSlice(normalized.IgnoredProductIDs),
	}
}

// MaskNotificationCenterSettingForAdmin ?????????
func MaskNotificationCenterSettingForAdmin(setting NotificationCenterSetting) models.JSON {
	normalized := NormalizeNotificationCenterSetting(setting)
	return models.JSON(NotificationCenterSettingToMap(normalized))
}

// GetNotificationCenterSetting ????????(?? settings,??????)
func (s *SettingService) GetNotificationCenterSetting() (NotificationCenterSetting, error) {
	fallback := NotificationCenterDefaultSetting()
	value, err := s.GetByKey(constants.SettingKeyNotificationCenterConfig)
	if err != nil {
		return fallback, err
	}
	if value == nil {
		return fallback, nil
	}
	parsed := notificationCenterSettingFromJSON(value, fallback)
	return NormalizeNotificationCenterSetting(parsed), nil
}

// PatchNotificationCenterSetting ????????????
func (s *SettingService) PatchNotificationCenterSetting(patch NotificationCenterSettingPatch) (NotificationCenterSetting, error) {
	current, err := s.GetNotificationCenterSetting()
	if err != nil {
		return NotificationCenterSetting{}, err
	}

	next := current
	if patch.DefaultLocale != nil {
		next.DefaultLocale = strings.TrimSpace(*patch.DefaultLocale)
	}
	if patch.DedupeTTLSeconds != nil {
		next.DedupeTTLSeconds = *patch.DedupeTTLSeconds
	}
	if patch.InventoryAlertIntervalSeconds != nil {
		next.InventoryAlertIntervalSeconds = *patch.InventoryAlertIntervalSeconds
	}
	if patch.IgnoredProductIDs != nil {
		next.IgnoredProductIDs = cloneUintSlice(*patch.IgnoredProductIDs)
	}
	if patch.Channels != nil {
		if patch.Channels.Email != nil {
			if patch.Channels.Email.Enabled != nil {
				next.Channels.Email.Enabled = *patch.Channels.Email.Enabled
			}
			if patch.Channels.Email.Recipients != nil {
				next.Channels.Email.Recipients = cloneStringSlice(*patch.Channels.Email.Recipients)
			}
		}
		if patch.Channels.Telegram != nil {
			if patch.Channels.Telegram.Enabled != nil {
				next.Channels.Telegram.Enabled = *patch.Channels.Telegram.Enabled
			}
			if patch.Channels.Telegram.Recipients != nil {
				next.Channels.Telegram.Recipients = cloneStringSlice(*patch.Channels.Telegram.Recipients)
			}
		}
	}
	if patch.Scenes != nil {
		if patch.Scenes.WalletRechargeSuccess != nil {
			next.Scenes.WalletRechargeSuccess = *patch.Scenes.WalletRechargeSuccess
		}
		if patch.Scenes.OrderPaidSuccess != nil {
			next.Scenes.OrderPaidSuccess = *patch.Scenes.OrderPaidSuccess
		}
		if patch.Scenes.ManualFulfillmentPending != nil {
			next.Scenes.ManualFulfillmentPending = *patch.Scenes.ManualFulfillmentPending
		}
		if patch.Scenes.ExceptionAlert != nil {
			next.Scenes.ExceptionAlert = *patch.Scenes.ExceptionAlert
		}
	}
	if patch.Templates != nil {
		if patch.Templates.WalletRechargeSuccess != nil {
			applyNotificationSceneTemplatePatch(&next.Templates.WalletRechargeSuccess, patch.Templates.WalletRechargeSuccess)
		}
		if patch.Templates.OrderPaidSuccess != nil {
			applyNotificationSceneTemplatePatch(&next.Templates.OrderPaidSuccess, patch.Templates.OrderPaidSuccess)
		}
		if patch.Templates.ManualFulfillmentPending != nil {
			applyNotificationSceneTemplatePatch(&next.Templates.ManualFulfillmentPending, patch.Templates.ManualFulfillmentPending)
		}
		if patch.Templates.ExceptionAlert != nil {
			applyNotificationSceneTemplatePatch(&next.Templates.ExceptionAlert, patch.Templates.ExceptionAlert)
		}
	}

	normalized := NormalizeNotificationCenterSetting(next)
	if err := ValidateNotificationCenterSetting(normalized); err != nil {
		return NotificationCenterSetting{}, err
	}
	if _, err := s.Update(constants.SettingKeyNotificationCenterConfig, NotificationCenterSettingToMap(normalized)); err != nil {
		return NotificationCenterSetting{}, err
	}
	return normalized, nil
}

// IsSceneEnabled ??????????
func (s NotificationSceneSetting) IsSceneEnabled(eventType string) bool {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case constants.NotificationEventWalletRechargeSuccess:
		return s.WalletRechargeSuccess
	case constants.NotificationEventOrderPaidSuccess:
		return s.OrderPaidSuccess
	case constants.NotificationEventManualFulfillmentPending:
		return s.ManualFulfillmentPending
	case constants.NotificationEventExceptionAlert, constants.NotificationEventExceptionAlertCheck:
		return s.ExceptionAlert
	default:
		return false
	}
}

// TemplateByEvent ??????
func (s NotificationTemplatesSetting) TemplateByEvent(eventType string) NotificationSceneTemplate {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case constants.NotificationEventWalletRechargeSuccess:
		return s.WalletRechargeSuccess
	case constants.NotificationEventOrderPaidSuccess:
		return s.OrderPaidSuccess
	case constants.NotificationEventManualFulfillmentPending:
		return s.ManualFulfillmentPending
	case constants.NotificationEventExceptionAlert, constants.NotificationEventExceptionAlertCheck:
		return s.ExceptionAlert
	default:
		return s.ExceptionAlert
	}
}

// ResolveLocaleTemplate ???????
func (s NotificationSceneTemplate) ResolveLocaleTemplate(locale string) NotificationLocalizedTemplate {
	normalized := normalizeNotificationLocale(locale)
	switch normalized {
	case constants.LocaleZhTW:
		return s.ZHTW
	case constants.LocaleEnUS:
		return s.ENUS
	default:
		return s.ZHCN
	}
}

func notificationCenterSettingFromJSON(raw models.JSON, fallback NotificationCenterSetting) NotificationCenterSetting {
	next := fallback
	if raw == nil {
		return next
	}
	legacyEnabled := readBool(raw, "enabled", true)
	next.DefaultLocale = readString(raw, "default_locale", next.DefaultLocale)
	next.DedupeTTLSeconds = readInt(raw, "dedupe_ttl_seconds", next.DedupeTTLSeconds)
	next.InventoryAlertIntervalSeconds = readInt(raw, "inventory_alert_interval_seconds", next.InventoryAlertIntervalSeconds)
	next.IgnoredProductIDs = readUintList(raw, "ignored_product_ids", next.IgnoredProductIDs)

	if channelsMap := toStringAnyMap(raw["channels"]); channelsMap != nil {
		if emailMap := toStringAnyMap(channelsMap["email"]); emailMap != nil {
			next.Channels.Email.Enabled = readBool(emailMap, "enabled", next.Channels.Email.Enabled)
			next.Channels.Email.Recipients = readStringList(emailMap, "recipients", next.Channels.Email.Recipients)
		}
		if telegramMap := toStringAnyMap(channelsMap["telegram"]); telegramMap != nil {
			next.Channels.Telegram.Enabled = readBool(telegramMap, "enabled", next.Channels.Telegram.Enabled)
			next.Channels.Telegram.Recipients = readStringList(telegramMap, "recipients", next.Channels.Telegram.Recipients)
		}
	}

	if scenesMap := toStringAnyMap(raw["scenes"]); scenesMap != nil {
		next.Scenes.WalletRechargeSuccess = readBool(scenesMap, "wallet_recharge_success", next.Scenes.WalletRechargeSuccess)
		next.Scenes.OrderPaidSuccess = readBool(scenesMap, "order_paid_success", next.Scenes.OrderPaidSuccess)
		next.Scenes.ManualFulfillmentPending = readBool(scenesMap, "manual_fulfillment_pending", next.Scenes.ManualFulfillmentPending)
		next.Scenes.ExceptionAlert = readBool(scenesMap, "exception_alert", next.Scenes.ExceptionAlert)
	}

	if templatesMap := toStringAnyMap(raw["templates"]); templatesMap != nil {
		if sceneMap := toStringAnyMap(templatesMap["wallet_recharge_success"]); sceneMap != nil {
			next.Templates.WalletRechargeSuccess = notificationSceneTemplateFromMap(sceneMap, next.Templates.WalletRechargeSuccess)
		}
		if sceneMap := toStringAnyMap(templatesMap["order_paid_success"]); sceneMap != nil {
			next.Templates.OrderPaidSuccess = notificationSceneTemplateFromMap(sceneMap, next.Templates.OrderPaidSuccess)
		}
		if sceneMap := toStringAnyMap(templatesMap["manual_fulfillment_pending"]); sceneMap != nil {
			next.Templates.ManualFulfillmentPending = notificationSceneTemplateFromMap(sceneMap, next.Templates.ManualFulfillmentPending)
		}
		if sceneMap := toStringAnyMap(templatesMap["exception_alert"]); sceneMap != nil {
			next.Templates.ExceptionAlert = notificationSceneTemplateFromMap(sceneMap, next.Templates.ExceptionAlert)
		}
	}
	if !legacyEnabled {
		next.Channels.Email.Enabled = false
		next.Channels.Telegram.Enabled = false
	}

	return next
}

func notificationSceneTemplateFromMap(raw map[string]interface{}, fallback NotificationSceneTemplate) NotificationSceneTemplate {
	next := fallback
	if zhCNMap := toStringAnyMap(raw[constants.LocaleZhCN]); zhCNMap != nil {
		next.ZHCN = notificationLocalizedTemplateFromMap(zhCNMap, next.ZHCN)
	}
	if zhTWMap := toStringAnyMap(raw[constants.LocaleZhTW]); zhTWMap != nil {
		next.ZHTW = notificationLocalizedTemplateFromMap(zhTWMap, next.ZHTW)
	}
	if enUSMap := toStringAnyMap(raw[constants.LocaleEnUS]); enUSMap != nil {
		next.ENUS = notificationLocalizedTemplateFromMap(enUSMap, next.ENUS)
	}
	return next
}

func notificationLocalizedTemplateFromMap(raw map[string]interface{}, fallback NotificationLocalizedTemplate) NotificationLocalizedTemplate {
	next := fallback
	next.Title = readString(raw, "title", next.Title)
	next.Body = readString(raw, "body", next.Body)
	return next
}

func notificationSceneTemplateToMap(template NotificationSceneTemplate) map[string]interface{} {
	return map[string]interface{}{
		constants.LocaleZhCN: map[string]interface{}{
			"title": strings.TrimSpace(template.ZHCN.Title),
			"body":  strings.TrimSpace(template.ZHCN.Body),
		},
		constants.LocaleZhTW: map[string]interface{}{
			"title": strings.TrimSpace(template.ZHTW.Title),
			"body":  strings.TrimSpace(template.ZHTW.Body),
		},
		constants.LocaleEnUS: map[string]interface{}{
			"title": strings.TrimSpace(template.ENUS.Title),
			"body":  strings.TrimSpace(template.ENUS.Body),
		},
	}
}

func normalizeNotificationTemplates(templates NotificationTemplatesSetting) NotificationTemplatesSetting {
	templates.WalletRechargeSuccess = normalizeNotificationSceneTemplate(templates.WalletRechargeSuccess)
	templates.OrderPaidSuccess = normalizeNotificationSceneTemplate(templates.OrderPaidSuccess)
	templates.ManualFulfillmentPending = normalizeNotificationSceneTemplate(templates.ManualFulfillmentPending)
	templates.ExceptionAlert = normalizeNotificationSceneTemplate(templates.ExceptionAlert)
	return templates
}

func normalizeNotificationSceneTemplate(template NotificationSceneTemplate) NotificationSceneTemplate {
	template.ZHCN = normalizeNotificationLocalizedTemplate(template.ZHCN)
	template.ZHTW = normalizeNotificationLocalizedTemplate(template.ZHTW)
	template.ENUS = normalizeNotificationLocalizedTemplate(template.ENUS)
	return template
}

func normalizeNotificationLocalizedTemplate(template NotificationLocalizedTemplate) NotificationLocalizedTemplate {
	template.Title = strings.TrimSpace(template.Title)
	template.Body = strings.TrimSpace(template.Body)
	return template
}

func normalizeNotificationLocale(locale string) string {
	normalized := strings.TrimSpace(locale)
	if _, ok := notificationSupportedLocales[normalized]; ok {
		return normalized
	}
	return constants.LocaleZhCN
}

func normalizeNotificationDedupeTTL(ttl int) int {
	if ttl < 30 || ttl > 86400 {
		return 300
	}
	return ttl
}

func normalizeNotificationInventoryAlertInterval(ttl int) int {
	if ttl < notificationInventoryAlertIntervalMinSeconds || ttl > notificationInventoryAlertIntervalMaxSeconds {
		return notificationInventoryAlertIntervalDefaultSeconds
	}
	return ttl
}

func normalizeEmailRecipients(items []string) []string {
	return normalizeStringList(items, true)
}

func normalizeTelegramRecipients(items []string) []string {
	result := normalizeStringList(items, false)
	filtered := make([]string, 0, len(result))
	for _, item := range result {
		if telegramChatIDPattern.MatchString(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func normalizeStringList(items []string, lower bool) []string {
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := trimmed
		if lower {
			key = strings.ToLower(trimmed)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if lower {
			trimmed = strings.ToLower(trimmed)
		}
		result = append(result, trimmed)
	}
	return result
}

func normalizeNotificationIgnoredProductIDs(items []uint) []uint {
	if len(items) == 0 {
		return []uint{}
	}
	result := make([]uint, 0, len(items))
	seen := make(map[uint]struct{}, len(items))
	for _, item := range items {
		if item == 0 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func applyNotificationSceneTemplatePatch(target *NotificationSceneTemplate, patch *NotificationSceneTemplatePatch) {
	if target == nil || patch == nil {
		return
	}
	applyNotificationLocalizedTemplatePatch(&target.ZHCN, patch.ZHCN)
	applyNotificationLocalizedTemplatePatch(&target.ZHTW, patch.ZHTW)
	applyNotificationLocalizedTemplatePatch(&target.ENUS, patch.ENUS)
}

func applyNotificationLocalizedTemplatePatch(target *NotificationLocalizedTemplate, patch *NotificationLocalizedTemplatePatch) {
	if target == nil || patch == nil {
		return
	}
	if patch.Title != nil {
		target.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Body != nil {
		target.Body = strings.TrimSpace(*patch.Body)
	}
}
