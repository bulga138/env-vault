// filename: bitwarden.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// BwsSecretValue defines the structure for the 'get' command JSON output.
type BwsSecretValue struct {
	Value string `json:"value"`
}

// BwsSecretListItem defines the structure for an item in the 'list' command output.
type BwsSecretListItem struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

// getMasterPasswordFromSecretsManager fetches a secret by its key (repo name) within a project.
func getMasterPasswordFromSecretsManager(repoName string) ([]byte, error) {
	fmt.Printf("🔑 Looking for secret with key '%s' in BitWarden...\n", repoName)

	// Get Project ID from environment.
	projectID := os.Getenv("BWS_PROJECT_ID")
	if projectID == "" {
		return nil, fmt.Errorf("BWS_PROJECT_ID environment variable must be set")
	}

	// Step 1: List all secrets in the project to find the ID.
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

	// Step 2: Find our secret's ID in the list.
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

	// Step 3: Get the secret's value using its ID.
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
