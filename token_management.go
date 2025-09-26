package main

import "github.com/zalando/go-keyring"

const (
	keyringService = "env-vault"
	keyringUser    = "github_access_token"
)

func saveToken(token string) error {
	return keyring.Set(keyringService, keyringUser, token)
}

func getToken() (string, error) {
	return keyring.Get(keyringService, keyringUser)
}
