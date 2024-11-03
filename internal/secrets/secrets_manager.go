// internal/secrets/secrets_manager.go

package secrets

import (
	"fmt"
	"os"
)

// GetSecret retrieves a secret from environment variables.
func GetSecret(key string) (string, error) {
	secret := os.Getenv(key)
	if secret == "" {
		return "", fmt.Errorf("required environment variable %s not set", key)
	}
	return secret, nil
}
