package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/dujiao-next/internal/adgateway"
	"github.com/dujiao-next/internal/logger"
)

// AdProxyService ??????? ad-system
type AdProxyService struct {
	client  *http.Client
	baseURL string
}

func NewAdProxyService() *AdProxyService {
	return &AdProxyService{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL: adgateway.ServerURL,
	}
}

// AdRenderSlotDTO ?????
type AdRenderSlotDTO struct {
	Code       string `json:"code"`
	Scene      string `json:"scene"`
	Layout     string `json:"layout"`
	RenderMode string `json:"render_mode"`
	MaxItems   int    `json:"max_items"`
}

// AdRenderItemDTO ?????
type AdRenderItemDTO struct {
	ID              int64  `json:"id"`
	AdvertiserName  string `json:"advertiser_name"`
	Title           string `json:"title"`
	Subtitle        string `json:"subtitle"`
	CTALabel        string `json:"cta_label"`
	Badge           string `json:"badge"`
	Image           string `json:"image"`
	MobileImage     string `json:"mobile_image"`
	Icon            string `json:"icon"`
	LinkType        string `json:"link_type"`
	OpenInNewTab    bool   `json:"open_in_new_tab"`
	Theme           string `json:"theme"`
	Dismissible     bool   `json:"dismissible"`
	ClickURL        string `json:"click_url"`
	ImpressionToken string `json:"impression_token"`
}

// AdRenderResponse ??????
type AdRenderResponse struct {
	Slot  AdRenderSlotDTO   `json:"slot"`
	Items []AdRenderItemDTO `json:"items"`
}

// RenderSlot ?? ad-system ???????
func (s *AdProxyService) RenderSlot(ctx context.Context, slotCode string, params map[string]string) (*AdRenderResponse, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/public/ad-slots/%s/render", s.baseURL, url.PathEscape(slotCode)))
	if err != nil {
		return nil, fmt.Errorf("ad_proxy: invalid url: %w", err)
	}

	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("ad_proxy: create request failed: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		logger.Warnw("ad_proxy_render_slot_failed", "slot_code", slotCode, "error", err)
		return nil, fmt.Errorf("ad_proxy: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ad_proxy: read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Warnw("ad_proxy_render_slot_non_ok", "slot_code", slotCode, "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("ad_proxy: upstream returned %d", resp.StatusCode)
	}

	var apiResp struct {
		StatusCode int               `json:"status_code"`
		Msg        string            `json:"msg"`
		Data       *AdRenderResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("ad_proxy: decode response failed: %w", err)
	}
	if apiResp.StatusCode != 0 || apiResp.Data == nil {
		return nil, fmt.Errorf("ad_proxy: upstream error: %s", apiResp.Msg)
	}

	return apiResp.Data, nil
}

// ReportImpression ??????
func (s *AdProxyService) ReportImpression(ctx context.Context, payload json.RawMessage) error {
	u := fmt.Sprintf("%s/api/v1/public/ad-events/impression", s.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("ad_proxy: create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		logger.Warnw("ad_proxy_report_impression_failed", "error", err)
		return fmt.Errorf("ad_proxy: request failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ad_proxy: upstream returned %d", resp.StatusCode)
	}
	return nil
}
