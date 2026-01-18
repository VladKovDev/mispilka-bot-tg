package crypto

import "errors"

var (
	ErrEncryptionFail = errors.New("encryption failed")
	ErrInvalidKeySize = errors.New("invalid key size, must be 16, 24, or 32 bytes")
	ErrDecryptionFail = errors.New("decryption failed")
	ErrUnknownKeyVersion = errors.New("unknown key version")
)
