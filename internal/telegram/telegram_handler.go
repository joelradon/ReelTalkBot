// internal/telegram/telegram_handler.go

package telegram

import (
	"log"
	"strings"

	"ReelTalkBot-Go/internal/handlers"
	"ReelTalkBot-Go/internal/types"
)

// TelegramHandler processes Telegram messages using a MessageProcessor interface.
type TelegramHandler struct {
	Processor handlers.MessageProcessor
}

// NewTelegramHandler initializes a new TelegramHandler with the provided MessageProcessor.
func NewTelegramHandler(processor handlers.MessageProcessor) *TelegramHandler {
	return &TelegramHandler{
		Processor: processor,
	}
}

// HandleTelegramMessage processes incoming Telegram messages and queries OpenAI or Knowledge Base.
func (th *TelegramHandler) HandleTelegramMessage(update *types.TelegramUpdate) (string, error) {
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
		return "", nil // Return empty string to avoid sending a message
	} else {
		log.Printf("No message to process. Received update payload: %+v", update)
		return "", nil // Return empty string to avoid sending a message
	}

	// Validate message structure
	if message.Chat.ID == 0 || message.Text == "" {
		log.Println("Invalid message structure: missing chat ID or text.")
		return "", nil // Return empty string to avoid sending a message
	}

	// Extract relevant fields from the message
	chatID := message.Chat.ID
	userQuestion := message.Text
	messageID := message.MessageID
	userID := message.From.ID
	username := message.From.Username

	log.Printf("Received message from user %d (%s) in chat %d: %s", userID, username, chatID, userQuestion)

	// Check if the message is a command (starts with "/")
	if strings.HasPrefix(message.Text, "/") {
		log.Printf("Message is a command: %s", message.Text)
		_, err := th.Processor.HandleCommand(message, userID, username)
		if err != nil {
			log.Printf("Error handling command: %v", err)
			return "", nil // Return empty string to avoid sending a message
		}
		return "", nil // Return empty string to avoid sending a message
	}

	// Determine if the message is a reply to another message
	isReply := message.ReplyToMessage != nil
	if isReply {
		log.Printf("Message is a reply to message ID %d from user %d", message.ReplyToMessage.MessageID, message.ReplyToMessage.From.ID)
	}

	// Check if the bot is mentioned (tagged) in the message
	isTagged := false
	if len(message.Entities) > 0 {
		for _, entity := range message.Entities {
			if entity.Type == "mention" {
				if entity.Offset+entity.Length > len(message.Text) {
					log.Println("Mention entity exceeds message length. Skipping.")
					continue // Prevent out-of-range slicing
				}
				mention := message.Text[entity.Offset : entity.Offset+entity.Length]
				log.Printf("Detected mention: %s", mention)
				if isTaggedMention(mention, th.Processor.GetBotUsername()) {
					isTagged = true
					userQuestion = removeMention(userQuestion, mention)
					log.Printf("Message is tagged with bot username: %s", th.Processor.GetBotUsername())
					break
				}
			}
		}
	}

	// If the message is not a direct message, a reply to the bot, or mentions the bot, ignore it
	if !isTagged && !(isReply && message.ReplyToMessage.From.IsBot) && message.Chat.Type != "private" {
		log.Printf("Ignoring message in group chat %d: %s", chatID, userQuestion)
		return "", nil // Return empty string to avoid sending a message
	}

	log.Printf("Processing message in chat %d: %s", chatID, userQuestion)

	// Process the message: Query Knowledge Base or fallback to OpenAI
	if err := th.Processor.ProcessMessage(chatID, userID, username, userQuestion, messageID); err != nil {
		log.Printf("Error processing message: %v", err)
		return "", nil // Return empty string to avoid sending a message
	}

	return "", nil // Return empty string to avoid sending a message
}

// isTaggedMention checks if the mention is the bot's username.
func isTaggedMention(mention, botUsername string) bool {
	return strings.ToLower(mention) == "@"+strings.ToLower(botUsername)
}

// removeMention removes the bot's mention from the message text.
func removeMention(text, mention string) string {
	return strings.TrimSpace(strings.Replace(text, mention, "", 1))
}
