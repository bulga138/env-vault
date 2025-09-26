// filename: token_manager.go
package main

import "github.com/zalando/go-keyring"

const (
    keyringService = "env-vault"
    keyringUser    = "github_access_token"
)

// saveToken guarda el token de acceso en el keyring del sistema.
func saveToken(token string) error {
    return keyring.Set(keyringService, keyringUser, token)
}

// getToken recupera el token de acceso desde el keyring del sistema.
func getToken() (string, error) {
    return keyring.Get(keyringService, keyringUser)
}