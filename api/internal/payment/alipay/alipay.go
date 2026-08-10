package alipay

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
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

var (
	ErrConfigInvalid    = errors.New("alipay config invalid")
	ErrSignGenerate     = errors.New("alipay sign generate failed")
	ErrRequestFailed    = errors.New("alipay request failed")
	ErrResponseInvalid  = errors.New("alipay response invalid")
	ErrSignatureInvalid = errors.New("alipay signature invalid")
)

const (
	alipaySignTypeRSA2 = "RSA2"
	alipaySignTypeRSA  = "RSA"

	alipayReqFormatJSON = "JSON"
	alipayReqCharset    = "utf-8"
	alipayReqVersion    = "1.0"

	alipayRespCodeSuccess = "10000"

	alipayMethodPrecreate = "alipay.trade.precreate"
	alipayMethodWAPPay    = "alipay.trade.wap.pay"
	alipayMethodPagePay   = "alipay.trade.page.pay"

	alipayProductCodeFaceToFace = "FACE_TO_FACE_PAYMENT"
	alipayProductCodeQuickWAP   = "QUICK_WAP_WAY"
	alipayProductCodeFastPay    = "FAST_INSTANT_TRADE_PAY"

	alipayGatewayDefault = "https://openapi.alipay.com/gateway.do"
)

// Config ????????
type Config struct {
	common.ExchangeRateConfig
	AppID            string `json:"app_id"`
	PrivateKey       string `json:"private_key"`
	AlipayPublicKey  string `json:"alipay_public_key"`
	GatewayURL       string `json:"gateway_url"`
	NotifyURL        string `json:"notify_url"`
	ReturnURL        string `json:"return_url"`
	SignType         string `json:"sign_type"`
	AppCertSN        string `json:"app_cert_sn"`
	AlipayRootCertSN string `json:"alipay_root_cert_sn"`
}

// CreateInput ????????
type CreateInput struct {
	OrderNo        string
	Amount         string
	Subject        string
	NotifyURL      string
	ReturnURL      string
	TimeoutExpress string
	QuitURL        string
}

// CreateResult ????????
type CreateResult struct {
	PayURL     string
	QRCode     string
	TradeNo    string
	OutTradeNo string
	Method     string
	Raw        map[string]interface{}
}

// ParseConfig ?????
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	return common.ParseConfig[Config](raw, ErrConfigInvalid)
}

