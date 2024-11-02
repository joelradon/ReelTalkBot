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

// KeywordRequest represents the payload for /extract_keywords
type KeywordRequest struct {
	UserID  int    `json:"user_id"`
	Message string `json:"message"`
}

// KeywordResponse represents the response from /extract_keywords
type KeywordResponse struct {
	Keywords []string `json:"keywords"`
}

// GenerateResponseRequest represents the payload for /generate_response
type GenerateResponseRequest struct {
	UserID  int    `json:"user_id"`
	Message string `json:"message"`
}

// GenerateResponseResponse represents the response from /generate_response
type GenerateResponseResponse struct {
	Response string `json:"response"`
}

// ExtractKeywords sends a request to the /extract_keywords endpoint of the AI microservice
func (a *App) ExtractKeywords(userID int, message string) ([]string, error) {
	url := fmt.Sprintf("%s/extract_keywords", a.AIServiceURL)

	payload := KeywordRequest{
		UserID:  userID,
		Message: message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal KeywordRequest: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Set a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to /extract_keywords failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("non-200 response from /extract_keywords: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var keywordResp KeywordResponse
	if err := json.NewDecoder(resp.Body).Decode(&keywordResp); err != nil {
		return nil, fmt.Errorf("failed to decode KeywordResponse: %w", err)
	}

	return keywordResp.Keywords, nil
}

// GenerateResponse sends a request to the /generate_response endpoint of the AI microservice
func (a *App) GenerateResponse(userID int, message string) (string, error) {
	url := fmt.Sprintf("%s/generate_response", a.AIServiceURL)

	payload := GenerateResponseRequest{
		UserID:  userID,
		Message: message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal GenerateResponseRequest: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Set a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request to /generate_response failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("non-200 response from /generate_response: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var generateResp GenerateResponseResponse
	if err := json.NewDecoder(resp.Body).Decode(&generateResp); err != nil {
		return "", fmt.Errorf("failed to decode GenerateResponseResponse: %w", err)
	}

	return generateResp.Response, nil
}
