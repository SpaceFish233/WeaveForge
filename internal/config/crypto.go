package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"runtime"
)

// machineKey derives a 32-byte AES key from machine-specific values.
// This provides per-machine encryption without requiring external key management.
func machineKey() []byte {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	h := sha256.New()
	h.Write([]byte("weaveforge-v1"))
	h.Write([]byte(hostname))
	h.Write([]byte(username))
	h.Write([]byte(runtime.GOOS))
	return h.Sum(nil)
}

// Encrypt encrypts plaintext using AES-GCM with a machine-derived key.
func encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key := machineKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("crypto: generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts ciphertext encrypted by encrypt().
func decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: decode base64: %w", err)
	}
	key := machineKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("crypto: ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt: %w", err)
	}
	return string(plaintext), nil
}

// isEncrypted checks if a string looks like an AES-GCM encrypted value.
// Encrypted values are base64-encoded and longer than plain base64-encoded keys.
func isEncrypted(s string) bool {
	if s == "" {
		return false
	}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return false
	}
	// AES-GCM ciphertext includes nonce (12 bytes) + tag (16 bytes) + payload
	// A valid encrypted value must be at least 28 bytes (nonce + tag)
	return len(data) >= 28
}
