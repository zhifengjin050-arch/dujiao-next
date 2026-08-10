package wechatpay

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/payment/common"

	"github.com/shopspring/decimal"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
)

var (
	ErrConfigInvalid    = errors.New("wechatpay config invalid")
	ErrRequestFailed    = errors.New("wechatpay request failed")
	ErrResponseInvalid  = errors.New("wechatpay response invalid")
	ErrSignatureInvalid = errors.New("wechatpay signature invalid")
)

const (
	defaultBaseURL = "https://api.mch.weixin.qq.com"
	defaultTimeout = 15 * time.Second

	wechatH5TypeWAP     = "WAP"
	wechatH5TypeIOS     = "IOS"
	wechatH5TypeAndroid = "ANDROID"

	wechatTradeStateSuccess    = "SUCCESS"
	wechatTradeStateRefund     = "REFUND"
	wechatTradeStateNotPay     = "NOTPAY"
	wechatTradeStateUserPaying = "USERPAYING"
	wechatTradeStateClosed     = "CLOSED"
	wechatTradeStateRevoked    = "REVOKED"
	wechatTradeStatePayError   = "PAYERROR"
)

// Config ?????????
type Config struct {
	common.ExchangeRateConfig
	AppID              string `json:"appid"`
	MerchantID         string `json:"mchid"`
	MerchantSerialNo   string `json:"merchant_serial_no"`
	MerchantPrivateKey string `json:"merchant_private_key"`
	APIV3Key           string `json:"api_v3_key"`
	NotifyURL          string `json:"notify_url"`
	H5RedirectURL      string `json:"h5_redirect_url"`
	H5Type             string `json:"h5_type"`
	H5WapURL           string `json:"h5_wap_url"`
	H5WapName          string `json:"h5_wap_name"`
	BaseURL            string `json:"base_url"`
}

// CreateInput ??????????
type CreateInput struct {
	OrderNo     string
	Amount      string
	Currency    string
	Description string
	ClientIP    string
	NotifyURL   string
}

// CreateResult ??????????
type CreateResult struct {
	PayURL   string
	QRCode   string
	PrepayID string
	Raw      map[string]interface{}
}

// QueryResult ?????????
type QueryResult struct {
	OrderNo       string
	TransactionID string
	Status        string
	Amount        string
	Currency      string
	Attach        string
	PaidAt        *time.Time
	Raw           map[string]interface{}
}

// WebhookResult ????????????
type WebhookResult struct {
	EventType     string
	OrderNo       string
	TransactionID string
	Status        string
	Amount        string
	Currency      string
	Attach        string
	PaidAt        *time.Time
	Raw           map[string]interface{}
}

// ParseConfig ?????
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	return common.ParseConfig[Config](raw, ErrConfigInvalid)
}

// ValidateConfig ?????
func ValidateConfig(cfg *Config, interactionMode string) error {
	if err := validateBaseConfig(cfg); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.H5RedirectURL) != "" {
		if _, err := url.ParseRequestURI(strings.TrimSpace(cfg.H5RedirectURL)); err != nil {
			return fmt.Errorf("%w: h5_redirect_url is invalid", ErrConfigInvalid)
		}
	}
	if strings.TrimSpace(cfg.H5WapURL) != "" {
		if _, err := url.ParseRequestURI(strings.TrimSpace(cfg.H5WapURL)); err != nil {
			return fmt.Errorf("%w: h5_wap_url is invalid", ErrConfigInvalid)
		}
	}
	if !IsSupportedInteractionMode(interactionMode) {
		return fmt.Errorf("%w: interaction_mode %s is not supported", ErrConfigInvalid, interactionMode)
	}
	if requiresH5RedirectURL(interactionMode) && strings.TrimSpace(cfg.H5RedirectURL) == "" {
		return fmt.Errorf("%w: h5_redirect_url is required for mode %s", ErrConfigInvalid, interactionMode)
	}
	if cfg.H5Type != "" {
		switch strings.ToUpper(strings.TrimSpace(cfg.H5Type)) {
		case wechatH5TypeWAP, wechatH5TypeIOS, wechatH5TypeAndroid:
		default:
			return fmt.Errorf("%w: h5_type is invalid", ErrConfigInvalid)
		}
	}
	return nil
}

