package main

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"

	"github.com/bulga138/env-vault/utils"
)

func initFirestoreClient(ctx context.Context) (*firestore.Client, error) {
	credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credPath == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}

	opt := option.WithCredentialsFile(credPath)
 firebaseProjectId := os.Getenv("FIREBASE_PROJECT_ID")
	conf := &firebase.Config{
		ProjectID: firebaseProjectId,
	}

	app, err := firebase.NewApp(ctx, conf, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing Firebase app: %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting Firestore client: %w", err)
	}

	return client, nil
}

func saveSecretToFirebase(ctx context.Context, client *firestore.Client, repoName string, secret StoredEnv) error {
	docID := utils.GetRepoKey(repoName)
	_, err := client.Collection("secrets").Doc(docID).Set(ctx, secret)
	if err != nil {
		return fmt.Errorf("failed to save secret for '%s' to Firestore: %w", repoName, err)
	}
	return nil
}

func loadSecretsFromFirebase(ctx context.Context, client *firestore.Client) error {
	localVault := make(vaultData)

	iter := client.Collection("secrets").Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return fmt.Errorf("failed to fetch documents from Firestore: %w", err)
	}

	if len(docs) == 0 {
		utils.Info("No secrets found in Firestore.")
		return nil
	}

	loadedCount := 0
	for _, doc := range docs {
		var secret StoredEnv
		if err := doc.DataTo(&secret); err != nil {
			utils.Warn("Warning: could not parse secret for doc ID '%s', skipping: %v\n", doc.Ref.ID, err)
			continue
		}
		localVault[doc.Ref.ID] = secret
		loadedCount++
	}

	if err := saveVault(localVault); err != nil {
		return fmt.Errorf("failed to write secrets to local vault.json: %w", err)
	}

	utils.Success("Loaded %d secrets from Firestore into vault.json.", loadedCount)
	return nil
}
