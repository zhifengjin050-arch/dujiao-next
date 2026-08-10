package service

import (
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
)

// OrderEmailLocalizedTemplate ?????????
type OrderEmailLocalizedTemplate struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// OrderEmailSceneTemplate ????????(???)
type OrderEmailSceneTemplate struct {
	ZHCN OrderEmailLocalizedTemplate `json:"zh-CN"`
	ZHTW OrderEmailLocalizedTemplate `json:"zh-TW"`
	ENUS OrderEmailLocalizedTemplate `json:"en-US"`
}

// OrderEmailGuestTip ????(???)
type OrderEmailGuestTip struct {
	ZHCN string `json:"zh-CN"`
	ZHTW string `json:"zh-TW"`
	ENUS string `json:"en-US"`
}

// OrderEmailFulfillmentAttachmentTip ????????(???)
type OrderEmailFulfillmentAttachmentTip struct {
	ZHCN string `json:"zh-CN"`
	ZHTW string `json:"zh-TW"`
	ENUS string `json:"en-US"`
}

// OrderEmailTemplatesSetting ????????????
type OrderEmailTemplatesSetting struct {
	Default              OrderEmailSceneTemplate `json:"default"`
	Paid                 OrderEmailSceneTemplate `json:"paid"`
	Delivered            OrderEmailSceneTemplate `json:"delivered"`
	DeliveredWithContent OrderEmailSceneTemplate `json:"delivered_with_content"`
	Canceled             OrderEmailSceneTemplate `json:"canceled"`
	Refunded             OrderEmailSceneTemplate `json:"refunded"`
	PartiallyRefunded    OrderEmailSceneTemplate `json:"partially_refunded"`
}

// OrderEmailTemplateSetting ????????
type OrderEmailTemplateSetting struct {
	Templates                OrderEmailTemplatesSetting         `json:"templates"`
	GuestTip                 OrderEmailGuestTip                 `json:"guest_tip"`
	FulfillmentAttachmentTip OrderEmailFulfillmentAttachmentTip `json:"fulfillment_attachment_tip"`
}

// --- Patch ?? ---

// OrderEmailLocalizedTemplatePatch ???????
type OrderEmailLocalizedTemplatePatch struct {
	Subject *string `json:"subject"`
	Body    *string `json:"body"`
}

// OrderEmailSceneTemplatePatch ??????
type OrderEmailSceneTemplatePatch struct {
	ZHCN *OrderEmailLocalizedTemplatePatch `json:"zh-CN"`
	ZHTW *OrderEmailLocalizedTemplatePatch `json:"zh-TW"`
	ENUS *OrderEmailLocalizedTemplatePatch `json:"en-US"`
}

// OrderEmailGuestTipPatch ??????
type OrderEmailGuestTipPatch struct {
	ZHCN *string `json:"zh-CN"`
	ZHTW *string `json:"zh-TW"`
	ENUS *string `json:"en-US"`
}

// OrderEmailFulfillmentAttachmentTipPatch ??????????
type OrderEmailFulfillmentAttachmentTipPatch struct {
	ZHCN *string `json:"zh-CN"`
	ZHTW *string `json:"zh-TW"`
	ENUS *string `json:"en-US"`
}

// OrderEmailTemplatesPatch ??????
type OrderEmailTemplatesPatch struct {
	Default              *OrderEmailSceneTemplatePatch `json:"default"`
	Paid                 *OrderEmailSceneTemplatePatch `json:"paid"`
	Delivered            *OrderEmailSceneTemplatePatch `json:"delivered"`
	DeliveredWithContent *OrderEmailSceneTemplatePatch `json:"delivered_with_content"`
	Canceled             *OrderEmailSceneTemplatePatch `json:"canceled"`
	Refunded             *OrderEmailSceneTemplatePatch `json:"refunded"`
	PartiallyRefunded    *OrderEmailSceneTemplatePatch `json:"partially_refunded"`
}

