package paypal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/payment/common"
	"github.com/shopspring/decimal"
)

var (
	ErrConfigInvalid       = errors.New("paypal config invalid")
	ErrAuthFailed          = errors.New("paypal auth failed")
	ErrRequestFailed       = errors.New("paypal request failed")
	ErrResponseInvalid     = errors.New("paypal response invalid")
	ErrWebhookVerifyFailed = errors.New("paypal webhook verify failed")
)

const (
	defaultSandboxBaseURL = "https://api-m.sandbox.paypal.com"
	defaultTimeout        = 12 * time.Second

	paypalWebhookVerifyStatusSuccess = "SUCCESS"

	paypalEventCaptureCompleted = "PAYMENT.CAPTURE.COMPLETED"
	paypalEventOrderCompleted   = "CHECKOUT.ORDER.COMPLETED"
	paypalEventCaptureDenied    = "PAYMENT.CAPTURE.DENIED"
	paypalEventCaptureDeclined  = "PAYMENT.CAPTURE.DECLINED"
	paypalEventCaptureFailed    = "PAYMENT.CAPTURE.FAILED"
	paypalEventOrderDenied      = "CHECKOUT.ORDER.DENIED"
	paypalEventCapturePending   = "PAYMENT.CAPTURE.PENDING"
	paypalEventOrderApproved    = "CHECKOUT.ORDER.APPROVED"

	paypalResourceStatusCompleted = "COMPLETED"
	paypalResourceStatusDenied    = "DENIED"
	paypalResourceStatusDeclined  = "DECLINED"
	paypalResourceStatusFailed    = "FAILED"
	paypalResourceStatusVoided    = "VOIDED"
	paypalResourceStatusPending   = "PENDING"
	paypalResourceStatusApproved  = "APPROVED"
	paypalResourceStatusCreated   = "CREATED"
	paypalResourceStatusSaved     = "SAVED"

	paypalUserActionPayNow         = "PAY_NOW"
	paypalShippingPreferenceNoShip = "NO_SHIPPING"
)

// Config PayPal ?????
type Config struct {
	ClientID           string `json:"client_id"`
	ClientSecret       string `json:"client_secret"`
	BaseURL            string `json:"base_url"`
	ReturnURL          string `json:"return_url"`
	CancelURL          string `json:"cancel_url"`
	WebhookID          string `json:"webhook_id"`
	BrandName          string `json:"brand_name"`
	Locale             string `json:"locale"`
	LandingPage        string `json:"landing_page"`
	UserAction         string `json:"user_action"`
	ShippingPreference string `json:"shipping_preference"`
	TargetCurrency     string `json:"target_currency"`
	ExchangeRate       string `json:"exchange_rate"`
}

// CreateInput ?? PayPal ?????
type CreateInput struct {
	OrderNo     string
	Amount      string
	Currency    string
	Description string
	ReturnURL   string
	CancelURL   string
}

// CreateResult ?? PayPal ?????
type CreateResult struct {
	OrderID     string
	ApprovalURL string
	Status      string
	Raw         map[string]interface{}
}

// CaptureResult ???????
type CaptureResult struct {
	OrderID   string
	CaptureID string
	Status    string
	Amount    string
	Currency  string
	PaidAt    *time.Time
	Raw       map[string]interface{}
}

// WebhookEvent PayPal Webhook ???
type WebhookEvent struct {
	ID         string                 `json:"id"`
	EventType  string                 `json:"event_type"`
	CreateTime string                 `json:"create_time"`
	Resource   map[string]interface{} `json:"resource"`
	Raw        map[string]interface{}
}

// ParseConfig ?????
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	return common.ParseConfig[Config](raw, ErrConfigInvalid)
}

