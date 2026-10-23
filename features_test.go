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

func TestResolveConflict(t *testing.T) {
	local := StoredEnv{Ciphertext: "local", TimeStamp: 5}
	remote := StoredEnv{Ciphertext: "remote", TimeStamp: 10}

	r, upd, conf := resolveConflict("newest", local, remote)
	if r.Ciphertext != "remote" || !upd || conf {
		t.Fatalf("newest remote-newer: got %v %v %v", r.Ciphertext, upd, conf)
	}
	r, upd, conf = resolveConflict("newest", remote, local)
	if r.Ciphertext != "remote" || upd || conf {
		t.Fatalf("newest local-newer: got %v %v %v", r.Ciphertext, upd, conf)
	}
	eq := StoredEnv{Ciphertext: "x", TimeStamp: 5}
	r, upd, conf = resolveConflict("newest", local, eq)
	if r.Ciphertext != "local" || upd || !conf {
		t.Fatalf("newest conflict: got %v %v %v", r.Ciphertext, upd, conf)
	}
	r, upd, conf = resolveConflict("remote", local, remote)
	if r.Ciphertext != "remote" || !upd || conf {
		t.Fatalf("remote: got %v %v %v", r.Ciphertext, upd, conf)
	}
	r, upd, conf = resolveConflict("local", local, remote)
	if r.Ciphertext != "local" || upd || conf {
		t.Fatalf("local: got %v %v %v", r.Ciphertext, upd, conf)
	}
	r, upd, conf = resolveConflict("skip", local, remote)
	if r.Ciphertext != "local" || upd || !conf {
		t.Fatalf("skip diff: got %v %v %v", r.Ciphertext, upd, conf)
	}
	same := StoredEnv{Ciphertext: "local", TimeStamp: 10}
	r, upd, conf = resolveConflict("skip", local, same)
	if r.Ciphertext != "local" || upd || conf {
		t.Fatalf("skip same: got %v %v %v", r.Ciphertext, upd, conf)
	}
}

func TestAuditLogReadWrite(t *testing.T) {
	setTempConfigDir(t)
	logAudit("test-action", "repo", true, "")
	logAudit("test-action", "repo", false, "boom")
	entries, err := readAuditEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Action != "test-action" || !entries[0].Success {
		t.Fatal("first entry mismatch")
	}
	if entries[1].Success || entries[1].ErrorMessage != "boom" {
		t.Fatal("second entry mismatch")
	}
}