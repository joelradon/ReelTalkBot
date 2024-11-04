// internal/types/types.go

package types

// TelegramUpdate represents an incoming update from Telegram.
type TelegramUpdate struct {
	UpdateID      int                    `json:"update_id"`
	Message       *TelegramMessage       `json:"message,omitempty"`
	EditedMessage *TelegramMessage       `json:"edited_message,omitempty"`
	ChannelPost   *TelegramMessage       `json:"channel_post,omitempty"`
	CallbackQuery *TelegramCallbackQuery `json:"callback_query,omitempty"`
}

// TelegramMessage represents a message in Telegram.
type TelegramMessage struct {
	MessageID      int              `json:"message_id"`
	From           TelegramUser     `json:"from"`
	Chat           TelegramChat     `json:"chat"`
	Date           int              `json:"date"`
	Text           string           `json:"text"`
	Entities       []TelegramEntity `json:"entities,omitempty"`
	ReplyToMessage *TelegramMessage `json:"reply_to_message,omitempty"`
}

// TelegramCallbackQuery represents a callback query from an inline keyboard.
type TelegramCallbackQuery struct {
	ID      string           `json:"id"`
	From    TelegramUser     `json:"from"`
	Message *TelegramMessage `json:"message,omitempty"`
	Data    string           `json:"data,omitempty"`
	// You can include other fields as needed based on Telegram's API.
}

// TelegramUser represents a user in Telegram.
type TelegramUser struct {
	ID           int    `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

// TelegramChat represents a chat in Telegram.
type TelegramChat struct {
	ID                          int64  `json:"id"`
	Type                        string `json:"type"`
	Title                       string `json:"title,omitempty"`
	FirstName                   string `json:"first_name,omitempty"`
	Username                    string `json:"username,omitempty"`
	AllMembersAreAdministrators bool   `json:"all_members_are_administrators,omitempty"`
	// You can include other fields as needed based on Telegram's API.
}

// TelegramEntity represents an entity in a Telegram message.
type TelegramEntity struct {
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	Type   string `json:"type"`
}

// KnowledgeEntryResponse represents a knowledge base entry.
type KnowledgeEntryResponse struct {
	ID                uint   `json:"id"`
	KBNumber          uint   `json:"kb_number"`
	BodyOfWater       string `json:"body_of_water"`
	FishSpecies       string `json:"fish_species"`
	WaterType         string `json:"water_type"`
	QuestionTemplate  string `json:"question_template"`
	Answer            string `json:"answer"`
	Category          string `json:"category"`
	SubCategory       string `json:"sub_category"`
	HelpfulRatings    int    `json:"helpful_ratings"`
	NotHelpfulRatings int    `json:"not_helpful_ratings"`
}

// OpenAIMessage represents a message in the OpenAI conversation.
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIQuery represents the payload sent to OpenAI's API.
type OpenAIQuery struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

// OpenAIResponse represents the response received from OpenAI's API.
type OpenAIResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []OpenAIResponseChoice `json:"choices"`
	Usage   OpenAIUsage            `json:"usage"`
}

// OpenAIResponseChoice represents a single choice in OpenAI's response.
type OpenAIResponseChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIUsage represents token usage information from OpenAI's response.
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// QueryParameters represents the parameters for querying the knowledge base.
type QueryParameters struct {
	BodyOfWater string `json:"body_of_water,omitempty"`
	FishSpecies string `json:"fish_species,omitempty"`
	WaterType   string `json:"water_type,omitempty"`
	Category    string `json:"category,omitempty"`
	Query       string `json:"query,omitempty"`
}

// FacebookUpdate represents the structure of a Facebook Messenger webhook event.
type FacebookUpdate struct {
	Object string          `json:"object"`
	Entry  []FacebookEntry `json:"entry"`
}

// FacebookEntry represents an entry in the Facebook Messenger webhook event.
type FacebookEntry struct {
	ID        string              `json:"id"`
	Time      int64               `json:"time"`
	Messaging []FacebookMessaging `json:"messaging"`
}

// FacebookMessaging represents a messaging event from a user.
type FacebookMessaging struct {
	Sender    FacebookUser      `json:"sender"`
	Recipient FacebookUser      `json:"recipient"`
	Timestamp int64             `json:"timestamp"`
	Message   *FacebookMessage  `json:"message,omitempty"`
	Postback  *FacebookPostback `json:"postback,omitempty"`
}

// FacebookUser represents a user in Facebook Messenger.
type FacebookUser struct {
	ID string `json:"id"`
}

// FacebookMessage represents a message sent by the user.
type FacebookMessage struct {
	Mid  string `json:"mid"`
	Text string `json:"text"`
}

// FacebookPostback represents a postback action (e.g., button click).
type FacebookPostback struct {
	Title   string `json:"title"`
	Payload string `json:"payload"`
}
