package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CQAQuery struct {
	Question string `json:"question"`
}

type OpenAIQuery struct {
	Messages    Message `json:"messages"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float32 `json:"temperature"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func QueryOpenAI(prompt, endpoint, apiKey string) (string, error) {
	query := OpenAIQuery{
		Messages: Message{
			Role:    "user",
			Content: prompt,
		},
		MaxTokens:   500,
		Temperature: 0.7,
	}
	body, err := json.Marshal(query)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	for i := 0; i < 5; i++ {
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &result)
			if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
				message := choices[0].(map[string]interface{})
				return message["content"].(string), nil
			}
		} else if resp.StatusCode == 429 {
			time.Sleep(time.Second * time.Duration(2<<i))
		}
	}
	return "Service unavailable due to rate limits. Please try again later.", nil
}

func QueryCQA(question, endpoint, apiKey string) (string, error) {
	cqaQuery := CQAQuery{
		Question: question,
	}
	body, err := json.Marshal(cqaQuery)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		json.Unmarshal(body, &result)
		if answers, ok := result["answers"].([]interface{}); ok && len(answers) > 0 {
			answer := answers[0].(map[string]interface{})
			return answer["answer"].(string), nil
		}
	}
	return "", fmt.Errorf("CQA API error: %v", resp.Status)
}