// CreatePayment ????????
func CreatePayment(ctx context.Context, cfg *Config, input CreateInput, interactionMode string) (*CreateResult, error) {
	if err := ValidateConfig(cfg, interactionMode); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.OrderNo) == "" || strings.TrimSpace(input.Amount) == "" {
		return nil, fmt.Errorf("%w: order input is invalid", ErrConfigInvalid)
	}
	amountFen, err := convertAmountToFen(input.Amount)
	if err != nil {
		return nil, err
	}

	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()
	client, err := createAPIClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	notifyURL := strings.TrimSpace(input.NotifyURL)
	if notifyURL == "" {
		notifyURL = cfg.NotifyURL
	}

	currency := constants.SiteCurrencyDefault
	payload := map[string]interface{}{
		"appid":        cfg.AppID,
		"mchid":        cfg.MerchantID,
		"description":  buildDescription(input.Description, input.OrderNo),
		"out_trade_no": input.OrderNo,
		"notify_url":   notifyURL,
		"amount": map[string]interface{}{
			"total":    amountFen,
			"currency": currency,
		},
	}

	mode := strings.ToLower(strings.TrimSpace(interactionMode))
	endpoint := ""
	clientIP := normalizeClientIP(input.ClientIP)
	switch mode {
	case constants.PaymentInteractionRedirect:
		endpoint = "/v3/pay/transactions/h5"
		h5Info := map[string]interface{}{
			"type": cfg.H5Type,
		}
		if strings.TrimSpace(cfg.H5WapName) != "" {
			h5Info["app_name"] = strings.TrimSpace(cfg.H5WapName)
		}
		if strings.TrimSpace(cfg.H5WapURL) != "" {
			h5Info["app_url"] = strings.TrimSpace(cfg.H5WapURL)
		}
		payload["scene_info"] = map[string]interface{}{
			"payer_client_ip": clientIP,
			"h5_info":         h5Info,
		}
	case constants.PaymentInteractionQR:
		endpoint = "/v3/pay/transactions/native"
		payload["scene_info"] = map[string]interface{}{
			"payer_client_ip": clientIP,
		}
	default:
		return nil, fmt.Errorf("%w: interaction_mode %s is not supported", ErrConfigInvalid, interactionMode)
	}

	requestURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/") + endpoint
	raw, err := doPostJSON(ctx, client, requestURL, payload)
	if err != nil {
		return nil, err
	}
	result := &CreateResult{Raw: raw}
	if prepayID := strings.TrimSpace(readString(raw, "prepay_id")); prepayID != "" {
		result.PrepayID = prepayID
	}

	switch mode {
	case constants.PaymentInteractionRedirect:
		h5URL := strings.TrimSpace(readString(raw, "h5_url"))
		if h5URL == "" {
			return nil, fmt.Errorf("%w: missing h5_url", ErrResponseInvalid)
		}
		result.PayURL = appendRedirectURL(h5URL, cfg.H5RedirectURL)
	case constants.PaymentInteractionQR:
		codeURL := strings.TrimSpace(readString(raw, "code_url"))
		if codeURL == "" {
			return nil, fmt.Errorf("%w: missing code_url", ErrResponseInvalid)
		}
		result.QRCode = codeURL
	}
	return result, nil
}

// QueryOrderByOutTradeNo ????????????????
func QueryOrderByOutTradeNo(ctx context.Context, cfg *Config, orderNo string) (*QueryResult, error) {
	if err := validateBaseConfig(cfg); err != nil {
		return nil, err
	}
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil, fmt.Errorf("%w: order no is required", ErrConfigInvalid)
	}
	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()
	client, err := createAPIClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	requestURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/") +
		"/v3/pay/transactions/out-trade-no/" + url.PathEscape(orderNo) +
		"?mchid=" + url.QueryEscape(cfg.MerchantID)

	raw, err := doGetJSON(ctx, client, requestURL)
	if err != nil {
		return nil, err
	}
	return parseQueryResult(raw, orderNo)
}

