// internal/secrets_manager.go

package internal

import (
	"fmt"
	"os"
)

// GetSecret retrieves a secret from environment variables.
func GetSecret(key string) (string, error) {
	secret := os.Getenv(key)
	if secret == "" {
		return "", fmt.Errorf("required environment variable not set")
	}
	return secret, nil
}
