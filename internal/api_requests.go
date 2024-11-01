// internal/api_requests.go

package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIQuery represents the request payload for OpenAI API
type OpenAIQuery struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float32         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

// OpenAIMessage represents a message in the OpenAI chat
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents the response from OpenAI API
type OpenAIResponse struct {
	Choices []struct {
		Index        int           `json:"index"`
		Message      OpenAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// QueryOpenAIWithMessages sends a request to OpenAI with given messages and returns response text
func (a *App) QueryOpenAIWithMessages(messages []OpenAIMessage) (string, error) {
	fullEndpoint := fmt.Sprintf("%s/chat/completions", a.OpenAIEndpoint)

	query := OpenAIQuery{
		Model:       "gpt-4o-mini", // Use the appropriate model
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   500,
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
	req.Header.Set("Authorization", "Bearer "+a.OpenAIKey)

	resp, err := a.HTTPClient.Do(req)
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
	var result OpenAIResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}

	// Extract content
	if len(result.Choices) > 0 {
		content := result.Choices[0].Message.Content
		if len(content) > 4096 { // Telegram's max message length
			content = SummarizeToLength(content, 4096)
		}
		return content, nil
	}

	return "", fmt.Errorf("no choices returned in OpenAI response")
}
