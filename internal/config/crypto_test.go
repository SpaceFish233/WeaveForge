package config

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty", ""},
		{"short key", "sk-abc123"},
		{"long key", "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890"},
		{"unicode key", "密钥测试-abc-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.plaintext == "" {
				enc, err := encrypt(tt.plaintext)
				if err != nil {
					t.Fatalf("encrypt empty: %v", err)
				}
				if enc != "" {
					t.Errorf("encrypt empty should return empty, got %q", enc)
				}
				return
			}
			enc, err := encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("encrypt: %v", err)
			}
			if enc == tt.plaintext {
				t.Error("encrypted value should differ from plaintext")
			}
			if !isEncrypted(enc) {
				t.Error("isEncrypted should return true for encrypted value")
			}
			dec, err := decrypt(enc)
			if err != nil {
				t.Fatalf("decrypt: %v", err)
			}
			if dec != tt.plaintext {
				t.Errorf("decrypt = %q, want %q", dec, tt.plaintext)
			}
		})
	}
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"empty", "", false},
		{"plain text", "hello", false},
		{"short base64", "aGVsbG8=", false}, // "hello" in base64 = 8 bytes, too short
		{"encrypted", func() string {
			enc, _ := encrypt("test-api-key-12345")
			return enc
		}(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEncrypted(tt.s); got != tt.want {
				t.Errorf("isEncrypted(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestDecodeKey_LegacyBase64(t *testing.T) {
	// Simulate old base64-encoded key
	legacyKey := EncodeKey("sk-test-legacy-key")
	// Force it to base64 format (bypass encryption)
	base64Key := "c2stdGVzdC1sZWdhY3kt" // "sk-test-legacy-" in base64 (short, won't be isEncrypted)

	decoded := DecodeKey(base64Key)
	// Should fall back to base64 decoding
	if decoded == "" {
		t.Error("DecodeKey should not return empty for valid base64")
	}
	_ = legacyKey
}
