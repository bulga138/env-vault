package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bulga138/env-vault/providers"
	"github.com/bulga138/env-vault/utils"
)

// Integration tests hit real Firestore / Bitwarden using credentials from the
// environment (or the package-dir .env loaded at init). They skip when the
// backend is not configured or unreachable. Run with:
//
//	go test ./... -run Integration -count=1 -v
//
// Firestore writes go to FIRESTORE_COLLECTION (default: env-vault-e2e when
// set by the test). Bitwarden secrets use unique keys and are deleted after.

func integrationRepoName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("github.com/e2e-int/%s-%d", t.Name(), time.Now().UnixNano())
}

func TestIntegration_Firestore_SaveLoadDelete(t *testing.T) {
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("GOOGLE_APPLICATION_CREDENTIALS not set")
	}
	// Isolate from the production `secrets` collection.
	t.Setenv("FIRESTORE_COLLECTION", "env-vault-e2e")
	t.Setenv("VAULT_FILE_PATH", filepath.Join(t.TempDir(), "vault.json"))
	t.Setenv("ENV_VAULT_PEPPER", "")

	ctx := context.Background()
	client, err := initFirestoreClient(ctx)
	if err != nil {
		t.Skipf("Firestore unavailable: %v", err)
	}
	defer client.Close()

	repoName := integrationRepoName(t)
	defer func() {
		if derr := deleteSecretFromFirebase(ctx, client, repoName); derr != nil {
			t.Logf("cleanup delete: %v", derr)
		}
	}()

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	password := []byte("integration-master-pass-1!")
	key := deriveKey(password, salt)
	content := []byte("INT_KEY=integration-value\n")
	ct, err := encrypt(content, key)
	if err != nil {
		t.Fatal(err)
	}
	ctB64 := base64.StdEncoding.EncodeToString(ct)
	saltB64 := base64.StdEncoding.EncodeToString(salt)
	secret := StoredEnv{
		Ciphertext: ctB64,
		Salt:       saltB64,
		HMAC:       computeHMAC([]byte(ctB64+saltB64), key),
		TimeStamp:  time.Now().Unix(),
	}

	if err := saveSecretToFirebase(ctx, client, repoName, secret); err != nil {
		t.Fatalf("saveSecretToFirebase: %v", err)
	}

	// Remote doc exists.
	docID := utils.GetRepoKey(repoName)
	snap, err := client.Collection(firestoreCollection()).Doc(docID).Get(ctx)
	if err != nil {
		t.Fatalf("get remote doc: %v", err)
	}
	var remote StoredEnv
	if err := snap.DataTo(&remote); err != nil {
		t.Fatalf("parse remote doc: %v", err)
	}
	if remote.Ciphertext != secret.Ciphertext {
		t.Fatal("remote ciphertext mismatch")
	}

	// Merge into local vault and decrypt through the normal path.
	if err := loadSecretsFromFirebase(ctx, client); err != nil {
		t.Fatalf("loadSecretsFromFirebase: %v", err)
	}
	got, err := getEnv(repoName, password)
	if err != nil {
		t.Fatalf("getEnv after sync-load: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("decrypted mismatch: got %q want %q", got, content)
	}

	// Local-only entries must survive a merge.
	localOnly := integrationRepoName(t) + "-local"
	if err := saveEnv(localOnly, []byte("LOCAL=1\n"), password); err != nil {
		t.Fatalf("saveEnv local-only: %v", err)
	}
	if err := loadSecretsFromFirebase(ctx, client); err != nil {
		t.Fatalf("loadSecretsFromFirebase #2: %v", err)
	}
	if _, err := getEnv(localOnly, password); err != nil {
		t.Fatalf("local-only entry lost after merge: %v", err)
	}

	if err := deleteSecretFromFirebase(ctx, client, repoName); err != nil {
		t.Fatalf("deleteSecretFromFirebase: %v", err)
	}
	if _, err := client.Collection(firestoreCollection()).Doc(docID).Get(ctx); err == nil {
		t.Fatal("doc still exists after delete")
	}
}

