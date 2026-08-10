package service

import (
	"regexp"
	"strings"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
)

var settingSupportedLanguages = append([]string(nil), constants.SupportedLocales...)
var settingCurrencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

const (
	settingSiteScriptsMaxCount       = 20
	settingSiteScriptNameMaxRuneSize = 120
	settingSiteScriptCodeMaxRuneSize = 20000

	settingSiteFooterLinksMaxCount       = 20
	settingSiteFooterLinkNameMaxRuneSize = 120
	settingSiteFooterLinkURLMaxRuneSize  = 2000

	settingNavCustomItemsMaxCount        = 10
	settingNavCustomItemTitleMaxRuneSize = 120
	settingNavCustomItemURLMaxRuneSize   = 2000
)

// normalizeSettingValueByKey ?????????,????????
func normalizeSettingValueByKey(key string, value map[string]interface{}) models.JSON {
	switch key {
	case constants.SettingKeyDashboardConfig:
		setting := dashboardSettingFromJSON(models.JSON(value), DashboardDefaultSetting())
		return DashboardSettingToMap(setting)
	case constants.SettingKeyOrderConfig:
		cfg := orderConfigFromJSON(models.JSON(value), DefaultOrderConfig())
		return OrderConfigToMap(cfg)
	case constants.SettingKeySiteConfig:
		return normalizeSiteSetting(value)
	case constants.SettingKeyTelegramAuthConfig:
		setting := telegramAuthSettingFromJSON(models.JSON(value), TelegramAuthDefaultSetting(config.TelegramAuthConfig{}))
		return TelegramAuthSettingToMap(setting)
	case constants.SettingKeyNotificationCenterConfig:
		setting := notificationCenterSettingFromJSON(models.JSON(value), NotificationCenterDefaultSetting())
		return NotificationCenterSettingToMap(setting)
	case constants.SettingKeyAffiliateConfig:
		return normalizeAffiliateSettingMap(value)
	case constants.SettingKeyTelegramBotConfig:
		return normalizeTelegramBotConfig(models.JSON(value))
	case constants.SettingKeyNavConfig:
		return normalizeNavConfig(value)
	case constants.SettingKeyRegistrationConfig:
		return normalizeRegistrationSetting(value)
	case constants.SettingKeyOrderRiskControlConfig:
		cfg := orderRiskControlConfigFromJSON(models.JSON(value), DefaultOrderRiskControlConfig())
		return OrderRiskControlConfigToMap(cfg)
	case constants.SettingKeyCallbackRoutesConfig:
		return normalizeCallbackRoutesSetting(value)
	default:
		return models.JSON(value)
	}
}

// normalizeSiteSetting ??????????
func normalizeSiteSetting(value map[string]interface{}) models.JSON {
	normalized := make(models.JSON, len(value)+8)
	for key, raw := range value {
		normalized[key] = raw
	}

	normalized["brand"] = normalizeSiteBrand(value["brand"])
	normalized["contact"] = normalizeSiteContact(value["contact"])
	normalized["seo"] = normalizeSiteLocalizedBlock(value["seo"], []string{"title", "keywords", "description"})
	normalized["legal"] = normalizeSiteLocalizedBlock(value["legal"], []string{"terms", "privacy"})
	normalized["about"] = normalizeSiteAbout(value["about"])
	normalized["scripts"] = normalizeSiteScripts(value["scripts"])
	normalized["footer_links"] = normalizeSiteFooterLinks(value["footer_links"])
	normalized[constants.SettingFieldSiteCurrency] = normalizeSiteCurrency(value[constants.SettingFieldSiteCurrency])
	normalized["template_mode"] = normalizeSiteTemplateMode(value["template_mode"])

	if raw, ok := value["languages"]; ok {
		normalized["languages"] = normalizeSiteLanguages(raw)
	}

	return normalized
}

func normalizeSiteScripts(raw interface{}) []interface{} {
	listRaw, ok := raw.([]interface{})
	if !ok {
		return make([]interface{}, 0)
	}

	result := make([]interface{}, 0, len(listRaw))
	for _, itemRaw := range listRaw {
		itemMap, ok := itemRaw.(map[string]interface{})
		if !ok {
			continue
		}

		code := normalizeSettingTextWithRuneLimit(itemMap["code"], settingSiteScriptCodeMaxRuneSize)
		if code == "" {
			continue
		}

		position := normalizeSettingText(itemMap["position"])
		if position != "head" && position != "body_end" {
			position = "head"
		}

		result = append(result, map[string]interface{}{
			"name":     normalizeSettingTextWithRuneLimit(itemMap["name"], settingSiteScriptNameMaxRuneSize),
			"enabled":  parseSettingBool(itemMap["enabled"]),
			"position": position,
			"code":     code,
		})

		if len(result) >= settingSiteScriptsMaxCount {
			break
		}
	}

	return result
}

