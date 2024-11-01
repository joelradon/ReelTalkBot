package internal

import (
	"fmt"
	"os"
)

// SecretsManager is responsible for managing sensitive data and secrets
func GetSecret(key string) (string, error) {
	secret := os.Getenv(key)
	if secret == "" {
		return "", fmt.Errorf("required environment variable not set")
	}
	return secret, nil
}
