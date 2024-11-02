// internal/telegram_handler.go

package internal

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// HandleTelegramMessage processes incoming Telegram messages and interacts with the AI microservice
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

	// Ignore messages in group chats unless the bot is tagged or it's a reply to the bot
	if !isTagged && !(isReply && message.ReplyToMessage.From.IsBot) && message.Chat.Type != "private" {
		log.Printf("Ignoring message in group chat %d: %s", chatID, userQuestion)
		return "No response needed", nil
	}

	log.Printf("Processing message in chat %d: %s", chatID, userQuestion)

	// Pass the parsed update to ProcessMessage
	err := a.ProcessMessage(chatID, userID, username, userQuestion, messageID)
	if err != nil {
		return "", err
	}

	return "Message processed", nil
}

// SendMessageWithThrottle sends a message to Telegram, considering rate limits
func (a *App) SendMessageWithThrottle(chatID int64, text string, replyToMessageID int, userID int) error {
	// Check if the user has exceeded the message limit
	noLimitUsers := strings.Split(os.Getenv("NO_LIMIT_USERS"), ",")
	isNoLimitUser := false
	for _, id := range noLimitUsers {
		if id == strconv.Itoa(userID) {
			isNoLimitUser = true
			break
		}
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

// Helper function to check if the mention is the bot's username
func isTaggedMention(mention, botUsername string) bool {
	return strings.ToLower(mention) == "@"+strings.ToLower(botUsername)
}

// Helper function to remove the mention from the message text
func removeMention(text, mention string) string {
	return strings.TrimSpace(strings.Replace(text, mention, "", 1))
}
