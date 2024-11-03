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

// GetKnowledgeEntries retrieves knowledge entries based on query parameters.
// Updated to accept a context.Context parameter.
func (k *KnowledgeBaseClient) GetKnowledgeEntries(ctx context.Context, params types.QueryParameters) ([]types.KnowledgeEntryResponse, error) {
	endpoint := k.BaseURL // Use BaseURL directly without appending

	payloadBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query parameters: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge base request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", k.APIKey)

	resp, err := k.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send knowledge base request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("knowledge base returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var entries []types.KnowledgeEntryResponse
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode knowledge base response: %w", err)
	}

	return entries, nil
}

// UpdateKnowledgeEntryRating updates the ratings of a KB entry based on user feedback
func (k *KnowledgeBaseClient) UpdateKnowledgeEntryRating(kbNumber int, rating string) error {
	endpoint := fmt.Sprintf("%s/rate", k.BaseURL) // Append /rate directly

	payload := map[string]string{
		"rating": rating,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal rating payload: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create rating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", k.APIKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req = req.WithContext(ctx)

	resp, err := k.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send rating request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("rating endpoint returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetKnowledgeEntry retrieves a single knowledge entry by KB number
func (k *KnowledgeBaseClient) GetKnowledgeEntry(ctx context.Context, kbNumber int) (*types.KnowledgeEntryResponse, error) {
	endpoint := fmt.Sprintf("%s/%d", k.BaseURL, kbNumber) // Append KB number directly

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge base get request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", k.APIKey)

	resp, err := k.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send knowledge base get request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("knowledge base get endpoint returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var entry types.KnowledgeEntryResponse
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("failed to decode knowledge base get response: %w", err)
	}

	return &entry, nil
}
