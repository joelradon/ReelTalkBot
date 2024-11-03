// internal/api/api_requests.go

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ReelTalkBot-Go/internal/types"
	"ReelTalkBot-Go/internal/utils"
)

// APIHandler handles OpenAI API interactions
type APIHandler struct {
	OpenAIKey      string
	OpenAIEndpoint string
	Client         *http.Client
}

// NewAPIHandler initializes a new APIHandler
func NewAPIHandler(openAIKey, openAIEndpoint string) *APIHandler {
	return &APIHandler{
		OpenAIKey:      openAIKey,
		OpenAIEndpoint: openAIEndpoint,
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// QueryOpenAIWithMessages sends a request to OpenAI with given messages and returns response text
func (api *APIHandler) QueryOpenAIWithMessages(messages []types.OpenAIMessage) (string, error) {
	fullEndpoint := fmt.Sprintf("%s/chat/completions", api.OpenAIEndpoint)

	query := types.OpenAIQuery{
		Model:       "gpt-4o-mini", // Corrected model name
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   4096, // Increased character limit
	}

	body, err := json.Marshal(query)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAI query: %w", err)
	}

	// Use context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", fullEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+api.OpenAIKey)

	resp, err := api.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request to OpenAI: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse and handle response
	var result types.OpenAIResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}

	// Extract content
	if len(result.Choices) > 0 {
		content := result.Choices[0].Message.Content
		if len(content) > 4096 { // Telegram's max message length
			content = utils.SummarizeToLength(content, 4096)
		}
		return content, nil
	}

	return "", fmt.Errorf("no choices returned in OpenAI response")
}
