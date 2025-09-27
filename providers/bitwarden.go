package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/bulga138/env-vault/utils"
)

type BwsSecretValue struct {
	Value string `json:"value"`
}

type BwsSecretListItem struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

type BitwardenProvider struct {
	projectID string
}

func NewBitwardenProvider() (*BitwardenProvider, error) {
	projectID := os.Getenv("BWS_PROJECT_ID")
	if projectID == "" {
		return nil, fmt.Errorf("BWS_PROJECT_ID not set")
	}
	return &BitwardenProvider{projectID: projectID}, nil
}

func (b *BitwardenProvider) Name() string {
	return "Bitwarden Secrets Manager"
}
func (b *BitwardenProvider) Get(key string) ([]byte, error) {
	namespacedKey := utils.GetSecretKey(key)
	utils.Debug("Looking for secret with key '%s' in Bitwarden...\n", namespacedKey)

	listCmd := exec.Command("bws", "secret", "list", "--output", "json", b.projectID)
	var listOut, listStderr bytes.Buffer
	listCmd.Stdout = &listOut
	listCmd.Stderr = &listStderr

	if err := listCmd.Run(); err != nil {
		return nil, fmt.Errorf("bws CLI failed during 'list': %s", listStderr.String())
	}

	var secretList []BwsSecretListItem
	if err := json.Unmarshal(listOut.Bytes(), &secretList); err != nil {
		return nil, fmt.Errorf("failed to parse secret list: %w", err)
	}

	var secretID string
	for _, secret := range secretList {
		if secret.Key == namespacedKey {
			secretID = secret.ID
			break
		}
	}

	if secretID == "" {
		return nil, fmt.Errorf("secret with key '%s' not found", namespacedKey)
	}

	utils.Debug("Found secret ID: %s", secretID)

	getCmd := exec.Command("bws", "secret", "get", "--output", "json", secretID)
	var getOut, getStderr bytes.Buffer
	getCmd.Stdout = &getOut
	getCmd.Stderr = &getStderr

	if err := getCmd.Run(); err != nil {
		return nil, fmt.Errorf("bws CLI failed during 'get': %s", getStderr.String())
	}

	var secretValue BwsSecretValue
	if err := json.Unmarshal(getOut.Bytes(), &secretValue); err != nil {
		return nil, fmt.Errorf("failed to parse secret value: %w", err)
	}

	utils.Debug("Master password successfully fetched and decrypted.")
	return []byte(secretValue.Value), nil
}

func (b *BitwardenProvider) Set(key string, value []byte) error {
	namespacedKey := utils.GetSecretKey(key)
	cmd := exec.Command("bws", "secret", "create", namespacedKey, string(value), b.projectID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to save secret to Bitwarden: %s", stderr.String())
	}
	return nil
}

// func (b *BitwardenProvider) namespacedKey(key string) string {
// 	pepper := os.Getenv("ENV_VAULT_PEPPER")
// 	hasher := sha256.New()
// 	hasher.Write([]byte(key))
// 	if pepper != "" {
// 		hasher.Write([]byte(pepper))
// 	}
// 	hash := hasher.Sum(nil)
// 	keyHash := base64.URLEncoding.EncodeToString(hash)[:16]
// 	return fmt.Sprintf("env-vault-%s", keyHash)
// }
