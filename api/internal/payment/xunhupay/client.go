package xunhupay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client ???????
type Client struct {
	appID      string
	appSecret  string
	gatewayURL string
	queryURL   string
	httpClient *http.Client
}

// NewClient ?????
func NewClient(appID, appSecret, gatewayURL, queryURL string) *Client {
	return &Client{
		appID:      strings.TrimSpace(appID),
		appSecret:  strings.TrimSpace(appSecret),
		gatewayURL: strings.TrimSpace(gatewayURL),
		queryURL:   strings.TrimSpace(queryURL),
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Post ????
func (c *Client) Post(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	if c == nil {
		return nil, ErrConfigInvalid
	}
	if strings.TrimSpace(endpoint) == "" {
		return nil, ErrConfigInvalid
	}
	form := make(map[string]string, len(params)+4)
	for k, v := range params {
		form[k] = strings.TrimSpace(v)
	}
	form["appid"] = c.appID
	form["time"] = fmt.Sprintf("%d", time.Now().Unix())
	form["nonce_str"] = randomNonce(16)
	content := buildSignContent(form)
	form["hash"] = signMD5(content, c.appSecret)

	values := url.Values{}
	for k, v := range form {
		if v == "" {
			continue
		}
		values.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("xunhupay request failed: status=%d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) Gateway() string { return c.gatewayURL }
func (c *Client) QueryGateway() string { return c.queryURL }

func randomNonce(n int) string {
	if n <= 0 {
		n = 16
	}
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func decodeJSON(body []byte, v any) error {
	return json.Unmarshal(body, v)
}
