package router

import (
	"io"
	"net/http"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/upstream"

	"github.com/gin-gonic/gin"
)

const upstreamUserIDKey = "upstream_user_id"
const upstreamCredentialIDKey = "upstream_credential_id"

// UpstreamAPIAuthMiddleware ?? API ???????
func UpstreamAPIAuthMiddleware(credRepo repository.ApiCredentialRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader(upstream.HeaderApiKey)
		timestampStr := c.GetHeader(upstream.HeaderTimestamp)
		signature := c.GetHeader(upstream.HeaderSignature)

		if apiKey == "" || timestampStr == "" || signature == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error_code": "missing_auth_headers", "error_message": "missing authentication headers"})
			return
		}

		timestamp, err := upstream.ParseTimestamp(timestampStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error_code": "invalid_timestamp", "error_message": "invalid timestamp"})
			return
		}

		if !upstream.IsTimestampValid(timestamp) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error_code": "timestamp_expired", "error_message": "timestamp expired"})
			return
		}

		cred, err := credRepo.GetByApiKey(apiKey)
		if err != nil {
			logger.Errorw("upstream_auth_db_error", "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ok": false, "error_code": "internal_error", "error_message": "internal error"})
			return
		}
		if cred == nil || cred.Status != constants.ApiCredentialStatusApproved || !cred.IsActive {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"ok": false, "error_code": "invalid_api_key", "error_message": "api key is invalid or disabled"})
			return
		}
		if cred.User == nil || cred.User.Status != constants.UserStatusActive {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"ok": false, "error_code": "user_disabled", "error_message": "user account is disabled"})
			return
		}

		// ?? body ??????(???? 10MB ??????)
		var body []byte
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
			body, err = io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"ok": false, "error_code": "bad_request", "error_message": "failed to read request body"})
				return
			}
			// ?? body ??? handler ??
			c.Request.Body = io.NopCloser(
				&bodyReader{data: body},
			)
		}

		method := c.Request.Method
		path := c.Request.URL.Path

		if !upstream.Verify(cred.ApiSecret, method, path, signature, timestamp, body) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error_code": "invalid_signature", "error_message": "signature verification failed"})
			return
		}

		// ????????(??,?????)
		now := time.Now()
		cred.LastUsedAt = &now
		go func() {
			if updateErr := credRepo.Update(cred); updateErr != nil {
				logger.Warnw("upstream_auth_update_last_used_failed", "error", updateErr)
			}
		}()

		// ??????? context
		c.Set(upstreamUserIDKey, cred.UserID)
		c.Set(upstreamCredentialIDKey, cred.ID)
		c.Set("upstream_api_key", cred.ApiKey)

		c.Next()
	}
}

// bodyReader ?? io.Reader,???? body
type bodyReader struct {
	data   []byte
	offset int
}

func (r *bodyReader) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}
