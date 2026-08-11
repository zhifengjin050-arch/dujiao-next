package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dujiao-next/internal/config"
	"github.com/gin-gonic/gin"
)

func TestResolveAllowedOriginWildcard(t *testing.T) {
	allowed := []string{"*"}
	result := resolveAllowedOrigin("https://example.com", allowed, false)
	if result != "*" {
		t.Fatalf("expected '*', got %q", result)
	}
}

func TestResolveAllowedOriginWildcardWithCredentials(t *testing.T) {
	allowed := []string{"*"}
	// 当 allowCredentials=true 且有 origin 时，应返回原始 origin 而非 "*"
	result := resolveAllowedOrigin("https://example.com", allowed, true)
	if result != "https://example.com" {
		t.Fatalf("expected 'https://example.com', got %q", result)
	}
}

func TestResolveAllowedOriginWildcardWithCredentialsEmptyOrigin(t *testing.T) {
	allowed := []string{"*"}
	// 当 allowCredentials=true 但 origin 为空时，应返回 "*"
	result := resolveAllowedOrigin("", allowed, true)
	if result != "*" {
		t.Fatalf("expected '*', got %q", result)
	}
}

func TestResolveAllowedOriginExactMatch(t *testing.T) {
	allowed := []string{"https://example.com", "https://app.example.com"}
	result := resolveAllowedOrigin("https://example.com", allowed, false)
	if result != "https://example.com" {
		t.Fatalf("expected 'https://example.com', got %q", result)
	}
}

func TestResolveAllowedOriginExactMatchCaseInsensitive(t *testing.T) {
	allowed := []string{"https://Example.COM"}
	result := resolveAllowedOrigin("https://example.com", allowed, false)
	if result != "https://example.com" {
		t.Fatalf("expected 'https://example.com', got %q", result)
	}
}

func TestResolveAllowedOriginNoMatch(t *testing.T) {
	allowed := []string{"https://example.com"}
	result := resolveAllowedOrigin("https://evil.com", allowed, false)
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestResolveAllowedOriginEmptyList(t *testing.T) {
	result := resolveAllowedOrigin("https://example.com", []string{}, false)
	if result != "" {
		t.Fatalf("expected empty string for empty list, got %q", result)
	}
}

func TestResolveAllowedOriginEmptyInput(t *testing.T) {
	allowed := []string{"https://example.com"}
	result := resolveAllowedOrigin("", allowed, false)
	if result != "" {
		t.Fatalf("expected empty string for empty origin, got %q", result)
	}
}

func TestCORSMiddlewareWithDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware(config.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           600,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected Allow-Origin=*, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Max-Age") != "600" {
		t.Fatalf("expected Max-Age=600, got %q", w.Header().Get("Access-Control-Max-Age"))
	}
}

func TestCORSMiddlewareWithCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware(config.CORSConfig{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("expected Allow-Credentials=true")
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("expected Allow-Origin=https://app.example.com, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddlewareDisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware(config.CORSConfig{
		AllowedOrigins:   []string{"https://trusted.com"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	router.ServeHTTP(w, req)

	// 非允许的 origin 不应设置 Allow-Origin 头
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no Allow-Origin for disallowed origin, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddlewareEmptyOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware(config.CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// 不设置 Origin header
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCORSMiddlewarePrefersConfiguredMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware(config.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"PUT", "DELETE"},
		AllowedHeaders:   []string{"X-Custom"},
		AllowCredentials: false,
	}))
	router.PUT("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://test.com")
	router.ServeHTTP(w, req)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods != "PUT, DELETE" {
		t.Fatalf("expected Allow-Methods='PUT, DELETE', got %q", methods)
	}
	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers != "X-Custom" {
		t.Fatalf("expected Allow-Headers='X-Custom', got %q", headers)
	}
}

func TestCORSMiddlewareMaxAgeZero(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware(config.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Content-Type"},
		MaxAge:           0,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://test.com")
	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Max-Age") != "" {
		t.Fatalf("expected no Max-Age when zero, got %q", w.Header().Get("Access-Control-Max-Age"))
	}
}