// ValidateConfig ????????
func ValidateConfig(cfg *Config, interactionMode string) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.AppID) == "" {
		return fmt.Errorf("%w: app_id is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.PrivateKey) == "" {
		return fmt.Errorf("%w: private_key is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.AlipayPublicKey) == "" {
		return fmt.Errorf("%w: alipay_public_key is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.GatewayURL) == "" {
		return fmt.Errorf("%w: gateway_url is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.NotifyURL) == "" {
		return fmt.Errorf("%w: notify_url is required", ErrConfigInvalid)
	}
	if _, err := url.ParseRequestURI(strings.TrimSpace(cfg.GatewayURL)); err != nil {
		return fmt.Errorf("%w: gateway_url is invalid", ErrConfigInvalid)
	}
	if _, err := url.ParseRequestURI(strings.TrimSpace(cfg.NotifyURL)); err != nil {
		return fmt.Errorf("%w: notify_url is invalid", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.ReturnURL) != "" {
		if _, err := url.ParseRequestURI(strings.TrimSpace(cfg.ReturnURL)); err != nil {
			return fmt.Errorf("%w: return_url is invalid", ErrConfigInvalid)
		}
	}
	if !IsSupportedInteractionMode(interactionMode) {
		return fmt.Errorf("%w: interaction_mode %s is not supported", ErrConfigInvalid, interactionMode)
	}
	if requiresReturnURL(interactionMode) && strings.TrimSpace(cfg.ReturnURL) == "" {
		return fmt.Errorf("%w: return_url is required for mode %s", ErrConfigInvalid, interactionMode)
	}
	if cfg.SignType != alipaySignTypeRSA2 && cfg.SignType != alipaySignTypeRSA {
		return fmt.Errorf("%w: sign_type is invalid", ErrConfigInvalid)
	}
	return nil
}

// CreatePayment ????????
func CreatePayment(ctx context.Context, cfg *Config, input CreateInput, interactionMode string) (*CreateResult, error) {
	if err := ValidateConfig(cfg, interactionMode); err != nil {
		return nil, err
	}
	input.OrderNo = strings.TrimSpace(input.OrderNo)
	input.Amount = strings.TrimSpace(input.Amount)
	if input.OrderNo == "" || input.Amount == "" {
		return nil, fmt.Errorf("%w: order_no/amount is required", ErrConfigInvalid)
	}
	amount, err := decimal.NewFromString(input.Amount)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("%w: amount is invalid", ErrConfigInvalid)
	}
	input.Amount = amount.Round(2).StringFixed(2)
	if strings.TrimSpace(input.Subject) == "" {
		input.Subject = input.OrderNo
	}

	mode := strings.ToLower(strings.TrimSpace(interactionMode))
	method, err := resolveMethod(mode)
	if err != nil {
		return nil, err
	}
	bizContent, err := buildBizContent(mode, input)
	if err != nil {
		return nil, err
	}
	bizContentBytes, err := json.Marshal(bizContent)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal biz_content failed", ErrConfigInvalid)
	}

	notifyURL := strings.TrimSpace(input.NotifyURL)
	if notifyURL == "" {
		notifyURL = cfg.NotifyURL
	}
	returnURL := strings.TrimSpace(input.ReturnURL)
	if returnURL == "" {
		returnURL = cfg.ReturnURL
	}

	params := map[string]string{
		"app_id":      cfg.AppID,
		"method":      method,
		"format":      alipayReqFormatJSON,
		"charset":     alipayReqCharset,
		"sign_type":   cfg.SignType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     alipayReqVersion,
		"notify_url":  notifyURL,
		"biz_content": string(bizContentBytes),
	}
	if returnURL != "" {
		params["return_url"] = returnURL
	}
	if strings.TrimSpace(cfg.AppCertSN) != "" {
		params["app_cert_sn"] = strings.TrimSpace(cfg.AppCertSN)
	}
	if strings.TrimSpace(cfg.AlipayRootCertSN) != "" {
		params["alipay_root_cert_sn"] = strings.TrimSpace(cfg.AlipayRootCertSN)
	}

	sign, err := signContent(buildSignContent(params), cfg.PrivateKey, cfg.SignType)
	if err != nil {
		return nil, err
	}
	params["sign"] = sign

	if mode == constants.PaymentInteractionQR {
		return requestPrecreate(ctx, cfg, method, params, input.OrderNo)
	}

	payURL := buildGatewayPayURL(cfg.GatewayURL, params)
	return &CreateResult{
		PayURL:     payURL,
		QRCode:     "",
		TradeNo:    "",
		OutTradeNo: input.OrderNo,
		Method:     method,
		Raw: map[string]interface{}{
			"pay_url":      payURL,
			"method":       method,
			"out_trade_no": input.OrderNo,
		},
	}, nil
}

// VerifyCallback ????????????
func VerifyCallback(cfg *Config, form map[string][]string) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrConfigInvalid)
	}
	if len(form) == 0 {
		return fmt.Errorf("%w: callback form is empty", ErrSignatureInvalid)
	}
	sign := strings.TrimSpace(firstFormValue(form, "sign"))
	if sign == "" {
		return fmt.Errorf("%w: sign is required", ErrSignatureInvalid)
	}
	signType := strings.ToUpper(strings.TrimSpace(firstFormValue(form, "sign_type")))
	if signType == "" {
		signType = strings.ToUpper(strings.TrimSpace(cfg.SignType))
	}
	if signType != alipaySignTypeRSA2 && signType != alipaySignTypeRSA {
		return fmt.Errorf("%w: sign_type is invalid", ErrSignatureInvalid)
	}
	content := buildSignContentFromForm(form)
	if content == "" {
		return fmt.Errorf("%w: sign content is empty", ErrSignatureInvalid)
	}
	publicKey, err := parsePublicKey(cfg.AlipayPublicKey)
	if err != nil {
		return err
	}
	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return fmt.Errorf("%w: decode sign failed", ErrSignatureInvalid)
	}
	var digest []byte
	var hashType crypto.Hash
	if signType == alipaySignTypeRSA {
		sum := sha1.Sum([]byte(content))
		digest = sum[:]
		hashType = crypto.SHA1
	} else {
		sum := sha256.Sum256([]byte(content))
		digest = sum[:]
		hashType = crypto.SHA256
	}
	if err := rsa.VerifyPKCS1v15(publicKey, hashType, digest, signBytes); err != nil {
		return fmt.Errorf("%w: verify failed", ErrSignatureInvalid)
	}
	return nil
}

