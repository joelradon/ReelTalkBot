package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type OpenAIQuery struct {
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float32   `json:"temperature"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// QueryOpenAIWithSummarization handles querying OpenAI and ensures summarized responses.
func QueryOpenAIWithSummarization(prompt, baseEndpoint, apiKey string) (string, error) {
	deploymentName := "gpt-4o-mini"
	apiVersion := "2024-08-01-preview"
	fullEndpoint := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", baseEndpoint, deploymentName, apiVersion)

	client := &http.Client{}
	query := OpenAIQuery{
		Messages:    []Message{{Role: "user", Content: prompt}},
		MaxTokens:   500,
		Temperature: 0.7,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAI query: %w", err)
	}
	req, err := http.NewRequest("POST", fullEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI request: %w", err)
	}
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request to OpenAI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse and handle response
	var result map[string]interface{}
	body, _ = io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}

	// Extract content, ensure it's under 1024 characters and summarize if necessary
	content := ""
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		message := choices[0].(map[string]interface{})["message"].(map[string]interface{})
		content = message["content"].(string)
		if len(content) > 1024 {
			content = SummarizeToLength(content, 1024)
		}
	}

	return content, nil
}

// SummarizeToLength shortens text to a max of `length` characters, rounding off sentences where possible.
func SummarizeToLength(text string, length int) string {
	if len(text) <= length {
		return text
	}

	// Attempt to round off at word boundaries, preferably after full stops
	rounded := text[:length]
	lastPeriod := strings.LastIndex(rounded, ".")
	if lastPeriod > length/2 { // Ensure at least half text is retained
		return rounded[:lastPeriod+1]
	}

	// Fallback: truncate with ellipsis
	return rounded + "..."
}
