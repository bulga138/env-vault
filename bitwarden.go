package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type BwsSecretValue struct {
	Value string `json:"value"`
}

type BwsSecretListItem struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

func getMasterPasswordFromSecretsManager(repoName string) ([]byte, error) {
	fmt.Printf("Looking for secret with key '%s' in BitWarden...\n", repoName)

	projectID, err := getOrPromptEnvVar("BWS_PROJECT_ID", "Enter your BitWarden Project ID:", false)
	if err != nil {
		return nil, err
	}

	listCmd := exec.Command("bws", "secret", "list", "--output", "json", projectID)
	var listOut, listStderr bytes.Buffer
	listCmd.Stdout = &listOut
	listCmd.Stderr = &listStderr

	if err := listCmd.Run(); err != nil {
		return nil, fmt.Errorf("bws CLI failed during 'list': %s", listStderr.String())
	}

	var secretList []BwsSecretListItem
	if err := json.Unmarshal(listOut.Bytes(), &secretList); err != nil {
		return nil, fmt.Errorf("failed to parse secret list from bws CLI: %w", err)
	}

	var secretID string
	for _, secret := range secretList {
		if secret.Key == repoName {
			secretID = secret.ID
			break
		}
	}

	if secretID == "" {
		return nil, fmt.Errorf("secret with key '%s' not found in project '%s'", repoName, projectID)
	}

	fmt.Println("   Found secret ID:", secretID)
	fmt.Println("   Fetching secret value...")

	getCmd := exec.Command("bws", "secret", "get", "--output", "json", secretID)
	var getOut, getStderr bytes.Buffer
	getCmd.Stdout = &getOut
	getCmd.Stderr = &getStderr

	if err := getCmd.Run(); err != nil {
		return nil, fmt.Errorf("bws CLI failed during 'get': %s", getStderr.String())
	}

	var secretValue BwsSecretValue
	if err := json.Unmarshal(getOut.Bytes(), &secretValue); err != nil {
		return nil, fmt.Errorf("failed to parse secret value from bws CLI: %w", err)
	}

	fmt.Println("✅ Master password successfully fetched and decrypted.")
	return []byte(secretValue.Value), nil
}

func saveMasterPasswordToBitWarden(repoName string, password []byte) error {
	projectID := os.Getenv("BWS_PROJECT_ID")
	if projectID == "" {
		return fmt.Errorf("BWS_PROJECT_ID environment variable must be set to save a new secret")
	}

	cmd := exec.Command("bws", "secret", "create", repoName, string(password), projectID)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to save secret to BitWarden: %s", stderr.String())
	}

	return nil
}
