package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

// FetchSecrets loads secrets from Azure Key Vault
func FetchSecrets() (map[string]string, error) {
	keyVaultName := os.Getenv("KEY_VAULT_NAME")
	if keyVaultName == "" {
		return nil, fmt.Errorf("KEY_VAULT_NAME environment variable is not set")
	}

	vaultUrl := fmt.Sprintf("https://%s.vault.azure.net/", keyVaultName)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain a credential: %v", err)
	}

	client, err := azsecrets.NewClient(vaultUrl, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret client: %v", err)
	}

	secrets := map[string]string{}
	secretNames := []string{"TELEGRAM-API-TOKEN", "CQA-ENDPOINT", "CQA-API-KEY", "OPENAI-ENDPOINT", "OPENAI-API-KEY"}

	for _, name := range secretNames {
		resp, err := client.GetSecret(context.TODO(), name, "", nil)
		if err != nil {
			log.Printf("Could not retrieve secret %s: %v\n", name, err)
			continue
		}
		secrets[name] = *resp.Value
	}

	return secrets, nil
}
