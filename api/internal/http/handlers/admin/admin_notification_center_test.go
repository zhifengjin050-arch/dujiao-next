package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/provider"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAdminNotificationCenterHandlerTest(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:admin_notification_center_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.NotificationLog{}); err != nil {
		t.Fatalf("auto migrate notification log failed: %v", err)
	}

	logRepo := repository.NewNotificationLogRepository(db)
	logSvc := service.NewNotificationLogService(logRepo)
	return &Handler{Container: &provider.Container{
		NotificationLogService: logSvc,
	}}, db
}

func TestListNotificationLogsFiltersStatusAndChannel(t *testing.T) {
	h, db := setupAdminNotificationCenterHandlerTest(t)
	now := time.Now().UTC().Truncate(time.Second)

	items := []models.NotificationLog{
		{
			EventType:    "order_paid_success",
			BizType:      "order",
			BizID:        11,
			Channel:      "email",
			Recipient:    "failed@example.com",
			Locale:       "en-US",
			Title:        "Order DJ-11",
			Body:         "failed body",
			Status:       "failed",
			ErrorMessage: "smtp disabled",
			IsTest:       false,
			CreatedAt:    now,
		},
		{
			EventType: "order_paid_success",
			BizType:   "order",
			BizID:     12,
			Channel:   "telegram",
			Recipient: "-100100",
			Locale:    "zh-CN",
			Title:     "Telegram notice",
			Body:      "ok",
			Status:    "success",
			IsTest:    true,
			CreatedAt: now.Add(time.Second),
		},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("seed notification logs failed: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/settings/notification-center/logs?channel=email&status=failed&is_test=false", nil)

	h.ListNotificationLogs(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status want 200 got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data       []models.NotificationLog `json:"data"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Pagination.Total != 1 {
		t.Fatalf("pagination total want 1 got %d", resp.Pagination.Total)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len want 1 got %d", len(resp.Data))
	}
	if resp.Data[0].Recipient != "failed@example.com" {
		t.Fatalf("unexpected recipient: %s", resp.Data[0].Recipient)
	}
	if resp.Data[0].Status != "failed" {
		t.Fatalf("unexpected status: %s", resp.Data[0].Status)
	}
}
