package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/repository"

	"github.com/redis/go-redis/v9"
)

var (
	ErrRiskIPBlacklisted        = errors.New("risk: ip blacklisted")
	ErrRiskEmailBlacklisted     = errors.New("risk: email blacklisted")
	ErrRiskTooManyPendingOrders = errors.New("risk: too many pending orders")
	ErrRiskOrderRateLimited     = errors.New("risk: order rate limited")
)

// RiskRateLimitedError ?? Retry-After ?????????
type RiskRateLimitedError struct {
	RetryAfter int64
}

func (e *RiskRateLimitedError) Error() string {
	return ErrRiskOrderRateLimited.Error()
}

func (e *RiskRateLimitedError) Is(target error) bool {
	return target == ErrRiskOrderRateLimited
}

// GetRetryAfter ?????? RetryAfter ??,?????? 0
func GetRetryAfter(err error) int64 {
	var rle *RiskRateLimitedError
	if errors.As(err, &rle) {
		return rle.RetryAfter
	}
	return 0
}

// RiskCheckInput ??????
type RiskCheckInput struct {
	UserID      uint
	GuestEmail  string
	ClientIP    string
	IsGuest     bool
	SkipIPCheck bool // ?? IP ????(??/Bot ??,? ClientIP ???? IP ???)
}

// parsedIPBlacklist ?????? IP ???
type parsedIPBlacklist struct {
	exactIPs map[string]struct{}
	cidrs    []*net.IPNet
	hash     string
}

// OrderRiskControlService ??????
type OrderRiskControlService struct {
	settingService *SettingService
	orderRepo      repository.OrderRepository

	mu              sync.RWMutex
	cachedBlacklist *parsedIPBlacklist
}

// NewOrderRiskControlService ??????
func NewOrderRiskControlService(settingService *SettingService, orderRepo repository.OrderRepository) *OrderRiskControlService {
	return &OrderRiskControlService{
		settingService: settingService,
		orderRepo:      orderRepo,
	}
}

// CheckOrderAllowed ????????
func (s *OrderRiskControlService) CheckOrderAllowed(input RiskCheckInput) error {
	if s == nil || s.settingService == nil {
		return nil
	}

	cfg, err := s.settingService.GetOrderRiskControlConfig()
	if err != nil {
		logger.Warnw("risk_control_get_config_error", "error", err)
		return nil // ?????????,???????
	}

	if !cfg.Enabled {
		return nil
	}

	// 1. IP ?????(?? IP ??????)
	if !input.SkipIPCheck && input.ClientIP != "" && len(cfg.IPBlacklist) > 0 {
		if s.isIPInBlacklist(input.ClientIP, cfg.IPBlacklist) {
			return ErrRiskIPBlacklisted
		}
	}

	// 2. ???????(????)
	if input.IsGuest && input.GuestEmail != "" && len(cfg.EmailBlacklist) > 0 {
		normalizedEmail := strings.ToLower(strings.TrimSpace(input.GuestEmail))
		for _, blocked := range cfg.EmailBlacklist {
			if normalizedEmail == blocked {
				return ErrRiskEmailBlacklisted
			}
		}
	}

	// 3. ??????????
	if err := s.checkPendingOrderLimits(input, cfg); err != nil {
		return err
	}

	// 4. ??????
	if cfg.OrderRateLimit.Enabled {
		if err := s.checkOrderRateLimit(input, cfg.OrderRateLimit); err != nil {
			return err
		}
	}

	return nil
}

// checkPendingOrderLimits ???????????
func (s *OrderRiskControlService) checkPendingOrderLimits(input RiskCheckInput, cfg OrderRiskControlConfig) error {
	// ????
	if input.UserID > 0 && cfg.MaxPendingOrdersPerUser > 0 {
		count, err := s.orderRepo.CountPendingByUserID(input.UserID)
		if err != nil {
			logger.Warnw("risk_control_count_pending_by_user_error", "user_id", input.UserID, "error", err)
		} else if count >= int64(cfg.MaxPendingOrdersPerUser) {
			return ErrRiskTooManyPendingOrders
		}
	}

	// IP ??(?? IP ??????)
	if !input.SkipIPCheck && input.ClientIP != "" && cfg.MaxPendingOrdersPerIP > 0 {
		count, err := s.orderRepo.CountPendingByClientIP(input.ClientIP)
		if err != nil {
			logger.Warnw("risk_control_count_pending_by_ip_error", "ip", input.ClientIP, "error", err)
		} else if count >= int64(cfg.MaxPendingOrdersPerIP) {
			return ErrRiskTooManyPendingOrders
		}
	}

	// ??????
	if input.IsGuest && input.GuestEmail != "" && cfg.MaxPendingOrdersPerGuestEmail > 0 {
		count, err := s.orderRepo.CountPendingByGuestEmail(input.GuestEmail)
		if err != nil {
			logger.Warnw("risk_control_count_pending_by_email_error", "email", input.GuestEmail, "error", err)
		} else if count >= int64(cfg.MaxPendingOrdersPerGuestEmail) {
			return ErrRiskTooManyPendingOrders
		}
	}

	return nil
}