// VerifyAndDecodeWebhook ??????????
func VerifyAndDecodeWebhook(ctx context.Context, cfg *Config, headers map[string]string, body []byte) (*WebhookResult, error) {
	if err := validateBaseConfig(cfg); err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("%w: empty webhook body", ErrResponseInvalid)
	}
	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()

	privateKey, err := parsePrivateKey(cfg.MerchantPrivateKey)
	if err != nil {
		return nil, err
	}

	mgr := downloader.MgrInstance()
	if !mgr.HasDownloader(ctx, cfg.MerchantID) {
		if err := mgr.RegisterDownloaderWithPrivateKey(ctx, privateKey, cfg.MerchantSerialNo, cfg.MerchantID, cfg.APIV3Key); err != nil {
			return nil, fmt.Errorf("%w: register certificate downloader failed", ErrRequestFailed)
		}
	}

	verifier := verifiers.NewSHA256WithRSAVerifier(mgr.GetCertificateVisitor(cfg.MerchantID))
	handler, err := notify.NewRSANotifyHandler(cfg.APIV3Key, verifier)
	if err != nil {
		return nil, fmt.Errorf("%w: init notify handler failed", ErrConfigInvalid)
	}

	notifyReq, transaction, err := parseNotifyTransaction(ctx, handler, headers, body)
	if err != nil {
		return nil, err
	}
	status, ok := ToPaymentStatus(pointerString(transaction.TradeState))
	if !ok {
		return nil, fmt.Errorf("%w: unsupported trade_state", ErrResponseInvalid)
	}

	amount := ""
	currency := ""
	if transaction.Amount != nil {
		if transaction.Amount.Total != nil {
			amount = fenToAmountString(*transaction.Amount.Total)
		}
		currency = strings.ToUpper(strings.TrimSpace(pointerString(transaction.Amount.Currency)))
	}

	raw := map[string]interface{}{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: decode webhook body failed", ErrResponseInvalid)
	}
	if notifyReq != nil && notifyReq.Resource != nil {
		if plaintext := strings.TrimSpace(notifyReq.Resource.Plaintext); plaintext != "" {
			resourcePlain := map[string]interface{}{}
			if err := json.Unmarshal([]byte(plaintext), &resourcePlain); err == nil {
				raw["resource_plaintext"] = resourcePlain
			}
		}
	}

	return &WebhookResult{
		EventType:     strings.TrimSpace(notifyReq.EventType),
		OrderNo:       strings.TrimSpace(pointerString(transaction.OutTradeNo)),
		TransactionID: strings.TrimSpace(pointerString(transaction.TransactionId)),
		Status:        status,
		Amount:        amount,
		Currency:      currency,
		Attach:        strings.TrimSpace(pointerString(transaction.Attach)),
		PaidAt:        parseTransactionTime(pointerString(transaction.SuccessTime)),
		Raw:           raw,
	}, nil
}

// ToPaymentStatus ?????????????????
func ToPaymentStatus(tradeState string) (string, bool) {
	state := strings.ToUpper(strings.TrimSpace(tradeState))
	switch state {
	case wechatTradeStateSuccess, wechatTradeStateRefund:
		return constants.PaymentStatusSuccess, true
	case wechatTradeStateNotPay, wechatTradeStateUserPaying:
		return constants.PaymentStatusPending, true
	case wechatTradeStateClosed, wechatTradeStateRevoked, wechatTradeStatePayError:
		return constants.PaymentStatusFailed, true
	default:
		return "", false
	}
}

// IsSupportedInteractionMode ?????????
func IsSupportedInteractionMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case constants.PaymentInteractionQR, constants.PaymentInteractionRedirect:
		return true
	default:
		return false
	}
}

func requiresH5RedirectURL(mode string) bool {
	return strings.ToLower(strings.TrimSpace(mode)) == constants.PaymentInteractionRedirect
}

func validateBaseConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.AppID) == "" {
		return fmt.Errorf("%w: appid is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.MerchantID) == "" {
		return fmt.Errorf("%w: mchid is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.MerchantSerialNo) == "" {
		return fmt.Errorf("%w: merchant_serial_no is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.MerchantPrivateKey) == "" {
		return fmt.Errorf("%w: merchant_private_key is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.APIV3Key) == "" {
		return fmt.Errorf("%w: api_v3_key is required", ErrConfigInvalid)
	}
	if len(strings.TrimSpace(cfg.APIV3Key)) != 32 {
		return fmt.Errorf("%w: api_v3_key must be 32 chars", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.NotifyURL) == "" {
		return fmt.Errorf("%w: notify_url is required", ErrConfigInvalid)
	}
	if _, err := url.ParseRequestURI(strings.TrimSpace(cfg.NotifyURL)); err != nil {
		return fmt.Errorf("%w: notify_url is invalid", ErrConfigInvalid)
	}
	if _, err := url.ParseRequestURI(strings.TrimSpace(cfg.BaseURL)); err != nil {
		return fmt.Errorf("%w: base_url is invalid", ErrConfigInvalid)
	}
	if err := validatePrivateKey(cfg.MerchantPrivateKey); err != nil {
		return err
	}
	return nil
}

func createAPIClient(ctx context.Context, cfg *Config) (*core.Client, error) {
	privateKey, err := parsePrivateKey(cfg.MerchantPrivateKey)
	if err != nil {
		return nil, err
	}
	client, err := core.NewClient(ctx,
		option.WithMerchantCredential(cfg.MerchantID, cfg.MerchantSerialNo, privateKey),
		option.WithoutValidator(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: init client failed", ErrConfigInvalid)
	}
	return client, nil
}

func doPostJSON(ctx context.Context, client *core.Client, requestURL string, payload map[string]interface{}) (map[string]interface{}, error) {
	result, err := client.Post(ctx, requestURL, payload)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return parseAPIResult(result)
}

func doGetJSON(ctx context.Context, client *core.Client, requestURL string) (map[string]interface{}, error) {
	result, err := client.Get(ctx, requestURL)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return parseAPIResult(result)
}

func wrapRequestError(err error) error {
	var apiErr *core.APIError
	if errors.As(err, &apiErr) {
		return fmt.Errorf("%w: %s", ErrResponseInvalid, strings.TrimSpace(apiErr.Message))
	}
	return fmt.Errorf("%w: %v", ErrRequestFailed, err)
}

func parseAPIResult(result *core.APIResult) (map[string]interface{}, error) {
	if result == nil || result.Response == nil || result.Response.Body == nil {
		return nil, fmt.Errorf("%w: empty response", ErrResponseInvalid)
	}
	defer result.Response.Body.Close()

	respBody, readErr := io.ReadAll(result.Response.Body)
	if readErr != nil {
		return nil, fmt.Errorf("%w: read response failed", ErrResponseInvalid)
	}
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 {
		if len(respBody) > 0 {
			return nil, fmt.Errorf("%w: status %d body %s", ErrResponseInvalid, result.Response.StatusCode, strings.TrimSpace(string(respBody)))
		}
		return nil, fmt.Errorf("%w: status %d", ErrResponseInvalid, result.Response.StatusCode)
	}
	if len(respBody) == 0 {
		return nil, fmt.Errorf("%w: empty response body", ErrResponseInvalid)
	}

	raw := map[string]interface{}{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("%w: decode response failed", ErrResponseInvalid)
	}
	return raw, nil
}

func parseQueryResult(raw map[string]interface{}, fallbackOrderNo string) (*QueryResult, error) {
	status, ok := ToPaymentStatus(readString(raw, "trade_state"))
	if !ok {
		return nil, fmt.Errorf("%w: unsupported trade_state", ErrResponseInvalid)
	}
	amount := ""
	if amountFen, ok := readInt64(raw, "amount", "total"); ok {
		amount = fenToAmountString(amountFen)
	}
	return &QueryResult{
		OrderNo:       pickFirstNonEmpty(readString(raw, "out_trade_no"), strings.TrimSpace(fallbackOrderNo)),
		TransactionID: readString(raw, "transaction_id"),
		Status:        status,
		Amount:        amount,
		Currency:      strings.ToUpper(strings.TrimSpace(readString(raw, "amount", "currency"))),
		Attach:        readString(raw, "attach"),
		PaidAt:        parseTransactionTime(readString(raw, "success_time")),
		Raw:           raw,
	}, nil
}

func parseNotifyTransaction(ctx context.Context, handler *notify.Handler, headers map[string]string, body []byte) (*notify.Request, *payments.Transaction, error) {
	requestURL := "https://notify.wechat.example/callback"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: build webhook request failed", ErrResponseInvalid)
	}
	for key, value := range headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		req.Header.Set(key, value)
	}

	content := new(payments.Transaction)
	notifyReq, err := handler.ParseNotifyRequest(ctx, req, content)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrSignatureInvalid, err)
	}
	return notifyReq, content, nil
}

func convertAmountToFen(amount string) (int64, error) {
	amountDec, err := decimal.NewFromString(strings.TrimSpace(amount))
	if err != nil {
		return 0, fmt.Errorf("%w: amount is invalid", ErrConfigInvalid)
	}
	if amountDec.LessThanOrEqual(decimal.Zero) {
		return 0, fmt.Errorf("%w: amount must be greater than zero", ErrConfigInvalid)
	}
	fen := amountDec.Mul(decimal.NewFromInt(100))
	if !fen.Equal(fen.Truncate(0)) {
		return 0, fmt.Errorf("%w: amount precision exceeds fen", ErrConfigInvalid)
	}
	return fen.IntPart(), nil
}

func fenToAmountString(amountFen int64) string {
	return decimal.NewFromInt(amountFen).Div(decimal.NewFromInt(100)).StringFixed(2)
}

func normalizeClientIP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "127.0.0.1"
	}
	if parsed := net.ParseIP(raw); parsed != nil {
		return parsed.String()
	}
	host, _, err := net.SplitHostPort(raw)
	if err == nil {
		if parsed := net.ParseIP(strings.TrimSpace(host)); parsed != nil {
			return parsed.String()
		}
	}
	return "127.0.0.1"
}

