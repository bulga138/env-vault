package main

import (
	"errors"
)

var (
	ErrRepoNotFound        = errors.New("repository not found in vault")
	ErrDecryptionFailed    = errors.New("decryption failed - wrong password or corrupted data")
	ErrVaultLocked         = errors.New("vault file is locked by another process")
	ErrInvalidHMAC         = errors.New("integrity check failed: possible tampering or corruption")
	ErrInvalidSalt         = errors.New("invalid salt in stored data")
	ErrInvalidCiphertext   = errors.New("invalid ciphertext in stored data")
	ErrNoSecretProvider    = errors.New("no secret provider configured")
	ErrBitwardenCLI        = errors.New("Bitwarden CLI error")
	ErrFirestoreConnection = errors.New("failed to connect to Firestore")
)
