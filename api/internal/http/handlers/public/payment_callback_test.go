package public

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPaymentCallbackRejectUnknownPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/callback", strings.NewReader(`{"payment_id":1,"status":"success"}`))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h := &Handler{}
	h.PaymentCallback(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}
