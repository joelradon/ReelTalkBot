// internal/telegram_handler.go

package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// HandleTelegramMessage processes incoming Telegram messages and queries OpenAI
func (a *App) HandleTelegramMessage(update *Update, r *http.Request) (string, error) {
	if update.Message == nil {
		return "No message to process", nil
	}

	message := update.Message
	if message.Chat == nil || message.Text == "" {
		return "Invalid message structure", nil
	}

	chatID := message.Chat.ID
	userQuestion := message.Text
	messageID := message.MessageID

	// Check if the message is a reply
	isReply := message.ReplyToMessage != nil

	// Check if the bot is mentioned (tagged) in the message
	isTagged := false
	for _, entity := range message.Entities {
		if entity.Type == "mention" {
			mention := message.Text[entity.Offset : entity.Offset+entity.Length]
			log.Printf("Detected mention: %s", mention)
			if isTaggedMention(mention, a.BotUsername) {
				isTagged = true
				// Remove the mention from the userQuestion
				userQuestion = removeMention(userQuestion, mention)
				break
			}
		}
	}

	// Ignore group messages that aren't replies or mentions
	if !isReply && !isTagged && message.Chat.Type != "private" {
		log.Printf("Ignoring message in group chat %d: %s", chatID, userQuestion)
		return "No response needed", nil
	}

	// Log incoming Telegram message
	log.Printf("Received message from Telegram chat %d: %s", chatID, userQuestion)

	// Prepare messages for OpenAI
	messages := []OpenAIMessage{
		{Role: "system", Content: "You are a helpful assistant."},
	}

	// Include previous bot message if the user is replying to it
	if isReply && message.ReplyToMessage.From.IsBot {
		previousBotMessage := message.ReplyToMessage
		messages = append(messages, OpenAIMessage{Role: "assistant", Content: previousBotMessage.Text})
		messages = append(messages, OpenAIMessage{Role: "user", Content: userQuestion})
	} else {
		messages = append(messages, OpenAIMessage{Role: "user", Content: userQuestion})
	}

	// Log that we're about to query OpenAI
	log.Printf("Querying OpenAI...")

	startTime := time.Now()

	// Use QueryOpenAIWithMessages to get the response
	responseText, err := a.QueryOpenAIWithMessagesSimple(messages)
	if err != nil {
		log.Printf("OpenAI query failed: %v", err)
		return "", err
	}

	// Calculate response time
	endTime := time.Now()
	responseTime := endTime.Sub(startTime)

	// Prepare the final message
	finalMessage := a.PrepareFinalMessageDetailed(responseText)

	// Send the response, incorporating throttling and replying to the user's message
	if err := a.SendMessageWithThrottle(chatID, finalMessage, messageID); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
		return "", err
	}

	// Log user data to S3
	a.LogToS3(message, userQuestion, responseTime)

	log.Printf("Sent message to chat %d: %s", chatID, responseText)
	return "Message processed", nil
}

// Helper function to check if the mention is the bot's username
func isTaggedMention(mention, botUsername string) bool {
	return mention == "@"+botUsername
}

// Helper function to remove the mention from the message text
func removeMention(text, mention string) string {
	return strings.TrimSpace(strings.Replace(text, mention, "", 1))
}

// PrepareFinalMessageSimple prepares the final message without sources
func (a *App) PrepareFinalMessageDetailed(responseText string) string {
	// Ensure the main message is within Telegram's character limits (4096 characters)
	maxLength := 4096
	if len(responseText) > maxLength {
		responseText = SummarizeToLength(responseText, maxLength)
	}

	return responseText
}

// QueryOpenAIWithMessagesSimple sends a request to OpenAI with given messages and returns only the response text
func (a *App) QueryOpenAIWithMessagesSimple(messages []OpenAIMessage) (string, error) {
	fullEndpoint := fmt.Sprintf("%s/chat/completions", a.OpenAIEndpoint)

	query := OpenAIQuery{
		Model:       "gpt-4", // Use the appropriate model
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
