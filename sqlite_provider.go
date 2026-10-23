package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/bulga138/env-vault/utils"
)

type SQLiteSyncProvider struct {
	DB *sql.DB
}

func NewSQLiteSyncProvider() (*SQLiteSyncProvider, error) {
	path := os.Getenv("SYNC_SQLITE_PATH")
	if path == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		dir := filepath.Join(configDir, "env-vault")
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
		path = filepath.Join(dir, "sync.db")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	stmt := `CREATE TABLE IF NOT EXISTS secrets (
		repo_key TEXT PRIMARY KEY,
		data TEXT NOT NULL,
		updated_at INTEGER NOT NULL
	);`
	if _, err := db.Exec(stmt); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteSyncProvider{DB: db}, nil
}

func (s *SQLiteSyncProvider) Name() string { return "SQLite Sync" }

func (s *SQLiteSyncProvider) Save(ctx context.Context, repoName string, secret StoredEnv) error {
	data, err := json.Marshal(secret)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO secrets(repo_key, data, updated_at) VALUES(?,?,?)
		 ON CONFLICT(repo_key) DO UPDATE SET data=excluded.data, updated_at=excluded.updated_at`,
		utils.GetRepoKey(repoName), string(data), time.Now().Unix())
	return err
}

func (s *SQLiteSyncProvider) LoadAll(ctx context.Context) (map[string]StoredEnv, error) {
	out := make(map[string]StoredEnv)
	rows, err := s.DB.QueryContext(ctx, `SELECT repo_key, data FROM secrets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key, data string
		if err := rows.Scan(&key, &data); err != nil {
			continue
		}
		var secret StoredEnv
		if err := json.Unmarshal([]byte(data), &secret); err != nil {
			continue
		}
		out[key] = secret
	}
	return out, rows.Err()
}

func (s *SQLiteSyncProvider) Delete(ctx context.Context, repoName string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM secrets WHERE repo_key = ?`, utils.GetRepoKey(repoName))
	return err
}

func (s *SQLiteSyncProvider) Close() error {
	if s.DB != nil {
		return s.DB.Close()
	}
	return nil
}

func init() {
	RegisterSyncProvider("sqlite", func() (SyncProvider, error) {
		return NewSQLiteSyncProvider()
	})
}