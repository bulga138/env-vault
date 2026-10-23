package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bulga138/env-vault/utils"
)

// SyncProvider abstracts cloud/local sync backends.
type SyncProvider interface {
	Name() string
	Save(ctx context.Context, repoName string, secret StoredEnv) error
	LoadAll(ctx context.Context) (map[string]StoredEnv, error)
	Delete(ctx context.Context, repoName string) error
}

var syncProviderFactories = map[string]func() (SyncProvider, error){}

func RegisterSyncProvider(name string, factory func() (SyncProvider, error)) {
	syncProviderFactories[name] = factory
}

func GetSyncProvider() (SyncProvider, error) {
	backend := os.Getenv("SYNC_BACKEND")
	if backend == "" {
		backend = "file"
	}
	factory, ok := syncProviderFactories[backend]
	if !ok {
		return nil, fmt.Errorf("unknown sync backend %q (available: %v)", backend, availableSyncBackends())
	}
	return factory()
}

func availableSyncBackends() []string {
	out := make([]string, 0, len(syncProviderFactories))
	for k := range syncProviderFactories {
		out = append(out, k)
	}
	return out
}

// --- Local file sync provider (no third-party service) ---

type FileSyncProvider struct {
	Dir string
}

func NewFileSyncProvider() (*FileSyncProvider, error) {
	dir := os.Getenv("SYNC_DIR")
	if dir == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(configDir, "env-vault", "sync")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &FileSyncProvider{Dir: dir}, nil
}

func (f *FileSyncProvider) Name() string { return "Local File Sync" }

func (f *FileSyncProvider) Save(ctx context.Context, repoName string, secret StoredEnv) error {
	path := filepath.Join(f.Dir, utils.GetRepoKey(repoName)+".json")
	data, err := json.MarshalIndent(secret, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	defer os.Remove(tmp)
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (f *FileSyncProvider) LoadAll(ctx context.Context) (map[string]StoredEnv, error) {
	out := make(map[string]StoredEnv)
	entries, err := os.ReadDir(f.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(f.Dir, e.Name()))
		if err != nil {
			continue
		}
		var secret StoredEnv
		if err := json.Unmarshal(data, &secret); err != nil {
			continue
		}
		key := e.Name()[:len(e.Name())-len(".json")]
		out[key] = secret
	}
	return out, nil
}

func (f *FileSyncProvider) Delete(ctx context.Context, repoName string) error {
	path := filepath.Join(f.Dir, utils.GetRepoKey(repoName)+".json")
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func init() {
	RegisterSyncProvider("file", func() (SyncProvider, error) {
		return NewFileSyncProvider()
	})
}

// resolveConflict decides how to merge a remote secret into an existing local one.
func resolveConflict(strategy string, existing, remote StoredEnv) (StoredEnv, bool, bool) {
	remoteNewer := remote.TimeStamp > existing.TimeStamp
	localNewer := existing.TimeStamp > remote.TimeStamp
	contentDiffers := remote.Ciphertext != existing.Ciphertext || remote.HMAC != existing.HMAC

	switch strategy {
	case "remote":
		return remote, true, false
	case "local":
		return existing, false, false
	case "skip":
		if contentDiffers {
			return existing, false, true
		}
		return existing, false, false
	default: // newest
		if remoteNewer {
			return remote, true, false
		}
		if localNewer {
			return existing, false, false
		}
		if contentDiffers {
			return existing, false, true
		}
		return existing, false, false
	}
}

func handleSyncSave() {
	utils.Info("☁️  Saving current repo's secret to sync backend...")
	ctx := context.Background()

	provider, err := GetSyncProvider()
	if err != nil {
		utils.Fatal("Error: %v", err)
		return
	}

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

	if err := provider.Save(ctx, repoName, secret); err != nil {
		logAudit("sync-save", repoName, false, fmt.Sprintf("Error saving to %s: %v", provider.Name(), err))
		utils.Fatal("Error saving to %s: %v", provider.Name(), err)
		return
	}

	logAudit("sync-save", repoName, true, "")
	utils.Info("✅ Secret for '%s' successfully saved via %s.", repoName, provider.Name())
}

func handleSyncLoad() {
	utils.Info("☁️  Loading all secrets from sync backend to local vault.json...")
	ctx := context.Background()

	provider, err := GetSyncProvider()
	if err != nil {
		utils.Fatal("Error: %v", err)
		return
	}

	secrets, err := provider.LoadAll(ctx)
	if err != nil {
		logAudit("sync-load", "", false, fmt.Sprintf("Error loading from %s: %v", provider.Name(), err))
		utils.Fatal("Error loading from %s: %v", provider.Name(), err)
		return
	}

	localVault, err := loadVault()
	if err != nil {
		utils.Warn("Could not load local vault, starting empty: %v", err)
		localVault = make(vaultData)
	}

	loadedCount := 0
	conflicts := 0
	for k, v := range secrets {
		existing, ok := localVault[k]
		if !ok {
			localVault[k] = v
			loadedCount++
			continue
		}
		resolved, updated, conflict := resolveConflict(*syncStrategy, existing, v)
		localVault[k] = resolved
		if updated {
			loadedCount++
		}
		if conflict {
			conflicts++
		}
	}
	if conflicts > 0 {
		utils.Warn("Detected %d conflict(s); kept local copy. Use --sync-strategy=remote to overwrite.", conflicts)
	}

	if err := saveVault(localVault); err != nil {
		logAudit("sync-load", "", false, fmt.Sprintf("Error writing local vault: %v", err))
		utils.Fatal("Error writing local vault: %v", err)
		return
	}

	logAudit("sync-load", "", true, "")
	utils.Success("Merged %d secrets from %s into local vault.json.", loadedCount, provider.Name())
}