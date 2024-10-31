package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func (a *App) Handler(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var reqBody map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Process the Telegram message using the loaded secrets
	response, err := a.HandleTelegramMessage(reqBody) // Call HandleTelegramMessage
	if err != nil {
		http.Error(w, "Error processing message", http.StatusInternalServerError)
		return
	}

	// Write the response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

// HandleTelegramMessage processes incoming Telegram messages
func (a *App) HandleTelegramMessage(update map[string]interface{}) (string, error) {
	// Extract chat and user information
	message := update["message"].(map[string]interface{})
	chat := message["chat"].(map[string]interface{})
	chatID := int64(chat["id"].(float64))
	userQuestion := message["text"].(string)

	// Query CQA and OpenAI as necessary
	cqaAnswer, err := QueryCQA(userQuestion, a.CQAEndpoint, a.CQAKey)
	if err != nil {
		log.Printf("CQA query failed: %v", err)
	}
	responseText := cqaAnswer
	if responseText == "" {
		responseText, err = QueryOpenAI(userQuestion, a.OpenAIEndpoint, a.OpenAIKey)
		if err != nil {
			log.Printf("OpenAI query failed: %v", err)
			return "", err
		}
	}

	// Simulate sending response to Telegram
	fmt.Printf("Sending message to chat %d: %s\n", chatID, responseText)
	return "Message processed", nil
}
