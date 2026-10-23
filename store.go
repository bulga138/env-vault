package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bulga138/env-vault/utils"
	"github.com/gofrs/flock"
)

type StoredEnv struct {
	Ciphertext string `json:"ciphertext"`
	Salt       string `json:"salt"`
	HMAC       string `json:"hmac"`
	TimeStamp  int64  `json:"timestamp"`
}

type vaultData map[string]StoredEnv

func saveEnv(repoName string, content []byte, password []byte) error {
	vaultKey := utils.GetRepoKey(repoName)
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("error generating salt: %w", err)
	}

	keyLB, key := deriveKey(password, salt)
	defer destroyLocked(keyLB)
	encryptedContent, err := encrypt(content, key)
	if err != nil {
		return err
	}

	ciphertextB64 := base64.StdEncoding.EncodeToString(encryptedContent)
	saltB64 := base64.StdEncoding.EncodeToString(salt)
	hmacInput := []byte(ciphertextB64 + saltB64)
	hmacValue := computeHMAC(hmacInput, key)
	timeStamp := time.Now().Unix()

	vault, err := loadVault()
	if err != nil {
		return err
	}
	vault[vaultKey] = StoredEnv{
		Ciphertext: ciphertextB64,
		Salt:       saltB64,
		HMAC:       hmacValue,
		TimeStamp:  timeStamp,
	}
	return saveVault(vault)
}

func getEnv(repoName string, password []byte) ([]byte, error) {
	vault, err := loadVault()
	if err != nil {
		return nil, fmt.Errorf("failed to load vault: %w", err)
	}

	vaultKey := utils.GetRepoKey(repoName)
	stored, ok := vault[vaultKey]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrRepoNotFound, repoName)
	}

	salt, err := base64.StdEncoding.DecodeString(stored.Salt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSalt, err)
	}
	encryptedContent, err := base64.StdEncoding.DecodeString(stored.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCiphertext, err)
	}

	keyLB, key := deriveKey(password, salt)
	defer destroyLocked(keyLB)

	hmacInput := []byte(stored.Ciphertext + stored.Salt)
	if !verifyHMAC(hmacInput, key, stored.HMAC) {
		return nil, ErrInvalidHMAC
	}

	plaintext, err := decrypt(encryptedContent, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

func getVaultFilePath() (string, error) {
	path := os.Getenv("VAULT_FILE_PATH")
	if path != "" {
		return path, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not get user config directory: %w", err)
	}

	appConfigDir := filepath.Join(configDir, "env-vault")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", fmt.Errorf("could not create app config directory: %w", err)
	}

	return filepath.Join(appConfigDir, "vault.json"), nil
}

func saveVault(vault vaultData) error {
	vaultFile, err := getVaultFilePath()
	if err != nil {
		return err
	}

	lock := flock.New(vaultFile + ".lock")
	locked, err := lock.TryLock()
	if err != nil || !locked {
		return ErrVaultLocked
	}
	defer lock.Unlock()

	tmpFile := vaultFile + ".tmp"
	defer os.Remove(tmpFile)

	data, err := json.MarshalIndent(vault, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vault data: %w", err)
	}

	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := os.Rename(tmpFile, vaultFile); err != nil {
		return fmt.Errorf("failed to atomic move vault file: %w", err)
	}

	return nil
}

func loadVault() (vaultData, error) {
	vaultFile, err := getVaultFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(vaultFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(vaultData), nil
		}
		return nil, err
	}

	var vault vaultData
	if err := json.Unmarshal(data, &vault); err != nil {
		return nil, err
	}
	return vault, nil
}

func verifyEntryHMAC(stored StoredEnv, password []byte) error {
	salt, err := base64.StdEncoding.DecodeString(stored.Salt)
	if err != nil {
		return fmt.Errorf("invalid salt encoding: %w", err)
	}

	keyLB, key := deriveKey(password, salt)
	defer destroyLocked(keyLB)

	hmacInput := []byte(stored.Ciphertext + stored.Salt)

	if !verifyHMAC(hmacInput, key, stored.HMAC) {
		return ErrInvalidHMAC
	}

	return nil
}

func VerifyAll(getPasswordFunc func(repoName string) ([]byte, error)) (map[string]error, error) {
	vault, err := loadVault()
	if err != nil {
		return nil, fmt.Errorf("failed to load vault: %w", err)
	}

	results := make(map[string]error)

	for hashedKey, stored := range vault {
		password, err := getPasswordFunc(hashedKey)
		if err != nil {
			results[hashedKey] = fmt.Errorf("failed to get password: %w", err)
			continue
		}

		if err := verifyEntryHMAC(stored, password); err != nil {
			results[hashedKey] = fmt.Errorf("HMAC verification failed: %w", err)
		} else {
			pt, err := getEnvByStoredEnv(stored, password)
			secureZero(pt)
			if err != nil {
				results[hashedKey] = fmt.Errorf("HMAC OK, but decryption failed (possible tampering or key issue): %w", err)
			} else {
				results[hashedKey] = nil
			}
		}
		secureZero(password)
	}
	return results, nil
}

func getEnvByStoredEnv(stored StoredEnv, password []byte) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(stored.Salt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSalt, err)
	}
	encryptedContent, err := base64.StdEncoding.DecodeString(stored.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCiphertext, err)
	}

	keyLB, key := deriveKey(password, salt)
	defer destroyLocked(keyLB)
	plaintext, err := decrypt(encryptedContent, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}
