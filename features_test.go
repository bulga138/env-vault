package main

import (
	"path/filepath"
	"testing"

	"github.com/bulga138/env-vault/utils"
)

func setTempConfigDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	return tmp
}

func TestUserStore(t *testing.T) {
	setTempConfigDir(t)

	if err := AddUser("alice", "admin"); err != nil {
		t.Fatal(err)
	}
	if err := AddUser("alice", "member"); err == nil {
		t.Fatal("expected duplicate error")
	}
	if err := AddUser("bob", "invalid"); err == nil {
		t.Fatal("expected invalid role error")
	}
	if err := AddUser("bob", "member"); err != nil {
		t.Fatal(err)
	}
	users, err := ListUsers()
	if err != nil || len(users) != 2 {
		t.Fatalf("expected 2 users, got %d err=%v", len(users), err)
	}
	if err := RemoveUser("alice"); err != nil {
		t.Fatal(err)
	}
	if err := RemoveUser("alice"); err == nil {
		t.Fatal("expected not found error")
	}
	users, _ = ListUsers()
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
}

func TestACL(t *testing.T) {
	setTempConfigDir(t)

	repo := "example.com/user/repo"
	if err := grantACL("alice", repo, "read"); err != nil {
		t.Fatal(err)
	}
	if !hasACL(repo) {
		t.Fatal("expected hasACL true")
	}
	if !checkPermission("alice", repo, "read") {
		t.Fatal("expected read permission")
	}
	if checkPermission("alice", repo, "write") {
		t.Fatal("read should not imply write")
	}
	if err := grantACL("alice", repo, "admin"); err != nil {
		t.Fatal(err)
	}
	if !checkPermission("alice", repo, "write") {
		t.Fatal("admin should imply write")
	}
	if err := revokeACL("alice", repo); err != nil {
		t.Fatal(err)
	}
	if hasACL(repo) {
		t.Fatal("expected hasACL false after revoke")
	}
	if err := revokeACL("alice", repo); err == nil {
		t.Fatal("expected revoke error")
	}
	_ = utils.GetRepoKey
}

func TestFileSyncProvider(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SYNC_DIR", tmp)
	p, err := NewFileSyncProvider()
	if err != nil {
		t.Fatal(err)
	}
	secret := StoredEnv{Ciphertext: "abc", Salt: "def", HMAC: "ghi", TimeStamp: 1}
	if err := p.Save(t.Context(), "repo", secret); err != nil {
		t.Fatal(err)
	}
	all, err := p.LoadAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1, got %d", len(all))
	}
	if err := p.Delete(t.Context(), "repo"); err != nil {
		t.Fatal(err)
	}
	all, _ = p.LoadAll(t.Context())
	if len(all) != 0 {
		t.Fatalf("expected 0, got %d", len(all))
	}
}

func TestSQLiteSyncProvider(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "sync.db")
	t.Setenv("SYNC_SQLITE_PATH", dbPath)
	p, err := NewSQLiteSyncProvider()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	secret := StoredEnv{Ciphertext: "abc", Salt: "def", HMAC: "ghi", TimeStamp: 1}
	if err := p.Save(t.Context(), "repo", secret); err != nil {
		t.Fatal(err)
	}
	all, err := p.LoadAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1, got %d", len(all))
	}
	if err := p.Delete(t.Context(), "repo"); err != nil {
		t.Fatal(err)
	}
	all, _ = p.LoadAll(t.Context())
	if len(all) != 0 {
		t.Fatalf("expected 0, got %d", len(all))
	}
}