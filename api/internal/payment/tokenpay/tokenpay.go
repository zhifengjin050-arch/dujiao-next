package tokenpay

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/payment/common"
)

const (
	createOrderPath = "/CreateOrder"
	DefaultCurrency = "USDT"
)

var (
	ErrConfigInvalid    = errors.New("tokenpay config invalid")
	ErrRequestFailed    = errors.New("tokenpay request failed")
	ErrResponseInvalid  = errors.New("tokenpay response invalid")
	ErrSignatureInvalid = errors.New("tokenpay signature invalid")
)

type Config struct {
	GatewayURL   string `json:"gateway_url"`
	NotifySecret string `json:"notify_secret"`
	Currency     string `json:"currency"`
	NotifyURL    string `json:"notify_url"`
	RedirectURL  string `json:"redirect_url"`
	BaseCurrency string `json:"base_currency"`
}

type CreateInput struct {
	OutOrderID   string
	OrderUserKey string
	ActualAmount string
	Currency     string
	NotifyURL    string
	RedirectURL  string
}

type CreateResult struct {
	PayURL       string
	TokenOrderID string
	QRCodeBase64 string
	QRCodeLink   string
	ToAddress    string
	Raw          map[string]interface{}
}

type CallbackData struct {
	Raw             map[string]interface{}
	Signature       string
	TokenOrderID    string
	OutOrderID      string
	OrderUserKey    string
	Status          int
	ActualAmount    string
	Amount          string
	BaseCurrency    string
	Currency        string
	PayTime         string
	PassThroughInfo string
}

type QueryResult struct {
	Raw map[string]interface{}
}

func ParseConfig(raw map[string]interface{}) (*Config, error) {
	return common.ParseConfig[Config](raw, ErrConfigInvalid)
}

func (c *Config) Normalize() {
	c.GatewayURL = strings.TrimRight(strings.TrimSpace(c.GatewayURL), "/")
	c.NotifySecret = strings.TrimSpace(c.NotifySecret)
	c.Currency = strings.TrimSpace(c.Currency)
	c.NotifyURL = strings.TrimSpace(c.NotifyURL)
	c.RedirectURL = strings.TrimSpace(c.RedirectURL)
	c.BaseCurrency = strings.ToUpper(strings.TrimSpace(c.BaseCurrency))
	if c.BaseCurrency == "" {
		c.BaseCurrency = constants.SiteCurrencyDefault
	}
}

func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.GatewayURL) == "" {
		return fmt.Errorf("%w: gateway_url is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.NotifySecret) == "" {
		return fmt.Errorf("%w: notify_secret is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.Currency) == "" {
		return fmt.Errorf("%w: currency is required", ErrConfigInvalid)
	}
	return nil
}

func CreatePayment(ctx context.Context, cfg *Config, input CreateInput) (*CreateResult, error) {
	if cfg == nil {
		return nil, ErrConfigInvalid
	}
	if strings.TrimSpace(input.OutOrderID) == "" || strings.TrimSpace(input.OrderUserKey) == "" || strings.TrimSpace(input.ActualAmount) == "" {
		return nil, ErrConfigInvalid
	}

	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = strings.TrimSpace(cfg.Currency)
	}
	if currency == "" {
		return nil, fmt.Errorf("%w: currency is required", ErrConfigInvalid)
	}
	notifyURL := strings.TrimSpace(input.NotifyURL)
	if notifyURL == "" {
		notifyURL = strings.TrimSpace(cfg.NotifyURL)
	}
	redirectURL := strings.TrimSpace(input.RedirectURL)
	if redirectURL == "" {
		redirectURL = strings.TrimSpace(cfg.RedirectURL)
	}

	payload := map[string]interface{}{
		"OutOrderId":   strings.TrimSpace(input.OutOrderID),
		"OrderUserKey": strings.TrimSpace(input.OrderUserKey),
		"ActualAmount": strings.TrimSpace(input.ActualAmount),
		"Currency":     currency,
	}
	if notifyURL != "" {
		payload["NotifyUrl"] = notifyURL
	}
	if redirectURL != "" {
		payload["RedirectUrl"] = redirectURL
	}
	payload["Signature"] = SignPayload(payload, cfg.NotifySecret)

	endpoint := cfg.GatewayURL + createOrderPath
	body, err := postJSON(ctx, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: decode response failed", ErrResponseInvalid)
	}
	success, _ := raw["success"].(bool)
	if !success {
		return nil, fmt.Errorf("%w: %s", ErrResponseInvalid, strings.TrimSpace(getString(raw, "message")))
	}

	result := &CreateResult{
		PayURL:       strings.TrimSpace(getString(raw, "data")),
		TokenOrderID: strings.TrimSpace(getStringFromMap(raw, "info", "Id")),
		QRCodeBase64: strings.TrimSpace(getStringFromMap(raw, "info", "QrCodeBase64")),
		QRCodeLink:   strings.TrimSpace(getStringFromMap(raw, "info", "QrCodeLink")),
		ToAddress:    strings.TrimSpace(getStringFromMap(raw, "info", "ToAddress")),
		Raw:          raw,
	}
	if result.PayURL == "" {
		result.PayURL = strings.TrimSpace(getStringFromMap(raw, "info", "PaymentUrl"))
	}
	return result, nil
}

