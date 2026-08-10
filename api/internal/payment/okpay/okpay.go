package okpay

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
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/payment/common"
	"github.com/shopspring/decimal"
)

const (
	defaultGatewayURL = "https://api.okaypay.me/shop"
	payLinkPath       = "/payLink"
)

var (
	ErrConfigInvalid    = errors.New("okpay config invalid")
	ErrRequestFailed    = errors.New("okpay request failed")
	ErrResponseInvalid  = errors.New("okpay response invalid")
	ErrSignatureInvalid = errors.New("okpay signature invalid")
)

type Config struct {
	GatewayURL    string `json:"gateway_url"`
	MerchantID    string `json:"merchant_id"`
	MerchantToken string `json:"merchant_token"`
	ReturnURL     string `json:"return_url"`
	CallbackURL   string `json:"callback_url"`
	DisplayName   string `json:"display_name"`
	ExchangeRate  string `json:"exchange_rate"`
	Coin          string `json:"coin"`
	Status        string `json:"status"`
}

type CreateInput struct {
	UniqueID    string
	Name        string
	Amount      string
	ReturnURL   string
	CallbackURL string
	Coin        string
	Status      string
}

type CreateResult struct {
	OrderID string
	PayURL  string
	Raw     map[string]interface{}
}

type OrderedPair struct {
	Key   string
	Value string
}

type CallbackData struct {
	RawPairs      []OrderedPair
	Raw           map[string]string
	MerchantID    string
	Code          string
	RequestStatus string
	Sign          string
	OrderID       string
	UniqueID      string
	PayUserID     string
	Amount        string
	Coin          string
	PaymentStatus string
	Type          string
}

func ParseConfig(raw map[string]interface{}) (*Config, error) {
	return common.ParseConfig[Config](raw, ErrConfigInvalid)
}

func (c *Config) Normalize() {
	c.GatewayURL = strings.TrimRight(strings.TrimSpace(c.GatewayURL), "/")
	c.MerchantID = strings.TrimSpace(c.MerchantID)
	c.MerchantToken = strings.TrimSpace(c.MerchantToken)
	c.ReturnURL = strings.TrimSpace(c.ReturnURL)
	c.CallbackURL = strings.TrimSpace(c.CallbackURL)
	c.DisplayName = strings.TrimSpace(c.DisplayName)
	c.ExchangeRate = strings.TrimSpace(c.ExchangeRate)
	c.Coin = strings.ToUpper(strings.TrimSpace(c.Coin))
	c.Status = strings.TrimSpace(c.Status)
	if c.GatewayURL == "" {
		c.GatewayURL = defaultGatewayURL
	}
	if c.ExchangeRate == "" {
		c.ExchangeRate = "1"
	}
}

