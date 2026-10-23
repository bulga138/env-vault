package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bulga138/env-vault/utils"
	"github.com/gofrs/flock"
)

type aclEntry struct {
	User       string    `json:"user"`
	RepoKey    string    `json:"repo_key"`
	Permission string    `json:"permission"` // read, write, admin
	GrantedAt  time.Time `json:"granted_at"`
}

type aclStore map[string]map[string]string // user -> repoKey -> permission

func getACLFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not get user config directory: %w", err)
	}
	appDir := filepath.Join(configDir, "env-vault")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return "", fmt.Errorf("could not create config directory: %w", err)
	}
	return filepath.Join(appDir, "acl.json"), nil
}

func loadACL() (aclStore, error) {
	path, err := getACLFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(aclStore), nil
		}
		return nil, err
	}
	var store aclStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store == nil {
		store = make(aclStore)
	}
	return store, nil
}

func saveACL(store aclStore) error {
	path, err := getACLFilePath()
	if err != nil {
		return err
	}
	lock := flock.New(path + ".lock")
	locked, err := lock.TryLock()
	if err != nil || !locked {
		return ErrVaultLocked
	}
	defer lock.Unlock()

	data, err := json.MarshalIndent(store, "", "  ")
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

func currentUser() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	if u := os.Getenv("USERNAME"); u != "" {
		return u
	}
	return "unknown"
}

func hasACL(repoName string) bool {
	store, err := loadACL()
	if err != nil {
		return false
	}
	repoKey := utils.GetRepoKey(repoName)
	for _, perms := range store {
		if _, ok := perms[repoKey]; ok {
			return true
		}
	}
	return false
}

func checkPermission(user, repoName, perm string) bool {
	store, err := loadACL()
	if err != nil {
		return false
	}
	repoKey := utils.GetRepoKey(repoName)
	userPerms, ok := store[user]
	if !ok {
		return false
	}
	p, ok := userPerms[repoKey]
	if !ok {
		return false
	}
	// admin implies write+read
	if p == "admin" {
		return true
	}
	if perm == "read" && (p == "read" || p == "write") {
		return true
	}
	if perm == "write" && p == "write" {
		return true
	}
	return false
}

func grantACL(user, repoName, perm string) error {
	if user == "" {
		return fmt.Errorf("user is required")
	}
	if perm != "read" && perm != "write" && perm != "admin" {
		return fmt.Errorf("invalid permission %q (use read, write, admin)", perm)
	}
	store, err := loadACL()
	if err != nil {
		return err
	}
	if _, ok := store[user]; !ok {
		store[user] = make(map[string]string)
	}
	store[user][utils.GetRepoKey(repoName)] = perm
	return saveACL(store)
}

func revokeACL(user, repoName string) error {
	store, err := loadACL()
	if err != nil {
		return err
	}
	userPerms, ok := store[user]
	if !ok {
		return fmt.Errorf("no ACL entries for user %q", user)
	}
	repoKey := utils.GetRepoKey(repoName)
	if _, ok := userPerms[repoKey]; !ok {
		return fmt.Errorf("no ACL entry for user %q on this repository", user)
	}
	delete(userPerms, repoKey)
	if len(userPerms) == 0 {
		delete(store, user)
	}
	return saveACL(store)
}

func listACL() ([]aclEntry, error) {
	store, err := loadACL()
	if err != nil {
		return nil, err
	}
	var out []aclEntry
	for user, perms := range store {
		for repoKey, perm := range perms {
			out = append(out, aclEntry{User: user, RepoKey: repoKey, Permission: perm})
		}
	}
	return out, nil
}
