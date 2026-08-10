package public

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStripeWebhookQueryBind(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("POST", "/api/v1/payments/webhook/stripe?channel_id=12", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	var query StripeWebhookQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		t.Fatalf("bind stripe query failed: %v", err)
	}
	if query.ChannelID != 12 {
		t.Fatalf("expected channel id 12, got %d", query.ChannelID)
	}
}
