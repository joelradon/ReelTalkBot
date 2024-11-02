// internal/telegram_handler.go

package internal

import (
	"log"
	"net/http"
	"strings"
)

// HandleTelegramMessage processes incoming Telegram messages and queries OpenAI or Knowledge Base
func (a *App) HandleTelegramMessage(update *Update, r *http.Request) (string, error) {
	var message *TelegramMessage

	// Determine the type of message received
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

	// Validate message structure
	if message.Chat == nil || message.Text == "" {
		return "Invalid message structure", nil
	}

	// Extract relevant fields from the message
	chatID := message.Chat.ID
	userQuestion := message.Text
	messageID := message.MessageID
	userID := message.From.ID
	username := message.From.Username

	// Determine if the message is a reply to another message
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

	// Check if the message is a command (starts with "/")
	if strings.HasPrefix(message.Text, "/") {
		return a.handleCommand(message, userID, username)
	}

	log.Printf("Processing message in chat %d: %s", chatID, userQuestion)

	// Process the message: Query Knowledge Base or fallback to OpenAI
	if err := a.ProcessMessage(chatID, userID, username, userQuestion, messageID); err != nil {
		log.Printf("Error processing message: %v", err)
		return "Error processing your request.", nil
	}

	return "Message processed", nil
}

// IdentifyTaxonomyCategories parses the user query to extract taxonomy categories.
func IdentifyTaxonomyCategories(query string) (bodyOfWater, fishSpecies, waterType string) {
	lowerQuery := strings.ToLower(query)

	bodyOfWaterKeywords := []string{"salmon river", "lake ontario", "great lake tributaries"}
	fishSpeciesKeywords := []string{"king salmon", "coho salmon", "steelhead", "brown trout"}
	waterTypeKeywords := []string{"adronomous", "lentic", "lotic"}

	for _, kw := range bodyOfWaterKeywords {
		if strings.Contains(lowerQuery, kw) {
			bodyOfWater = kw
			break
		}
	}

	for _, kw := range fishSpeciesKeywords {
		if strings.Contains(lowerQuery, kw) {
			fishSpecies = kw
			break
		}
	}

	for _, kw := range waterTypeKeywords {
		if strings.Contains(lowerQuery, kw) {
			waterType = kw
			break
		}
	}

	return
}

// isTaggedMention checks if the mention is the bot's username.
func isTaggedMention(mention, botUsername string) bool {
	return strings.ToLower(mention) == "@"+strings.ToLower(botUsername)
}

// removeMention removes the bot's mention from the message text.
func removeMention(text, mention string) string {
	return strings.TrimSpace(strings.Replace(text, mention, "", 1))
}
