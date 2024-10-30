package main

import (
	"fmt"
	"log"
)

// HandleTelegramMessage processes incoming Telegram messages
func HandleTelegramMessage(update map[string]interface{}, secrets map[string]string) (string, error) {
	// Extract chat and user information
	message := update["message"].(map[string]interface{})
	chat := message["chat"].(map[string]interface{})
	chatID := int64(chat["id"].(float64))
	userQuestion := message["text"].(string)

	// Query CQA and OpenAI as necessary
	cqaAnswer, err := QueryCQA(userQuestion, secrets["CQA_ENDPOINT"], secrets["CQA_API_KEY"])
	if err != nil {
		log.Printf("CQA query failed: %v", err)
	}
	responseText := cqaAnswer
	if responseText == "" {
		responseText, err = QueryOpenAI(userQuestion, secrets["OPENAI_ENDPOINT"], secrets["OPENAI_API_KEY"])
		if err != nil {
			log.Printf("OpenAI query failed: %v", err)
			return "", err
		}
	}

	// Simulate sending response to Telegram
	fmt.Printf("Sending message to chat %d: %s\n", chatID, responseText)
	return "Message processed", nil
}
