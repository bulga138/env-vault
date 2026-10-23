package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

type User struct {
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type userStore map[string]User

func getUsersFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not get user config directory: %w", err)
	}
	appDir := filepath.Join(configDir, "env-vault")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return "", fmt.Errorf("could not create config directory: %w", err)
	}
	return filepath.Join(appDir, "users.json"), nil
}

func loadUsers() (userStore, error) {
	path, err := getUsersFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(userStore), nil
		}
		return nil, err
	}
	var store userStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store == nil {
		store = make(userStore)
	}
	return store, nil
}

func saveUsers(store userStore) error {
	path, err := getUsersFilePath()
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

func validRole(role string) bool {
	switch role {
	case "admin", "member", "read-only", "readonly":
		return true
	}
	return false
}

func normalizeRole(role string) string {
	if role == "readonly" {
		return "read-only"
	}
	return role
}

func AddUser(name, role string) error {
	if name == "" {
		return fmt.Errorf("user name is required")
	}
	if !validRole(role) {
		return fmt.Errorf("invalid role %q (use admin, member, read-only)", role)
	}
	store, err := loadUsers()
	if err != nil {
		return err
	}
	if _, exists := store[name]; exists {
		return fmt.Errorf("user %q already exists", name)
	}
	store[name] = User{Name: name, Role: normalizeRole(role), CreatedAt: time.Now()}
	return saveUsers(store)
}

func RemoveUser(name string) error {
	store, err := loadUsers()
	if err != nil {
		return err
	}
	if _, exists := store[name]; !exists {
		return fmt.Errorf("user %q not found", name)
	}
	delete(store, name)
	return saveUsers(store)
}

func ListUsers() ([]User, error) {
	store, err := loadUsers()
	if err != nil {
		return nil, err
	}
	users := make([]User, 0, len(store))
	for _, u := range store {
		users = append(users, u)
	}
	return users, nil
}
