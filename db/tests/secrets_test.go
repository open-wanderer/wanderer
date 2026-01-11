package util

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"os"
	"testing"

	"github.com/pocketbase/pocketbase/tools/security"

	"pocketbase/util"
)

func TestLooksLikeEncrypted(t *testing.T) {
	// Sub-test for non-base64 input.
	t.Run("NonBase64", func(t *testing.T) {
		if util.LooksLikeEncrypted("not a base64 string!") {
			t.Errorf("Expected false for non-base64 input")
		}
	})

	// Sub-test for an empty string.
	t.Run("EmptyString", func(t *testing.T) {
		if util.LooksLikeEncrypted("") {
			t.Errorf("Expected false for empty string")
		}
	})

	// Create a dummy byte slice to determine the nonce size using a dummy 32-byte key.
	dummyKey := make([]byte, 32)
	block, err := aes.NewCipher(dummyKey)
	if err != nil {
		t.Fatalf("Failed to create dummy cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("Failed to create dummy GCM: %v", err)
	}
	nonceSize := gcm.NonceSize()
	minSize := nonceSize + 16

	// Sub-test for data length less than nonce+16 bytes.
	t.Run("ShortData", func(t *testing.T) {
		shortData := make([]byte, minSize-1)
		encodedShort := base64.StdEncoding.EncodeToString(shortData)
		if util.LooksLikeEncrypted(encodedShort) {
			t.Errorf("Expected false for data length < nonce+16, got true")
		}
	})

	// Sub-test for data length exactly equal to nonce+16 bytes.
	t.Run("ExactData", func(t *testing.T) {
		exactData := make([]byte, minSize)
		encodedExact := base64.StdEncoding.EncodeToString(exactData)
		if !util.LooksLikeEncrypted(encodedExact) {
			t.Errorf("Expected true for data length == nonce+16")
		}
	})

	// Sub-test for data length greater than nonce+16 bytes.
	t.Run("LongData", func(t *testing.T) {
		longData := make([]byte, minSize+10)
		encodedLong := base64.StdEncoding.EncodeToString(longData)
		if !util.LooksLikeEncrypted(encodedLong) {
			t.Errorf("Expected true for data length > nonce+16")
		}
	})
}

func TestCanDecryptSecret(t *testing.T) {
	// Sub-test: When POCKETBASE_ENCRYPTION_KEY is not set.
	t.Run("NoEncryptionKey", func(t *testing.T) {
		if err := os.Unsetenv("POCKETBASE_ENCRYPTION_KEY"); err != nil {
			t.Fatalf("Failed to unset encryption key: %v", err)
		}
		if util.CanDecryptSecret("anyciphertext") {
			t.Errorf("Expected false when POCKETBASE_ENCRYPTION_KEY is not set")
		}
	})

	// Set a valid 32-byte key (for AES-256).
	encryptionKey := "0123456789abcdef0123456789abcdef" // exactly 32 bytes
	if err := os.Setenv("POCKETBASE_ENCRYPTION_KEY", encryptionKey); err != nil {
		t.Fatalf("Failed to set encryption key: %v", err)
	}
	// Cleanup environment variable after the test.
	t.Cleanup(func() {
		if err := os.Unsetenv("POCKETBASE_ENCRYPTION_KEY"); err != nil {
			t.Fatalf("Failed to unset encryption key: %v", err)
		}
	})

	// Sub-test: Valid ciphertext should return true.
	t.Run("ValidCiphertext", func(t *testing.T) {
		plaintext := "my secret message"
		// Cast plaintext to []byte as required by security.Encrypt.
		ciphertext, err := security.Encrypt([]byte(plaintext), encryptionKey)
		if err != nil {
			t.Fatalf("Failed to encrypt secret: %v", err)
		}
		if !util.CanDecryptSecret(ciphertext) {
			t.Errorf("Expected true for valid ciphertext with correct key")
		}
	})

	// Sub-test: Invalid ciphertext should return false.
	t.Run("InvalidCiphertext", func(t *testing.T) {
		invalidCiphertext := "invalid_ciphertext"
		if util.CanDecryptSecret(invalidCiphertext) {
			t.Errorf("Expected false for invalid ciphertext")
		}
	})
}