// ValidateConfig ?????
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		return fmt.Errorf("%w: client_id is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return fmt.Errorf("%w: client_secret is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return fmt.Errorf("%w: base_url is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.ReturnURL) == "" {
		return fmt.Errorf("%w: return_url is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.CancelURL) == "" {
		return fmt.Errorf("%w: cancel_url is required", ErrConfigInvalid)
	}
	if _, err := url.ParseRequestURI(strings.TrimSpace(cfg.BaseURL)); err != nil {
		return fmt.Errorf("%w: base_url is invalid", ErrConfigInvalid)
	}
	if _, err := url.ParseRequestURI(strings.TrimSpace(cfg.ReturnURL)); err != nil {
		return fmt.Errorf("%w: return_url is invalid", ErrConfigInvalid)
	}
	if _, err := url.ParseRequestURI(strings.TrimSpace(cfg.CancelURL)); err != nil {
		return fmt.Errorf("%w: cancel_url is invalid", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.WebhookID) == "" {
		return fmt.Errorf("%w: webhook_id is required", ErrConfigInvalid)
	}
	return nil
}

// CreateOrder ?? PayPal ???
func CreateOrder(ctx context.Context, cfg *Config, input CreateInput) (*CreateResult, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.OrderNo) == "" || strings.TrimSpace(input.Amount) == "" || strings.TrimSpace(input.Currency) == "" {
		return nil, fmt.Errorf("%w: order input is invalid", ErrConfigInvalid)
	}
	returnURL := strings.TrimSpace(input.ReturnURL)
	if returnURL == "" {
		returnURL = cfg.ReturnURL
	}
	cancelURL := strings.TrimSpace(input.CancelURL)
	if cancelURL == "" {
		cancelURL = cfg.CancelURL
	}

	token, err := getAccessToken(ctx, cfg)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"invoice_id": input.OrderNo,
				"amount": map[string]string{
					"currency_code": strings.ToUpper(strings.TrimSpace(input.Currency)),
					"value":         strings.TrimSpace(input.Amount),
				},
				"description": strings.TrimSpace(input.Description),
			},
		},
		"application_context": buildApplicationContext(cfg, returnURL, cancelURL),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal request failed", ErrRequestFailed)
	}

	respBody, statusCode, err := doJSONRequest(ctx, cfg, http.MethodPost, "/v2/checkout/orders", token, body)
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("%w: create order status %d", ErrResponseInvalid, statusCode)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("%w: decode response failed", ErrResponseInvalid)
	}

	result := &CreateResult{Raw: raw}
	result.OrderID = strings.TrimSpace(readString(raw, "id"))
	result.Status = strings.TrimSpace(readString(raw, "status"))
	result.ApprovalURL = extractLinkByRel(raw, "approve")
	if result.OrderID == "" || result.ApprovalURL == "" {
		return nil, fmt.Errorf("%w: missing order id or approve url", ErrResponseInvalid)
	}
	return result, nil
}

// CaptureOrder ?? PayPal ???
func CaptureOrder(ctx context.Context, cfg *Config, orderID string) (*CaptureResult, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, fmt.Errorf("%w: order id is empty", ErrConfigInvalid)
	}

	token, err := getAccessToken(ctx, cfg)
	if err != nil {
		return nil, err
	}

	endpoint := "/v2/checkout/orders/" + url.PathEscape(orderID) + "/capture"
	respBody, statusCode, err := doJSONRequest(ctx, cfg, http.MethodPost, endpoint, token, []byte("{}"))
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("%w: capture status %d", ErrResponseInvalid, statusCode)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("%w: decode response failed", ErrResponseInvalid)
	}

	result := &CaptureResult{Raw: raw}
	result.OrderID = strings.TrimSpace(readString(raw, "id"))
	result.Status = strings.TrimSpace(readString(raw, "status"))

	captures := readArray(raw, "purchase_units", "0", "payments", "captures")
	if len(captures) > 0 {
		if captureMap, ok := captures[0].(map[string]interface{}); ok {
			result.CaptureID = strings.TrimSpace(readString(captureMap, "id"))
			if status := strings.TrimSpace(readString(captureMap, "status")); status != "" {
				result.Status = status
			}
			result.Amount = strings.TrimSpace(readString(captureMap, "amount", "value"))
			result.Currency = strings.TrimSpace(readString(captureMap, "amount", "currency_code"))
			if rawTime := strings.TrimSpace(readString(captureMap, "create_time")); rawTime != "" {
				if parsed, err := time.Parse(time.RFC3339, rawTime); err == nil {
					result.PaidAt = &parsed
				}
			}
		}
	}

	if result.OrderID == "" {
		result.OrderID = orderID
	}
	if result.Status == "" {
		return nil, fmt.Errorf("%w: missing capture status", ErrResponseInvalid)
	}
	return result, nil
}

