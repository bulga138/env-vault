package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gofrs/flock"
)

const vaultFile = "vault.json"

type StoredEnv struct {
	Ciphertext string `json:"ciphertext"`
	Salt       string `json:"salt"`
}

type vaultData map[string]StoredEnv

func saveEnv(repoName string, content []byte, password []byte) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("error al generar la sal: %w", err)
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
		return nil, fmt.Errorf("no se encontró ningún .env para el repositorio '%s'", repoName)
	}

	salt, err := base64.StdEncoding.DecodeString(stored.Salt)
	if err != nil {
		return nil, fmt.Errorf("sal corrupta: %w", err)
	}
	encryptedContent, err := base64.StdEncoding.DecodeString(stored.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("contenido cifrado corrupto: %w", err)
	}

	key := deriveKey(password, salt)

	return decrypt(encryptedContent, key)
}

func saveVault(vault vaultData) error {
	vaultFile := getVaultFilePath()
	lock := flock.New(vaultFile + ".lock")
	locked, err := lock.TryLock()
	if err != nil || !locked {
		return fmt.Errorf("no se pudo obtener el bloqueo del archivo del almacén, inténtalo de nuevo")
	}
	defer lock.Unlock()

	data, err := json.MarshalIndent(vault, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vaultFile, data, 0600)
}

func loadVault() (vaultData, error) {
	vaultFile := getVaultFilePath()
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

func getVaultFilePath() string {
	path := os.Getenv("VAULT_FILE_PATH")
	if path == "" {
		return "vault.json"
	}
	return path
}
