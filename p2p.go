package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/bulga138/env-vault/utils"
)

// P2PSyncProvider syncs encrypted vault entries through a relay/rendezvous
// server (or an in-process store when no relay URL is configured).
type P2PSyncProvider struct {
	RelayURL string
	Room     string
	Token    string
	Client   *http.Client
}

// NewP2PSyncProvider builds a provider. Arguments (all optional) are:
// relay URL, room/code, auth token. Missing values fall back to env vars.
func NewP2PSyncProvider(args ...string) (*P2PSyncProvider, error) {
	relay := strings.TrimSpace(os.Getenv("P2P_RELAY_URL"))
	if relay == "" {
		relay = strings.TrimSpace(os.Getenv("P2P_URL"))
	}
	if relay == "" {
		relay = strings.TrimSpace(os.Getenv("P2P_PEER_URL"))
	}
	if relay == "" {
		relay = strings.TrimSpace(os.Getenv("RELAY_URL"))
	}
	room := strings.TrimSpace(os.Getenv("P2P_ROOM"))
	if room == "" {
		room = strings.TrimSpace(os.Getenv("P2P_CODE"))
	}
	if room == "" {
		room = strings.TrimSpace(os.Getenv("ROOM"))
	}
	if room == "" {
		room = "default"
	}
	token := strings.TrimSpace(os.Getenv("P2P_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("P2P_SECRET"))
	}

	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		relay = strings.TrimSpace(args[0])
	}
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		room = strings.TrimSpace(args[1])
	}
	if len(args) > 2 && strings.TrimSpace(args[2]) != "" {
		token = strings.TrimSpace(args[2])
	}
	if room == "" {
		room = "default"
	}

	return &P2PSyncProvider{
		RelayURL: strings.TrimRight(relay, "/"),
		Room:     room,
		Token:    token,
		Client:   http.DefaultClient,
	}, nil
}

func NewP2PProvider(args ...string) (*P2PSyncProvider, error)   { return NewP2PSyncProvider(args...) }
func NewPeerProvider(args ...string) (*P2PSyncProvider, error)  { return NewP2PSyncProvider(args...) }
func NewRelayProvider(args ...string) (*P2PSyncProvider, error) { return NewP2PSyncProvider(args...) }

func (p *P2PSyncProvider) Name() string { return "P2P Relay Sync" }

func (p *P2PSyncProvider) Close() error { return nil }

type p2pRoomData struct {
	mu      sync.RWMutex
	secrets map[string]StoredEnv
	peers   []string
}

type p2pStore struct {
	mu    sync.RWMutex
	rooms map[string]*p2pRoomData
}

var globalP2PStore = &p2pStore{rooms: make(map[string]*p2pRoomData)}

func (s *p2pStore) room(name string) *p2pRoomData {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.rooms[name]; ok {
		return r
	}
	r := &p2pRoomData{secrets: make(map[string]StoredEnv)}
	s.rooms[name] = r
	return r
}

func (s *p2pStore) Save(room, key string, secret StoredEnv) {
	r := s.room(room)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.secrets[key] = secret
}

func (s *p2pStore) LoadAll(room string) map[string]StoredEnv {
	r := s.room(room)
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]StoredEnv, len(r.secrets))
	for k, v := range r.secrets {
		out[k] = v
	}
	return out
}

func (s *p2pStore) Delete(room, key string) {
	r := s.room(room)
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.secrets, key)
}

func (s *p2pStore) RegisterPeer(room, addr string) {
	r := s.room(room)
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.peers {
		if p == addr {
			return
		}
	}
	r.peers = append(r.peers, addr)
}

func (s *p2pStore) Peers(room string) []string {
	r := s.room(room)
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.peers))
	copy(out, r.peers)
	return out
}