func appendRedirectURL(h5URL string, redirectURL string) string {
	h5URL = strings.TrimSpace(h5URL)
	redirectURL = strings.TrimSpace(redirectURL)
	if h5URL == "" || redirectURL == "" {
		return h5URL
	}
	parsed, err := url.Parse(h5URL)
	if err != nil {
		return h5URL
	}
	query := parsed.Query()
	query.Set("redirect_url", redirectURL)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func readString(raw map[string]interface{}, keys ...string) string {
	if len(keys) == 0 {
		return ""
	}
	var current interface{} = raw
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			return ""
		}
		mapValue, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		next, ok := mapValue[key]
		if !ok {
			return ""
		}
		current = next
	}
	switch value := current.(type) {
	case string:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func readInt64(raw map[string]interface{}, keys ...string) (int64, bool) {
	if len(keys) == 0 {
		return 0, false
	}
	var current interface{} = raw
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			return 0, false
		}
		mapValue, ok := current.(map[string]interface{})
		if !ok {
			return 0, false
		}
		next, ok := mapValue[key]
		if !ok {
			return 0, false
		}
		current = next
	}
	switch value := current.(type) {
	case float64:
		return int64(value), true
	case int64:
		return value, true
	case json.Number:
		parsed, err := value.Int64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func pointerString(val *string) string {
	if val == nil {
		return ""
	}
	return strings.TrimSpace(*val)
}

func parseTransactionTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &parsed
}

func pickFirstNonEmpty(values ...string) string {
	for _, val := range values {
		trimmed := strings.TrimSpace(val)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func buildDescription(description string, orderNo string) string {
	description = strings.TrimSpace(description)
	if description != "" {
		return description
	}
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return "??????"
	}
	return "?? " + orderNo
}

func validatePrivateKey(raw string) error {
	if _, err := parsePrivateKey(raw); err != nil {
		return err
	}
	return nil
}

func parsePrivateKey(raw string) (*rsa.PrivateKey, error) {
	normalized := normalizePrivateKey(raw)
	if normalized == "" {
		return nil, fmt.Errorf("%w: merchant_private_key is empty", ErrConfigInvalid)
	}
	block, _ := pem.Decode([]byte(normalized))
	if block == nil {
		return nil, fmt.Errorf("%w: merchant_private_key pem decode failed", ErrConfigInvalid)
	}
	parsedPKCS8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		privateKey, ok := parsedPKCS8.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("%w: merchant_private_key type is not rsa", ErrConfigInvalid)
		}
		return privateKey, nil
	}
	privateKey, parseErr := x509.ParsePKCS1PrivateKey(block.Bytes)
	if parseErr == nil {
		return privateKey, nil
	}
	return nil, fmt.Errorf("%w: parse merchant_private_key failed", ErrConfigInvalid)
}

func normalizePrivateKey(raw string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(raw, "\\n", "\n"))
	if normalized == "" {
		return ""
	}
	if !strings.Contains(normalized, "BEGIN") {
		return "-----BEGIN PRIVATE KEY-----\n" + normalized + "\n-----END PRIVATE KEY-----"
	}
	return normalized
}

func (c *Config) Normalize() {
	c.AppID = strings.TrimSpace(c.AppID)
	c.MerchantID = strings.TrimSpace(c.MerchantID)
	c.MerchantSerialNo = strings.TrimSpace(c.MerchantSerialNo)
	c.MerchantPrivateKey = strings.TrimSpace(c.MerchantPrivateKey)
	c.APIV3Key = strings.TrimSpace(c.APIV3Key)
	c.NotifyURL = strings.TrimSpace(c.NotifyURL)
	c.H5RedirectURL = strings.TrimSpace(c.H5RedirectURL)
	c.H5Type = strings.ToUpper(strings.TrimSpace(c.H5Type))
	c.H5WapURL = strings.TrimSpace(c.H5WapURL)
	c.H5WapName = strings.TrimSpace(c.H5WapName)
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if c.H5Type == "" {
		c.H5Type = wechatH5TypeWAP
	}
	if c.BaseURL == "" {
		c.BaseURL = defaultBaseURL
	}
	c.ExchangeRateConfig.NormalizeExchangeRate()
}

func withDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, defaultTimeout)
}
