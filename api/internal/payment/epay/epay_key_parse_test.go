package epay

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"
)

func TestParseRSAPrivateKeyWithBase64Body(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key failed: %v", err)
	}

	rawDER := x509.MarshalPKCS1PrivateKey(key)
	rawBase64 := base64.StdEncoding.EncodeToString(rawDER)

	parsed, err := parseRSAPrivateKey(rawBase64)
	if err != nil {
		t.Fatalf("parse private key from base64 body failed: %v", err)
	}
	if parsed.N.Cmp(key.N) != 0 {
		t.Fatalf("private key modulus mismatch")
	}
}

func TestParseRSAPrivateKeyWithPEM(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key failed: %v", err)
	}

	rawDER := x509.MarshalPKCS1PrivateKey(key)
	rawPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: rawDER})

	parsed, err := parseRSAPrivateKey(string(rawPEM))
	if err != nil {
		t.Fatalf("parse private key from pem failed: %v", err)
	}
	if parsed.N.Cmp(key.N) != 0 {
		t.Fatalf("private key modulus mismatch")
	}
}

func TestParseRSAPublicKeyWithBase64Body(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key failed: %v", err)
	}

	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key failed: %v", err)
	}
	rawBase64 := base64.StdEncoding.EncodeToString(publicDER)

	parsed, err := parseRSAPublicKey(rawBase64)
	if err != nil {
		t.Fatalf("parse public key from base64 body failed: %v", err)
	}
	if parsed.N.Cmp(key.PublicKey.N) != 0 {
		t.Fatalf("public key modulus mismatch")
	}
}

func TestParseRSAPublicKeyWithPEM(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key failed: %v", err)
	}

	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key failed: %v", err)
	}
	rawPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})

	parsed, err := parseRSAPublicKey(string(rawPEM))
	if err != nil {
		t.Fatalf("parse public key from pem failed: %v", err)
	}
	if parsed.N.Cmp(key.PublicKey.N) != 0 {
		t.Fatalf("public key modulus mismatch")
	}
}
