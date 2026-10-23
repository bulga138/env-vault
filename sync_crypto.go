package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

const syncEncPrefix = "enc:"

func syncDBPassphrase() string {
	return os.Getenv("SYNC_DB_PASSPHRASE")
}

func syncKey() []byte {
	p := syncDBPassphrase()
	if p == "" {
		return nil
	}
	h := sha256.Sum256([]byte(p))
	return h[:]
}

func encryptSyncData(plaintext []byte) (string, error) {
	key := syncKey()
	if key == nil {
		return string(plaintext), nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, plaintext, nil)
	return syncEncPrefix + base64.StdEncoding.EncodeToString(ct), nil
}

func decryptSyncData(stored string) ([]byte, error) {
	if !hasPrefix(stored, syncEncPrefix) {
		return []byte(stored), nil
	}
	key := syncKey()
	if key == nil {
		return nil, fmt.Errorf("encrypted data but SYNC_DB_PASSPHRASE not set")
	}
	raw, err := base64.StdEncoding.DecodeString(stored[len(syncEncPrefix):])
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(raw) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}