// OrderEmailTemplateSettingPatch ??????????
type OrderEmailTemplateSettingPatch struct {
	Templates                *OrderEmailTemplatesPatch                `json:"templates"`
	GuestTip                 *OrderEmailGuestTipPatch                 `json:"guest_tip"`
	FulfillmentAttachmentTip *OrderEmailFulfillmentAttachmentTipPatch `json:"fulfillment_attachment_tip"`
}

// --- ??? ---

// OrderEmailTemplateDefaultSetting ??????????(?? i18n ??????)
func OrderEmailTemplateDefaultSetting() OrderEmailTemplateSetting {
	return OrderEmailTemplateSetting{
		Templates: OrderEmailTemplatesSetting{
			Default: OrderEmailSceneTemplate{
				ZHCN: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n??:{{amount}} {{currency}}\n\n???????\n\n{{site_name}} ???:{{site_url}}",
				},
				ZHTW: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n??:{{amount}} {{currency}}\n\n???????\n\n{{site_name}} ???:{{site_url}}",
				},
				ENUS: OrderEmailLocalizedTemplate{
					Subject: "Order status updated: {{status}}",
					Body:    "Order No: {{order_no}}\nStatus: {{status}}\nAmount: {{amount}} {{currency}}\n\nThank you for your purchase.\n\n{{site_name}}'s Site URL: {{site_url}}",
				},
			},
			Paid: OrderEmailSceneTemplate{
				ZHCN: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n??:{{amount}} {{currency}}\n\n?????????,????????\n\n{{site_name}} ???:{{site_url}}",
				},
				ZHTW: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n??:{{amount}} {{currency}}\n\n?????,????????\n\n{{site_name}} ???:{{site_url}}",
				},
				ENUS: OrderEmailLocalizedTemplate{
					Subject: "Order status updated: {{status}}",
					Body:    "Order No: {{order_no}}\nStatus: {{status}}\nAmount: {{amount}} {{currency}}\n\nWe have received your payment and will deliver soon.\n\n{{site_name}}'s Site URL: {{site_url}}",
				},
			},
			Delivered: OrderEmailSceneTemplate{
				ZHCN: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n??:{{amount}} {{currency}}\n\n?????,???????\n\n{{site_name}} ???:{{site_url}}",
				},
				ZHTW: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n??:{{amount}} {{currency}}\n\n?????,???????\n\n{{site_name}} ???:{{site_url}}",
				},
				ENUS: OrderEmailLocalizedTemplate{
					Subject: "Order status updated: {{status}}",
					Body:    "Order No: {{order_no}}\nStatus: {{status}}\nAmount: {{amount}} {{currency}}\n\nDelivery completed. Thank you for your purchase.\n\n{{site_name}}'s Site URL: {{site_url}}",
				},
			},
			DeliveredWithContent: OrderEmailSceneTemplate{
				ZHCN: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n??:{{amount}} {{currency}}\n\n????:\n{{fulfillment_info}}\n\n????:\n{{instructions}}\n\n???????\n\n{{site_name}} ???:{{site_url}}",
				},
				ZHTW: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n??:{{amount}} {{currency}}\n\n????:\n{{fulfillment_info}}\n\n????:\n{{instructions}}\n\n???????\n\n{{site_name}} ???:{{site_url}}",
				},
				ENUS: OrderEmailLocalizedTemplate{
					Subject: "Order status updated: {{status}}",
					Body:    "Order No: {{order_no}}\nStatus: {{status}}\nAmount: {{amount}} {{currency}}\n\nDelivery content:\n{{fulfillment_info}}\n\nUsage instructions:\n{{instructions}}\n\nThank you for your purchase.\n\n{{site_name}}'s Site URL: {{site_url}}",
				},
			},
			Canceled: OrderEmailSceneTemplate{
				ZHCN: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n??:{{amount}} {{currency}}\n\n?????,???????????\n\n{{site_name}} ???:{{site_url}}",
				},
				ZHTW: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n??:{{amount}} {{currency}}\n\n?????,???????????\n\n{{site_name}} ???:{{site_url}}",
				},
				ENUS: OrderEmailLocalizedTemplate{
					Subject: "Order status updated: {{status}}",
					Body:    "Order No: {{order_no}}\nStatus: {{status}}\nAmount: {{amount}} {{currency}}\n\nThe order has been canceled. Please contact admin if needed.\n\n{{site_name}}'s Site URL: {{site_url}}",
				},
			},
			Refunded: OrderEmailSceneTemplate{
				ZHCN: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n????:{{refund_amount}} {{currency}}\n????:{{refund_reason}}\n\n?????,???????????\n\n{{site_name}} ???:{{site_url}}",
				},
				ZHTW: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n????:{{refund_amount}} {{currency}}\n????:{{refund_reason}}\n\n?????,???????????\n\n{{site_name}} ???:{{site_url}}",
				},
				ENUS: OrderEmailLocalizedTemplate{
					Subject: "Order status updated: {{status}}",
					Body:    "Order No: {{order_no}}\nStatus: {{status}}\nRefund Amount: {{refund_amount}} {{currency}}\nReason for refund: {{refund_reason}}\n\nThe order has been refunded. Please contact admin if needed.\n\n{{site_name}}'s Site URL: {{site_url}}",
				},
			},
			PartiallyRefunded: OrderEmailSceneTemplate{
				ZHCN: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n????:{{refund_amount}} {{currency}}\n????:{{refund_reason}}\n\n???????,???????????\n\n{{site_name}} ???:{{site_url}}",
				},
				ZHTW: OrderEmailLocalizedTemplate{
					Subject: "??????:{{status}}",
					Body:    "???:{{order_no}}\n??:{{status}}\n????:{{refund_amount}} {{currency}}\n????:{{refund_reason}}\n\n???????,???????????\n\n{{site_name}} ???:{{site_url}}",
				},
				ENUS: OrderEmailLocalizedTemplate{
					Subject: "Order status updated: {{status}}",
					Body:    "Order No: {{order_no}}\nStatus: {{status}}\nRefund Amount: {{refund_amount}} {{currency}}\nReason for refund: {{refund_reason}}\n\nThe order has been partially refunded. Please contact admin if needed.\n\n{{site_name}}'s Site URL: {{site_url}}",
				},
			},
		},
		GuestTip: OrderEmailGuestTip{
			ZHCN: "??????????????????????????",
			ZHTW: "??????????????????????????",
			ENUS: "Guest orders can be queried on the site using the checkout email and order password.",
		},
		FulfillmentAttachmentTip: OrderEmailFulfillmentAttachmentTip{
			ZHCN: "??????,???????,????????????????",
			ZHTW: "??????,???????,????????????????",
			ENUS: "The delivery content is included as an attachment. Please check the email attachment for the full content.",
		},
	}
}

