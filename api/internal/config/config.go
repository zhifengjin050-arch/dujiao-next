package config

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"

	"github.com/spf13/viper"
)

// Config ??????
type Config struct {
	App          AppConfig          `mapstructure:"app"`
	Server       ServerConfig       `mapstructure:"server"`
	Log          LogConfig          `mapstructure:"log"`
	Database     DatabaseConfig     `mapstructure:"database"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	UserJWT      JWTConfig          `mapstructure:"user_jwt"`
	Bootstrap    BootstrapConfig    `mapstructure:"bootstrap"`
	TelegramAuth TelegramAuthConfig `mapstructure:"telegram_auth"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Queue        QueueConfig        `mapstructure:"queue"`
	Upload       UploadConfig       `mapstructure:"upload"`
	CORS         CORSConfig         `mapstructure:"cors"`
	Security     SecurityConfig     `mapstructure:"security"`
	Email        EmailConfig        `mapstructure:"email"`
	Order        OrderConfig        `mapstructure:"order"`
	Captcha      CaptchaConfig      `mapstructure:"captcha"`
}

// AppConfig ?????
type AppConfig struct {
	SecretKey string `mapstructure:"secret_key"` // ??????(AES-256,??????????)
}

// ServerConfig ?????
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug / release
}

// LogConfig ????
type LogConfig struct {
	Dir        string `mapstructure:"dir"`
	Filename   string `mapstructure:"filename"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
	Compress   bool   `mapstructure:"compress"`
}

// ToLoggerOptions ??? logger ??
func (c LogConfig) ToLoggerOptions() logger.Options {
	return logger.Options{
		Dir:        c.Dir,
		Filename:   c.Filename,
		MaxSizeMB:  c.MaxSizeMB,
		MaxBackups: c.MaxBackups,
		MaxAgeDays: c.MaxAgeDays,
		Compress:   c.Compress,
	}
}

// DatabasePoolConfig ????????
type DatabasePoolConfig struct {
	MaxOpenConns           int `mapstructure:"max_open_conns"`
	MaxIdleConns           int `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeSeconds int `mapstructure:"conn_max_lifetime_seconds"`
	ConnMaxIdleTimeSeconds int `mapstructure:"conn_max_idle_time_seconds"`
}

// DatabaseConfig ?????
type DatabaseConfig struct {
	Driver string             `mapstructure:"driver"` // ?????(sqlite/postgres)
	DSN    string             `mapstructure:"dsn"`    // ??????
	Pool   DatabasePoolConfig `mapstructure:"pool"`
}

// JWTConfig JWT ??
type JWTConfig struct {
	SecretKey             string `mapstructure:"secret"`
	ExpireHours           int    `mapstructure:"expire_hours"`
	RememberMeExpireHours int    `mapstructure:"remember_me_expire_hours"`
}

// BootstrapConfig ???????
type BootstrapConfig struct {
	DefaultAdminUsername string `mapstructure:"default_admin_username"`
	DefaultAdminPassword string `mapstructure:"default_admin_password"`
}

// TelegramAuthConfig Telegram ????
type TelegramAuthConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	BotUsername        string `mapstructure:"bot_username"`
	BotToken           string `mapstructure:"bot_token"`
	MiniAppURL         string `mapstructure:"mini_app_url"`
	LoginExpireSeconds int    `mapstructure:"login_expire_seconds"`
	ReplayTTLSeconds   int    `mapstructure:"replay_ttl_seconds"`
}

// RedisConfig Redis ??
type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	Prefix   string `mapstructure:"prefix"`
}

// QueueConfig ??????
type QueueConfig struct {
	Enabled              bool           `mapstructure:"enabled"`
	Host                 string         `mapstructure:"host"`
	Port                 int            `mapstructure:"port"`
	Password             string         `mapstructure:"password"`
	DB                   int            `mapstructure:"db"`
	Concurrency          int            `mapstructure:"concurrency"`
	Queues               map[string]int `mapstructure:"queues"`
	UpstreamSyncInterval string         `mapstructure:"upstream_sync_interval"` // ????????,? "5m"?"10m",?? "5m"
}

// OrderConfig ????
type OrderConfig struct {
	PaymentExpireMinutes int `mapstructure:"payment_expire_minutes"`
	MaxRefundDays        int `mapstructure:"max_refund_days"`
}

// EmailConfig ??????
type EmailConfig struct {
	Enabled    bool             `mapstructure:"enabled"`
	Host       string           `mapstructure:"host"`
	Port       int              `mapstructure:"port"`
	Username   string           `mapstructure:"username"`
	Password   string           `mapstructure:"password"`
	From       string           `mapstructure:"from"`
	FromName   string           `mapstructure:"from_name"`
	UseTLS     bool             `mapstructure:"use_tls"`
	UseSSL     bool             `mapstructure:"use_ssl"`
	VerifyCode VerifyCodeConfig `mapstructure:"verify_code"`
}

// VerifyCodeConfig ???????
type VerifyCodeConfig struct {
	ExpireMinutes       int `mapstructure:"expire_minutes"`
	SendIntervalSeconds int `mapstructure:"send_interval_seconds"`
	MaxAttempts         int `mapstructure:"max_attempts"`
	Length              int `mapstructure:"length"`
}