func ParseCallback(body []byte) (*CallbackData, error) {
	if len(body) == 0 {
		return nil, ErrResponseInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var payload map[string]interface{}
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("%w: decode callback failed", ErrResponseInvalid)
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("%w: empty callback payload", ErrResponseInvalid)
	}
	callback := &CallbackData{
		Raw:             payload,
		Signature:       strings.TrimSpace(pickString(payload, "Signature", "signature")),
		TokenOrderID:    strings.TrimSpace(pickString(payload, "Id", "id")),
		OutOrderID:      strings.TrimSpace(pickString(payload, "OutOrderId", "out_order_id")),
		OrderUserKey:    strings.TrimSpace(pickString(payload, "OrderUserKey", "order_user_key")),
		Status:          pickInt(payload, "Status", "status"),
		ActualAmount:    strings.TrimSpace(pickString(payload, "ActualAmount", "actual_amount")),
		Amount:          strings.TrimSpace(pickString(payload, "Amount", "amount")),
		BaseCurrency:    strings.ToUpper(strings.TrimSpace(pickString(payload, "BaseCurrency", "base_currency"))),
		Currency:        strings.TrimSpace(pickString(payload, "Currency", "currency")),
		PayTime:         strings.TrimSpace(pickString(payload, "PayTime", "pay_time")),
		PassThroughInfo: strings.TrimSpace(pickString(payload, "PassThroughInfo", "pass_through_info")),
	}
	return callback, nil
}

func VerifyCallback(data *CallbackData, notifySecret string) error {
	if data == nil {
		return ErrConfigInvalid
	}
	if strings.TrimSpace(notifySecret) == "" {
		return ErrConfigInvalid
	}
	expected := SignPayload(data.Raw, notifySecret)
	if !strings.EqualFold(expected, strings.TrimSpace(data.Signature)) {
		return ErrSignatureInvalid
	}
	return nil
}

func ParsePaidAt(raw string) *time.Time {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, text, time.Local)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func ParseAmount(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	num, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return ""
	}
	return strconv.FormatFloat(num, 'f', 2, 64)
}

func ToPaymentStatus(status int) string {
	switch status {
	case 1:
		return constants.PaymentStatusSuccess
	case 2:
		return constants.PaymentStatusExpired
	default:
		return constants.PaymentStatusPending
	}
}

func SignPayload(payload map[string]interface{}, notifySecret string) string {
	keys := make([]string, 0, len(payload))
	for key, value := range payload {
		if strings.EqualFold(strings.TrimSpace(key), "Signature") {
			continue
		}
		if isEmptyValue(value) {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+normalizeSignValue(payload[key]))
	}
	signText := strings.Join(parts, "&") + strings.TrimSpace(notifySecret)
	sum := md5.Sum([]byte(signText))
	return strings.ToLower(hex.EncodeToString(sum[:]))
}

func isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case json.Number:
		return strings.TrimSpace(v.String()) == ""
	default:
		return false
	}
}

func normalizeSignValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return strings.TrimSpace(string(data))
	}
}

func pickString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			if text, ok := val.(string); ok {
				return text
			}
			if num, ok := val.(json.Number); ok {
				return num.String()
			}
			if val != nil {
				return fmt.Sprintf("%v", val)
			}
		}
	}
	return ""
}

func pickInt(data map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		val, ok := data[key]
		if !ok || val == nil {
			continue
		}
		switch v := val.(type) {
		case int:
			return v
		case int8:
			return int(v)
		case int16:
			return int(v)
		case int32:
			return int(v)
		case int64:
			return int(v)
		case uint:
			return int(v)
		case uint8:
			return int(v)
		case uint16:
			return int(v)
		case uint32:
			return int(v)
		case uint64:
			return int(v)
		case float64:
			return int(v)
		case json.Number:
			parsed, err := v.Int64()
			if err == nil {
				return int(parsed)
			}
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(v))
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func getString(data map[string]interface{}, key string) string {
	return pickString(data, key)
}

func getStringFromMap(data map[string]interface{}, parent string, key string) string {
	raw, ok := data[parent]
	if !ok || raw == nil {
		return ""
	}
	mapping, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	return pickString(mapping, key)
}

func postJSON(ctx context.Context, endpoint string, payload map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}
	return respBody, nil
}