// --- Normalize / Validate ---

// NormalizeOrderEmailTemplateSetting ???????????
func NormalizeOrderEmailTemplateSetting(setting OrderEmailTemplateSetting) OrderEmailTemplateSetting {
	setting.Templates.Default = normalizeOrderEmailSceneTemplate(setting.Templates.Default)
	setting.Templates.Paid = normalizeOrderEmailSceneTemplate(setting.Templates.Paid)
	setting.Templates.Delivered = normalizeOrderEmailSceneTemplate(setting.Templates.Delivered)
	setting.Templates.DeliveredWithContent = normalizeOrderEmailSceneTemplate(setting.Templates.DeliveredWithContent)
	setting.Templates.Canceled = normalizeOrderEmailSceneTemplate(setting.Templates.Canceled)
	setting.Templates.Refunded = normalizeOrderEmailSceneTemplate(setting.Templates.Refunded)
	setting.Templates.PartiallyRefunded = normalizeOrderEmailSceneTemplate(setting.Templates.PartiallyRefunded)
	setting.GuestTip.ZHCN = strings.TrimSpace(setting.GuestTip.ZHCN)
	setting.GuestTip.ZHTW = strings.TrimSpace(setting.GuestTip.ZHTW)
	setting.GuestTip.ENUS = strings.TrimSpace(setting.GuestTip.ENUS)
	setting.FulfillmentAttachmentTip.ZHCN = strings.TrimSpace(setting.FulfillmentAttachmentTip.ZHCN)
	setting.FulfillmentAttachmentTip.ZHTW = strings.TrimSpace(setting.FulfillmentAttachmentTip.ZHTW)
	setting.FulfillmentAttachmentTip.ENUS = strings.TrimSpace(setting.FulfillmentAttachmentTip.ENUS)
	return setting
}

