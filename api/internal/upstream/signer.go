package upstream

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"
)

const (
	// HeaderApiKey API Key header
	HeaderApiKey = "Dujiao-Next-Api-Key"
	// HeaderTimestamp ??? header
	HeaderTimestamp = "Dujiao-Next-Timestamp"
	// HeaderSignature ?? header
	HeaderSignature = "Dujiao-Next-Signature"

	// MaxTimestampSkew ???????(?)
	MaxTimestampSkew = 60
)

// Sign ?? HMAC-SHA256 ??
// signString = "{method}\n{path}\n{timestamp}\n{body_md5}"
func Sign(secret, method, path string, timestamp int64, body []byte) string {
	bodyMD5 := md5Hex(body)
	signString := fmt.Sprintf("%s\n%s\n%d\n%s", method, path, timestamp, bodyMD5)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signString))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify ????
func Verify(secret, method, path, signature string, timestamp int64, body []byte) bool {
	expected := Sign(secret, method, path, timestamp, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// IsTimestampValid ?????????????
func IsTimestampValid(timestamp int64) bool {
	now := time.Now().Unix()
	return math.Abs(float64(now-timestamp)) <= MaxTimestampSkew
}

// ParseTimestamp ????????
func ParseTimestamp(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func md5Hex(data []byte) string {
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
