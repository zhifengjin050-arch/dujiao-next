package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dujiao-next/internal/models"
)

func TestCategoryRespOmitsSensitiveFields(t *testing.T) {
	cat := &models.Category{
		ID:        1,
		ParentID:  0,
		Slug:      "games",
		NameJSON:  models.JSON{"zh-CN": "??"},
		Icon:      "/icons/game.png",
		SortOrder: 10,
	}

	resp := NewCategoryResp(cat)
	data, _ := json.Marshal(resp)
	jsonStr := string(data)

	if strings.Contains(jsonStr, `"created_at"`) {
		t.Error("created_at should not appear")
	}
	if !strings.Contains(jsonStr, `"slug"`) {
		t.Error("slug should appear")
	}
	if resp.Name["zh-CN"] != "??" {
		t.Error("name should be preserved")
	}
}

func TestCategoryRespListEmpty(t *testing.T) {
	result := NewCategoryRespList(nil)
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d", len(result))
	}
}
