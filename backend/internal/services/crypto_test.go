package services

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := []byte("thisisa32bytekeytousefortesting!") // 32 bytes (AES-256)
	plaintext := []byte("Hello, world! This is a test of AES-GCM encryption.")

	// 1. Encrypt the plaintext
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// 2. Decrypt the ciphertext
	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	// 3. Verify the decrypted plaintext matches original
	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("decrypted plaintext does not match original: got %q, want %q", string(decrypted), string(plaintext))
	}
}

func TestAESGCMKnownValues(t *testing.T) {
	// Known values computed with standard AES-256-GCM
	key := []byte("thisisa32bytekeytousefortesting!") // 32 bytes
	nonce := []byte("12bytenonce!")                   // 12 bytes
	plaintext := []byte("AES-GCM known value test")

	// Hex representation of expected ciphertext + tag (excluding nonce)
	expectedCiphertextHex := "ffe015daa3149f6826ed822f797e59d4bd7d285586e3d85dc22b18d38530c861d60cdf181efae0a4"
	expectedCiphertext, err := hex.DecodeString(expectedCiphertextHex)
	if err != nil {
		t.Fatalf("failed to decode expected ciphertext hex: %v", err)
	}

	// 1. Test EncryptWithNonce matches the known value
	ciphertext, err := EncryptWithNonce(plaintext, key, nonce)
	if err != nil {
		t.Fatalf("EncryptWithNonce failed: %v", err)
	}

	if !bytes.Equal(ciphertext, expectedCiphertext) {
		t.Errorf("EncryptWithNonce result mismatch:\n got:  %x\n want: %x", ciphertext, expectedCiphertext)
	}

	// 2. Test Decrypt with prefixed nonce matches the known plaintext
	// The Decrypt function expects the nonce prefixed to the ciphertext + tag
	prefixedCiphertext := append([]byte{}, nonce...)
	prefixedCiphertext = append(prefixedCiphertext, expectedCiphertext...)

	decrypted, err := Decrypt(prefixedCiphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed on known ciphertext: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypt result mismatch:\n got:  %q\n want: %q", string(decrypted), string(plaintext))
	}
}

func TestDecryptFailureCases(t *testing.T) {
	key := []byte("thisisa32bytekeytousefortesting!")
	wrongKey := []byte("wrong32bytekeyfortestingpurpose!")
	plaintext := []byte("Secret message")

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// 1. Decrypt with wrong key should fail
	_, err = Decrypt(ciphertext, wrongKey)
	if err == nil {
		t.Error("expected decryption to fail with a wrong key, but it succeeded")
	}

	// 2. Decrypt with tampered ciphertext should fail
	tamperedCiphertext := make([]byte, len(ciphertext))
	copy(tamperedCiphertext, ciphertext)
	if len(tamperedCiphertext) > 15 {
		// Tamper with the ciphertext payload/tag (after the nonce)
		tamperedCiphertext[len(tamperedCiphertext)-1] ^= 0xFF
	}

	_, err = Decrypt(tamperedCiphertext, key)
	if err == nil {
		t.Error("expected decryption to fail with tampered ciphertext, but it succeeded")
	}

	// 3. Decrypt with too short ciphertext should fail
	shortCiphertext := []byte("too_short")
	_, err = Decrypt(shortCiphertext, key)
	if err == nil {
		t.Error("expected decryption to fail with too short ciphertext, but it succeeded")
	}
}