func normalizeOrderEmailSceneTemplate(t OrderEmailSceneTemplate) OrderEmailSceneTemplate {
	t.ZHCN.Subject = strings.TrimSpace(t.ZHCN.Subject)
	t.ZHCN.Body = strings.TrimSpace(t.ZHCN.Body)
	t.ZHTW.Subject = strings.TrimSpace(t.ZHTW.Subject)
	t.ZHTW.Body = strings.TrimSpace(t.ZHTW.Body)
	t.ENUS.Subject = strings.TrimSpace(t.ENUS.Subject)
	t.ENUS.Body = strings.TrimSpace(t.ENUS.Body)
	return t
}

// ValidateOrderEmailTemplateSetting ??????????
func ValidateOrderEmailTemplateSetting(setting OrderEmailTemplateSetting) error {
	scenes := []OrderEmailSceneTemplate{
		setting.Templates.Default,
		setting.Templates.Paid,
		setting.Templates.Delivered,
		setting.Templates.DeliveredWithContent,
		setting.Templates.Canceled,
		setting.Templates.Refunded,
		setting.Templates.PartiallyRefunded,
	}
	for _, scene := range scenes {
		locales := []OrderEmailLocalizedTemplate{scene.ZHCN, scene.ZHTW, scene.ENUS}
		for _, lt := range locales {
			if lt.Subject == "" || lt.Body == "" {
				return ErrOrderEmailTemplateConfigInvalid
			}
		}
	}
	return nil
}

// --- ToMap / Mask ---

// OrderEmailTemplateSettingToMap ???? map
func OrderEmailTemplateSettingToMap(setting OrderEmailTemplateSetting) map[string]interface{} {
	normalized := NormalizeOrderEmailTemplateSetting(setting)
	return map[string]interface{}{
		"templates": map[string]interface{}{
			"default":                orderEmailSceneTemplateToMap(normalized.Templates.Default),
			"paid":                   orderEmailSceneTemplateToMap(normalized.Templates.Paid),
			"delivered":              orderEmailSceneTemplateToMap(normalized.Templates.Delivered),
			"delivered_with_content": orderEmailSceneTemplateToMap(normalized.Templates.DeliveredWithContent),
			"canceled":               orderEmailSceneTemplateToMap(normalized.Templates.Canceled),
			"refunded":               orderEmailSceneTemplateToMap(normalized.Templates.Refunded),
			"partially_refunded":     orderEmailSceneTemplateToMap(normalized.Templates.PartiallyRefunded),
		},
		"guest_tip": map[string]interface{}{
			constants.LocaleZhCN: normalized.GuestTip.ZHCN,
			constants.LocaleZhTW: normalized.GuestTip.ZHTW,
			constants.LocaleEnUS: normalized.GuestTip.ENUS,
		},
		"fulfillment_attachment_tip": map[string]interface{}{
			constants.LocaleZhCN: normalized.FulfillmentAttachmentTip.ZHCN,
			constants.LocaleZhTW: normalized.FulfillmentAttachmentTip.ZHTW,
			constants.LocaleEnUS: normalized.FulfillmentAttachmentTip.ENUS,
		},
	}
}

