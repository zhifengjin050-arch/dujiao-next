package crypto

import "testing"

func TestEncryptDecrypt(t *testing.T) {
	key := DeriveKey("my-secret-key")
	plaintext := "super-secret-api-key-12345"

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if encrypted == plaintext {
		t.Fatal("encrypted should differ from plaintext")
	}

	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := DeriveKey("key-1")
	key2 := DeriveKey("key-2")

	encrypted, err := Encrypt(key1, "hello")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	_, err = Decrypt(key2, encrypted)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	k1 := DeriveKey("test")
	k2 := DeriveKey("test")
	if string(k1) != string(k2) {
		t.Fatal("same input should produce same key")
	}
	if len(k1) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(k1))
	}
}
