package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bulga138/env-vault/utils"
)

func TestP2PProviderBasic(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	p, err := NewP2PSyncProvider()
	if err != nil {
		t.Fatal(err)
	}
	secret := StoredEnv{Ciphertext: "abc", Salt: "def", HMAC: "ghi", TimeStamp: 1}
	if err := p.Save(t.Context(), "repo", secret); err != nil {
		t.Fatal(err)
	}
	all, err := p.LoadAll(t.Context())
	if err != nil || len(all) != 1 {
		t.Fatalf("expected 1, got %d err=%v", len(all), err)
	}
	if err := p.Delete(t.Context(), "repo"); err != nil {
		t.Fatal(err)
	}
	all, _ = p.LoadAll(t.Context())
	if len(all) != 0 {
		t.Fatalf("expected 0, got %d", len(all))
	}
}

func TestP2PRelayHTTP(t *testing.T) {
	srv := httptest.NewServer(newP2PRelayHandler("token"))
	defer srv.Close()

	p, err := NewP2PSyncProvider(srv.URL, "room", "token")
	if err != nil {
		t.Fatal(err)
	}
	secret := StoredEnv{Ciphertext: "abc", Salt: "def", HMAC: "ghi", TimeStamp: 1}
	if err := p.Save(t.Context(), "repo", secret); err != nil {
		t.Fatal(err)
	}
	all, err := p.LoadAll(t.Context())
	if err != nil || len(all) != 1 {
		t.Fatalf("expected 1, got %d err=%v", len(all), err)
	}
	// Relay must reject requests without a token.
	resp, err := http.Get(srv.URL + "/p2p/room")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	_ = utils.GetRepoKey
}