// orderRateLimitScript Redis Lua ??:??????,?? {current, ttl}
var orderRateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[1])
end
if tonumber(ARGV[2]) > 0 and tonumber(ARGV[3]) > 0 and current == tonumber(ARGV[2]) + 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[3])
end
local ttl = redis.call("TTL", KEYS[1])
return {current, ttl}
`)

// checkOrderRateLimit ??????
func (s *OrderRiskControlService) checkOrderRateLimit(input RiskCheckInput, rl OrderRateLimitConfig) error {
	client := cache.Client()
	if client == nil {
		return nil // Redis ??????
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// IP ??????(?? IP ??????)
	if !input.SkipIPCheck && input.ClientIP != "" {
		if err := s.checkSingleRateLimit(ctx, client,
			fmt.Sprintf("dj:risk:order_rate:ip:%s", input.ClientIP), rl); err != nil {
			return err
		}
	}

	// ????????(????????)
	if input.UserID > 0 {
		if err := s.checkSingleRateLimit(ctx, client,
			fmt.Sprintf("dj:risk:order_rate:user:%d", input.UserID), rl); err != nil {
			return err
		}
	}

	return nil
}

// checkSingleRateLimit ?????????????
func (s *OrderRiskControlService) checkSingleRateLimit(ctx context.Context, client *redis.Client, key string, rl OrderRateLimitConfig) error {
	result, err := orderRateLimitScript.Run(ctx, client, []string{key},
		rl.WindowSeconds, rl.MaxRequests, rl.BlockSeconds,
	).Result()
	if err != nil {
		logger.Warnw("risk_control_rate_limit_script_error", "key", key, "error", err)
		return nil // ?????????
	}

	values, ok := result.([]interface{})
	if !ok || len(values) < 2 {
		return nil
	}

	current, _ := values[0].(int64)
	ttl, _ := values[1].(int64)

	if current > int64(rl.MaxRequests) {
		if ttl < 0 {
			ttl = 0
		}
		return &RiskRateLimitedError{RetryAfter: ttl}
	}
	return nil
}

// getOrBuildBlacklist ???????? IP ???
func (s *OrderRiskControlService) getOrBuildBlacklist(blacklist []string) *parsedIPBlacklist {
	hash := hashBlacklist(blacklist)

	s.mu.RLock()
	if s.cachedBlacklist != nil && s.cachedBlacklist.hash == hash {
		cached := s.cachedBlacklist
		s.mu.RUnlock()
		return cached
	}
	s.mu.RUnlock()

	// ??
	parsed := &parsedIPBlacklist{
		exactIPs: make(map[string]struct{}, len(blacklist)),
		hash:     hash,
	}
	for _, entry := range blacklist {
		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err == nil {
				parsed.cidrs = append(parsed.cidrs, cidr)
			}
		} else {
			parsed.exactIPs[entry] = struct{}{}
		}
	}

	s.mu.Lock()
	if s.cachedBlacklist == nil || s.cachedBlacklist.hash != hash {
		s.cachedBlacklist = parsed
	}
	s.mu.Unlock()

	return parsed
}

// hashBlacklist ??????????????????
func hashBlacklist(list []string) string {
	h := sha256.New()
	for _, s := range list {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// isIPInBlacklist ?? IP ???????(?? CIDR,????)
func (s *OrderRiskControlService) isIPInBlacklist(clientIP string, blacklist []string) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	parsed := s.getOrBuildBlacklist(blacklist)

	// ?? IP ??(O(1) ????)
	if _, ok := parsed.exactIPs[clientIP]; ok {
		return true
	}

	// CIDR ??
	for _, cidr := range parsed.cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}
