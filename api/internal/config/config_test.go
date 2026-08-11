package config

import (
	"testing"
)

func TestDefaultCORSAllowedOriginsReturnsCopy(t *testing.T) {
	origins1 := DefaultCORSAllowedOrigins()
	origins2 := DefaultCORSAllowedOrigins()

	// 两次调用应该返回不同的底层数组（副本）
	if len(origins1) != 1 || origins1[0] != "*" {
		t.Fatalf("expected [\"*\"], got %v", origins1)
	}
	if len(origins2) != 1 || origins2[0] != "*" {
		t.Fatalf("expected [\"*\"], got %v", origins2)
	}

	// 修改其中一个副本不应影响默认值
	origins1[0] = "modified"
	origins3 := DefaultCORSAllowedOrigins()
	if origins3[0] != "*" {
		t.Fatal("modifying a copy should not affect the default value")
	}
}

func TestDefaultCORSAllowedMethodsReturnsCopy(t *testing.T) {
	methods := DefaultCORSAllowedMethods()
	if len(methods) == 0 {
		t.Fatal("expected non-empty methods list")
	}
	// 验证包含核心方法
	hasGet := false
	hasPost := false
	for _, m := range methods {
		if m == "GET" {
			hasGet = true
		}
		if m == "POST" {
			hasPost = true
		}
	}
	if !hasGet || !hasPost {
		t.Fatal("expected GET and POST in default allowed methods")
	}

	// 验证副本独立性
	methodsCopy := DefaultCORSAllowedMethods()
	methods[0] = "CUSTOM"
	anotherCopy := DefaultCORSAllowedMethods()
	if anotherCopy[0] == "CUSTOM" {
		t.Fatal("modifying a copy should not affect the default value")
	}
	_ = methodsCopy
}

func TestDefaultCORSAllowedHeadersReturnsCopy(t *testing.T) {
	headers := DefaultCORSAllowedHeaders()
	if len(headers) == 0 {
		t.Fatal("expected non-empty headers list")
	}
	// 验证包含关键头
	hasAuth := false
	for _, h := range headers {
		if h == "Authorization" {
			hasAuth = true
			break
		}
	}
	if !hasAuth {
		t.Fatal("expected Authorization in default allowed headers")
	}

	// 验证副本独立性
	headers[0] = "X-Custom"
	anotherCopy := DefaultCORSAllowedHeaders()
	if anotherCopy[0] == "X-Custom" {
		t.Fatal("modifying a copy should not affect the default value")
	}
}

func TestLogConfigToLoggerOptions(t *testing.T) {
	cfg := LogConfig{
		Dir:        "/var/log/app",
		Filename:   "app.log",
		MaxSizeMB:  100,
		MaxBackups: 7,
		MaxAgeDays: 30,
		Compress:   true,
	}

	opts := cfg.ToLoggerOptions()

	if opts.Dir != "/var/log/app" {
		t.Fatalf("expected Dir=/var/log/app, got %s", opts.Dir)
	}
	if opts.Filename != "app.log" {
		t.Fatalf("expected Filename=app.log, got %s", opts.Filename)
	}
	if opts.MaxSizeMB != 100 {
		t.Fatalf("expected MaxSizeMB=100, got %d", opts.MaxSizeMB)
	}
	if opts.MaxBackups != 7 {
		t.Fatalf("expected MaxBackups=7, got %d", opts.MaxBackups)
	}
	if opts.MaxAgeDays != 30 {
		t.Fatalf("expected MaxAgeDays=30, got %d", opts.MaxAgeDays)
	}
	if !opts.Compress {
		t.Fatal("expected Compress=true")
	}
}

