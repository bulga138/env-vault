package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

type StoredEnv struct {
	Ciphertext string `json:"ciphertext"`
	Salt       string `json:"salt"`
}

type vaultData map[string]StoredEnv

func saveEnv(repoName string, content []byte, password []byte) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("error generating salt: %w", err)
	}

	key := deriveKey(password, salt)

	encryptedContent, err := encrypt(content, key)
	if err != nil {
		return err
	}

	vault, err := loadVault()
	if err != nil {
		return err
	}
	vault[repoName] = StoredEnv{
		Ciphertext: base64.StdEncoding.EncodeToString(encryptedContent),
		Salt:       base64.StdEncoding.EncodeToString(salt),
	}
	return saveVault(vault)
}

func getEnv(repoName string, password []byte) ([]byte, error) {
	vault, err := loadVault()
	if err != nil {
		return nil, err
	}

	stored, ok := vault[repoName]
	if !ok {
		return nil, fmt.Errorf("no env found for '%s'", repoName)
	}

	salt, err := base64.StdEncoding.DecodeString(stored.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid salt: %w", err)
	}
	encryptedContent, err := base64.StdEncoding.DecodeString(stored.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext: %w", err)
	}

	key := deriveKey(password, salt)

	return decrypt(encryptedContent, key)
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
		return fmt.Errorf("could not obtain lock on vault file, please try again")
	}
	defer lock.Unlock()

	data, err := json.MarshalIndent(vault, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vaultFile, data, 0600)
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