func normalizeSiteFooterLinks(raw interface{}) []interface{} {
	listRaw, ok := raw.([]interface{})
	if !ok {
		return make([]interface{}, 0)
	}

	result := make([]interface{}, 0, len(listRaw))
	for _, itemRaw := range listRaw {
		itemMap, ok := itemRaw.(map[string]interface{})
		if !ok {
			continue
		}

		name := normalizeSettingTextWithRuneLimit(itemMap["name"], settingSiteFooterLinkNameMaxRuneSize)
		if name == "" {
			continue
		}

		url := normalizeSettingTextWithRuneLimit(itemMap["url"], settingSiteFooterLinkURLMaxRuneSize)

		result = append(result, map[string]interface{}{
			"name": name,
			"url":  url,
		})

		if len(result) >= settingSiteFooterLinksMaxCount {
			break
		}
	}

	return result
}

func normalizeSiteContact(raw interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"telegram": "",
		"whatsapp": "",
	}
	contactMap, ok := raw.(map[string]interface{})
	if !ok {
		return result
	}
	result["telegram"] = normalizeSettingText(contactMap["telegram"])
	result["whatsapp"] = normalizeSettingText(contactMap["whatsapp"])
	return result
}

func normalizeSiteBrand(raw interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"site_name":        "",
		"site_url":         "",
		"site_description": normalizeSiteLocalizedField(nil),
	}
	brandMap, ok := raw.(map[string]interface{})
	if !ok {
		return result
	}
	result["site_name"] = normalizeSettingText(brandMap["site_name"])
	result["site_url"] = strings.TrimRight(normalizeSettingText(brandMap["site_url"]), "/")
	result["site_description"] = normalizeSiteLocalizedField(brandMap["site_description"])
	return result
}

func normalizeSiteLocalizedBlock(raw interface{}, fields []string) map[string]interface{} {
	result := make(map[string]interface{}, len(fields))
	blockMap, _ := raw.(map[string]interface{})

	for _, field := range fields {
		if blockMap == nil {
			result[field] = normalizeSiteLocalizedField(nil)
			continue
		}
		result[field] = normalizeSiteLocalizedField(blockMap[field])
	}

	return result
}

func normalizeSiteLocalizedField(raw interface{}) map[string]interface{} {
	fieldResult := make(map[string]interface{}, len(settingSupportedLanguages))
	for _, language := range settingSupportedLanguages {
		fieldResult[language] = ""
	}

	fieldRaw, ok := raw.(map[string]interface{})
	if !ok {
		return fieldResult
	}

	for _, language := range settingSupportedLanguages {
		fieldResult[language] = normalizeSettingText(fieldRaw[language])
	}

	return fieldResult
}

func normalizeSiteLocalizedList(raw interface{}, maxItems int) []interface{} {
	listRaw, ok := raw.([]interface{})
	if !ok {
		return make([]interface{}, 0)
	}

	result := make([]interface{}, 0, len(listRaw))
	for _, item := range listRaw {
		normalized := normalizeSiteLocalizedField(item)
		hasText := false
		for _, language := range settingSupportedLanguages {
			text, _ := normalized[language].(string)
			if text != "" {
				hasText = true
				break
			}
		}
		if !hasText {
			continue
		}

		result = append(result, normalized)
		if maxItems > 0 && len(result) >= maxItems {
			break
		}
	}

	return result
}

func normalizeSiteAbout(raw interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"hero":         normalizeSiteLocalizedBlock(nil, []string{"title", "subtitle"}),
		"introduction": normalizeSiteLocalizedField(nil),
		"services": map[string]interface{}{
			"title": normalizeSiteLocalizedField(nil),
			"items": make([]interface{}, 0),
		},
		"contact": map[string]interface{}{
			"title": normalizeSiteLocalizedField(nil),
			"text":  normalizeSiteLocalizedField(nil),
		},
	}

	aboutMap, ok := raw.(map[string]interface{})
	if !ok {
		return result
	}

	result["hero"] = normalizeSiteLocalizedBlock(aboutMap["hero"], []string{"title", "subtitle"})
	result["introduction"] = normalizeSiteLocalizedField(aboutMap["introduction"])

	services := map[string]interface{}{
		"title": normalizeSiteLocalizedField(nil),
		"items": make([]interface{}, 0),
	}
	if servicesRaw, ok := aboutMap["services"].(map[string]interface{}); ok {
		services["title"] = normalizeSiteLocalizedField(servicesRaw["title"])
		services["items"] = normalizeSiteLocalizedList(servicesRaw["items"], 12)
	}
	result["services"] = services

	contact := map[string]interface{}{
		"title": normalizeSiteLocalizedField(nil),
		"text":  normalizeSiteLocalizedField(nil),
	}
	if contactRaw, ok := aboutMap["contact"].(map[string]interface{}); ok {
		contact["title"] = normalizeSiteLocalizedField(contactRaw["title"])
		contact["text"] = normalizeSiteLocalizedField(contactRaw["text"])
	}
	result["contact"] = contact

	return result
}