func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.MerchantID) == "" {
		return fmt.Errorf("%w: merchant_id is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.MerchantToken) == "" {
		return fmt.Errorf("%w: merchant_token is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.ReturnURL) == "" {
		return fmt.Errorf("%w: return_url is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.CallbackURL) == "" {
		return fmt.Errorf("%w: callback_url is required", ErrConfigInvalid)
	}
	if gatewayURL := strings.TrimSpace(cfg.GatewayURL); gatewayURL == "" {
		return fmt.Errorf("%w: gateway_url is required", ErrConfigInvalid)
	}
	if _, err := ParseExchangeRate(cfg.ExchangeRate); err != nil {
		return err
	}
	if cfg.Coin != "" && !isSupportedCoin(cfg.Coin) {
		return fmt.Errorf("%w: unsupported coin %s", ErrConfigInvalid, cfg.Coin)
	}
	return nil
}

func IsSupportedChannelType(channelType string) bool {
	switch strings.ToLower(strings.TrimSpace(channelType)) {
	case constants.PaymentChannelTypeUsdt, constants.PaymentChannelTypeTrx:
		return true
	default:
		return false
	}
}

func ResolveCoin(channelType string) string {
	switch strings.ToLower(strings.TrimSpace(channelType)) {
	case constants.PaymentChannelTypeUsdt:
		return "USDT"
	case constants.PaymentChannelTypeTrx:
		return "TRX"
	default:
		return ""
	}
}

func CreatePayment(ctx context.Context, cfg *Config, input CreateInput) (*CreateResult, error) {
	if cfg == nil {
		return nil, ErrConfigInvalid
	}
	if strings.TrimSpace(input.UniqueID) == "" || strings.TrimSpace(input.Amount) == "" {
		return nil, ErrConfigInvalid
	}
	coin := strings.ToUpper(strings.TrimSpace(input.Coin))
	if coin == "" {
		coin = strings.ToUpper(strings.TrimSpace(cfg.Coin))
	}
	if !isSupportedCoin(coin) {
		return nil, fmt.Errorf("%w: coin is required", ErrConfigInvalid)
	}
	returnURL := strings.TrimSpace(input.ReturnURL)
	if returnURL == "" {
		returnURL = strings.TrimSpace(cfg.ReturnURL)
	}
	callbackURL := strings.TrimSpace(input.CallbackURL)
	if callbackURL == "" {
		callbackURL = strings.TrimSpace(cfg.CallbackURL)
	}
	if returnURL == "" || callbackURL == "" {
		return nil, ErrConfigInvalid
	}
	convertedAmount, err := ConvertAmountByRate(strings.TrimSpace(input.Amount), cfg.ExchangeRate)
	if err != nil {
		return nil, err
	}

	payload := map[string]string{
		"unique_id":    strings.TrimSpace(input.UniqueID),
		"amount":       convertedAmount.StringFixed(8),
		"return_url":   returnURL,
		"callback_url": callbackURL,
		"coin":         coin,
	}
	if name := strings.TrimSpace(input.Name); name != "" {
		payload["name"] = name
	} else if name := strings.TrimSpace(cfg.DisplayName); name != "" {
		payload["name"] = name
	}
	if status := strings.TrimSpace(input.Status); status != "" {
		payload["status"] = status
	} else if status := strings.TrimSpace(cfg.Status); status != "" {
		payload["status"] = status
	}

	signedPayload := SignPayload(payload, cfg.MerchantID, cfg.MerchantToken)
	body, err := postForm(ctx, cfg.GatewayURL+payLinkPath, signedPayload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: decode response failed", ErrResponseInvalid)
	}
	data := common.ReadMap(raw, "data")
	if data == nil {
		if items, ok := raw["data"].([]interface{}); ok && len(items) > 0 {
			if first, ok := items[0].(map[string]interface{}); ok {
				data = first
			}
		}
	}

	result := &CreateResult{
		OrderID: strings.TrimSpace(common.ReadString(data, "order_id")),
		PayURL:  strings.TrimSpace(common.ReadString(data, "pay_url")),
		Raw:     raw,
	}
	if result.OrderID == "" || result.PayURL == "" {
		return nil, fmt.Errorf("%w: missing order_id/pay_url", ErrResponseInvalid)
	}
	return result, nil
}

func SignPayload(payload map[string]string, merchantID string, merchantToken string) map[string]string {
	values := make([]OrderedPair, 0, len(payload)+1)
	for key, value := range payload {
		values = append(values, OrderedPair{
			Key:   strings.TrimSpace(key),
			Value: strings.TrimSpace(value),
		})
	}
	values = append(values, OrderedPair{Key: "id", Value: strings.TrimSpace(merchantID)})
	sort.Slice(values, func(i, j int) bool {
		return values[i].Key < values[j].Key
	})
	sign := buildSignature(values, merchantToken)

	result := make(map[string]string, len(values)+1)
	for _, item := range values {
		if item.Key == "" || item.Value == "" {
			continue
		}
		result[item.Key] = item.Value
	}
	result["sign"] = sign
	return result
}

func ParseCallback(body []byte) (*CallbackData, error) {
	var (
		pairs []OrderedPair
		err   error
	)
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, ErrResponseInvalid
	}
	if strings.HasPrefix(trimmed, "{") {
		pairs, err = parseOrderedJSON(body)
	} else {
		pairs = parseOrderedForm(body)
	}
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, ErrResponseInvalid
	}
	raw := make(map[string]string, len(pairs))
	for _, item := range pairs {
		raw[item.Key] = item.Value
	}
	callback := &CallbackData{
		RawPairs:      pairs,
		Raw:           raw,
		MerchantID:    strings.TrimSpace(raw["id"]),
		Code:          strings.TrimSpace(raw["code"]),
		RequestStatus: strings.TrimSpace(raw["status"]),
		Sign:          strings.TrimSpace(raw["sign"]),
		OrderID:       strings.TrimSpace(raw["data[order_id]"]),
		UniqueID:      strings.TrimSpace(raw["data[unique_id]"]),
		PayUserID:     strings.TrimSpace(raw["data[pay_user_id]"]),
		Amount:        strings.TrimSpace(raw["data[amount]"]),
		Coin:          strings.ToUpper(strings.TrimSpace(raw["data[coin]"])),
		PaymentStatus: strings.TrimSpace(raw["data[status]"]),
		Type:          strings.TrimSpace(raw["data[type]"]),
	}
	if callback.Sign == "" {
		return nil, ErrResponseInvalid
	}
	return callback, nil
}