// VerifyCallbackOwnership ?????????,??????????
func VerifyCallbackOwnership(cfg *Config, form map[string][]string) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrConfigInvalid)
	}
	if len(form) == 0 {
		return fmt.Errorf("%w: callback form is empty", ErrSignatureInvalid)
	}
	callbackAppID := strings.TrimSpace(firstFormValue(form, "app_id"))
	if callbackAppID == "" {
		callbackAppID = strings.TrimSpace(firstFormValue(form, "appid"))
	}
	if callbackAppID == "" {
		return fmt.Errorf("%w: app_id is required", ErrSignatureInvalid)
	}
	if !strings.EqualFold(callbackAppID, strings.TrimSpace(cfg.AppID)) {
		return fmt.Errorf("%w: app_id mismatch", ErrSignatureInvalid)
	}
	return nil
}

func requestPrecreate(ctx context.Context, cfg *Config, method string, params map[string]string, fallbackOrderNo string) (*CreateResult, error) {
	responseBody, err := postGateway(ctx, cfg.GatewayURL, params)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(responseBody, &raw); err != nil {
		return nil, fmt.Errorf("%w: decode response failed", ErrResponseInvalid)
	}
	responseKey := strings.ReplaceAll(method, ".", "_") + "_response"
	responseNode, ok := raw[responseKey].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: %s not found", ErrResponseInvalid, responseKey)
	}

	code := strings.TrimSpace(readString(responseNode, "code"))
	if code != alipayRespCodeSuccess {
		errMsg := strings.TrimSpace(readString(responseNode, "sub_msg"))
		if errMsg == "" {
			errMsg = strings.TrimSpace(readString(responseNode, "msg"))
		}
		if errMsg == "" {
			errMsg = "code=" + code
		}
		return nil, fmt.Errorf("%w: %s", ErrResponseInvalid, errMsg)
	}

	result := &CreateResult{
		PayURL:     "",
		QRCode:     strings.TrimSpace(readString(responseNode, "qr_code")),
		TradeNo:    strings.TrimSpace(readString(responseNode, "trade_no")),
		OutTradeNo: strings.TrimSpace(readString(responseNode, "out_trade_no")),
		Method:     method,
		Raw:        raw,
	}
	if result.OutTradeNo == "" {
		result.OutTradeNo = strings.TrimSpace(fallbackOrderNo)
	}
	if result.QRCode == "" {
		return nil, fmt.Errorf("%w: qr_code is empty", ErrResponseInvalid)
	}
	return result, nil
}

func resolveMethod(mode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case constants.PaymentInteractionQR:
		return alipayMethodPrecreate, nil
	case constants.PaymentInteractionWAP:
		return alipayMethodWAPPay, nil
	case constants.PaymentInteractionPage:
		return alipayMethodPagePay, nil
	default:
		return "", fmt.Errorf("%w: interaction_mode %s is not supported", ErrConfigInvalid, mode)
	}
}

