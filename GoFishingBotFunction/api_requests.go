package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func QueryCQA(question, endpoint, apiKey string) (string, error) {
	data := map[string]string{"question": question}
	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
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
		body, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(body, &result)
		if answers, ok := result["answers"].([]interface{}); ok && len(answers) > 0 {
			answer := answers[0].(map[string]interface{})
			return answer["answer"].(string), nil
		}
	}
	return "", fmt.Errorf("CQA API error: %v", resp.Status)
}

func QueryOpenAI(prompt, endpoint, apiKey string) (string, error) {
	data := map[string]interface{}{
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":  500,
		"temperature": 0.7,
	}
	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
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
			body, _ := ioutil.ReadAll(resp.Body)
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
