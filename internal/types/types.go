// internal/types/types.go

package types

// OpenAIMessage represents a message in the OpenAI chat
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIQuery represents the request payload for OpenAI API
type OpenAIQuery struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float32         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

// OpenAIResponse represents the response from OpenAI API
type OpenAIResponse struct {
	Choices []struct {
		Index        int           `json:"index"`
		Message      OpenAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// KnowledgeEntryResponse represents the structure of a Knowledge Entry from the Knowledge Base
type KnowledgeEntryResponse struct {
	ID               uint   `json:"id"`
	BodyOfWater      string `json:"body_of_water"`
	FishSpecies      string `json:"fish_species"`
	WaterType        string `json:"water_type"`
	QuestionTemplate string `json:"question_template"`
	Answer           string `json:"answer"`
	Category         string `json:"category"` // New field for categorization
}

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

// QueryParameters represents the parameters for querying the Knowledge Base
type QueryParameters struct {
	BodyOfWater string `json:"body_of_water"`
	FishSpecies string `json:"fish_species"`
	WaterType   string `json:"water_type"`
	Category    string `json:"category"`
	Query       string `json:"query"`
}
