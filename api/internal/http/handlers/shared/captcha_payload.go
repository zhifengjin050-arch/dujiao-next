package shared

import (
	"strings"

	"github.com/dujiao-next/internal/service"
)

// CaptchaPayloadRequest ????????
type CaptchaPayloadRequest struct {
	CaptchaID      string `json:"captcha_id"`
	CaptchaCode    string `json:"captcha_code"`
	TurnstileToken string `json:"turnstile_token"`
}

// ToServicePayload ??? service ???????
func (r CaptchaPayloadRequest) ToServicePayload() service.CaptchaVerifyPayload {
	return service.CaptchaVerifyPayload{
		CaptchaID:      strings.TrimSpace(r.CaptchaID),
		CaptchaCode:    strings.TrimSpace(r.CaptchaCode),
		TurnstileToken: strings.TrimSpace(r.TurnstileToken),
	}
}