func orderEmailSceneTemplateToMap(t OrderEmailSceneTemplate) map[string]interface{} {
	return map[string]interface{}{
		constants.LocaleZhCN: map[string]interface{}{
			"subject": t.ZHCN.Subject,
			"body":    t.ZHCN.Body,
		},
		constants.LocaleZhTW: map[string]interface{}{
			"subject": t.ZHTW.Subject,
			"body":    t.ZHTW.Body,
		},
		constants.LocaleEnUS: map[string]interface{}{
			"subject": t.ENUS.Subject,
			"body":    t.ENUS.Body,
		},
	}
}

// MaskOrderEmailTemplateSettingForAdmin ?????????(?????)
func MaskOrderEmailTemplateSettingForAdmin(setting OrderEmailTemplateSetting) models.JSON {
	normalized := NormalizeOrderEmailTemplateSetting(setting)
	return models.JSON(OrderEmailTemplateSettingToMap(normalized))
}

// --- Get / Patch ---

// GetOrderEmailTemplateSetting ??????????(?? settings,??????)
func (s *SettingService) GetOrderEmailTemplateSetting() (OrderEmailTemplateSetting, error) {
	fallback := OrderEmailTemplateDefaultSetting()
	value, err := s.GetByKey(constants.SettingKeyOrderEmailTemplateConfig)
	if err != nil {
		return fallback, err
	}
	if value == nil {
		return fallback, nil
	}
	parsed := orderEmailTemplateSettingFromJSON(value, fallback)
	return NormalizeOrderEmailTemplateSetting(parsed), nil
}

// PatchOrderEmailTemplateSetting ??????????????
func (s *SettingService) PatchOrderEmailTemplateSetting(patch OrderEmailTemplateSettingPatch) (OrderEmailTemplateSetting, error) {
	current, err := s.GetOrderEmailTemplateSetting()
	if err != nil {
		return OrderEmailTemplateSetting{}, err
	}

	next := current
	if patch.Templates != nil {
		if patch.Templates.Default != nil {
			applyOrderEmailSceneTemplatePatch(&next.Templates.Default, patch.Templates.Default)
		}
		if patch.Templates.Paid != nil {
			applyOrderEmailSceneTemplatePatch(&next.Templates.Paid, patch.Templates.Paid)
		}
		if patch.Templates.Delivered != nil {
			applyOrderEmailSceneTemplatePatch(&next.Templates.Delivered, patch.Templates.Delivered)
		}
		if patch.Templates.DeliveredWithContent != nil {
			applyOrderEmailSceneTemplatePatch(&next.Templates.DeliveredWithContent, patch.Templates.DeliveredWithContent)
		}
		if patch.Templates.Canceled != nil {
			applyOrderEmailSceneTemplatePatch(&next.Templates.Canceled, patch.Templates.Canceled)
		}
		if patch.Templates.Refunded != nil {
			applyOrderEmailSceneTemplatePatch(&next.Templates.Refunded, patch.Templates.Refunded)
		}
		if patch.Templates.PartiallyRefunded != nil {
			applyOrderEmailSceneTemplatePatch(&next.Templates.PartiallyRefunded, patch.Templates.PartiallyRefunded)
		}
	}
	if patch.GuestTip != nil {
		if patch.GuestTip.ZHCN != nil {
			next.GuestTip.ZHCN = strings.TrimSpace(*patch.GuestTip.ZHCN)
		}
		if patch.GuestTip.ZHTW != nil {
			next.GuestTip.ZHTW = strings.TrimSpace(*patch.GuestTip.ZHTW)
		}
		if patch.GuestTip.ENUS != nil {
			next.GuestTip.ENUS = strings.TrimSpace(*patch.GuestTip.ENUS)
		}
	}
	if patch.FulfillmentAttachmentTip != nil {
		if patch.FulfillmentAttachmentTip.ZHCN != nil {
			next.FulfillmentAttachmentTip.ZHCN = strings.TrimSpace(*patch.FulfillmentAttachmentTip.ZHCN)
		}
		if patch.FulfillmentAttachmentTip.ZHTW != nil {
			next.FulfillmentAttachmentTip.ZHTW = strings.TrimSpace(*patch.FulfillmentAttachmentTip.ZHTW)
		}
		if patch.FulfillmentAttachmentTip.ENUS != nil {
			next.FulfillmentAttachmentTip.ENUS = strings.TrimSpace(*patch.FulfillmentAttachmentTip.ENUS)
		}
	}

	normalized := NormalizeOrderEmailTemplateSetting(next)
	if err := ValidateOrderEmailTemplateSetting(normalized); err != nil {
		return OrderEmailTemplateSetting{}, err
	}
	if _, err := s.Update(constants.SettingKeyOrderEmailTemplateConfig, OrderEmailTemplateSettingToMap(normalized)); err != nil {
		return OrderEmailTemplateSetting{}, err
	}
	return normalized, nil
}