// CaptchaConfig ?????
type CaptchaConfig struct {
	Provider  string                 `mapstructure:"provider"`
	Scenes    CaptchaSceneConfig     `mapstructure:"scenes"`
	Image     CaptchaImageConfig     `mapstructure:"image"`
	Turnstile CaptchaTurnstileConfig `mapstructure:"turnstile"`
}

// CaptchaSceneConfig ???????
type CaptchaSceneConfig struct {
	Login            bool `mapstructure:"login"`
	RegisterSendCode bool `mapstructure:"register_send_code"`
	ResetSendCode    bool `mapstructure:"reset_send_code"`
	GuestCreateOrder bool `mapstructure:"guest_create_order"`
	GiftCardRedeem   bool `mapstructure:"gift_card_redeem"`
}

// CaptchaImageConfig ???????
type CaptchaImageConfig struct {
	Length        int `mapstructure:"length"`
	Width         int `mapstructure:"width"`
	Height        int `mapstructure:"height"`
	NoiseCount    int `mapstructure:"noise_count"`
	ShowLine      int `mapstructure:"show_line"`
	ExpireSeconds int `mapstructure:"expire_seconds"`
	MaxStore      int `mapstructure:"max_store"`
}

// CaptchaTurnstileConfig Cloudflare Turnstile ??
type CaptchaTurnstileConfig struct {
	SiteKey   string `mapstructure:"site_key"`
	SecretKey string `mapstructure:"secret_key"`
	VerifyURL string `mapstructure:"verify_url"`
	TimeoutMS int    `mapstructure:"timeout_ms"`
}

// UploadConfig ??????
type UploadConfig struct {
	MaxSize           int64    `mapstructure:"max_size"`
	AllowedTypes      []string `mapstructure:"allowed_types"`
	AllowedExtensions []string `mapstructure:"allowed_extensions"`
	MaxWidth          int      `mapstructure:"max_width"`
	MaxHeight         int      `mapstructure:"max_height"`
}

// CORSConfig ????
type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

var (
	defaultCORSAllowedOrigins = []string{"*"}
	defaultCORSAllowedMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodPatch}
	defaultCORSAllowedHeaders = []string{
		"Content-Type",
		"Content-Length",
		"Accept-Encoding",
		"Authorization",
		"Cache-Control",
		"X-Requested-With",
		"X-CSRF-Token",
	}
)

// DefaultCORSAllowedOrigins ??????????(??,????????????)
func DefaultCORSAllowedOrigins() []string {
	return append([]string(nil), defaultCORSAllowedOrigins...)
}

// DefaultCORSAllowedMethods ??????????(??,????????????)
func DefaultCORSAllowedMethods() []string {
	return append([]string(nil), defaultCORSAllowedMethods...)
}

// DefaultCORSAllowedHeaders ???????????(??,????????????)
func DefaultCORSAllowedHeaders() []string {
	return append([]string(nil), defaultCORSAllowedHeaders...)
}

// SecurityConfig ????
type SecurityConfig struct {
	LoginRateLimit LoginRateLimitConfig `mapstructure:"login_rate_limit"`
	PasswordPolicy PasswordPolicyConfig `mapstructure:"password_policy"`
}

// LoginRateLimitConfig ??????
type LoginRateLimitConfig struct {
	WindowSeconds int `mapstructure:"window_seconds"`
	MaxAttempts   int `mapstructure:"max_attempts"`
	BlockSeconds  int `mapstructure:"block_seconds"`
}

// PasswordPolicyConfig ??????
type PasswordPolicyConfig struct {
	MinLength      int  `mapstructure:"min_length"`
	RequireUpper   bool `mapstructure:"require_upper"`
	RequireLower   bool `mapstructure:"require_lower"`
	RequireNumber  bool `mapstructure:"require_number"`
	RequireSpecial bool `mapstructure:"require_special"`
}