func normalizeSiteLanguages(raw interface{}) []string {
	list := make([]string, 0)
	switch value := raw.(type) {
	case []string:
		list = append(list, value...)
	case []interface{}:
		for _, item := range value {
			list = append(list, normalizeSettingText(item))
		}
	default:
		return append([]string(nil), settingSupportedLanguages...)
	}

	result := make([]string, 0, len(list))
	seen := make(map[string]struct{}, len(list))
	for _, item := range list {
		lang := strings.TrimSpace(item)
		if lang == "" {
			continue
		}
		if _, exists := seen[lang]; exists {
			continue
		}
		seen[lang] = struct{}{}
		result = append(result, lang)
	}
	if len(result) == 0 {
		return append([]string(nil), settingSupportedLanguages...)
	}
	return result
}

// normalizeSiteTemplateMode ?????????,?? "card" ? "list",?? "card"?
func normalizeSiteTemplateMode(raw interface{}) string {
	mode := normalizeSettingText(raw)
	if mode == "list" {
		return "list"
	}
	return "card"
}

// normalizeRegistrationSetting ????????
func normalizeNavConfig(value map[string]interface{}) models.JSON {
	// builtin: blog / notice / about ??,?? true
	builtin := map[string]interface{}{
		"blog":   true,
		"notice": true,
		"about":  true,
	}
	if builtinRaw, ok := value["builtin"].(map[string]interface{}); ok {
		for _, key := range []string{"blog", "notice", "about"} {
			if raw, exists := builtinRaw[key]; exists {
				builtin[key] = parseSettingBool(raw)
			}
		}
	}

	// custom_items: ??????
	customItems := make([]interface{}, 0)
	if itemsRaw, ok := value["custom_items"].([]interface{}); ok {
		for _, itemRaw := range itemsRaw {
			itemMap, ok := itemRaw.(map[string]interface{})
			if !ok {
				continue
			}

			title := normalizeSiteLocalizedField(itemMap["title"])
			// ???????
			allEmpty := true
			for _, lang := range settingSupportedLanguages {
				if s, _ := title[lang].(string); s != "" {
					allEmpty = false
					break
				}
			}
			if allEmpty {
				continue
			}
			// ??????
			for _, lang := range settingSupportedLanguages {
				if s, _ := title[lang].(string); s != "" {
					title[lang] = normalizeSettingTextWithRuneLimit(s, settingNavCustomItemTitleMaxRuneSize)
				}
			}

			linkType := normalizeSettingText(itemMap["link_type"])
			if linkType != "internal" && linkType != "external" {
				linkType = "internal"
			}

			target := normalizeSettingText(itemMap["target"])
			if target != "_self" && target != "_blank" {
				target = "_self"
			}

			url := normalizeSettingTextWithRuneLimit(itemMap["url"], settingNavCustomItemURLMaxRuneSize)

			sortOrder := 0
			if v, err := parseSettingInt(itemMap["sort_order"]); err == nil {
				sortOrder = v
			}

			icon := normalizeSettingText(itemMap["icon"])
			if icon == "" {
				icon = "link"
			}

			// ??????? id
			id := float64(0)
			if v, ok := itemMap["id"].(float64); ok {
				id = v
			}

			customItems = append(customItems, map[string]interface{}{
				"id":         id,
				"title":      title,
				"link_type":  linkType,
				"url":        url,
				"target":     target,
				"sort_order": sortOrder,
				"enabled":    parseSettingBool(itemMap["enabled"]),
				"icon":       icon,
			})

			if len(customItems) >= settingNavCustomItemsMaxCount {
				break
			}
		}
	}

	return models.JSON{
		"builtin":      builtin,
		"custom_items": customItems,
	}
}

func normalizeRegistrationSetting(value map[string]interface{}) models.JSON {
	normalized := make(models.JSON, 2)
	registrationEnabled := true
	if raw, ok := value[constants.SettingFieldRegistrationEnabled]; ok {
		registrationEnabled = parseSettingBool(raw)
	}
	normalized[constants.SettingFieldRegistrationEnabled] = registrationEnabled

	emailVerificationEnabled := true
	if raw, ok := value[constants.SettingFieldEmailVerificationEnabled]; ok {
		emailVerificationEnabled = parseSettingBool(raw)
	}
	normalized[constants.SettingFieldEmailVerificationEnabled] = emailVerificationEnabled

	return normalized
}
