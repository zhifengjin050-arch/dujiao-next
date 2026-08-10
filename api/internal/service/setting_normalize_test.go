package service

import (
	"testing"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
)

func TestUpdateOrderSettingNormalized(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	result, err := svc.Update(constants.SettingKeyOrderConfig, map[string]interface{}{
		constants.SettingFieldPaymentExpireMinutes: "20000",
		"extra": "keep",
	})
	if err != nil {
		t.Fatalf("update order config failed: %v", err)
	}

	minutes, err := parseSettingInt(result[constants.SettingFieldPaymentExpireMinutes])
	if err != nil {
		t.Fatalf("parse payment_expire_minutes failed: %v", err)
	}
	if minutes != 10080 {
		t.Fatalf("unexpected payment_expire_minutes, expected 10080 got %d", minutes)
	}
	if _, ok := result["extra"]; ok {
		t.Fatalf("unexpected extra field kept in normalized order config: %v", result["extra"])
	}
}

func TestGetSiteBrand(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	repo.store[constants.SettingKeySiteConfig] = map[string]interface{}{
		"brand": map[string]interface{}{
			"site_name": "  Demo Shop  ",
			"site_url":  "  https://example.com/path/  ",
		},
	}
	brand, err := svc.GetSiteBrand()
	if err != nil {
		t.Fatalf("get site brand failed: %v", err)
	}
	if brand.SiteName != "Demo Shop" {
		t.Fatalf("unexpected site name: %q", brand.SiteName)
	}
	if brand.SiteURL != "https://example.com/path" {
		t.Fatalf("unexpected site url: %q", brand.SiteURL)
	}

	repo.store[constants.SettingKeySiteConfig] = map[string]interface{}{
		"brand": map[string]interface{}{},
	}
	brand, err = svc.GetSiteBrand()
	if err != nil {
		t.Fatalf("get site brand with missing field failed: %v", err)
	}
	if brand.SiteName != "" || brand.SiteURL != "" {
		t.Fatalf("expected empty site brand, got %+v", brand)
	}
}