func buildBizContent(mode string, input CreateInput) (map[string]interface{}, error) {
	subject := strings.TrimSpace(input.Subject)
	if subject == "" {
		subject = strings.TrimSpace(input.OrderNo)
	}
	if subject == "" {
		return nil, fmt.Errorf("%w: subject is required", ErrConfigInvalid)
	}
	bizContent := map[string]interface{}{
		"out_trade_no": strings.TrimSpace(input.OrderNo),
		"total_amount": strings.TrimSpace(input.Amount),
		"subject":      subject,
	}
	if strings.TrimSpace(input.TimeoutExpress) != "" {
		bizContent["timeout_express"] = strings.TrimSpace(input.TimeoutExpress)
	}
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case constants.PaymentInteractionQR:
		bizContent["product_code"] = alipayProductCodeFaceToFace
	case constants.PaymentInteractionWAP:
		bizContent["product_code"] = alipayProductCodeQuickWAP
		if strings.TrimSpace(input.QuitURL) != "" {
			bizContent["quit_url"] = strings.TrimSpace(input.QuitURL)
		}
	case constants.PaymentInteractionPage:
		bizContent["product_code"] = alipayProductCodeFastPay
	default:
		return nil, fmt.Errorf("%w: interaction_mode %s is not supported", ErrConfigInvalid, mode)
	}
	return bizContent, nil
}

func buildSignContent(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		key = strings.TrimSpace(key)
		if key == "" || key == "sign" {
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func signContent(content, privateKeyRaw, signType string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("%w: empty sign content", ErrSignGenerate)
	}
	privateKey, err := parsePrivateKey(privateKeyRaw)
	if err != nil {
		return "", err
	}
	var hashType crypto.Hash
	var digest []byte
	signType = strings.ToUpper(strings.TrimSpace(signType))
	if signType == alipaySignTypeRSA {
		sum := sha1.Sum([]byte(content))
		hashType = crypto.SHA1
		digest = sum[:]
	} else {
		sum := sha256.Sum256([]byte(content))
		hashType = crypto.SHA256
		digest = sum[:]
	}
	signBytes, err := rsa.SignPKCS1v15(rand.Reader, privateKey, hashType, digest)
	if err != nil {
		return "", fmt.Errorf("%w: sign failed", ErrSignGenerate)
	}
	return base64.StdEncoding.EncodeToString(signBytes), nil
}

func parsePrivateKey(raw string) (*rsa.PrivateKey, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(raw, "\\n", "\n"))
	if normalized == "" {
		return nil, fmt.Errorf("%w: private key is empty", ErrSignGenerate)
	}
	if !strings.Contains(normalized, "BEGIN") {
		normalized = "-----BEGIN PRIVATE KEY-----\n" + normalized + "\n-----END PRIVATE KEY-----"
	}
	block, _ := pem.Decode([]byte(normalized))
	if block == nil {
		return nil, fmt.Errorf("%w: private key pem decode failed", ErrSignGenerate)
	}
	parsedPKCS8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		if privateKey, ok := parsedPKCS8.(*rsa.PrivateKey); ok {
			return privateKey, nil
		}
		return nil, fmt.Errorf("%w: private key type is not rsa", ErrSignGenerate)
	}
	privateKey, parseErr := x509.ParsePKCS1PrivateKey(block.Bytes)
	if parseErr == nil {
		return privateKey, nil
	}
	return nil, fmt.Errorf("%w: parse private key failed", ErrSignGenerate)
}