func VerifyCallback(cfg *Config, data *CallbackData) error {
	if cfg == nil || data == nil {
		return ErrConfigInvalid
	}
	if strings.TrimSpace(cfg.MerchantID) != "" && data.MerchantID != "" && data.MerchantID != strings.TrimSpace(cfg.MerchantID) {
		return ErrSignatureInvalid
	}
	if data.Sign == "" {
		return ErrSignatureInvalid
	}
	expected := buildSignature(stripSignPairs(data.RawPairs), cfg.MerchantToken)
	if strings.EqualFold(expected, data.Sign) {
		return nil
	}
	sortedPairs := make([]OrderedPair, 0, len(data.Raw))
	for key, value := range data.Raw {
		if strings.EqualFold(strings.TrimSpace(key), "sign") {
			continue
		}
		sortedPairs = append(sortedPairs, OrderedPair{Key: key, Value: value})
	}
	sort.SliceStable(sortedPairs, func(i, j int) bool {
		return sortedPairs[i].Key < sortedPairs[j].Key
	})
	expected = buildSignature(sortedPairs, cfg.MerchantToken)
	if !strings.EqualFold(expected, data.Sign) {
		return ErrSignatureInvalid
	}
	return nil
}

func ParseExchangeRate(raw string) (decimal.Decimal, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "1"
	}
	rate, err := decimal.NewFromString(trimmed)
	if err != nil {
		return decimal.Zero, fmt.Errorf("%w: exchange_rate invalid", ErrConfigInvalid)
	}
	if rate.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, fmt.Errorf("%w: exchange_rate must be greater than 0", ErrConfigInvalid)
	}
	return rate, nil
}

func ConvertAmountByRate(baseAmount string, exchangeRate string) (decimal.Decimal, error) {
	amount, err := decimal.NewFromString(strings.TrimSpace(baseAmount))
	if err != nil {
		return decimal.Zero, fmt.Errorf("%w: invalid amount", ErrConfigInvalid)
	}
	rate, err := ParseExchangeRate(exchangeRate)
	if err != nil {
		return decimal.Zero, err
	}
	return amount.Mul(rate).Round(8), nil
}

func ToPaymentStatus(requestStatus string, paymentStatus string) string {
	if !strings.EqualFold(strings.TrimSpace(requestStatus), "success") {
		return constants.PaymentStatusFailed
	}
	switch strings.TrimSpace(paymentStatus) {
	case "1":
		return constants.PaymentStatusSuccess
	case "2":
		return constants.PaymentStatusFailed
	default:
		return constants.PaymentStatusPending
	}
}

func postForm(ctx context.Context, endpoint string, payload map[string]string) ([]byte, error) {
	values := url.Values{}
	for key, value := range payload {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		values.Set(key, value)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return body, nil
}

func parseOrderedForm(body []byte) []OrderedPair {
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		return nil
	}
	items := strings.Split(raw, "&")
	result := make([]OrderedPair, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		key, err := url.QueryUnescape(parts[0])
		if err != nil {
			key = parts[0]
		}
		value := ""
		if len(parts) == 2 {
			if decoded, decodeErr := url.QueryUnescape(parts[1]); decodeErr == nil {
				value = decoded
			} else {
				value = parts[1]
			}
		}
		result = append(result, OrderedPair{
			Key:   strings.TrimSpace(key),
			Value: strings.TrimSpace(value),
		})
	}
	return result
}

