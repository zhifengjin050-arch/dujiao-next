package epusdt

import (
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

var (
	ErrConfigInvalid       = errors.New("epusdt config invalid")
	ErrRequestFailed       = errors.New("epusdt request failed")
	ErrResponseInvalid     = errors.New("epusdt response invalid")
	ErrSignatureInvalid    = errors.New("epusdt signature invalid")
	ErrTradeTypeNotSupport = errors.New("epusdt trade type not supported")
)

// ??????
const (
	StatusWaiting = 1 // ????
	StatusSuccess = 2 // ????
	StatusExpired = 3 // ????

	epusdtTradeTypeUSDTTRC20 = "usdt.trc20"
	epusdtTradeTypeUSDTERC20 = "usdt.erc20"
	epusdtTradeTypeUSDTBEP20 = "usdt.bep20"
	epusdtTradeTypeUSDTPOLY  = "usdt.polygon"
	epusdtTradeTypeUSDCTRC20 = "usdc.trc20"
	epusdtTradeTypeUSDCERC20 = "usdc.erc20"
	epusdtTradeTypeUSDCPOLY  = "usdc.polygon"
	epusdtTradeTypeUSDCBEP20 = "usdc.bep20"
	epusdtTradeTypeTRX       = "tron.trx"
	epusdtTradeTypeETH       = "eth.eth"
	epusdtTradeTypeBNB       = "bsc.bnb"

	epusdtChannelTypeUSDT      = "usdt"
	epusdtChannelTypeUSDTTRC20 = "usdt-trc20"
	epusdtChannelTypeUSDCTRC20 = "usdc-trc20"
	epusdtChannelTypeTRX       = "trx"

	epusdtCreateTransactionPath = "/api/v1/order/create-transaction"
	epusdtStatusSuccessMsg      = "status is not success"
)

// Config BEpusdt ??
type Config struct {
	GatewayURL string `json:"gateway_url"` // ????,? https://usdt.example.com
	AuthToken  string `json:"auth_token"`  // API Token
	TradeType  string `json:"trade_type"`  // ????,? usdt.trc20
	Fiat       string `json:"fiat"`        // ????,?? CNY
	NotifyURL  string `json:"notify_url"`  // ??????
	ReturnURL  string `json:"return_url"`  // ??????
}

// CreateInput ??????
type CreateInput struct {
	OrderNo   string
	Amount    string
	Name      string
	NotifyURL string
	ReturnURL string
}

// CreateResult ??????
type CreateResult struct {
	TradeID      string                 // ???? ID
	OrderID      string                 // ??????
	Amount       string                 // ??????(??)
	ActualAmount string                 // ??????(????)
	Token        string                 // ????
	PaymentURL   string                 // ?????
	Raw          map[string]interface{} // ????
}

// CallbackData ????
type CallbackData struct {
	TradeID            string      `json:"trade_id"`
	OrderID            string      `json:"order_id"`
	Amount             interface{} `json:"amount"`        // ??? float64 ? string
	ActualAmount       interface{} `json:"actual_amount"` // ??? float64 ? string
	Token              string      `json:"token"`
	BlockTransactionID string      `json:"block_transaction_id"`
	Signature          string      `json:"signature"`
	Status             int         `json:"status"`
}

// GetAmount ????(float64)
func (c *CallbackData) GetAmount() float64 {
	switch v := c.Amount.(type) {
	case float64:
		return v
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

// GetActualAmount ??????(float64)
func (c *CallbackData) GetActualAmount() float64 {
	switch v := c.ActualAmount.(type) {
	case float64:
		return v
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

// ParseConfig ????
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	return common.ParseConfig[Config](raw, ErrConfigInvalid)
}

// ValidateConfig ????
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.GatewayURL) == "" {
		return fmt.Errorf("%w: gateway_url is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.AuthToken) == "" {
		return fmt.Errorf("%w: auth_token is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.NotifyURL) == "" {
		return fmt.Errorf("%w: notify_url is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.ReturnURL) == "" {
		return fmt.Errorf("%w: return_url is required", ErrConfigInvalid)
	}
	return nil
}

func (c *Config) Normalize() {
	c.GatewayURL = strings.TrimRight(strings.TrimSpace(c.GatewayURL), "/")
	c.AuthToken = strings.TrimSpace(c.AuthToken)
	c.TradeType = strings.TrimSpace(c.TradeType)
	c.Fiat = strings.TrimSpace(c.Fiat)
	c.NotifyURL = strings.TrimSpace(c.NotifyURL)
	c.ReturnURL = strings.TrimSpace(c.ReturnURL)
	if c.TradeType == "" {
		c.TradeType = epusdtTradeTypeUSDTTRC20
	}
	if c.Fiat == "" {
		c.Fiat = constants.SiteCurrencyDefault
	}
}

// CreatePayment ??????
func CreatePayment(ctx context.Context, cfg *Config, input CreateInput) (*CreateResult, error) {
	if cfg == nil {
		return nil, ErrConfigInvalid
	}
	if input.OrderNo == "" || input.Amount == "" {
		return nil, ErrConfigInvalid
	}

	notifyURL := input.NotifyURL
	if notifyURL == "" {
		notifyURL = cfg.NotifyURL
	}
	returnURL := input.ReturnURL
	if returnURL == "" {
		returnURL = cfg.ReturnURL
	}

	// ? amount ??????? float64
	amountFloat, err := strconv.ParseFloat(input.Amount, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid amount", ErrConfigInvalid)
	}

	params := map[string]interface{}{
		"order_id":     input.OrderNo,
		"amount":       amountFloat,
		"notify_url":   notifyURL,
		"redirect_url": returnURL,
		"trade_type":   cfg.TradeType,
		"fiat":         cfg.Fiat,
	}
	if input.Name != "" {
		params["name"] = input.Name
	}

	// ????
	signature := Sign(params, cfg.AuthToken)
	params["signature"] = signature

	endpoint := cfg.GatewayURL + epusdtCreateTransactionPath
	respBytes, err := postJSON(ctx, endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	var resp struct {
		StatusCode int    `json:"status_code"`
		Message    string `json:"message"`
		Data       struct {
			Fiat           string `json:"fiat"`
			TradeID        string `json:"trade_id"`
			OrderID        string `json:"order_id"`
			Amount         string `json:"amount"`
			ActualAmount   string `json:"actual_amount"`
			Token          string `json:"token"`
			ExpirationTime int    `json:"expiration_time"`
			PaymentURL     string `json:"payment_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrResponseInvalid, err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%w: %s", ErrResponseInvalid, resp.Message)
	}

	var raw map[string]interface{}
	_ = json.Unmarshal(respBytes, &raw)

	return &CreateResult{
		TradeID:      resp.Data.TradeID,
		OrderID:      resp.Data.OrderID,
		Amount:       resp.Data.Amount,
		ActualAmount: resp.Data.ActualAmount,
		Token:        resp.Data.Token,
		PaymentURL:   resp.Data.PaymentURL,
		Raw:          raw,
	}, nil
}

// VerifyCallback ??????
func VerifyCallback(cfg *Config, data *CallbackData) error {
	if cfg == nil || data == nil {
		return ErrConfigInvalid
	}

	if data.Status != StatusSuccess {
		return fmt.Errorf("%w: %s", ErrResponseInvalid, epusdtStatusSuccessMsg)
	}

	params := map[string]interface{}{
		"trade_id":             data.TradeID,
		"order_id":             data.OrderID,
		"amount":               data.GetAmount(),
		"actual_amount":        data.GetActualAmount(),
		"token":                data.Token,
		"block_transaction_id": data.BlockTransactionID,
		"status":               data.Status,
	}

	expected := Sign(params, cfg.AuthToken)
	if !strings.EqualFold(expected, data.Signature) {
		return ErrSignatureInvalid
	}
	return nil
}

// ParseCallback ??????
func ParseCallback(body []byte) (*CallbackData, error) {
	if len(body) == 0 {
		return nil, ErrResponseInvalid
	}
	var data CallbackData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrResponseInvalid, err)
	}
	return &data, nil
}

// Sign ????
// ????:
// 1. ???????? signature ???
// 2. ???? ASCII ???????
// 3. ? key=value ????,?? & ??
// 4. ????? AuthToken(? & ??)
// 5. MD5 ??????
func Sign(params map[string]interface{}, authToken string) string {
	var keys []string
	for k, v := range params {
		if k == "signature" {
			continue
		}
		if isEmptyValue(v) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		v := params[k]
		pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
	}

	content := strings.Join(pairs, "&") + authToken
	sum := md5.Sum([]byte(content))
	return strings.ToLower(hex.EncodeToString(sum[:]))
}

func isEmptyValue(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val) == ""
	case int, int8, int16, int32, int64:
		return false
	case uint, uint8, uint16, uint32, uint64:
		return false
	case float32, float64:
		return false
	case bool:
		return false
	default:
		return false
	}
}

func postJSON(ctx context.Context, endpoint string, params map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// IsSupportedChannelType ???????????
func IsSupportedChannelType(channelType string) bool {
	return ResolveTradeType(channelType) != ""
}

// ResolveTradeType ?? channel_type ?? trade_type
func ResolveTradeType(channelType string) string {
	switch strings.ToLower(strings.TrimSpace(channelType)) {
	case epusdtChannelTypeUSDT, epusdtChannelTypeUSDTTRC20:
		return epusdtTradeTypeUSDTTRC20
	case epusdtChannelTypeUSDCTRC20:
		return epusdtTradeTypeUSDCTRC20
	case epusdtChannelTypeTRX:
		return epusdtTradeTypeTRX
	default:
		return ""
	}
}

// IsSupportedTradeType ???????????
func IsSupportedTradeType(tradeType string) bool {
	supported := []string{
		epusdtTradeTypeUSDTTRC20, epusdtTradeTypeUSDTERC20, epusdtTradeTypeUSDTBEP20, epusdtTradeTypeUSDTPOLY,
		epusdtTradeTypeTRX, epusdtTradeTypeETH, epusdtTradeTypeBNB,
		epusdtTradeTypeUSDCTRC20, epusdtTradeTypeUSDCERC20, epusdtTradeTypeUSDCPOLY, epusdtTradeTypeUSDCBEP20,
	}
	t := strings.ToLower(strings.TrimSpace(tradeType))
	for _, s := range supported {
		if s == t {
			return true
		}
	}
	// ???? trade_type,? BEpusdt ?????
	return true
}

// ToPaymentStatus ? BEpusdt ?????????
func ToPaymentStatus(status int) string {
	switch status {
	case StatusSuccess:
		return constants.PaymentStatusSuccess
	case StatusExpired:
		return constants.PaymentStatusExpired
	default:
		return constants.PaymentStatusPending
	}
}
