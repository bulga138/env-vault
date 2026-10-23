package main

import (
	"bytes"
	"crypto/rand"
	"path/filepath"
	"testing"
)

func TestCryptoRoundtrip(t *testing.T) {
	password := []byte("test-password-123!")
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		t.Fatalf("salt: %v", err)
	}

	_, key := deriveKey(password, salt)
	plaintext := []byte("FOO=bar\nBAZ=qux\n")

	ct, err := encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if bytes.Equal(ct, plaintext) {
		t.Fatal("ciphertext equals plaintext")
	}

	hmacInput := append(append([]byte{}, ct...), salt...)
	hmacValue := computeHMAC(hmacInput, key)
	if !verifyHMAC(hmacInput, key, hmacValue) {
		t.Fatal("HMAC did not verify")
	}
	if verifyHMAC(hmacInput, []byte("wrong-key-32-bytes-long!!!!!!"), hmacValue) {
		t.Fatal("HMAC verified with wrong key")
	}

	pt, err := decrypt(ct, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Fatalf("roundtrip mismatch: got %q want %q", pt, plaintext)
	}
}

func TestVaultSaveGetRoundtrip(t *testing.T) {
	t.Setenv("VAULT_FILE_PATH", filepath.Join(t.TempDir(), "vault.json"))

	password := []byte("vault-master-pass-1!")
	content := []byte("API_KEY=abc\nSECRET=xyz\n")

	if err := saveEnv("github.com/user/repo", content, password); err != nil {
		t.Fatalf("saveEnv: %v", err)
	}

	got, err := getEnv("github.com/user/repo", password)
	if err != nil {
		t.Fatalf("getEnv: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("vault roundtrip mismatch: got %q want %q", got, content)
	}

	if _, err := getEnv("github.com/user/other", password); err == nil {
		t.Fatal("expected ErrRepoNotFound for unknown repo")
	}

	if _, err := getEnv("github.com/user/repo", []byte("wrong-password")); err == nil {
		t.Fatal("expected error for wrong password")
	}
}