func TestLogConfigToLoggerOptionsDefaults(t *testing.T) {
	cfg := LogConfig{}
	opts := cfg.ToLoggerOptions()

	if opts.Dir != "" {
		t.Fatalf("expected empty Dir, got %s", opts.Dir)
	}
	if opts.Filename != "" {
		t.Fatalf("expected empty Filename, got %s", opts.Filename)
	}
	if opts.MaxSizeMB != 0 {
		t.Fatalf("expected MaxSizeMB=0, got %d", opts.MaxSizeMB)
	}
	if opts.Compress {
		t.Fatal("expected Compress=false with zero value")
	}
}

func TestConfigStructTypes(t *testing.T) {
	// 验证配置结构体字段类型正确
	cfg := Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: "8080",
			Mode: "debug",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "./db/test.db",
		},
		JWT: JWTConfig{
			SecretKey:   "test-secret",
			ExpireHours: 24,
		},
	}

	if cfg.Server.Port != "8080" {
		t.Fatalf("expected Port=8080, got %s", cfg.Server.Port)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Fatalf("expected Driver=sqlite, got %s", cfg.Database.Driver)
	}
	if cfg.JWT.ExpireHours != 24 {
		t.Fatalf("expected ExpireHours=24, got %d", cfg.JWT.ExpireHours)
	}
}

func TestSecurityConfigDefaults(t *testing.T) {
	cfg := SecurityConfig{
		LoginRateLimit: LoginRateLimitConfig{
			WindowSeconds: 300,
			MaxAttempts:   5,
			BlockSeconds:  900,
		},
		PasswordPolicy: PasswordPolicyConfig{
			MinLength:      8,
			RequireUpper:   true,
			RequireLower:   true,
			RequireNumber:  true,
			RequireSpecial: false,
		},
	}

	if cfg.LoginRateLimit.MaxAttempts != 5 {
		t.Fatalf("expected MaxAttempts=5, got %d", cfg.LoginRateLimit.MaxAttempts)
	}
	if cfg.PasswordPolicy.MinLength != 8 {
		t.Fatalf("expected MinLength=8, got %d", cfg.PasswordPolicy.MinLength)
	}
	if !cfg.PasswordPolicy.RequireUpper {
		t.Fatal("expected RequireUpper=true")
	}
	if cfg.PasswordPolicy.RequireSpecial {
		t.Fatal("expected RequireSpecial=false")
	}
}

func TestCORSConfigStruct(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           600,
	}

	if len(cfg.AllowedOrigins) != 1 {
		t.Fatalf("expected 1 origin, got %d", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0] != "https://example.com" {
		t.Fatalf("expected https://example.com, got %s", cfg.AllowedOrigins[0])
	}
	if !cfg.AllowCredentials {
		t.Fatal("expected AllowCredentials=true")
	}
	if cfg.MaxAge != 600 {
		t.Fatalf("expected MaxAge=600, got %d", cfg.MaxAge)
	}
}

func TestOrderConfigDefaults(t *testing.T) {
	cfg := OrderConfig{
		PaymentExpireMinutes: 15,
		MaxRefundDays:        30,
	}

	if cfg.PaymentExpireMinutes != 15 {
		t.Fatalf("expected PaymentExpireMinutes=15, got %d", cfg.PaymentExpireMinutes)
	}
	if cfg.MaxRefundDays != 30 {
		t.Fatalf("expected MaxRefundDays=30, got %d", cfg.MaxRefundDays)
	}
}

func TestUploadConfigDefaults(t *testing.T) {
	cfg := UploadConfig{
		MaxSize:           10485760,
		AllowedTypes:      []string{"image/jpeg", "image/png"},
		AllowedExtensions: []string{".jpg", ".png"},
		MaxWidth:          4096,
		MaxHeight:         4096,
	}

	if cfg.MaxSize != 10485760 {
		t.Fatalf("expected MaxSize=10485760, got %d", cfg.MaxSize)
	}
	if len(cfg.AllowedTypes) != 2 {
		t.Fatalf("expected 2 allowed types, got %d", len(cfg.AllowedTypes))
	}
	if cfg.MaxWidth != 4096 {
		t.Fatalf("expected MaxWidth=4096, got %d", cfg.MaxWidth)
	}
}