func TestUpdateSiteSettingNormalized(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	result, err := svc.Update(constants.SettingKeySiteConfig, map[string]interface{}{
		"brand": map[string]interface{}{
			"site_name": 123,
			"site_url":  "  https://example.com/path/  ",
		},
		"contact": map[string]interface{}{
			"telegram": "  https://t.me/demo  ",
			"whatsapp": 123,
		},
		"seo": map[string]interface{}{
			"title": map[string]interface{}{
				"zh-CN": "  ??  ",
				"en-US": "  Title  ",
			},
		},
		"about": map[string]interface{}{
			"hero": map[string]interface{}{
				"title": map[string]interface{}{
					"zh-CN": "  ????  ",
					"en-US": "  About Us  ",
				},
				"subtitle": map[string]interface{}{
					"zh-CN": "  ?????????  ",
				},
			},
			"introduction": map[string]interface{}{
				"zh-CN": "  ??????????????  ",
				"zh-TW": 123,
			},
			"services": map[string]interface{}{
				"title": map[string]interface{}{
					"zh-CN": "  ?????  ",
				},
				"items": []interface{}{
					map[string]interface{}{
						"zh-CN": "  ????  ",
						"en-US": "  Service One  ",
					},
					map[string]interface{}{
						"zh-CN": "",
						"zh-TW": "",
						"en-US": "",
					},
					"invalid",
				},
			},
			"contact": map[string]interface{}{
				"title": map[string]interface{}{
					"zh-CN": "  ????  ",
				},
				"text": map[string]interface{}{
					"zh-CN": "  ??????????  ",
					"en-US": "  Contact us via channels below  ",
				},
			},
		},
		"scripts": []interface{}{
			map[string]interface{}{
				"name":     "  Google Analytics  ",
				"enabled":  "true",
				"position": "head",
				"code":     "  window.dataLayer = window.dataLayer || [];  ",
			},
			map[string]interface{}{
				"name":     "Footer Tracker",
				"enabled":  1,
				"position": "body_end",
				"code":     "window.__footerTracker = true;",
			},
			map[string]interface{}{
				"name":     "Fallback Position",
				"enabled":  false,
				"position": "invalid",
				"code":     "console.log('fallback');",
			},
			map[string]interface{}{
				"name":     "Skip Empty",
				"enabled":  true,
				"position": "head",
				"code":     "   ",
			},
			"invalid",
		},
		"languages": []interface{}{" zh-CN ", "en-US", "", "en-US"},
		"currency":  "usdx",
	})
	if err != nil {
		t.Fatalf("update site config failed: %v", err)
	}

	brand, ok := result["brand"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid brand payload type: %T", result["brand"])
	}
	if brand["site_name"] != "" {
		t.Fatalf("unexpected brand.site_name: %v", brand["site_name"])
	}
	if brand["site_url"] != "https://example.com/path" {
		t.Fatalf("unexpected brand.site_url: %v", brand["site_url"])
	}

	contact, ok := result["contact"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid contact payload type: %T", result["contact"])
	}
	if contact["telegram"] != "https://t.me/demo" {
		t.Fatalf("unexpected telegram: %v", contact["telegram"])
	}
	if contact["whatsapp"] != "" {
		t.Fatalf("unexpected whatsapp: %v", contact["whatsapp"])
	}

	seo, ok := result["seo"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid seo payload type: %T", result["seo"])
	}
	title, ok := seo["title"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid seo.title payload type: %T", seo["title"])
	}
	if title["zh-CN"] != "??" {
		t.Fatalf("unexpected seo.title.zh-CN: %v", title["zh-CN"])
	}
	if title["en-US"] != "Title" {
		t.Fatalf("unexpected seo.title.en-US: %v", title["en-US"])
	}
	if title["zh-TW"] != "" {
		t.Fatalf("unexpected seo.title.zh-TW: %v", title["zh-TW"])
	}

	legal, ok := result["legal"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid legal payload type: %T", result["legal"])
	}
	privacy, ok := legal["privacy"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid legal.privacy payload type: %T", legal["privacy"])
	}
	if privacy["zh-CN"] != "" || privacy["zh-TW"] != "" || privacy["en-US"] != "" {
		t.Fatalf("unexpected legal.privacy payload: %+v", privacy)
	}

	about, ok := result["about"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about payload type: %T", result["about"])
	}

	hero, ok := about["hero"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about.hero payload type: %T", about["hero"])
	}
	heroTitle, ok := hero["title"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about.hero.title payload type: %T", hero["title"])
	}
	if heroTitle["zh-CN"] != "????" || heroTitle["en-US"] != "About Us" || heroTitle["zh-TW"] != "" {
		t.Fatalf("unexpected about.hero.title payload: %+v", heroTitle)
	}

	introduction, ok := about["introduction"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about.introduction payload type: %T", about["introduction"])
	}
	if introduction["zh-CN"] != "??????????????" || introduction["zh-TW"] != "" {
		t.Fatalf("unexpected about.introduction payload: %+v", introduction)
	}

	services, ok := about["services"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about.services payload type: %T", about["services"])
	}
	serviceItems, ok := services["items"].([]interface{})
	if !ok {
		t.Fatalf("invalid about.services.items payload type: %T", services["items"])
	}
	if len(serviceItems) != 1 {
		t.Fatalf("unexpected about.services.items size: %d", len(serviceItems))
	}
	serviceItem, ok := serviceItems[0].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about.services.items[0] payload type: %T", serviceItems[0])
	}
	if serviceItem["zh-CN"] != "????" || serviceItem["en-US"] != "Service One" || serviceItem["zh-TW"] != "" {
		t.Fatalf("unexpected about.services.items[0] payload: %+v", serviceItem)
	}

	aboutContact, ok := about["contact"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about.contact payload type: %T", about["contact"])
	}
	contactText, ok := aboutContact["text"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about.contact.text payload type: %T", aboutContact["text"])
	}
	if contactText["zh-CN"] != "??????????" || contactText["en-US"] != "Contact us via channels below" {
		t.Fatalf("unexpected about.contact.text payload: %+v", contactText)
	}

	languages, ok := result["languages"].([]string)
	if !ok {
		t.Fatalf("invalid languages payload type: %T", result["languages"])
	}
	if len(languages) != 2 || languages[0] != "zh-CN" || languages[1] != "en-US" {
		t.Fatalf("unexpected languages: %+v", languages)
	}
	currency, ok := result[constants.SettingFieldSiteCurrency].(string)
	if !ok {
		t.Fatalf("invalid currency payload type: %T", result[constants.SettingFieldSiteCurrency])
	}
	if currency != constants.SiteCurrencyDefault {
		t.Fatalf("unexpected currency: %s", currency)
	}

	scripts, ok := result["scripts"].([]interface{})
	if !ok {
		t.Fatalf("invalid scripts payload type: %T", result["scripts"])
	}
	if len(scripts) != 3 {
		t.Fatalf("unexpected scripts size: %d", len(scripts))
	}
	firstScript, ok := scripts[0].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid scripts[0] payload type: %T", scripts[0])
	}
	if firstScript["name"] != "Google Analytics" || firstScript["enabled"] != true || firstScript["position"] != "head" || firstScript["code"] != "window.dataLayer = window.dataLayer || [];" {
		t.Fatalf("unexpected scripts[0]: %+v", firstScript)
	}
	secondScript, ok := scripts[1].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid scripts[1] payload type: %T", scripts[1])
	}
	if secondScript["position"] != "body_end" || secondScript["enabled"] != true {
		t.Fatalf("unexpected scripts[1]: %+v", secondScript)
	}
	thirdScript, ok := scripts[2].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid scripts[2] payload type: %T", scripts[2])
	}
	if thirdScript["position"] != "head" || thirdScript["enabled"] != false {
		t.Fatalf("unexpected scripts[2]: %+v", thirdScript)
	}
}

