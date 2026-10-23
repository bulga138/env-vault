//go:build cloud

package main

import (
	"context"
)

// FirestoreSyncProvider adapts the existing Firestore helpers to SyncProvider.
type FirestoreSyncProvider struct{}

func NewFirestoreSyncProvider() (*FirestoreSyncProvider, error) {
	return &FirestoreSyncProvider{}, nil
}

func (f *FirestoreSyncProvider) Name() string { return "Firestore Sync" }

func (f *FirestoreSyncProvider) Save(ctx context.Context, repoName string, secret StoredEnv) error {
	client, err := initFirestoreClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	return saveSecretToFirebase(ctx, client, repoName, secret)
}

func (f *FirestoreSyncProvider) LoadAll(ctx context.Context) (map[string]StoredEnv, error) {
	client, err := initFirestoreClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	out := make(map[string]StoredEnv)
	iter := client.Collection(firestoreCollection()).Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, err
	}
	for _, doc := range docs {
		var secret StoredEnv
		if err := doc.DataTo(&secret); err != nil {
			continue
		}
		out[doc.Ref.ID] = secret
	}
	return out, nil
}

func (f *FirestoreSyncProvider) Delete(ctx context.Context, repoName string) error {
	client, err := initFirestoreClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	return deleteSecretFromFirebase(ctx, client, repoName)
}

func init() {
	RegisterSyncProvider("firestore", func() (SyncProvider, error) {
		return NewFirestoreSyncProvider()
	})
}

