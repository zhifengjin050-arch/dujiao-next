package channel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dujiao-next/internal/provider"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/dujiao-next/internal/models"
)

func TestNormalizeChannelManualFormSchemaUsesLocaleText(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":      "account",
				"type":     "text",
				"required": true,
				"label": map[string]interface{}{
					"zh-CN": "????",
					"en-US": "Account",
				},
				"placeholder": map[string]interface{}{
					"zh-CN": "?????",
					"en-US": "Enter account",
				},
			},
			map[string]interface{}{
				"key":      "server",
				"type":     "radio",
				"required": false,
				"label":    "??",
				"options":  []interface{}{"??", "???"},
			},
		},
	}

	got := normalizeChannelManualFormSchema(schema, "zh-CN", "en-US")
	fields, ok := got["fields"].([]gin.H)
	if !ok || len(fields) != 2 {
		t.Fatalf("expected 2 fields, got=%T len=%d", got["fields"], len(fields))
	}
	if fields[0]["label"] != "????" {
		t.Fatalf("expected localized label, got=%v", fields[0]["label"])
	}
	if fields[0]["placeholder"] != "?????" {
		t.Fatalf("expected localized placeholder, got=%v", fields[0]["placeholder"])
	}
	options, ok := fields[1]["options"].([]string)
	if !ok || len(options) != 2 {
		t.Fatalf("expected options list, got=%T %#v", fields[1]["options"], fields[1]["options"])
	}
}

type channelCatalogTestResponse struct {
	StatusCode int            `json:"status_code"`
	Data       map[string]any `json:"data"`
}

func TestGetCategoriesIncludesParentIDAndVisibleParent(t *testing.T) {
	dsn := fmt.Sprintf("file:channel_catalog_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Category{}, &models.Product{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	parent := models.Category{
		Slug:     "games",
		NameJSON: models.JSON{"zh-CN": "??"},
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("create parent category failed: %v", err)
	}
	child := models.Category{
		ParentID: parent.ID,
		Slug:     "steam",
		NameJSON: models.JSON{"zh-CN": "Steam"},
	}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child category failed: %v", err)
	}
	hidden := models.Category{
		Slug:     "hidden",
		NameJSON: models.JSON{"zh-CN": "hidden"},
	}
	if err := db.Create(&hidden).Error; err != nil {
		t.Fatalf("create hidden category failed: %v", err)
	}

	product := models.Product{
		CategoryID:  child.ID,
		Slug:        "steam-product",
		TitleJSON:   models.JSON{"zh-CN": "Steam Product"},
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:    true,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	categoryRepo := repository.NewCategoryRepository(db)
	handler := New(&provider.Container{
		CategoryRepo:    categoryRepo,
		CategoryService: service.NewCategoryService(categoryRepo),
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/channel/catalog/categories", handler.GetCategories)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channel/catalog/categories?locale=zh-CN", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected http 200, got %d", recorder.Code)
	}

	var payload channelCatalogTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	itemsValue, ok := payload.Data["items"].([]any)
	if !ok {
		t.Fatalf("expected items array, got %T", payload.Data["items"])
	}
	if len(itemsValue) != 2 {
		t.Fatalf("expected 2 visible categories, got %d", len(itemsValue))
	}

	itemsBySlug := make(map[string]map[string]any, len(itemsValue))
	for _, item := range itemsValue {
		row, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("expected item object, got %T", item)
		}
		slug, _ := row["slug"].(string)
		itemsBySlug[slug] = row
	}

	parentItem, ok := itemsBySlug["games"]
	if !ok {
		t.Fatalf("expected parent category to be returned")
	}
	if parentItem["parent_id"] != float64(0) {
		t.Fatalf("expected parent parent_id=0, got %v", parentItem["parent_id"])
	}
	if parentItem["product_count"] != float64(0) {
		t.Fatalf("expected parent product_count=0, got %v", parentItem["product_count"])
	}

	childItem, ok := itemsBySlug["steam"]
	if !ok {
		t.Fatalf("expected child category to be returned")
	}
	if childItem["parent_id"] != float64(parent.ID) {
		t.Fatalf("expected child parent_id=%d, got %v", parent.ID, childItem["parent_id"])
	}
	if childItem["product_count"] != float64(1) {
		t.Fatalf("expected child product_count=1, got %v", childItem["product_count"])
	}

	if _, ok := itemsBySlug["hidden"]; ok {
		t.Fatalf("expected hidden category without products or visible children to be filtered out")
	}
}
