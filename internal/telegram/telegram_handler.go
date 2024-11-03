// internal/telegram/telegram_handler.go

package telegram

import (
	"log"
	"strings"

	"ReelTalkBot-Go/internal/app"
	"ReelTalkBot-Go/internal/types"
)

// TelegramHandler handles Telegram message processing
type TelegramHandler struct {
	App *app.App
}

// NewTelegramHandler initializes a new TelegramHandler
func NewTelegramHandler(app *app.App) *TelegramHandler {
	return &TelegramHandler{
		App: app,
	}
}

// HandleTelegramMessage processes incoming Telegram messages and queries OpenAI or Knowledge Base
func (th *TelegramHandler) HandleTelegramMessage(update *types.Update) (string, error) {
	var message *types.TelegramMessage

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
				if isTaggedMention(mention, th.App.BotUsername) {
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
		return th.App.HandleCommand(message, userID, username)
	}

	log.Printf("Processing message in chat %d: %s", chatID, userQuestion)

	// Process the message: Query Knowledge Base or fallback to OpenAI
	if err := th.App.ProcessMessage(chatID, userID, username, userQuestion, messageID); err != nil {
		log.Printf("Error processing message: %v", err)
		return "Error processing your request.", nil
	}

	return "Message processed", nil
}

// isTaggedMention checks if the mention is the bot's username.
func isTaggedMention(mention, botUsername string) bool {
	return strings.ToLower(mention) == "@"+strings.ToLower(botUsername)
}

// removeMention removes the bot's mention from the message text.
func removeMention(text, mention string) string {
	return strings.TrimSpace(strings.Replace(text, mention, "", 1))
}
