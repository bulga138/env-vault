// filename: bitwarden.go
package main

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/floriaanpost/go-bitwarden-client"
	"golang.org/x/term"
)

// getMasterPasswordFromBitwarden fetches the env-vault master password from a personal Bitwarden vault.
func getMasterPasswordFromBitwarden() ([]byte, error) {
	fmt.Println("🔑 Your vault is locked. Unlocking with Bitwarden...")

	// 1. Get the Bitwarden Item ID from an environment variable.
	itemID := os.Getenv("BITWARDEN_ITEM_ID")
	if itemID == "" {
		return nil, fmt.Errorf("BITWARDEN_ITEM_ID environment variable must be set")
	}

	// 2. Prompt the user for their main Bitwarden master password to unlock the vault.
	fmt.Print("Enter your Bitwarden master password: ")
	bitwardenPassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, fmt.Errorf("failed to read Bitwarden password: %w", err)
	}
	fmt.Println()

	// 3. Initialize the Bitwarden client.
	bw := bitwarden.New()
	defer bw.Close()

	// 4. Unlock the vault for this session.
	if err := bw.Unlock(context.TODO(), string(bitwardenPassword)); err != nil {
		return nil, fmt.Errorf("failed unlocking Bitwarden vault: %w", err)
	}
	// Lock the vault again when the function finishes.
	defer bw.Lock(context.TODO())

	// 5. Fetch the stored password item.
	// You can also use bw.GetSecureNote if you stored it as a note.
	item, err := bw.GetSecureNote(context.TODO(), itemID)
	if err != nil {
		return nil, fmt.Errorf("failed getting item with ID '%s' from Bitwarden: %w", itemID, err)
	}

	fmt.Println("✅ Master password successfully fetched from Bitwarden.")
	return []byte(item), nil
}
