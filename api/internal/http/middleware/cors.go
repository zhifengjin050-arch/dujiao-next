package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/config"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware ?????
func CORSMiddleware(cfg config.CORSConfig) gin.HandlerFunc {
	allowedOrigins := cfg.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = config.DefaultCORSAllowedOrigins()
	}
	allowedMethods := cfg.AllowedMethods
	if len(allowedMethods) == 0 {
		allowedMethods = config.DefaultCORSAllowedMethods()
	}
	allowedHeaders := cfg.AllowedHeaders
	if len(allowedHeaders) == 0 {
		allowedHeaders = config.DefaultCORSAllowedHeaders()
	}
	methodsHeader := strings.Join(allowedMethods, ", ")
	headersHeader := strings.Join(allowedHeaders, ", ")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowedOrigin := resolveAllowedOrigin(origin, allowedOrigins, cfg.AllowCredentials)
		if allowedOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			if allowedOrigin != "*" {
				c.Writer.Header().Add("Vary", "Origin")
			}
		}
		if cfg.AllowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", headersHeader)
		c.Writer.Header().Set("Access-Control-Allow-Methods", methodsHeader)
		if cfg.MaxAge > 0 {
			c.Writer.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func resolveAllowedOrigin(origin string, allowedOrigins []string, allowCredentials bool) string {
	if len(allowedOrigins) == 0 {
		return ""
	}
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			if allowCredentials && origin != "" {
				return origin
			}
			return "*"
		}
	}
	if origin == "" {
		return ""
	}
	for _, allowed := range allowedOrigins {
		if strings.EqualFold(allowed, origin) {
			return origin
		}
	}
	return ""
}
