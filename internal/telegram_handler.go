package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

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

func (a *App) HandleTelegramMessage(update map[string]interface{}) (string, error) {
	message := update["message"].(map[string]interface{})
	chat := message["chat"].(map[string]interface{})
	chatID := int64(chat["id"].(float64))
	userQuestion := message["text"].(string)

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

	fmt.Printf("Sending message to chat %d: %s\n", chatID, responseText)
	return "Message processed", nil
}
