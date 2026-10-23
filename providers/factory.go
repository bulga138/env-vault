package providers

import (
	"fmt"
	"os"
)

func GetSecretProvider() (SecretProvider, error) {
	// Priority: Bitwarden (if configured)
	if os.Getenv("BWS_PROJECT_ID") != "" {
		return NewBitwardenProvider()
	}

	// No provider configured
	return nil, fmt.Errorf("no secret provider configured. Set BWS_PROJECT_ID or use password prompt")
}
