package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const e2eRepoRemote = "https://github.com/e2e-user/e2e-repo.git"
const e2eRepoName = "github.com/e2e-user/e2e-repo"

func e2eSetup(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}

	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=e2e", "GIT_AUTHOR_EMAIL=e2e@test",
			"GIT_COMMITTER_NAME=e2e", "GIT_COMMITTER_EMAIL=e2e@test",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init")
	git("remote", "add", "origin", e2eRepoRemote)

	t.Chdir(dir)
	t.Setenv("VAULT_FILE_PATH", filepath.Join(dir, "vault.json"))
	t.Setenv("ENV_VAULT_PASSWORD", "")
	t.Setenv("ENV_VAULT_NEW_PASSWORD", "")
	t.Setenv("BWS_PROJECT_ID", "")
	t.Setenv("ENV_VAULT_PEPPER", "")
	t.Setenv("ENV_VAULT_EXISTING_FILENAME", "")
	// Drop any pepper/BWS leakage from the package-dir .env loaded at init.
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("API_KEY=e2e-value\nDB_URL=postgres://local\n"), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
}

func e2eVaultPath(t *testing.T) string {
	t.Helper()
	p := os.Getenv("VAULT_FILE_PATH")
	if p == "" {
		t.Fatal("VAULT_FILE_PATH not set")
	}
	return p
}

func TestE2E_GetRepoName_RealGit(t *testing.T) {
	e2eSetup(t)
	name, err := getCurrentRepoName()
	if err != nil {
		t.Fatalf("getCurrentRepoName: %v", err)
	}
	if name != e2eRepoName {
		t.Fatalf("repo name = %q, want %q", name, e2eRepoName)
	}
}

func TestE2E_SaveGetRotateVerify(t *testing.T) {
	e2eSetup(t)

	const pass1 = "e2e-master-pass-1!"
	const pass2 = "e2e-master-pass-2!"

	t.Setenv("ENV_VAULT_PASSWORD", pass1)
	handleSave()

	vaultPath := e2eVaultPath(t)
	if _, err := os.Stat(vaultPath); err != nil {
		t.Fatalf("vault.json missing after save: %v", err)
	}

	if _, err := getEnv(e2eRepoName, []byte(pass1)); err != nil {
		t.Fatalf("getEnv after save: %v", err)
	}

	if err := os.Remove(".env"); err != nil {
		t.Fatalf("remove .env: %v", err)
	}
	handleGet()
	data, err := os.ReadFile(".env")
	if err != nil {
		t.Fatalf("read recovered .env: %v", err)
	}
	if !bytes.Contains(data, []byte("API_KEY=e2e-value")) || !bytes.Contains(data, []byte("DB_URL=postgres://local")) {
		t.Fatalf("recovered .env content mismatch:\n%s", data)
	}

	t.Setenv("ENV_VAULT_NEW_PASSWORD", pass2)
	handleRotatePassword()

	if _, err := getEnv(e2eRepoName, []byte(pass1)); err == nil {
		t.Fatal("old password still decrypts after rotate")
	}
	if _, err := getEnv(e2eRepoName, []byte(pass2)); err != nil {
		t.Fatalf("new password fails after rotate: %v", err)
	}

	if err := os.Remove(".env"); err != nil {
		t.Fatalf("remove .env before get: %v", err)
	}
	t.Setenv("ENV_VAULT_PASSWORD", pass2)
	handleGet()
	data, err = os.ReadFile(".env")
	if err != nil {
		t.Fatalf("read .env after rotate get: %v", err)
	}
	if !bytes.Contains(data, []byte("API_KEY=e2e-value")) {
		t.Fatalf("post-rotate .env missing key:\n%s", data)
	}

	// Wrong password must fail at the vault layer.
	if _, err := getEnv(e2eRepoName, []byte("totally-wrong-pass")); err == nil {
		t.Fatal("wrong password decrypted vault")
	}

	t.Setenv("ENV_VAULT_PASSWORD", pass2)
	handleVerify()
}

func TestE2E_GetExistingLocalEnvRecovered(t *testing.T) {
	e2eSetup(t)

	const pass = "e2e-recover-pass-!"
	t.Setenv("ENV_VAULT_PASSWORD", pass)
	handleSave()

	// Local .env still present → get must not overwrite it.
	handleGet()
	if _, err := os.Stat(".env.recovered"); err != nil {
		t.Fatalf("expected .env.recovered: %v", err)
	}
	orig, _ := os.ReadFile(".env")
	if !bytes.Contains(orig, []byte("API_KEY=e2e-value")) {
		t.Fatalf("original .env was modified:\n%s", orig)
	}
	rec, _ := os.ReadFile(".env.recovered")
	if !bytes.Contains(rec, []byte("API_KEY=e2e-value")) {
		t.Fatalf(".env.recovered content mismatch:\n%s", rec)
	}
}

func TestE2E_CLI_Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	bin := filepath.Join(t.TempDir(), "env-vault-e2e")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	work := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = work
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("env-vault %v: %v\n%s", args, err, out)
		}
		return string(out)
	}

	help := run()
	if !strings.Contains(help, "save") || !strings.Contains(help, "ENV_VAULT_PASSWORD") {
		t.Fatalf("help output missing expected sections:\n%s", help)
	}

	gen := run("generate-pass")
	if !strings.Contains(gen, "secure password") && !strings.Contains(gen, "password") {
		t.Fatalf("generate-pass output unexpected:\n%s", gen)
	}

	// Full CLI lifecycle against a temp git repo (non-interactive env passwords).
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init")
	git("remote", "add", "origin", e2eRepoRemote)
	if err := os.WriteFile(filepath.Join(repo, ".env"), []byte("CLI_KEY=cli-value\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cliEnv := append(os.Environ(),
		"VAULT_FILE_PATH="+filepath.Join(repo, "vault.json"),
		"ENV_VAULT_PASSWORD=cli-e2e-pass-1!",
		"ENV_VAULT_NEW_PASSWORD=cli-e2e-pass-2!",
		"BWS_PROJECT_ID=",
		"ENV_VAULT_PEPPER=",
	)
	runCli := func(args ...string) {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = repo
		cmd.Env = cliEnv
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("env-vault %v in repo: %v\n%s", args, err, out)
		}
	}

	runCli("save")
	if _, err := os.Stat(filepath.Join(repo, "vault.json")); err != nil {
		t.Fatalf("CLI save did not create vault: %v", err)
	}

	if err := os.Remove(filepath.Join(repo, ".env")); err != nil {
		t.Fatal(err)
	}
	runCli("get")
	data, err := os.ReadFile(filepath.Join(repo, ".env"))
	if err != nil {
		t.Fatalf("CLI get did not write .env: %v", err)
	}
	if !bytes.Contains(data, []byte("CLI_KEY=cli-value")) {
		t.Fatalf("CLI .env mismatch:\n%s", data)
	}

	runCli("rotate-pass")
	runCli("verify")
}
