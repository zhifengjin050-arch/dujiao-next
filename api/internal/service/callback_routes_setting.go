package service

import (
	"strings"
	"sync"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
)

// CallbackRoutesSetting ??????
type CallbackRoutesSetting struct {
	PaymentCallback  string
	PaypalWebhook    string
	StripeWebhook    string
	UpstreamCallback string
}

// HasCustomRoutes ????????????????
func (s *CallbackRoutesSetting) HasCustomRoutes() bool {
	return s.PaymentCallback != "" || s.PaypalWebhook != "" ||
		s.StripeWebhook != "" || s.UpstreamCallback != ""
}

// callbackRoutesSettingFromJSON ? JSON map ????????
func callbackRoutesSettingFromJSON(value models.JSON) CallbackRoutesSetting {
	return CallbackRoutesSetting{
		PaymentCallback:  normalizeCallbackRoutePath(readString(value, constants.SettingFieldPaymentCallback, "")),
		PaypalWebhook:    normalizeCallbackRoutePath(readString(value, constants.SettingFieldPaypalWebhook, "")),
		StripeWebhook:    normalizeCallbackRoutePath(readString(value, constants.SettingFieldStripeWebhook, "")),
		UpstreamCallback: normalizeCallbackRoutePath(readString(value, constants.SettingFieldUpstreamCallback, "")),
	}
}

// CallbackRoutesSettingToMap ??????????? JSON map
func CallbackRoutesSettingToMap(s CallbackRoutesSetting) models.JSON {
	return models.JSON{
		constants.SettingFieldPaymentCallback:  s.PaymentCallback,
		constants.SettingFieldPaypalWebhook:    s.PaypalWebhook,
		constants.SettingFieldStripeWebhook:    s.StripeWebhook,
		constants.SettingFieldUpstreamCallback: s.UpstreamCallback,
	}
}

// normalizeCallbackRoutesSetting ?????????
func normalizeCallbackRoutesSetting(value map[string]interface{}) models.JSON {
	setting := callbackRoutesSettingFromJSON(models.JSON(value))
	deduplicateCallbackRoutes(&setting)
	return CallbackRoutesSettingToMap(setting)
}

// reservedRoutePrefixes ??????,?????????????
var reservedRoutePrefixes = []string{
	"/api/v1/public/",
	"/api/v1/admin/",
	"/api/v1/auth/",
	"/api/v1/guest/",
	"/api/v1/channel/",
	"/api/v1/upstream/api/",
	"/api/v1/user/",
}

// normalizeCallbackRoutePath ????????????
// ???????????;??????? /api/ ??,?? query string,
// ?????????????
func normalizeCallbackRoutePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	// ?? query string ? fragment
	if idx := strings.IndexAny(path, "?#"); idx != -1 {
		path = path[:idx]
	}

	// ??????
	path = strings.TrimRight(path, "/")

	// ??? /api/ ??
	if !strings.HasPrefix(path, "/api/") {
		return ""
	}

	// ???????????
	pathWithSlash := path + "/"
	for _, prefix := range reservedRoutePrefixes {
		if strings.HasPrefix(pathWithSlash, prefix) || strings.HasPrefix(prefix, pathWithSlash) {
			return ""
		}
	}

	return path
}

// deduplicateCallbackRoutes ??????:???????????
func deduplicateCallbackRoutes(s *CallbackRoutesSetting) {
	seen := make(map[string]bool, 4)
	fields := []*string{
		&s.PaymentCallback,
		&s.PaypalWebhook,
		&s.StripeWebhook,
		&s.UpstreamCallback,
	}
	for _, f := range fields {
		if *f == "" {
			continue
		}
		if seen[*f] {
			*f = "" // ??????
		} else {
			seen[*f] = true
		}
	}
}

// --- ???????? ---

var callbackRoutesCache struct {
	mu      sync.RWMutex
	routes  *CallbackRoutesSetting
	loaded  bool
	expires time.Time
}

const callbackRoutesCacheTTL = 5 * time.Minute

// InvalidateCallbackRoutesCache ??????????(??????????)
func (s *SettingService) InvalidateCallbackRoutesCache() {
	callbackRoutesCache.mu.Lock()
	callbackRoutesCache.loaded = false
	callbackRoutesCache.routes = nil
	callbackRoutesCache.expires = time.Time{}
	callbackRoutesCache.mu.Unlock()
}

// GetCallbackRoutesCached ????????????????,??????? DB ???
func (s *SettingService) GetCallbackRoutesCached() *CallbackRoutesSetting {
	callbackRoutesCache.mu.RLock()
	if callbackRoutesCache.loaded && time.Now().Before(callbackRoutesCache.expires) {
		routes := callbackRoutesCache.routes
		callbackRoutesCache.mu.RUnlock()
		return routes
	}
	callbackRoutesCache.mu.RUnlock()

	callbackRoutesCache.mu.Lock()
	defer callbackRoutesCache.mu.Unlock()

	// ????
	if callbackRoutesCache.loaded && time.Now().Before(callbackRoutesCache.expires) {
		return callbackRoutesCache.routes
	}

	routes := s.GetCallbackRoutes()
	callbackRoutesCache.routes = routes
	callbackRoutesCache.loaded = true
	callbackRoutesCache.expires = time.Now().Add(callbackRoutesCacheTTL)
	return routes
}
