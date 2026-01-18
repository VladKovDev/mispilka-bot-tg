package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestAESEncryptor_EncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to read random key: %v", err)
	}

	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatalf("NewAESEncryptor error: %v", err)
	}

	plain := []byte("super-secret-token-123")
	ct, err := enc.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	if bytes.Equal(ct, plain) {
		t.Fatalf("ciphertext should not equal plaintext")
	}

	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}

	if !bytes.Equal(pt, plain) {
		t.Fatalf("decrypted mismatch: got %v want %v", pt, plain)
	}
}

func TestAESEncryptor_WrongKeyDecryptFails(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	if _, err := rand.Read(key1); err != nil {
		t.Fatalf("rand read key1: %v", err)
	}
	if _, err := rand.Read(key2); err != nil {
		t.Fatalf("rand read key2: %v", err)
	}

	enc1, err := NewAESEncryptor(key1)
	if err != nil {
		t.Fatalf("NewAESEncryptor key1: %v", err)
	}

	ct, err := enc1.Encrypt([]byte("data-to-encrypt"))
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	enc2, err := NewAESEncryptor(key2)
	if err != nil {
		t.Fatalf("NewAESEncryptor key2: %v", err)
	}

	if _, err := enc2.Decrypt(ct); err == nil {
		t.Fatalf("expected decrypt to fail with wrong key, but it succeeded")
	}
}