func TestIntegration_Bitwarden_SetGetRotateDelete(t *testing.T) {
	if _, err := exec.LookPath("bws"); err != nil {
		t.Skip("bws CLI not in PATH")
	}
	if os.Getenv("BWS_PROJECT_ID") == "" {
		t.Skip("BWS_PROJECT_ID not set")
	}

	provider, err := providers.NewBitwardenProvider()
	if err != nil {
		t.Skipf("Bitwarden provider: %v", err)
	}

	// Probe: list must work (auth + project access).
	repoName := integrationRepoName(t)
	hashed := utils.GetRepoKey(repoName)
	_ = hashed

	// Ensure a clean slate, then exercise create → get → edit → get → delete.
	_ = provider.Delete(repoName)

	const pass1 = "bws-int-pass-1!"
	const pass2 = "bws-int-pass-2!"

	if err := provider.Set(repoName, []byte(pass1)); err != nil {
		t.Fatalf("Set (create): %v", err)
	}
	defer func() {
		if derr := provider.Delete(repoName); derr != nil {
			t.Logf("cleanup delete: %v", derr)
		}
	}()

	got, err := provider.Get(repoName)
	if err != nil {
		t.Fatalf("Get after create: %v", err)
	}
	if string(got) != pass1 {
		t.Fatalf("Get value = %q, want %q", got, pass1)
	}

	// Second Set must take the edit path, not fail on duplicate create.
	if err := provider.Set(repoName, []byte(pass2)); err != nil {
		t.Fatalf("Set (edit): %v", err)
	}
	got, err = provider.Get(repoName)
	if err != nil {
		t.Fatalf("Get after edit: %v", err)
	}
	if string(got) != pass2 {
		t.Fatalf("Get after edit = %q, want %q", got, pass2)
	}

	if err := provider.Delete(repoName); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := provider.Get(repoName); err == nil {
		t.Fatal("Get succeeded after delete")
	}
}

func TestIntegration_CLI_BitwardenSaveGet(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	if _, err := exec.LookPath("bws"); err != nil {
		t.Skip("bws CLI not in PATH")
	}
	if os.Getenv("BWS_PROJECT_ID") == "" {
		t.Skip("BWS_PROJECT_ID not set")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}

	bin := filepath.Join(t.TempDir(), "env-vault-int")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
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
	// handleSave uses git remote, so set origin to a synthetic host/path.
	git("remote", "add", "origin", "https://github.com/e2e-int/cli-bws.git")

	if err := os.WriteFile(filepath.Join(repo, ".env"), []byte("BW_KEY=bw-cli-value\n"), 0600); err != nil {
		t.Fatal(err)
	}

	detect := exec.Command("git", "-C", repo, "remote", "get-url", "origin")
	urlOut, err := detect.Output()
	if err != nil {
		t.Fatal(err)
	}
	detectedRepo, err := parseGitRemoteURL(string(urlOut))
	if err != nil {
		t.Fatalf("parse remote: %v", err)
	}
	defer func() {
		p, perr := providers.NewBitwardenProvider()
		if perr != nil {
			return
		}
		if derr := p.Delete(utils.GetRepoKey(detectedRepo)); derr != nil {
			t.Logf("cleanup delete: %v", derr)
		}
	}()

	cliEnv := append(os.Environ(),
		"VAULT_FILE_PATH="+filepath.Join(repo, "vault.json"),
		"BWS_PROJECT_ID="+os.Getenv("BWS_PROJECT_ID"),
		"BWS_ACCESS_TOKEN="+os.Getenv("BWS_ACCESS_TOKEN"),
		"ENV_VAULT_PEPPER=",
		"FIRESTORE_COLLECTION=",
	)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = repo
		cmd.Env = cliEnv
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("env-vault %v: %v\n%s", args, err, out)
		}
	}

	// Seed a known master password in Bitwarden, then exercise --use-bitwarden
	// save / get / rotate against the real bws CLI.
	// CLI handlers pass GetRepoKey(repo) into the provider (which hashes again).
	provider, err := providers.NewBitwardenProvider()
	if err != nil {
		t.Skipf("provider: %v", err)
	}
	if err := provider.Set(utils.GetRepoKey(detectedRepo), []byte("cli-bws-master-1!")); err != nil {
		t.Fatalf("seed BW secret: %v", err)
	}

	run("--use-bitwarden", "save")
	if _, err := os.Stat(filepath.Join(repo, "vault.json")); err != nil {
		t.Fatalf("vault not written: %v", err)
	}

	if err := os.Remove(filepath.Join(repo, ".env")); err != nil {
		t.Fatal(err)
	}
	run("--use-bitwarden", "get")
	data, err := os.ReadFile(filepath.Join(repo, ".env"))
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if !bytes.Contains(data, []byte("BW_KEY=bw-cli-value")) {
		t.Fatalf("CLI BW .env mismatch:\n%s", data)
	}

	cliEnv = append(cliEnv, "ENV_VAULT_NEW_PASSWORD=cli-bws-master-2!")
	run("--use-bitwarden", "rotate-pass")

	got, err := provider.Get(utils.GetRepoKey(detectedRepo))
	if err != nil {
		t.Fatalf("BW get after rotate: %v", err)
	}
	if string(got) != "cli-bws-master-2!" {
		t.Fatalf("BW value after rotate = %q", got)
	}
}