// --- Locale ?? ---

// ResolveOrderEmailLocaleTemplate ? locale ????
func ResolveOrderEmailLocaleTemplate(t OrderEmailSceneTemplate, locale string) OrderEmailLocalizedTemplate {
	switch locale {
	case constants.LocaleZhTW:
		return t.ZHTW
	case constants.LocaleEnUS:
		return t.ENUS
	default:
		return t.ZHCN
	}
}

// ResolveOrderEmailFulfillmentAttachmentTip ? locale ??????????
func ResolveOrderEmailFulfillmentAttachmentTip(tip OrderEmailFulfillmentAttachmentTip, locale string) string {
	switch locale {
	case constants.LocaleZhTW:
		return tip.ZHTW
	case constants.LocaleEnUS:
		return tip.ENUS
	default:
		return tip.ZHCN
	}
}

// ResolveOrderEmailGuestTip ? locale ??????
func ResolveOrderEmailGuestTip(tip OrderEmailGuestTip, locale string) string {
	switch locale {
	case constants.LocaleZhTW:
		return tip.ZHTW
	case constants.LocaleEnUS:
		return tip.ENUS
	default:
		return tip.ZHCN
	}
}

// --- JSON ?? ---

func orderEmailTemplateSettingFromJSON(raw models.JSON, fallback OrderEmailTemplateSetting) OrderEmailTemplateSetting {
	next := fallback
	if raw == nil {
		return next
	}

	if templatesMap := toStringAnyMap(raw["templates"]); templatesMap != nil {
		if sceneMap := toStringAnyMap(templatesMap["default"]); sceneMap != nil {
			next.Templates.Default = orderEmailSceneTemplateFromMap(sceneMap, next.Templates.Default)
		}
		if sceneMap := toStringAnyMap(templatesMap["paid"]); sceneMap != nil {
			next.Templates.Paid = orderEmailSceneTemplateFromMap(sceneMap, next.Templates.Paid)
		}
		if sceneMap := toStringAnyMap(templatesMap["delivered"]); sceneMap != nil {
			next.Templates.Delivered = orderEmailSceneTemplateFromMap(sceneMap, next.Templates.Delivered)
		}
		if sceneMap := toStringAnyMap(templatesMap["delivered_with_content"]); sceneMap != nil {
			next.Templates.DeliveredWithContent = orderEmailSceneTemplateFromMap(sceneMap, next.Templates.DeliveredWithContent)
		}
		if sceneMap := toStringAnyMap(templatesMap["canceled"]); sceneMap != nil {
			next.Templates.Canceled = orderEmailSceneTemplateFromMap(sceneMap, next.Templates.Canceled)
		}
		if sceneMap := toStringAnyMap(templatesMap["refunded"]); sceneMap != nil {
			next.Templates.Refunded = orderEmailSceneTemplateFromMap(sceneMap, next.Templates.Refunded)
		}
		if sceneMap := toStringAnyMap(templatesMap["partially_refunded"]); sceneMap != nil {
			next.Templates.PartiallyRefunded = orderEmailSceneTemplateFromMap(sceneMap, next.Templates.PartiallyRefunded)
		}
	}

	if guestTipMap := toStringAnyMap(raw["guest_tip"]); guestTipMap != nil {
		next.GuestTip.ZHCN = readString(guestTipMap, constants.LocaleZhCN, next.GuestTip.ZHCN)
		next.GuestTip.ZHTW = readString(guestTipMap, constants.LocaleZhTW, next.GuestTip.ZHTW)
		next.GuestTip.ENUS = readString(guestTipMap, constants.LocaleEnUS, next.GuestTip.ENUS)
	}

	if attachTipMap := toStringAnyMap(raw["fulfillment_attachment_tip"]); attachTipMap != nil {
		next.FulfillmentAttachmentTip.ZHCN = readString(attachTipMap, constants.LocaleZhCN, next.FulfillmentAttachmentTip.ZHCN)
		next.FulfillmentAttachmentTip.ZHTW = readString(attachTipMap, constants.LocaleZhTW, next.FulfillmentAttachmentTip.ZHTW)
		next.FulfillmentAttachmentTip.ENUS = readString(attachTipMap, constants.LocaleEnUS, next.FulfillmentAttachmentTip.ENUS)
	}

	return next
}

