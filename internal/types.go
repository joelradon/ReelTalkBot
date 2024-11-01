// internal/types.go

package internal

// Update represents an incoming Telegram update
type Update struct {
	UpdateID      int              `json:"update_id"`
	Message       *TelegramMessage `json:"message,omitempty"`
	EditedMessage *TelegramMessage `json:"edited_message,omitempty"`
	ChannelPost   *TelegramMessage `json:"channel_post,omitempty"`
	CallbackQuery *CallbackQuery   `json:"callback_query,omitempty"`
	// Add more fields as necessary based on the payload.
}

// CallbackQuery represents an incoming callback query from inline buttons.
type CallbackQuery struct {
	ID           string           `json:"id"`
	From         *User            `json:"from"`
	Data         string           `json:"data,omitempty"`
	Message      *TelegramMessage `json:"message,omitempty"`
	ChatInstance string           `json:"chat_instance"`
}

// TelegramMessage represents a Telegram message
type TelegramMessage struct {
	MessageID      int              `json:"message_id"`
	From           *User            `json:"from"`
	Chat           *Chat            `json:"chat"`
	Date           int              `json:"date"`
	Text           string           `json:"text"`
	ReplyToMessage *TelegramMessage `json:"reply_to_message,omitempty"`
	Entities       []MessageEntity  `json:"entities,omitempty"`
}

// User represents a Telegram user or bot
type User struct {
	ID           int    `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

// Chat represents a Telegram chat
type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
}

// MessageEntity represents special entities in text messages
type MessageEntity struct {
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	Type   string `json:"type"`
}