func TestUpdateSiteSettingNormalizedDefaultAbout(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	result, err := svc.Update(constants.SettingKeySiteConfig, map[string]interface{}{})
	if err != nil {
		t.Fatalf("update site config failed: %v", err)
	}

	brand, ok := result["brand"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid brand payload type: %T", result["brand"])
	}
	if brand["site_name"] != "" {
		t.Fatalf("unexpected default brand payload: %+v", brand)
	}
	if brand["site_url"] != "" {
		t.Fatalf("unexpected default brand payload: %+v", brand)
	}
	scripts, ok := result["scripts"].([]interface{})
	if !ok {
		t.Fatalf("invalid scripts payload type: %T", result["scripts"])
	}
	if len(scripts) != 0 {
		t.Fatalf("unexpected default scripts payload: %+v", scripts)
	}
	if result[constants.SettingFieldSiteCurrency] != constants.SiteCurrencyDefault {
		t.Fatalf("unexpected default currency: %v", result[constants.SettingFieldSiteCurrency])
	}

	about, ok := result["about"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about payload type: %T", result["about"])
	}

	hero, ok := about["hero"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about.hero payload type: %T", about["hero"])
	}
	heroTitle, ok := hero["title"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about.hero.title payload type: %T", hero["title"])
	}
	if heroTitle["zh-CN"] != "" || heroTitle["zh-TW"] != "" || heroTitle["en-US"] != "" {
		t.Fatalf("unexpected default about.hero.title: %+v", heroTitle)
	}

	services, ok := about["services"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid about.services payload type: %T", about["services"])
	}
	serviceItems, ok := services["items"].([]interface{})
	if !ok {
		t.Fatalf("invalid about.services.items payload type: %T", services["items"])
	}
	if len(serviceItems) != 0 {
		t.Fatalf("unexpected default about.services.items size: %d", len(serviceItems))
	}
}

func TestUpdateSiteSettingNormalizedCurrency(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	result, err := svc.Update(constants.SettingKeySiteConfig, map[string]interface{}{
		"currency": " usd ",
	})
	if err != nil {
		t.Fatalf("update site config failed: %v", err)
	}
	if result[constants.SettingFieldSiteCurrency] != "USD" {
		t.Fatalf("unexpected normalized currency: %v", result[constants.SettingFieldSiteCurrency])
	}
}

func TestUpdateOrderRefundSettingNormalized(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	result, err := svc.Update(constants.SettingKeyOrderConfig, map[string]interface{}{
		orderConfigFieldMaxRefundDays: "0",
	})
	if err != nil {
		t.Fatalf("update order refund config failed: %v", err)
	}
	maxDays, err := parseSettingInt(result[orderConfigFieldMaxRefundDays])
	if err != nil {
		t.Fatalf("parse normalized max_refund_days failed: %v", err)
	}
	if maxDays != 0 {
		t.Fatalf("unexpected normalized max_refund_days: %v", result[orderConfigFieldMaxRefundDays])
	}

	result, err = svc.Update(constants.SettingKeyOrderConfig, map[string]interface{}{
		orderConfigFieldMaxRefundDays: "-1",
	})
	if err != nil {
		t.Fatalf("update order refund config failed: %v", err)
	}
	maxDays, err = parseSettingInt(result[orderConfigFieldMaxRefundDays])
	if err != nil {
		t.Fatalf("parse normalized max_refund_days failed: %v", err)
	}
	if maxDays != DefaultOrderRefundConfig().MaxRefundDays {
		t.Fatalf("unexpected normalized max_refund_days for negative value: %v", result[orderConfigFieldMaxRefundDays])
	}

	result, err = svc.Update(constants.SettingKeyOrderConfig, map[string]interface{}{
		orderConfigFieldMaxRefundDays: "5000",
	})
	if err != nil {
		t.Fatalf("update order refund config failed: %v", err)
	}
	maxDays, err = parseSettingInt(result[orderConfigFieldMaxRefundDays])
	if err != nil {
		t.Fatalf("parse normalized max_refund_days failed: %v", err)
	}
	if maxDays != NormalizeOrderRefundConfig(OrderRefundConfig{MaxRefundDays: 5000}).MaxRefundDays {
		t.Fatalf("unexpected clamped max_refund_days: %v", result[orderConfigFieldMaxRefundDays])
	}
}