func (p *P2PSyncProvider) Save(ctx context.Context, repoName string, secret StoredEnv) error {
	key := utils.GetRepoKey(repoName)
	if p.RelayURL == "" {
		globalP2PStore.Save(p.Room, key, secret)
		return nil
	}
	body, err := json.Marshal(secret)
	if err != nil {
		return err
	}
	u := fmt.Sprintf("%s/p2p/%s/%s", p.RelayURL, url.PathEscape(p.Room), url.PathEscape(key))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.Token != "" {
		req.Header.Set("X-P2P-Token", p.Token)
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("p2p save failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}

func (p *P2PSyncProvider) LoadAll(ctx context.Context) (map[string]StoredEnv, error) {
	if p.RelayURL == "" {
		return globalP2PStore.LoadAll(p.Room), nil
	}
	u := fmt.Sprintf("%s/p2p/%s", p.RelayURL, url.PathEscape(p.Room))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if p.Token != "" {
		req.Header.Set("X-P2P-Token", p.Token)
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("p2p load failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var out map[string]StoredEnv
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out == nil {
		out = make(map[string]StoredEnv)
	}
	return out, nil
}

func (p *P2PSyncProvider) Delete(ctx context.Context, repoName string) error {
	key := utils.GetRepoKey(repoName)
	if p.RelayURL == "" {
		globalP2PStore.Delete(p.Room, key)
		return nil
	}
	u := fmt.Sprintf("%s/p2p/%s/%s", p.RelayURL, url.PathEscape(p.Room), url.PathEscape(key))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	if p.Token != "" {
		req.Header.Set("X-P2P-Token", p.Token)
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("p2p delete failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}

// StartP2PRelay starts an HTTP relay/rendezvous server. It returns the server.
func StartP2PRelay(addr string, token ...string) (*http.Server, error) {
	if addr == "" {
		addr = ":8787"
	}
	tk := ""
	if len(token) > 0 {
		tk = token[0]
	}
	mux := http.NewServeMux()
	handler := newP2PRelayHandler(tk)
	mux.Handle("/", handler)
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		_ = srv.ListenAndServe()
	}()
	return srv, nil
}

// RunP2PRelay blocks serving a relay on addr.
func RunP2PRelay(addr string, token ...string) error {
	tk := ""
	if len(token) > 0 {
		tk = token[0]
	}
	if addr == "" {
		addr = ":8787"
	}
	mux := http.NewServeMux()
	mux.Handle("/", newP2PRelayHandler(tk))
	fmt.Printf("P2P relay listening on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func newP2PRelayHandler(token ...string) http.Handler {
	tk := ""
	if len(token) > 0 {
		tk = token[0]
	}
	mux := http.NewServeMux()

	auth := func(w http.ResponseWriter, r *http.Request) bool {
		if tk == "" {
			return true
		}
		if r.Header.Get("X-P2P-Token") != tk {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
		return true
	}

	mux.HandleFunc("/p2p/{room}", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		room := r.PathValue("room")
		if r.Method == http.MethodGet {
			data := globalP2PStore.LoadAll(room)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(data)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/p2p/{room}/{repo}", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		room := r.PathValue("room")
		repo := r.PathValue("repo")
		switch r.Method {
		case http.MethodPut:
			var secret StoredEnv
			if err := json.NewDecoder(r.Body).Decode(&secret); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			globalP2PStore.Save(room, repo, secret)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			globalP2PStore.Delete(room, repo)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/rendezvous/{code}", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		code := r.PathValue("code")
		switch r.Method {
		case http.MethodPost:
			var body struct {
				Addr string `json:"addr"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.Addr != "" {
				globalP2PStore.RegisterPeer(code, body.Addr)
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			peers := globalP2PStore.Peers(code)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string][]string{"peers": peers})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}

func startP2PRelay(addr, token string) (*http.Server, error) { return StartP2PRelay(addr, token) }
func runP2PRelay(addr, token string) error                   { return RunP2PRelay(addr, token) }
func handleP2PRelay()                                        { handleP2P() }
func handleP2PHost()                                         { handleP2P() }

func handleP2P() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: env-vault p2p <relay|sync> [args]")
		fmt.Println("  relay [--port 8787] [--token CODE]   Start a P2P relay/rendezvous server")
		fmt.Println("  sync [push|pull]                    Sync using P2P backend (set SYNC_BACKEND=p2p)")
		return
	}
	sub := flag.Arg(1)
	switch sub {
	case "relay", "serve", "host":
		addr := ":8787"
		token := ""
		args := flag.Args()[2:]
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--port", "-port":
				if i+1 < len(args) {
					i++
					addr = ":" + strings.TrimPrefix(args[i], ":")
				}
			case "--token", "-token":
				if i+1 < len(args) {
					i++
					token = args[i]
				}
			}
		}
		if err := RunP2PRelay(addr, token); err != nil {
			utils.Fatal("P2P relay error: %v", err)
		}
	case "sync", "push", "pull":
		// Reuse existing sync commands with p2p backend.
		os.Setenv("SYNC_BACKEND", "p2p")
		if sub == "push" {
			handleSyncSave()
		} else if sub == "pull" {
			handleSyncLoad()
		} else {
			// default: load
			handleSyncLoad()
		}
	default:
		fmt.Println("Unknown p2p subcommand. Use relay or sync.")
	}
}

type P2PProvider = P2PSyncProvider
type PeerProvider = P2PSyncProvider
type RelayProvider = P2PSyncProvider
type PeerSyncProvider = P2PSyncProvider

func handleP2PSync()  { handleP2P() }
func handleP2PSave()  { os.Setenv("SYNC_BACKEND", "p2p"); handleSyncSave() }
func handleP2PLoad()  { os.Setenv("SYNC_BACKEND", "p2p"); handleSyncLoad() }
func handleP2PServe() { handleP2P() }
func handleP2PJoin()  { handleP2P() }

func init() {
	RegisterSyncProvider("p2p", func() (SyncProvider, error) {
		return NewP2PSyncProvider()
	})
	RegisterSyncProvider("peer", func() (SyncProvider, error) {
		return NewP2PSyncProvider()
	})
	RegisterSyncProvider("relay", func() (SyncProvider, error) {
		return NewP2PSyncProvider()
	})
	RegisterSyncProvider("rendezvous", func() (SyncProvider, error) {
		return NewP2PSyncProvider()
	})
}