// VerifyWebhookSignature ?? PayPal Webhook ???
func VerifyWebhookSignature(ctx context.Context, cfg *Config, headers http.Header, event map[string]interface{}) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.WebhookID) == "" {
		return fmt.Errorf("%w: webhook_id is required", ErrConfigInvalid)
	}
	token, err := getAccessToken(ctx, cfg)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"transmission_id":   strings.TrimSpace(headers.Get("Paypal-Transmission-Id")),
		"transmission_time": strings.TrimSpace(headers.Get("Paypal-Transmission-Time")),
		"cert_url":          strings.TrimSpace(headers.Get("Paypal-Cert-Url")),
		"auth_algo":         strings.TrimSpace(headers.Get("Paypal-Auth-Algo")),
		"transmission_sig":  strings.TrimSpace(headers.Get("Paypal-Transmission-Sig")),
		"webhook_id":        strings.TrimSpace(cfg.WebhookID),
		"webhook_event":     event,
	}

	for _, key := range []string{"transmission_id", "transmission_time", "cert_url", "auth_algo", "transmission_sig"} {
		if strings.TrimSpace(readString(payload, key)) == "" {
			return fmt.Errorf("%w: missing %s", ErrWebhookVerifyFailed, key)
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%w: marshal verify payload failed", ErrWebhookVerifyFailed)
	}

	respBody, statusCode, err := doJSONRequest(ctx, cfg, http.MethodPost, "/v1/notifications/verify-webhook-signature", token, body)
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("%w: verify status %d", ErrWebhookVerifyFailed, statusCode)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("%w: decode verify response failed", ErrWebhookVerifyFailed)
	}
	if strings.ToUpper(strings.TrimSpace(readString(resp, "verification_status"))) != paypalWebhookVerifyStatusSuccess {
		return fmt.Errorf("%w: verify result is not success", ErrWebhookVerifyFailed)
	}
	return nil
}

// ParseWebhookEvent ?? Webhook ???
func ParseWebhookEvent(body []byte) (*WebhookEvent, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("%w: webhook body is empty", ErrResponseInvalid)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: webhook body invalid", ErrResponseInvalid)
	}
	event := &WebhookEvent{
		ID:         strings.TrimSpace(readString(raw, "id")),
		EventType:  strings.TrimSpace(readString(raw, "event_type")),
		CreateTime: strings.TrimSpace(readString(raw, "create_time")),
		Raw:        raw,
	}
	if resource, ok := raw["resource"].(map[string]interface{}); ok {
		event.Resource = resource
	} else {
		event.Resource = map[string]interface{}{}
	}
	if event.EventType == "" {
		return nil, fmt.Errorf("%w: event_type is missing", ErrResponseInvalid)
	}
	return event, nil
}

// RelatedOrderID ????? PayPal ????
func (e *WebhookEvent) RelatedOrderID() string {
	if e == nil {
		return ""
	}
	if val := strings.TrimSpace(readString(e.Resource, "supplementary_data", "related_ids", "order_id")); val != "" {
		return val
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(e.EventType)), "CHECKOUT.ORDER") {
		if val := strings.TrimSpace(readString(e.Resource, "id")); val != "" {
			return val
		}
	}
	if val := strings.TrimSpace(readString(e.Resource, "order_id")); val != "" {
		return val
	}
	return ""
}

// RelatedInvoiceID ? webhook event resource ? purchase_units ??? invoice_id?
func (e *WebhookEvent) RelatedInvoiceID() string {
	if e == nil {
		return ""
	}
	units := readArray(e.Resource, "purchase_units")
	if len(units) > 0 {
		if unitMap, ok := units[0].(map[string]interface{}); ok {
			if val := strings.TrimSpace(readString(unitMap, "invoice_id")); val != "" {
				return val
			}
		}
	}
	return ""
}