// Load ? config.yml ????
func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")     // ???????
	viper.AddConfigPath("./")    // ????
	viper.AddConfigPath("../")   // ??? cmd/server ??
	viper.AddConfigPath("./etc") // etc ???

	// ?????(??)
	viper.SetDefault("app.secret_key", "change-me-32-byte-secret-key!!")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("log.dir", "")
	viper.SetDefault("log.filename", "app.log")
	viper.SetDefault("log.max_size_mb", 100)
	viper.SetDefault("log.max_backups", 7)
	viper.SetDefault("log.max_age_days", 30)
	viper.SetDefault("log.compress", true)
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.dsn", "./db/dujiao.db")
	viper.SetDefault("database.pool.max_open_conns", 1)
	viper.SetDefault("database.pool.max_idle_conns", 1)
	viper.SetDefault("database.pool.conn_max_lifetime_seconds", 0)
	viper.SetDefault("database.pool.conn_max_idle_time_seconds", 0)
	viper.SetDefault("jwt.secret", "change-me-in-production")
	viper.SetDefault("jwt.expire_hours", 24)
	viper.SetDefault("user_jwt.secret", "user-change-me-in-production")
	viper.SetDefault("user_jwt.expire_hours", 24)
	viper.SetDefault("user_jwt.remember_me_expire_hours", 168)
	viper.SetDefault("bootstrap.default_admin_username", "")
	viper.SetDefault("bootstrap.default_admin_password", "")
	viper.SetDefault("telegram_auth.enabled", false)
	viper.SetDefault("telegram_auth.bot_username", "")
	viper.SetDefault("telegram_auth.bot_token", "")
	viper.SetDefault("telegram_auth.login_expire_seconds", 300)
	viper.SetDefault("telegram_auth.replay_ttl_seconds", 300)
	viper.SetDefault("redis.enabled", true)
	viper.SetDefault("redis.host", "127.0.0.1")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.prefix", constants.RedisPrefixDefault)
	viper.SetDefault("queue.enabled", true)
	viper.SetDefault("queue.host", "127.0.0.1")
	viper.SetDefault("queue.port", 6379)
	viper.SetDefault("queue.password", "")
	viper.SetDefault("queue.db", 1)
	viper.SetDefault("queue.concurrency", 10)
	viper.SetDefault("queue.queues", map[string]int{
		"default":  10,
		"critical": 5,
	})
	viper.SetDefault("upload.max_size", 10485760)
	viper.SetDefault("upload.allowed_types", []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	})
	viper.SetDefault("upload.allowed_extensions", []string{
		".jpg",
		".jpeg",
		".png",
		".gif",
		".webp",
	})
	viper.SetDefault("upload.max_width", 4096)
	viper.SetDefault("upload.max_height", 4096)
	viper.SetDefault("cors.allowed_origins", DefaultCORSAllowedOrigins())
	viper.SetDefault("cors.allowed_methods", DefaultCORSAllowedMethods())
	viper.SetDefault("cors.allowed_headers", DefaultCORSAllowedHeaders())
	viper.SetDefault("cors.allow_credentials", true)
	viper.SetDefault("cors.max_age", 600)
	viper.SetDefault("security.login_rate_limit.window_seconds", 300)
	viper.SetDefault("security.login_rate_limit.max_attempts", 5)
	viper.SetDefault("security.login_rate_limit.block_seconds", 900)
	viper.SetDefault("security.password_policy.min_length", 8)
	viper.SetDefault("security.password_policy.require_upper", true)
	viper.SetDefault("security.password_policy.require_lower", true)
	viper.SetDefault("security.password_policy.require_number", true)
	viper.SetDefault("security.password_policy.require_special", false)
	viper.SetDefault("email.enabled", false)
	viper.SetDefault("email.host", "")
	viper.SetDefault("email.port", 587)
	viper.SetDefault("email.username", "")
	viper.SetDefault("email.password", "")
	viper.SetDefault("email.from", "")
	viper.SetDefault("email.from_name", "")
	viper.SetDefault("email.use_tls", true)
	viper.SetDefault("email.use_ssl", false)
	viper.SetDefault("email.verify_code.expire_minutes", 10)
	viper.SetDefault("email.verify_code.send_interval_seconds", 60)
	viper.SetDefault("email.verify_code.max_attempts", 5)
	viper.SetDefault("email.verify_code.length", 6)
	viper.SetDefault("order.payment_expire_minutes", 15)
	viper.SetDefault("order.max_refund_days", 30)
	viper.SetDefault("captcha.provider", "none")
	viper.SetDefault("captcha.scenes.login", false)
	viper.SetDefault("captcha.scenes.register_send_code", false)
	viper.SetDefault("captcha.scenes.reset_send_code", false)
	viper.SetDefault("captcha.scenes.guest_create_order", false)
	viper.SetDefault("captcha.scenes.gift_card_redeem", false)
	viper.SetDefault("captcha.image.length", 5)
	viper.SetDefault("captcha.image.width", 240)
	viper.SetDefault("captcha.image.height", 80)
	viper.SetDefault("captcha.image.noise_count", 2)
	viper.SetDefault("captcha.image.show_line", 2)
	viper.SetDefault("captcha.image.expire_seconds", 300)
	viper.SetDefault("captcha.image.max_store", 10240)
	viper.SetDefault("captcha.turnstile.site_key", "")
	viper.SetDefault("captcha.turnstile.secret_key", "")
	viper.SetDefault("captcha.turnstile.verify_url", "https://challenges.cloudflare.com/turnstile/v0/siteverify")
	viper.SetDefault("captcha.turnstile.timeout_ms", 2000)

	// ??????
	viper.AutomaticEnv()                                   // ????????
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // ? . ??? _ (?? server.port -> SERVER_PORT)

	// ??????
	if err := viper.ReadInConfig(); err != nil {
		logger.Warnw("config_file_read_failed",
			"error", err,
			"fallback", "env_or_defaults",
		)
	} else {
		logger.Infow("config_file_loaded", "file", viper.ConfigFileUsed())
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		logger.Errorw("config_unmarshal_failed", "error", err)
		panic(fmt.Errorf("??????: %w", err))
	}

	return &cfg
}