func orderEmailSceneTemplateFromMap(raw map[string]interface{}, fallback OrderEmailSceneTemplate) OrderEmailSceneTemplate {
	next := fallback
	if zhCNMap := toStringAnyMap(raw[constants.LocaleZhCN]); zhCNMap != nil {
		next.ZHCN.Subject = readString(zhCNMap, "subject", next.ZHCN.Subject)
		next.ZHCN.Body = readString(zhCNMap, "body", next.ZHCN.Body)
	}
	if zhTWMap := toStringAnyMap(raw[constants.LocaleZhTW]); zhTWMap != nil {
		next.ZHTW.Subject = readString(zhTWMap, "subject", next.ZHTW.Subject)
		next.ZHTW.Body = readString(zhTWMap, "body", next.ZHTW.Body)
	}
	if enUSMap := toStringAnyMap(raw[constants.LocaleEnUS]); enUSMap != nil {
		next.ENUS.Subject = readString(enUSMap, "subject", next.ENUS.Subject)
		next.ENUS.Body = readString(enUSMap, "body", next.ENUS.Body)
	}
	return next
}

// --- Patch ?? ---

func applyOrderEmailSceneTemplatePatch(target *OrderEmailSceneTemplate, patch *OrderEmailSceneTemplatePatch) {
	if target == nil || patch == nil {
		return
	}
	if patch.ZHCN != nil {
		if patch.ZHCN.Subject != nil {
			target.ZHCN.Subject = strings.TrimSpace(*patch.ZHCN.Subject)
		}
		if patch.ZHCN.Body != nil {
			target.ZHCN.Body = strings.TrimSpace(*patch.ZHCN.Body)
		}
	}
	if patch.ZHTW != nil {
		if patch.ZHTW.Subject != nil {
			target.ZHTW.Subject = strings.TrimSpace(*patch.ZHTW.Subject)
		}
		if patch.ZHTW.Body != nil {
			target.ZHTW.Body = strings.TrimSpace(*patch.ZHTW.Body)
		}
	}
	if patch.ENUS != nil {
		if patch.ENUS.Subject != nil {
			target.ENUS.Subject = strings.TrimSpace(*patch.ENUS.Subject)
		}
		if patch.ENUS.Body != nil {
			target.ENUS.Body = strings.TrimSpace(*patch.ENUS.Body)
		}
	}
}
