package service

import "testing"

func TestTelegramBotConfigHelpRoundTrip(t *testing.T) {
	t.Parallel()

	original := TelegramBotConfigDefault()
	original.Help.Title["en-US"] = "Support Center"
	original.Help.CenterHint["en-US"] = "Configured center hint"
	original.Help.Items = append(original.Help.Items, TelegramBotHelpItem{
		Key:     "custom",
		Enabled: true,
		Order:   9,
		Summary: LocalizedText{"zh-CN": "?? ???", "zh-TW": "?? ??", "en-US": "?? Custom"},
		Title:   LocalizedText{"zh-CN": "?? ???", "zh-TW": "?? ??", "en-US": "?? Custom"},
		Content: LocalizedText{"zh-CN": "??", "zh-TW": "??", "en-US": "Content"},
	})

	serialized := TelegramBotConfigToMap(original)
	parsed := telegramBotConfigFromJSON(serialized, TelegramBotConfigDefault())
	if parsed.Help.Title["en-US"] != "Support Center" {
		t.Fatalf("expected help title to survive round trip, got=%q", parsed.Help.Title["en-US"])
	}
	if parsed.Help.CenterHint["en-US"] != "Configured center hint" {
		t.Fatalf("expected help center hint to survive round trip, got=%q", parsed.Help.CenterHint["en-US"])
	}
	if len(parsed.Help.Items) != len(original.Help.Items) {
		t.Fatalf("expected help items to survive round trip, got=%d", len(parsed.Help.Items))
	}
	if parsed.Help.Items[len(parsed.Help.Items)-1].Key != "custom" {
		t.Fatalf("expected custom help item key, got=%q", parsed.Help.Items[len(parsed.Help.Items)-1].Key)
	}
}

func TestNormalizeTelegramBotConfigNormalizesHelpTexts(t *testing.T) {
	t.Parallel()

	normalized := normalizeTelegramBotConfig(map[string]interface{}{
		"help": map[string]interface{}{
			"enabled": true,
			"title": map[string]interface{}{
				"zh-CN": "  ????  ",
			},
			"center_hint": map[string]interface{}{
				"zh-CN": "  ????  ",
			},
			"items": []interface{}{
				map[string]interface{}{
					"key":     "  faq  ",
					"enabled": true,
					"order":   1,
					"summary": map[string]interface{}{"zh-CN": "  ??  "},
					"title":   map[string]interface{}{"zh-CN": "  ??  "},
					"content": map[string]interface{}{"zh-CN": "  ??  "},
				},
			},
		},
	})

	helpRaw := normalized["help"].(map[string]interface{})
	title := helpRaw["title"].(map[string]interface{})
	if title["zh-CN"] != "????" {
		t.Fatalf("expected trimmed help title, got=%q", title["zh-CN"])
	}
	centerHint := helpRaw["center_hint"].(map[string]interface{})
	if centerHint["zh-CN"] != "????" {
		t.Fatalf("expected trimmed help center hint, got=%q", centerHint["zh-CN"])
	}

	items := helpRaw["items"].([]interface{})
	first := items[0].(map[string]interface{})
	if first["key"] != "faq" {
		t.Fatalf("expected trimmed help key, got=%q", first["key"])
	}
	summary := first["summary"].(map[string]interface{})
	if summary["zh-CN"] != "??" {
		t.Fatalf("expected trimmed help summary, got=%q", summary["zh-CN"])
	}
}