func TestGetOrderRefundConfig(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	cfg, err := svc.GetOrderRefundConfig()
	if err != nil {
		t.Fatalf("get order refund config failed: %v", err)
	}
	if cfg.MaxRefundDays != DefaultOrderRefundConfig().MaxRefundDays {
		t.Fatalf("unexpected default max refund days: %d", cfg.MaxRefundDays)
	}

	repo.store[constants.SettingKeyOrderConfig] = map[string]interface{}{
		orderConfigFieldMaxRefundDays: 45,
	}
	cfg, err = svc.GetOrderRefundConfig()
	if err != nil {
		t.Fatalf("get order refund config failed: %v", err)
	}
	if cfg.MaxRefundDays != 45 {
		t.Fatalf("unexpected max refund days: %d", cfg.MaxRefundDays)
	}
}

func TestGetOrderRefundConfigFallsBackToConfigOrder(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, config.OrderConfig{
		MaxRefundDays: 7,
	})

	cfg, err := svc.GetOrderRefundConfig()
	if err != nil {
		t.Fatalf("get order refund config failed: %v", err)
	}
	if cfg.MaxRefundDays != 7 {
		t.Fatalf("unexpected max refund days fallback from config order: %d", cfg.MaxRefundDays)
	}
}

func TestGetOrderRefundConfigFallsBackToConfigOrderUnlimited(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, config.OrderConfig{
		MaxRefundDays: 0,
	})

	cfg, err := svc.GetOrderRefundConfig()
	if err != nil {
		t.Fatalf("get order refund config failed: %v", err)
	}
	if cfg.MaxRefundDays != 0 {
		t.Fatalf("unexpected unlimited max refund days fallback from config order: %d", cfg.MaxRefundDays)
	}
}

func TestGetOrderConfigFallsBackToConfigWhenMissing(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	cfg, err := svc.GetOrderConfig(config.OrderConfig{
		PaymentExpireMinutes: 19,
	})
	if err != nil {
		t.Fatalf("get order config failed: %v", err)
	}
	if cfg.PaymentExpireMinutes != 19 {
		t.Fatalf("unexpected fallback payment_expire_minutes: %d", cfg.PaymentExpireMinutes)
	}
	if cfg.MaxRefundDays != DefaultOrderConfig().MaxRefundDays {
		t.Fatalf("unexpected default max_refund_days: %d", cfg.MaxRefundDays)
	}
}

func TestUpdateSiteSettingNormalizedScriptsLimit(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	scripts := make([]interface{}, 0, 25)
	for i := 0; i < 25; i++ {
		scripts = append(scripts, map[string]interface{}{
			"name":     "script",
			"enabled":  true,
			"position": "head",
			"code":     "console.log('ok');",
		})
	}

	result, err := svc.Update(constants.SettingKeySiteConfig, map[string]interface{}{
		"scripts": scripts,
	})
	if err != nil {
		t.Fatalf("update site config failed: %v", err)
	}

	normalizedScripts, ok := result["scripts"].([]interface{})
	if !ok {
		t.Fatalf("invalid scripts payload type: %T", result["scripts"])
	}
	if len(normalizedScripts) != settingSiteScriptsMaxCount {
		t.Fatalf("unexpected scripts size: %d", len(normalizedScripts))
	}
}

func TestUpdateTelegramAuthSettingNormalized(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	result, err := svc.Update(constants.SettingKeyTelegramAuthConfig, map[string]interface{}{
		"enabled":              true,
		"bot_username":         " @demo_bot ",
		"bot_token":            " token-abc ",
		"mini_app_url":         " https://example.com/mini-app ",
		"login_expire_seconds": -10,
		"replay_ttl_seconds":   1,
	})
	if err != nil {
		t.Fatalf("update telegram auth config failed: %v", err)
	}

	if result["bot_username"] != "demo_bot" {
		t.Fatalf("unexpected bot_username: %v", result["bot_username"])
	}
	if result["bot_token"] != "token-abc" {
		t.Fatalf("unexpected bot_token: %v", result["bot_token"])
	}
	if result["mini_app_url"] != "https://example.com/mini-app" {
		t.Fatalf("unexpected mini_app_url: %v", result["mini_app_url"])
	}
	if result["login_expire_seconds"] != 300 {
		t.Fatalf("unexpected login_expire_seconds: %v", result["login_expire_seconds"])
	}
	if result["replay_ttl_seconds"] != 60 {
		t.Fatalf("unexpected replay_ttl_seconds: %v", result["replay_ttl_seconds"])
	}
}
