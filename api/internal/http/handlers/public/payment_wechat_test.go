package public

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsWechatCallbackRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"id":"EV-1","resource":{"algorithm":"AEAD_AES_256_GCM"}}`
	req := httptest.NewRequest("POST", "/api/v1/payments/callback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Wechatpay-Signature", "mock-sign")
	req.Header.Set("Wechatpay-Timestamp", "1760000000")
	req.Header.Set("Wechatpay-Nonce", "mock-nonce")
	req.Header.Set("Wechatpay-Serial", "mock-serial")
	c.Request = req

	if !isWechatCallbackRequest(c, []byte(body)) {
		t.Fatalf("expected wechat callback request")
	}
}

func TestIsWechatCallbackRequestMissingHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"id":"EV-1","resource":{"algorithm":"AEAD_AES_256_GCM"}}`
	req := httptest.NewRequest("POST", "/api/v1/payments/callback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	if isWechatCallbackRequest(c, []byte(body)) {
		t.Fatalf("expected non-wechat callback request")
	}
}
