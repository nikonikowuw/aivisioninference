// Package cryptoutil provides encryption utilities for sensitive data.
package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// sshKeyEncryptionKey returns the AES encryption key for SSH private keys.
// The key is read from the SSH_KEY_ENC_KEY environment variable and must be
// exactly 32 bytes (64 hex characters) for AES-256-GCM.
func sshKeyEncryptionKey() ([]byte, error) {
	keyHex := os.Getenv("SSH_KEY_ENC_KEY")
	if keyHex == "" {
		return nil, fmt.Errorf("SSH_KEY_ENC_KEY environment variable is not set")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("SSH_KEY_ENC_KEY is not valid hex: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("SSH_KEY_ENC_KEY must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}
	return key, nil
}

// EncryptSSHKey encrypts a plaintext SSH private key using AES-256-GCM.
// Returns a byte slice containing (nonce || ciphertext).
func EncryptSSHKey(plaintext []byte) ([]byte, error) {
	key, err := sshKeyEncryptionKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends ciphertext to nonce, returning (nonce || ciphertext).
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptSSHKey decrypts an SSH private key that was encrypted with EncryptSSHKey.
// The ciphertext must be (nonce || ciphertext).
func DecryptSSHKey(ciphertext []byte) ([]byte, error) {
	key, err := sshKeyEncryptionKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, cryptoText := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, cryptoText, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt SSH key: %w", err)
	}

	return plaintext, nil
}
