package encryption

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := randomKey(t)
	plaintext := []byte("ntm encryption round-trip test")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if len(ciphertext) <= headerSize {
		t.Fatalf("ciphertext too short: %d", len(ciphertext))
	}

	got, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round-trip mismatch: got %q want %q", string(got), string(plaintext))
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key := randomKey(t)
	wrongKey := randomKey(t)
	plaintext := []byte("ntm encryption wrong-key test")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	_, err = Decrypt(wrongKey, ciphertext)
	if err == nil {
		t.Fatal("Decrypt: expected error")
	}
	if !IsWrongKey(err) {
		t.Fatalf("expected wrong-key error, got %v", err)
	}
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	key := randomKey(t)
	plaintext := []byte("ntm encryption corrupted test")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if len(ciphertext) <= headerSize {
		t.Fatalf("ciphertext too short: %d", len(ciphertext))
	}

	// Truncate to simulate corruption (remove ciphertext entirely).
	corrupted := ciphertext[:headerSize]
	_, err = Decrypt(key, corrupted)
	if err == nil {
		t.Fatal("Decrypt: expected error")
	}
	if !IsCorruptedData(err) {
		t.Fatalf("expected corrupted-data error, got %v", err)
	}
}

func TestEncryptInvalidKey(t *testing.T) {
	_, err := Encrypt([]byte("short"), []byte("data"))
	if err == nil {
		t.Fatal("Encrypt: expected error")
	}
	if !IsInvalidKey(err) {
		t.Fatalf("expected invalid-key error, got %v", err)
	}
}

func randomKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return key
}
