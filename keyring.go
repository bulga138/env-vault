package main

import (
	"github.com/zalando/go-keyring"
)

const keyringService = "env-vault"

func keyringGet(repoKey string) ([]byte, error) {
	v, err := keyring.Get(keyringService, repoKey)
	if err != nil {
		return nil, err
	}
	return []byte(v), nil
}

func keyringSet(repoKey string, password []byte) error {
	return keyring.Set(keyringService, repoKey, string(password))
}

func keyringDelete(repoKey string) error {
	return keyring.Delete(keyringService, repoKey)
}

func keyringAvailable() bool {
	// Best-effort probe
	err := keyring.Set(keyringService, "__probe__", "x")
	if err != nil {
		return false
	}
	_ = keyring.Delete(keyringService, "__probe__")
	return true
}

func keyringName() string { return "OS Keyring" }