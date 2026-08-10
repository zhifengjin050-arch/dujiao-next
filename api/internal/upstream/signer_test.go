package upstream

import (
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	secret := "test-secret-key-12345"
	method := "POST"
	path := "/api/v1/upstream/orders"
	timestamp := time.Now().Unix()
	body := []byte(`{"sku_id":1,"quantity":1}`)

	sig := Sign(secret, method, path, timestamp, body)
	if sig == "" {
		t.Fatal("signature should not be empty")
	}

	// ???????????
	if !Verify(secret, method, path, sig, timestamp, body) {
		t.Fatal("signature verification should pass")
	}

	// ??? secret ??????
	if Verify("wrong-secret", method, path, sig, timestamp, body) {
		t.Fatal("signature verification should fail with wrong secret")
	}

	// ??? body ??????
	if Verify(secret, method, path, sig, timestamp, []byte(`{"sku_id":2}`)) {
		t.Fatal("signature verification should fail with different body")
	}

	// ??? path ??????
	if Verify(secret, method, "/api/v1/upstream/ping", sig, timestamp, body) {
		t.Fatal("signature verification should fail with different path")
	}
}

func TestSignEmptyBody(t *testing.T) {
	secret := "test-secret"
	sig1 := Sign(secret, "GET", "/test", 1000, nil)
	sig2 := Sign(secret, "GET", "/test", 1000, []byte{})

	// nil ?? []byte ? MD5 ??
	if sig1 != sig2 {
		t.Fatal("nil body and empty body should produce same signature")
	}
}

func TestIsTimestampValid(t *testing.T) {
	now := time.Now().Unix()

	if !IsTimestampValid(now) {
		t.Fatal("current timestamp should be valid")
	}

	if !IsTimestampValid(now - (MaxTimestampSkew / 2)) {
		t.Fatalf("timestamp within skew window should be valid: skew=%d", MaxTimestampSkew)
	}

	if IsTimestampValid(now - (MaxTimestampSkew + 1)) {
		t.Fatalf("timestamp older than skew window should be invalid: skew=%d", MaxTimestampSkew)
	}

	if IsTimestampValid(now + (MaxTimestampSkew + 1)) {
		t.Fatalf("future timestamp beyond skew window should be invalid: skew=%d", MaxTimestampSkew)
	}
}

func TestParseTimestamp(t *testing.T) {
	ts, err := ParseTimestamp("1709625600")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts != 1709625600 {
		t.Fatalf("expected 1709625600, got %d", ts)
	}

	_, err = ParseTimestamp("not-a-number")
	if err == nil {
		t.Fatal("expected error for non-numeric string")
	}
}