func parseOrderedJSON(body []byte) ([]OrderedPair, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	token, err := decoder.Token()
	if err != nil {
		return nil, ErrResponseInvalid
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return nil, ErrResponseInvalid
	}

	pairs, err := decodeJSONObject(decoder, "")
	if err != nil {
		return nil, err
	}
	return pairs, nil
}

func decodeJSONObject(decoder *json.Decoder, prefix string) ([]OrderedPair, error) {
	result := make([]OrderedPair, 0)
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, ErrResponseInvalid
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, ErrResponseInvalid
		}
		nextToken, err := decoder.Token()
		if err != nil {
			return nil, ErrResponseInvalid
		}

		fullKey := key
		if prefix != "" {
			fullKey = prefix + "[" + key + "]"
		}

		switch token := nextToken.(type) {
		case json.Delim:
			switch token {
			case '{':
				nested, err := decodeJSONObject(decoder, fullKey)
				if err != nil {
					return nil, err
				}
				result = append(result, nested...)
			case '[':
				nested, err := decodeJSONArray(decoder, fullKey)
				if err != nil {
					return nil, err
				}
				result = append(result, nested...)
			default:
				return nil, ErrResponseInvalid
			}
		default:
			result = append(result, OrderedPair{
				Key:   strings.TrimSpace(fullKey),
				Value: scalarTokenToString(nextToken),
			})
		}
	}

	endToken, err := decoder.Token()
	if err != nil {
		return nil, ErrResponseInvalid
	}
	if delim, ok := endToken.(json.Delim); !ok || delim != '}' {
		return nil, ErrResponseInvalid
	}
	return result, nil
}

func decodeJSONArray(decoder *json.Decoder, prefix string) ([]OrderedPair, error) {
	result := make([]OrderedPair, 0)
	index := 0
	for decoder.More() {
		nextToken, err := decoder.Token()
		if err != nil {
			return nil, ErrResponseInvalid
		}
		fullKey := fmt.Sprintf("%s[%d]", prefix, index)
		switch token := nextToken.(type) {
		case json.Delim:
			switch token {
			case '{':
				nested, err := decodeJSONObject(decoder, fullKey)
				if err != nil {
					return nil, err
				}
				result = append(result, nested...)
			case '[':
				nested, err := decodeJSONArray(decoder, fullKey)
				if err != nil {
					return nil, err
				}
				result = append(result, nested...)
			default:
				return nil, ErrResponseInvalid
			}
		default:
			result = append(result, OrderedPair{
				Key:   strings.TrimSpace(fullKey),
				Value: scalarTokenToString(nextToken),
			})
		}
		index++
	}

	endToken, err := decoder.Token()
	if err != nil {
		return nil, ErrResponseInvalid
	}
	if delim, ok := endToken.(json.Delim); !ok || delim != ']' {
		return nil, ErrResponseInvalid
	}
	return result, nil
}

func scalarTokenToString(token interface{}) string {
	switch value := token.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case json.Number:
		return value.String()
	case bool:
		if value {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func stripSignPairs(pairs []OrderedPair) []OrderedPair {
	result := make([]OrderedPair, 0, len(pairs))
	for _, item := range pairs {
		if strings.EqualFold(strings.TrimSpace(item.Key), "sign") {
			continue
		}
		result = append(result, item)
	}
	return result
}

func buildSignature(pairs []OrderedPair, merchantToken string) string {
	filtered := make([]string, 0, len(pairs))
	for _, item := range pairs {
		key := strings.TrimSpace(item.Key)
		value := strings.TrimSpace(item.Value)
		if key == "" || value == "" {
			continue
		}
		filtered = append(filtered, key+"="+value)
	}
	query := strings.Join(filtered, "&")
	sum := md5.Sum([]byte(query + "&token=" + strings.TrimSpace(merchantToken)))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func isSupportedCoin(coin string) bool {
	switch strings.ToUpper(strings.TrimSpace(coin)) {
	case "USDT", "TRX":
		return true
	default:
		return false
	}
}
