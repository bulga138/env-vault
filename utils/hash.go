package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
)

func GetRepoKey(repoName string) string {
	pepper := os.Getenv("ENV_VAULT_PEPPER")
	hasher := sha256.New()
	hasher.Write([]byte(repoName))
	if pepper != "" {
		hasher.Write([]byte(pepper))
	}
	hash := hasher.Sum(nil)
	return base64.URLEncoding.EncodeToString(hash)
}

func GetSecretKey(repoName string) string {
	keyHash := GetRepoKey(repoName)[:16]
	return fmt.Sprintf("env-vault-%s", keyHash)
}