// CaptureAmount ??????????
func (e *WebhookEvent) CaptureAmount() (string, string) {
	if e == nil {
		return "", ""
	}
	value := strings.TrimSpace(readString(e.Resource, "amount", "value"))
	currency := strings.TrimSpace(readString(e.Resource, "amount", "currency_code"))
	if value != "" && currency != "" {
		return value, currency
	}

	value = strings.TrimSpace(readString(e.Resource, "purchase_units", "0", "amount", "value"))
	currency = strings.TrimSpace(readString(e.Resource, "purchase_units", "0", "amount", "currency_code"))
	if value != "" && currency != "" {
		return value, currency
	}

	value = strings.TrimSpace(readString(e.Resource, "purchase_units", "0", "payments", "captures", "0", "amount", "value"))
	currency = strings.TrimSpace(readString(e.Resource, "purchase_units", "0", "payments", "captures", "0", "amount", "currency_code"))
	return value, currency
}

// PaidAt ???????
func (e *WebhookEvent) PaidAt() *time.Time {
	if e == nil {
		return nil
	}
	for _, key := range []string{"create_time", "update_time"} {
		raw := strings.TrimSpace(readString(e.Resource, key))
		if raw == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, raw)
		if err == nil {
			return &parsed
		}
	}
	if e.CreateTime != "" {
		parsed, err := time.Parse(time.RFC3339, e.CreateTime)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

// ResourceStatus ???????
func (e *WebhookEvent) ResourceStatus() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(readString(e.Resource, "status"))
}

// ToPaymentStatus ?? PayPal ??????????
func ToPaymentStatus(eventType, resourceStatus string) (string, bool) {
	eventType = strings.ToUpper(strings.TrimSpace(eventType))
	resourceStatus = strings.ToUpper(strings.TrimSpace(resourceStatus))

	switch eventType {
	case paypalEventCaptureCompleted, paypalEventOrderCompleted:
		return constants.PaymentStatusSuccess, true
	case paypalEventCaptureDenied, paypalEventCaptureDeclined, paypalEventCaptureFailed, paypalEventOrderDenied:
		return constants.PaymentStatusFailed, true
	case paypalEventCapturePending, paypalEventOrderApproved:
		return constants.PaymentStatusPending, true
	}

	switch resourceStatus {
	case paypalResourceStatusCompleted:
		return constants.PaymentStatusSuccess, true
	case paypalResourceStatusDenied, paypalResourceStatusDeclined, paypalResourceStatusFailed, paypalResourceStatusVoided:
		return constants.PaymentStatusFailed, true
	case paypalResourceStatusPending, paypalResourceStatusApproved, paypalResourceStatusCreated, paypalResourceStatusSaved:
		return constants.PaymentStatusPending, true
	}

	return "", false
}

func (c *Config) Normalize() {
	c.ClientID = strings.TrimSpace(c.ClientID)
	c.ClientSecret = strings.TrimSpace(c.ClientSecret)
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if c.BaseURL == "" {
		c.BaseURL = defaultSandboxBaseURL
	}
	c.ReturnURL = strings.TrimSpace(c.ReturnURL)
	c.CancelURL = strings.TrimSpace(c.CancelURL)
	c.WebhookID = strings.TrimSpace(c.WebhookID)
	c.BrandName = strings.TrimSpace(c.BrandName)
	c.Locale = strings.TrimSpace(c.Locale)
	c.LandingPage = strings.TrimSpace(c.LandingPage)
	c.UserAction = strings.TrimSpace(c.UserAction)
	if c.UserAction == "" {
		c.UserAction = paypalUserActionPayNow
	}
	c.ShippingPreference = strings.TrimSpace(c.ShippingPreference)
	if c.ShippingPreference == "" {
		c.ShippingPreference = paypalShippingPreferenceNoShip
	}
	c.TargetCurrency = strings.ToUpper(strings.TrimSpace(c.TargetCurrency))
	c.ExchangeRate = strings.TrimSpace(c.ExchangeRate)
}

// NeedsCurrencyConversion ?????????
func (c *Config) NeedsCurrencyConversion() bool {
	return c.TargetCurrency != "" && c.ExchangeRate != ""
}

