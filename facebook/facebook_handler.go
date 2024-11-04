// internal/facebook/facebook_handler.go

package facebook

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"ReelTalkBot-Go/internal/app"
	"ReelTalkBot-Go/internal/types"
)

// HandleFacebookRequest handles Facebook Messenger webhook verification and messages.
func HandleFacebookRequest(w http.ResponseWriter, r *http.Request, verifyToken, pageAccessToken string, botApp *app.App) {
	switch r.Method {
	case http.MethodGet:
		verifyWebhook(w, r, verifyToken)
	case http.MethodPost:
		handleMessages(w, r, pageAccessToken, botApp)
	default:
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

// verifyWebhook verifies the webhook with the token provided by Facebook.
func verifyWebhook(w http.ResponseWriter, r *http.Request, verifyToken string) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == verifyToken {
		fmt.Fprint(w, challenge)
	} else {
		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}

// handleMessages processes incoming messages from Facebook Messenger.
func handleMessages(w http.ResponseWriter, r *http.Request, pageAccessToken string, botApp *app.App) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var fbUpdate types.FacebookUpdate
	if err := json.Unmarshal(body, &fbUpdate); err != nil {
		log.Printf("Failed to unmarshal Facebook update: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Process messaging events
	for _, entry := range fbUpdate.Entry {
		for _, messaging := range entry.Messaging {
			if messaging.Message != nil {
				go processMessage(messaging, pageAccessToken, botApp)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

// processMessage handles an individual message from a user.
func processMessage(messaging types.FacebookMessaging, pageAccessToken string, botApp *app.App) {
	senderID := messaging.Sender.ID
	messageText := messaging.Message.Text

	log.Printf("Received message from user %s: %s", senderID, messageText)

	// Process commands like /help or /learn
	if strings.HasPrefix(messageText, "/") {
		response, err := botApp.HandleFacebookCommand(senderID, messageText)
		if err != nil {
			log.Printf("Error handling command: %v", err)
			return
		}
		if response != "" {
			sendTextMessage(senderID, response, pageAccessToken)
		}
		return
	}

	// Process the message using the bot's logic
	_, err := botApp.ProcessFacebookMessage(senderID, messageText)
	if err != nil {
		log.Printf("Error processing message: %v", err)
		return
	}
}

// sendTextMessage sends a text message to the user via Facebook Messenger.
func sendTextMessage(recipientID, text, pageAccessToken string) {
	url := "https://graph.facebook.com/v11.0/me/messages?access_token=" + pageAccessToken

	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"message": map[string]string{
			"text": text,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal message payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		log.Printf("Failed to create send message request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Failed to send message, status %d: %s", resp.StatusCode, string(bodyBytes))
	}
}
