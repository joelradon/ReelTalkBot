// internal/knowledgebase/knowledgebase.go

package knowledgebase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ReelTalkBot-Go/internal/types"
)

// KnowledgeBaseClient handles communication with the Knowledge Base microservice
type KnowledgeBaseClient struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

// NewKnowledgeBaseClient initializes a new KnowledgeBaseClient
func NewKnowledgeBaseClient(baseURL, apiKey string) *KnowledgeBaseClient {
	return &KnowledgeBaseClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetKnowledgeEntries retrieves knowledge entries based on query parameters
func (k *KnowledgeBaseClient) GetKnowledgeEntries(params types.QueryParameters) ([]types.KnowledgeEntryResponse, error) {
	endpoint := fmt.Sprintf("%s/api/knowledge", k.BaseURL)

	payloadBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query parameters: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create Knowledge Base request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", k.APIKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := k.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send Knowledge Base request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Knowledge Base returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var entries []types.KnowledgeEntryResponse
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode Knowledge Base response: %w", err)
	}

	return entries, nil
}
