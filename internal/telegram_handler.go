// internal/telegram_handler.go

package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// HandleTelegramMessage processes incoming Telegram messages and queries OpenAI
func (a *App) HandleTelegramMessage(update *Update, r *http.Request) (string, error) {
	var message *TelegramMessage
	if update.Message != nil {
		message = update.Message
	} else if update.EditedMessage != nil {
		message = update.EditedMessage
	} else if update.ChannelPost != nil {
		message = update.ChannelPost
	} else if update.CallbackQuery != nil {
		// Handle callback queries separately if needed
		log.Printf("Received callback query: %+v", update.CallbackQuery)
		return "Callback query received", nil
	} else {
		log.Printf("No message to process. Received update payload: %+v", update)
		return "No message to process", nil
	}

	if message.Chat == nil || message.Text == "" {
		return "Invalid message structure", nil
	}

	chatID := message.Chat.ID
	userQuestion := message.Text
	messageID := message.MessageID
	userID := message.From.ID
	username := message.From.Username

	// Check if the message is a reply
	isReply := message.ReplyToMessage != nil

	// Check if the bot is mentioned (tagged) in the message
	isTagged := false
	if message.Entities != nil {
		for _, entity := range message.Entities {
			if entity.Type == "mention" {
				mention := message.Text[entity.Offset : entity.Offset+entity.Length]
				log.Printf("Detected mention: %s", mention)
				if isTaggedMention(mention, a.BotUsername) {
					isTagged = true
					userQuestion = removeMention(userQuestion, mention)
					break
				}
			}
		}
	}

	// If the message is not a direct message, a reply to the bot, or mentions the bot, ignore it
	if !isTagged && !(isReply && message.ReplyToMessage.From.IsBot) && message.Chat.Type != "private" {
		log.Printf("Ignoring message in group chat %d: %s", chatID, userQuestion)
		return "No response needed", nil
	}

	// Check if the message is a command
	if strings.HasPrefix(message.Text, "/") {
		return a.handleCommand(message, userID, username)
	}

	log.Printf("Processing message in chat %d: %s", chatID, userQuestion)

	// Prepare messages for OpenAI
	messages := []OpenAIMessage{
		{Role: "system", Content: "You are a helpful assistant."},
	}

	if isReply && message.ReplyToMessage.From.IsBot {
		previousBotMessage := message.ReplyToMessage
		messages = append(messages, OpenAIMessage{Role: "assistant", Content: previousBotMessage.Text})
	}

	messages = append(messages, OpenAIMessage{Role: "user", Content: userQuestion})

	startTime := time.Now()

	// Query OpenAI API
	responseText, err := a.QueryOpenAIWithMessagesSimple(messages)
	if err != nil {
		log.Printf("OpenAI query failed: %v", err)
		return "", err
	}

	endTime := time.Now()
	responseTime := endTime.Sub(startTime)

	// Prepare the final message to fit within Telegram's character limit
	finalMessage := a.PrepareFinalMessageDetailed(responseText)

	// Send the response, incorporating throttling
	if err := a.SendMessageWithThrottle(chatID, finalMessage, messageID, userID); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
		return "", err
	}

	// Determine if the user is rate limited after sending the message
	isRateLimited := !a.UsageCache.CanUserChat(userID)

	// Extract keywords from userQuestion
	keywords := ExtractKeywords(userQuestion)

	// Log the interaction in S3, tracking if the user is rate-limited
	a.logToS3(userID, username, userQuestion, keywords, responseTime, isRateLimited)

	log.Printf("Sent message to chat %d: %s", chatID, responseText)
	return "Message processed", nil
}

// handleCommand processes Telegram commands such as /learn.
func (a *App) handleCommand(message *TelegramMessage, userID int, username string) (string, error) {
	command := strings.Split(message.Text, " ")[0]

	switch command {
	case "/learn":
		// Check if the knowledge base feature is active
		if !a.KnowledgeBaseActive {
			msg := "Knowledge base training is currently disabled."
			a.sendMessage(message.Chat.ID, msg, message.MessageID)
			return "Knowledge base is off", nil
		}

		// Check if the user is authorized
		if _, ok := a.NoLimitUsers[userID]; !ok {
			msg := "You are not authorized to train the knowledge base."
			a.sendMessage(message.Chat.ID, msg, message.MessageID)
			return "Unauthorized training attempt", nil
		}

		// Extract training data from the message
		// For example, assume the command is "/learn [your training data here]"
		parts := strings.SplitN(message.Text, " ", 2)
		if len(parts) < 2 {
			msg := "Please provide the training data. Usage: /learn [your data]"
			a.sendMessage(message.Chat.ID, msg, message.MessageID)
			return "Incomplete training command", nil
		}
		trainingData := parts[1]

		// Send training data to the knowledge base microservice
		err := a.sendTrainingData(trainingData)
		if err != nil {
			log.Printf("Failed to send training data: %v", err)
			msg := "Failed to train the knowledge base. Please try again later."
			a.sendMessage(message.Chat.ID, msg, message.MessageID)
			return "Training failed", err
		}

		msg := "Training data received and is being processed."
		a.sendMessage(message.Chat.ID, msg, message.MessageID)
		return "Training command processed", nil

	default:
		msg := "Unknown command."
		a.sendMessage(message.Chat.ID, msg, message.MessageID)
		return "Unknown command", nil
	}
}

// sendTrainingData sends training data to the knowledge base microservice.
func (a *App) sendTrainingData(data string) error {
	// Define the knowledge base microservice endpoint
	trainingEndpoint := os.Getenv("KNOWLEDGE_BASE_TRAIN_ENDPOINT")
	if trainingEndpoint == "" {
		return fmt.Errorf("KNOWLEDGE_BASE_TRAIN_ENDPOINT not set")
	}

	// Prepare the payload
	payload := map[string]string{
		"data": data,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal training data: %w", err)
	}

	// Send the training data
	resp, err := a.HTTPClient.Post(trainingEndpoint, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to send training data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("training endpoint returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// SendMessageWithThrottle sends a message to the user, considering rate limits.
func (a *App) SendMessageWithThrottle(chatID int64, text string, replyToMessageID int, userID int) error {
	// Check if the user has exceeded the message limit
	isNoLimitUser := false
	if _, ok := a.NoLimitUsers[userID]; ok {
		isNoLimitUser = true
	}

	if !isNoLimitUser && !a.UsageCache.CanUserChat(userID) {
		// Calculate the remaining time until limit reset
		timeRemaining := a.UsageCache.TimeUntilLimitReset(userID)
		minutes := int(timeRemaining.Minutes())
		seconds := int(timeRemaining.Seconds()) % 60

		// Customize the rate limit message to include time until reset
		limitMsg := fmt.Sprintf(
			"Thanks for using ReelTalkBot. We restrict to 10 messages per 10 minutes to keep costs low and allow everyone to use the tool. Please try again in %d minutes and %d seconds.",
			minutes, seconds,
		)
		return a.sendMessage(chatID, limitMsg, replyToMessageID)
	}

	// Track usage
	a.UsageCache.AddUsage(userID)

	// Proceed to send the message through Telegram API using the `sendMessage` function from `app.go`
	return a.sendMessage(chatID, text, replyToMessageID)
}

// isTaggedMention checks if the mention is the bot's username.
func isTaggedMention(mention, botUsername string) bool {
	return strings.ToLower(mention) == "@"+strings.ToLower(botUsername)
}

// removeMention removes the bot's mention from the message text.
func removeMention(text, mention string) string {
	return strings.TrimSpace(strings.Replace(text, mention, "", 1))
}
