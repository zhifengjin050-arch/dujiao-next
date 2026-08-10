package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
)

func TestBannerRespOmitsSensitiveFields(t *testing.T) {
	now := time.Now()
	banner := &models.Banner{
		ID:           1,
		Name:         "Admin-Only-Name",
		Position:     "home_hero",
		TitleJSON:    models.JSON{"zh-CN": "??"},
		SubtitleJSON: models.JSON{"zh-CN": "???"},
		Image:        "/img/banner.png",
		LinkType:     "url",
		LinkValue:    "https://example.com",
		OpenInNewTab: true,
		IsActive:     true,
		StartAt:      &now,
		EndAt:        &now,
		SortOrder:    5,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	resp := NewBannerResp(banner)
	data, _ := json.Marshal(resp)
	jsonStr := string(data)

	sensitiveFields := []string{"name", "is_active", "start_at", "end_at", "sort_order", "created_at", "updated_at"}
	for _, field := range sensitiveFields {
		if strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("sensitive field %q should not appear", field)
		}
	}

	if !strings.Contains(jsonStr, `"position"`) {
		t.Error("position should appear")
	}
	if !strings.Contains(jsonStr, `"image"`) {
		t.Error("image should appear")
	}
	if !strings.Contains(jsonStr, `"link_type"`) {
		t.Error("link_type should appear")
	}
}
