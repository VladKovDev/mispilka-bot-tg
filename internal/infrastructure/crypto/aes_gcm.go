package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

type aesGCMEncryptor struct {
	aead cipher.AEAD
}

func NewAESEncryptor(key []byte) (Encryptor, error) {
	if !ValidateAESKey(key) {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrUnknownKeyVersion
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrUnknownKeyVersion
	}

	return &aesGCMEncryptor{aead: gcm}, nil
}

func (e *aesGCMEncryptor) Encrypt(plainText []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, ErrEncryptionFail
	}

	// Prepend nonce to ciphertext so caller can decrypt
	cipherText := e.aead.Seal(nil, nonce, plainText, nil)
	out := make([]byte, 0, nonceSize+len(cipherText))
	out = append(out, nonce...)
	out = append(out, cipherText...)
	return out, nil
}

func (e *aesGCMEncryptor) Decrypt(cipherText []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	if len(cipherText) < nonceSize {
		return nil, ErrDecryptionFail
	}

	nonce := cipherText[:nonceSize]
	ct := cipherText[nonceSize:]

	plain, err := e.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, ErrDecryptionFail
	}

	return plain, nil
}

func ValidateAESKey(key []byte) bool {
	switch len(key) {
	case 16, 24, 32:
		return true
	default:
		return false
	}
}