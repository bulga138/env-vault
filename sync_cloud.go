//go:build cloud

package main

import (
	"context"
	"fmt"

	"github.com/bulga138/env-vault/utils"
)

func handleSyncSave() {
	utils.Info("☁️  Saving current repo's secret to Firestore...")
	ctx := context.Background()

	client, err := initFirestoreClient(ctx)
	if err != nil {
		utils.Fatal("Error: Could not connect to Firebase: %v", err)
		return
	}
	defer client.Close()

	repoName, err := getCurrentRepoName()
	if err != nil {
		logAudit("sync-save", "", false, fmt.Sprintf("Failed to get repository name: %v", err))
		utils.Fatal("Error getting repo name: %v", err)
		return
	}

	vault, err := loadVault()
	if err != nil {
		logAudit("sync-save", repoName, false, fmt.Sprintf("Error loading local vault: %v", err))
		utils.Fatal("Error loading local vault: %v", err)
		return
	}

	hashedRepoName := utils.GetRepoKey(repoName)
	secret, ok := vault[hashedRepoName]
	if !ok {
		utils.Fatal("Error: No secret found for '%s' in local vault.json. Please run 'save' first.", repoName)
		logAudit("sync-save", repoName, false, "No secret found for repository in local vault.json.")
		return
	}

	if err := saveSecretToFirebase(ctx, client, repoName, secret); err != nil {
		logAudit("sync-save", repoName, false, fmt.Sprintf("Error saving to Firestore: %v", err))
		utils.Fatal("Error saving to Firestore: %v", err)
		return
	}

	logAudit("sync-save", repoName, true, "")
	utils.Info("✅ Secret for '%s' successfully saved to Firestore.", repoName)
}

func handleSyncLoad() {
	utils.Info("☁️  Loading all secrets from Firestore to local vault.json...")
	ctx := context.Background()

	client, err := initFirestoreClient(ctx)
	if err != nil {
		logAudit("sync-load", "", false, fmt.Sprintf("Could not connect to Firebase: %v", err))
		utils.Fatal("Error: Could not connect to Firebase: %v", err)
		return
	}
	defer client.Close()

	if err := loadSecretsFromFirebase(ctx, client); err != nil {
		logAudit("sync-load", "", false, fmt.Sprintf("Error loading from Firestore: %v", err))
		utils.Fatal("Error loading from Firestore: %v", err)
		return
	}
	logAudit("sync-load", "", true, "")
}