// ConvertAmount ?????????????????,??????????????
func (c *Config) ConvertAmount(amount, currency string) (string, string, error) {
	if !c.NeedsCurrencyConversion() {
		return amount, currency, nil
	}
	amountDec, err := decimal.NewFromString(strings.TrimSpace(amount))
	if err != nil {
		return "", "", fmt.Errorf("%w: invalid amount %q", ErrConfigInvalid, amount)
	}
	rate, err := decimal.NewFromString(c.ExchangeRate)
	if err != nil || rate.LessThanOrEqual(decimal.Zero) {
		return "", "", fmt.Errorf("%w: invalid exchange_rate %q", ErrConfigInvalid, c.ExchangeRate)
	}
	converted := amountDec.Mul(rate).Round(2)
	return converted.String(), c.TargetCurrency, nil
}

func buildApplicationContext(cfg *Config, returnURL, cancelURL string) map[string]string {
	ctx := map[string]string{
		"return_url":          strings.TrimSpace(returnURL),
		"cancel_url":          strings.TrimSpace(cancelURL),
		"user_action":         strings.TrimSpace(cfg.UserAction),
		"shipping_preference": strings.TrimSpace(cfg.ShippingPreference),
	}
	if cfg.BrandName != "" {
		ctx["brand_name"] = cfg.BrandName
	}
	if cfg.Locale != "" {
		ctx["locale"] = cfg.Locale
	}
	if cfg.LandingPage != "" {
		ctx["landing_page"] = cfg.LandingPage
	}
	return ctx
}

func getAccessToken(ctx context.Context, cfg *Config) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()

	values := url.Values{}
	values.Set("grant_type", "client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+"/v1/oauth2/token", strings.NewReader(values.Encode()))
	if err != nil {
		return "", fmt.Errorf("%w: build token request failed", ErrAuthFailed)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: request token failed", ErrAuthFailed)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: read token response failed", ErrAuthFailed)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%w: token status %d", ErrAuthFailed, resp.StatusCode)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("%w: decode token response failed", ErrAuthFailed)
	}
	token := strings.TrimSpace(readString(parsed, "access_token"))
	if token == "" {
		return "", fmt.Errorf("%w: access_token is empty", ErrAuthFailed)
	}
	return token, nil
}

func doJSONRequest(ctx context.Context, cfg *Config, method, endpoint, token string, body []byte) ([]byte, int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(cfg.BaseURL, "/")+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("%w: build request failed", ErrRequestFailed)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: http request failed", ErrRequestFailed)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("%w: read response failed", ErrRequestFailed)
	}
	return respBody, resp.StatusCode, nil
}

func withDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, defaultTimeout)
}

func extractLinkByRel(raw map[string]interface{}, rel string) string {
	links, ok := raw["links"].([]interface{})
	if !ok {
		return ""
	}
	rel = strings.ToLower(strings.TrimSpace(rel))
	for _, item := range links {
		linkMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if strings.ToLower(strings.TrimSpace(readString(linkMap, "rel"))) != rel {
			continue
		}
		if href := strings.TrimSpace(readString(linkMap, "href")); href != "" {
			return href
		}
	}
	return ""
}

func readString(raw map[string]interface{}, path ...string) string {
	if raw == nil {
		return ""
	}
	var current interface{} = raw
	for _, seg := range path {
		if idx, err := strconv.Atoi(seg); err == nil {
			arr, ok := current.([]interface{})
			if !ok || idx < 0 || idx >= len(arr) {
				return ""
			}
			current = arr[idx]
			continue
		}
		next, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = next[seg]
	}
	if current == nil {
		return ""
	}
	if str, ok := current.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", current)
}

func readArray(raw map[string]interface{}, path ...string) []interface{} {
	if raw == nil {
		return nil
	}
	var current interface{} = raw
	for _, seg := range path {
		if idx, err := strconv.Atoi(seg); err == nil {
			arr, ok := current.([]interface{})
			if !ok || idx < 0 || idx >= len(arr) {
				return nil
			}
			current = arr[idx]
			continue
		}
		next, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = next[seg]
	}
	arr, ok := current.([]interface{})
	if !ok {
		return nil
	}
	return arr
}