func postGateway(ctx context.Context, gatewayURL string, params map[string]string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := common.WithDefaultTimeout(ctx)
	defer cancel()

	form := url.Values{}
	for key, value := range params {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		form.Set(key, value)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(gatewayURL), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: build request failed", ErrRequestFailed)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: http request failed", ErrRequestFailed)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read response failed", ErrRequestFailed)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", ErrResponseInvalid, resp.StatusCode)
	}
	return body, nil
}

func buildGatewayPayURL(gatewayURL string, params map[string]string) string {
	form := url.Values{}
	for key, value := range params {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		form.Set(key, value)
	}
	baseURL := strings.TrimSpace(gatewayURL)
	if baseURL == "" {
		return ""
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		if strings.Contains(baseURL, "?") {
			return baseURL + "&" + form.Encode()
		}
		return baseURL + "?" + form.Encode()
	}
	parsed.RawQuery = form.Encode()
	return parsed.String()
}

func buildSignContentFromForm(form map[string][]string) string {
	params := make(map[string]string, len(form))
	for key, values := range form {
		if len(values) == 0 {
			continue
		}
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		if strings.EqualFold(normalizedKey, "sign") || strings.EqualFold(normalizedKey, "sign_type") {
			continue
		}
		value := values[0]
		if value == "" {
			continue
		}
		params[normalizedKey] = value
	}
	return buildSignContent(params)
}

func firstFormValue(form map[string][]string, key string) string {
	if values, ok := form[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

func parsePublicKey(raw string) (*rsa.PublicKey, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(raw, "\\n", "\n"))
	if normalized == "" {
		return nil, fmt.Errorf("%w: public key is empty", ErrSignatureInvalid)
	}
	if !strings.Contains(normalized, "BEGIN") {
		normalized = "-----BEGIN PUBLIC KEY-----\n" + normalized + "\n-----END PUBLIC KEY-----"
	}
	block, _ := pem.Decode([]byte(normalized))
	if block == nil {
		return nil, fmt.Errorf("%w: public key pem decode failed", ErrSignatureInvalid)
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		if publicKey, ok := parsed.(*rsa.PublicKey); ok {
			return publicKey, nil
		}
		return nil, fmt.Errorf("%w: public key type is not rsa", ErrSignatureInvalid)
	}
	publicKey, parseErr := x509.ParsePKCS1PublicKey(block.Bytes)
	if parseErr == nil {
		return publicKey, nil
	}
	return nil, fmt.Errorf("%w: parse public key failed", ErrSignatureInvalid)
}

func readString(raw map[string]interface{}, key string) string {
	if raw == nil {
		return ""
	}
	value, ok := raw[key]
	if !ok || value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}

func IsSupportedInteractionMode(mode string) bool {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case constants.PaymentInteractionQR, constants.PaymentInteractionWAP, constants.PaymentInteractionPage:
		return true
	default:
		return false
	}
}

func requiresReturnURL(mode string) bool {
	mode = strings.ToLower(strings.TrimSpace(mode))
	return mode == constants.PaymentInteractionWAP || mode == constants.PaymentInteractionPage
}

func (c *Config) Normalize() {
	c.AppID = strings.TrimSpace(c.AppID)
	c.PrivateKey = strings.TrimSpace(c.PrivateKey)
	c.AlipayPublicKey = strings.TrimSpace(c.AlipayPublicKey)
	c.GatewayURL = strings.TrimSpace(c.GatewayURL)
	c.NotifyURL = strings.TrimSpace(c.NotifyURL)
	c.ReturnURL = strings.TrimSpace(c.ReturnURL)
	c.SignType = strings.ToUpper(strings.TrimSpace(c.SignType))
	c.AppCertSN = strings.TrimSpace(c.AppCertSN)
	c.AlipayRootCertSN = strings.TrimSpace(c.AlipayRootCertSN)
	if c.SignType == "" {
		c.SignType = alipaySignTypeRSA2
	}
	if c.GatewayURL == "" {
		c.GatewayURL = alipayGatewayDefault
	}
	c.ExchangeRateConfig.NormalizeExchangeRate()
}
