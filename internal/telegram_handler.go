package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Handler processes incoming HTTP requests from Telegram
func (a *App) Handler(w http.ResponseWriter, r *http.Request) {
	var reqBody map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	response, err := a.HandleTelegramMessage(reqBody)
	if err != nil {
		http.Error(w, "Error processing message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

// HandleTelegramMessage processes incoming Telegram messages and queries OpenAI
func (a *App) HandleTelegramMessage(update map[string]interface{}) (string, error) {
	message := update["message"].(map[string]interface{})
	chat := message["chat"].(map[string]interface{})
	chatID := int64(chat["id"].(float64))
	userQuestion := message["text"].(string)

	// Log incoming Telegram message
	log.Printf("Received message from Telegram chat %d: %s", chatID, userQuestion)

	// Custom Answer functionality commented out
	/*
		cqaAnswer, err := QueryCQA(userQuestion, a.CQAEndpoint, a.CQAKey)
		if err != nil {
			log.Printf("CQA query failed: %v", err)
		}
		if cqaAnswer != "" {
			responseText := summarizeText(cqaAnswer, 1024)
			log.Printf("Sending summarized CQA response to chat %d: %s", chatID, responseText)
			return responseText, nil
		}
	*/

	// Log that we're about to query OpenAI
	log.Printf("Querying OpenAI...")
	// Use QueryOpenAIWithSummarization instead of QueryOpenAI for 1024-character summarization
	responseText, err := QueryOpenAIWithSummarization(userQuestion, a.OpenAIEndpoint, a.OpenAIKey)
	if err != nil {
		log.Printf("OpenAI query failed: %v", err)
		return "", err
	}

	// Summarize response to 1024 characters
	responseText = summarizeText(responseText, 1024)

	// Send the response
	if err := a.SendMessage(chatID, responseText); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
		return "", err
	}

	log.Printf("Sent message to chat %d: %s", chatID, responseText)
	return "Message processed", nil
}

// SendMessage sends a message to a Telegram chat
func (a *App) SendMessage(chatID int64, text string) error {
	url := "https://api.telegram.org/bot" + a.TelegramToken + "/sendMessage"
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}

// summarizeText truncates text to a specified character limit, ensuring proper sentence completion
func summarizeText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	words := strings.Fields(text)
	truncated := ""
	for _, word := range words {
		if len(truncated)+len(word)+1 > limit {
			break
		}
		truncated += " " + word
	}
	return strings.TrimSpace(truncated) + "..." // Ensures a proper end to the text
}
