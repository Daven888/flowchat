package cryptoutil

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := DeriveKey("test-secret-123")
	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := "sk-this-is-a-test-api-key-12345"
	encoded, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if encoded == "" {
		t.Fatal("encoded is empty")
	}
	if encoded == plaintext {
		t.Fatal("encoded equals plaintext, not encrypted")
	}

	decoded, err := enc.Decrypt(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != plaintext {
		t.Fatalf("decrypt mismatch: got %q, want %q", decoded, plaintext)
	}
}

func TestDecryptFailsWithDifferentKey(t *testing.T) {
	key1 := DeriveKey("key-alpha")
	key2 := DeriveKey("key-beta")
	enc1, _ := NewAESEncryptor(key1)
	enc2, _ := NewAESEncryptor(key2)

	plaintext := "secret-value"
	encoded, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	_, err = enc2.Decrypt(encoded)
	if err == nil {
		t.Fatal("expected decrypt to fail with wrong key, got nil error")
	}
}

func TestNewAESEncryptorInvalidKeySize(t *testing.T) {
	_, err := NewAESEncryptor([]byte("short"))
	if err == nil {
		t.Fatal("expected error for short key")
	}
	if err != ErrInvalidKeySize {
		t.Fatalf("expected ErrInvalidKeySize, got %v", err)
	}
}

func TestDecryptCorruptedData(t *testing.T) {
	key := DeriveKey("test-secret")
	enc, _ := NewAESEncryptor(key)

	_, err := enc.Decrypt("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}

	_, err = enc.Decrypt("dG9vLXNob3J0")
	if err == nil {
		t.Fatal("expected error for short ciphertext")
	}
}

func TestEncryptDeterministicallyDifferent(t *testing.T) {
	key := DeriveKey("test-secret")
	enc, _ := NewAESEncryptor(key)

	plaintext := "same-value"
	e1, _ := enc.Encrypt(plaintext)
	e2, _ := enc.Encrypt(plaintext)

	if e1 == e2 {
		t.Fatal("same plaintext produced identical ciphertext, nonce reuse suspected")
	}
}
