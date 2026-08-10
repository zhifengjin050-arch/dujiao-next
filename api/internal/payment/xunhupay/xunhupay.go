package xunhupay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// CreateInput ????????
type CreateInput struct {
	OrderNo     string
	Amount      string
	Title       string
	NotifyURL   string
	ReturnURL   string
	CallbackURL string
	WapName     string
	TradeType   string
}

// CreateResult ????????
type CreateResult struct {
	PayURL  string
	QRCode  string
	TradeNo string
	Raw     map[string]interface{}
}

// CreatePayment ??????
func CreatePayment(ctx context.Context, cfg *Config, input CreateInput) (*CreateResult, error) {
	if cfg == nil {
		return nil, ErrConfigInvalid
	}
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.OrderNo) == "" || strings.TrimSpace(input.Amount) == "" {
		return nil, fmt.Errorf("%w: order_no/amount is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(input.Title) == "" {
		input.Title = input.OrderNo
	}
	if strings.TrimSpace(input.NotifyURL) == "" {
		input.NotifyURL = cfg.NotifyURL
	}
	if strings.TrimSpace(input.ReturnURL) == "" {
		input.ReturnURL = cfg.ReturnURL
	}
	if strings.TrimSpace(input.WapName) == "" {
		input.WapName = cfg.WapName
	}

	params := map[string]string{
		"version":         "1.1",
		"trade_order_id":  input.OrderNo,
		"total_fee":       input.Amount,
		"title":           input.Title,
		"notify_url":      input.NotifyURL,
		"return_url":      input.ReturnURL,
		"wap_name":        input.WapName,
		"callback_url":    input.CallbackURL,
		"type":            input.TradeType,
	}
	if strings.TrimSpace(input.CallbackURL) == "" {
		params["callback_url"] = cfg.CallbackURL
	}

	client := NewClient(cfg.AppID, cfg.AppSecret, cfg.GatewayURL, cfg.QueryURL)
	body, err := client.Post(ctx, client.Gateway(), params)
	if err != nil {
		return nil, err
	}
	body, err = normalizeResponseBody(body)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	_ = json.Unmarshal(body, &raw)

	var resp struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
		Data   struct {
			JumpURL string `json:"jump_url"`
			URL     string `json:"url"`
			QRCode  string `json:"qrcode"`
			TradeNo string `json:"trade_order_id"`
		} `json:"data"`
		TradeNo string `json:"trade_order_id"`
		URL     string `json:"url"`
		QRCode  string `json:"qrcode"`
		JumpURL string `json:"jump_url"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Status != 1 && resp.Status != 0 {
		return nil, fmt.Errorf("xunhupay response invalid: %s", resp.Msg)
	}

	result := &CreateResult{Raw: raw}
	result.TradeNo = strings.TrimSpace(firstNonEmpty(resp.TradeNo, resp.Data.TradeNo))
	result.PayURL = strings.TrimSpace(firstNonEmpty(resp.JumpURL, resp.URL, resp.Data.JumpURL, resp.Data.URL))
	result.QRCode = strings.TrimSpace(firstNonEmpty(resp.QRCode, resp.Data.QRCode))
	if result.PayURL == "" && result.QRCode != "" {
		result.PayURL = result.QRCode
	}
	return result, nil
}

// QueryPayment ??????
func QueryPayment(ctx context.Context, cfg *Config, tradeOrderID string) (map[string]interface{}, error) {
	if cfg == nil {
		return nil, ErrConfigInvalid
	}
	if strings.TrimSpace(tradeOrderID) == "" {
		return nil, fmt.Errorf("%w: trade_order_id is required", ErrConfigInvalid)
	}
	client := NewClient(cfg.AppID, cfg.AppSecret, cfg.GatewayURL, cfg.QueryURL)
	params := map[string]string{
		"trade_order_id": tradeOrderID,
	}
	body, err := client.Post(ctx, client.QueryGateway(), params)
	if err != nil {
		return nil, err
	}
	body, err = normalizeResponseBody(body)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func normalizeResponseBody(body []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return trimmed, nil
	}
	if trimmed[0] != '"' {
		return trimmed, nil
	}
	var inner string
	if err := json.Unmarshal(trimmed, &inner); err != nil {
		return nil, err
	}
	return bytes.TrimSpace([]byte(inner)), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
