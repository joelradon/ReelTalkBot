// internal/app/app.go

package app

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"ReelTalkBot-Go/internal/api"
	"ReelTalkBot-Go/internal/cache"
	"ReelTalkBot-Go/internal/conversation"
	"ReelTalkBot-Go/internal/handlers"
	"ReelTalkBot-Go/internal/knowledgebase"
	"ReelTalkBot-Go/internal/telegram"
	"ReelTalkBot-Go/internal/types"
	"ReelTalkBot-Go/internal/usage"
	"ReelTalkBot-Go/internal/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

// Ensure App implements handlers.MessageProcessor
var _ handlers.MessageProcessor = (*App)(nil)

// App represents the main application with all necessary configurations and dependencies.
type App struct {
	TelegramToken        string
	OpenAIKey            string
	OpenAIEndpoint       string
	BotUsername          string
	Cache                *cache.Cache
	HTTPClient           *http.Client
	RateLimiter          *rate.Limiter
	S3BucketName         string
	S3Endpoint           string
	S3Region             string
	S3Client             *s3.S3
	UsageCache           *usage.UsageCache
	NoLimitUsers         map[int]struct{}                // Map of user IDs with no rate limits
	KnowledgeBaseActive  bool                            // Indicates if the knowledge base is active
	isKnowledgeBaseDown  bool                            // Flag to indicate if Knowledge Base is down
	logMutex             sync.Mutex                      // Mutex to ensure thread-safe logging
	KnowledgeBaseURL     string                          // URL of the Knowledge Base API
	KnowledgeBaseAPIKey  string                          // API Key for authenticating with Knowledge Base
	ConversationContexts *conversation.ConversationCache // Cache for maintaining conversation contexts
	KnowledgeBaseClient  *knowledgebase.KnowledgeBaseClient
	APIHandler           *api.APIHandler           // APIHandler for OpenAI interactions
	promptMap            map[string]string         // Mapping of callback_data to prompts
	TelegramHandler      *telegram.TelegramHandler // TelegramHandler for message processing
}

// NewApp initializes the App with configurations from environment variables.
func NewApp() *App {
	// Load environment variables from .env file if present
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found. Proceeding with environment variables.")
	}

	// Parse NO_LIMIT_USERS
	noLimitUsersRaw := os.Getenv("NO_LIMIT_USERS")
	noLimitUsers := parseNoLimitUsers(noLimitUsersRaw)

	// Parse KNOWLEDGE_BASE (default to OFF)
	knowledgeBaseEnv := strings.ToUpper(os.Getenv("KNOWLEDGE_BASE"))
	knowledgeBaseActive := false
	if knowledgeBaseEnv == "ON" {
		knowledgeBaseActive = true
	}

	// Initialize AWS S3 Client
	sess, err := session.NewSession(&aws.Config{
		Region:   aws.String(os.Getenv("AWS_REGION")),
		Endpoint: aws.String(os.Getenv("AWS_ENDPOINT_URL_S3")),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	s3Client := s3.New(sess)

	// Initialize APIHandler for OpenAI
	apiHandler := api.NewAPIHandler(os.Getenv("OPENAI_KEY"), os.Getenv("OPENAI_ENDPOINT"))

	app := &App{
		TelegramToken:        os.Getenv("TELEGRAM_TOKEN"),
		OpenAIKey:            os.Getenv("OPENAI_KEY"),
		OpenAIEndpoint:       os.Getenv("OPENAI_ENDPOINT"),
		BotUsername:          os.Getenv("BOT_USERNAME"),
		Cache:                cache.NewCache(),
		HTTPClient:           &http.Client{Timeout: 15 * time.Second},
		RateLimiter:          rate.NewLimiter(rate.Every(time.Second), 5),
		S3BucketName:         os.Getenv("BUCKET_NAME"),
		S3Endpoint:           os.Getenv("AWS_ENDPOINT_URL_S3"),
		S3Region:             os.Getenv("AWS_REGION"),
		S3Client:             s3Client,
		UsageCache:           usage.NewUsageCache(),
		NoLimitUsers:         noLimitUsers,
		KnowledgeBaseActive:  knowledgeBaseActive,
		isKnowledgeBaseDown:  false, // Initialize as not down
		KnowledgeBaseURL:     os.Getenv("KNOWLEDGE_BASE_TRAIN_ENDPOINT"),
		KnowledgeBaseAPIKey:  os.Getenv("API_KEY"),
		ConversationContexts: conversation.NewConversationCache(),
		APIHandler:           apiHandler, // Initialize APIHandler
		promptMap:            make(map[string]string),
	}

	if app.BotUsername == "" {
		log.Println("Warning: BOT_USERNAME environment variable is missing. The bot will not respond to mentions.")
	} else {
		log.Printf("Bot username is set to: %s", app.BotUsername)
	}

	// Initialize Knowledge Base Client
	if app.KnowledgeBaseActive && app.KnowledgeBaseURL != "" && app.KnowledgeBaseAPIKey != "" {
		app.KnowledgeBaseClient = knowledgebase.NewKnowledgeBaseClient(app.KnowledgeBaseURL, app.KnowledgeBaseAPIKey)
	}

	// Initialize TelegramHandler with the App as the MessageProcessor
	app.TelegramHandler = telegram.NewTelegramHandler(app)

	// Start Health Check Routine
	app.StartHealthCheckRoutine(30 * time.Second)

	return app
}

// parseNoLimitUsers parses the NO_LIMIT_USERS environment variable into a map of user IDs.
func parseNoLimitUsers(raw string) map[int]struct{} {
	userMap := make(map[int]struct{})
	ids := strings.Split(raw, ",")
	for _, idStr := range ids {
		idStr = strings.Trim(idStr, " \"") // Remove spaces and quotes
		if id, err := strconv.Atoi(idStr); err == nil {
			userMap[id] = struct{}{}
		}
	}
	return userMap
}

// ProcessMessage processes a user's message, queries Knowledge Base or OpenAI, sends the response, and logs the interaction.
func (a *App) ProcessMessage(chatID int64, userID int, username, userQuestion string, messageID int) error {
	// Rate limit check
	isNoLimitUser := false
	if _, ok := a.NoLimitUsers[userID]; ok {
		isNoLimitUser = true
	}

	isRateLimited := false
	if !isNoLimitUser && !a.UsageCache.CanUserChat(userID) {
		isRateLimited = true
		// Calculate remaining time until limit reset
		timeRemaining := a.UsageCache.TimeUntilLimitReset(userID)
		minutes := int(timeRemaining.Minutes())
		seconds := int(timeRemaining.Seconds()) % 60

		limitMsg := fmt.Sprintf(
			"Thanks for using ReelTalkBot. We restrict to 10 messages per 10 minutes to keep costs low and allow everyone to use the tool. Please try again in %d minutes and %d seconds.",
			minutes, seconds,
		)
		if err := a.SendMessage(chatID, limitMsg, messageID); err != nil {
			log.Printf("Failed to send rate limit message to Telegram: %v", err)
		}

		// Extract keywords from userQuestion
		keywords := utils.ExtractKeywords(userQuestion)

		// Log the attempt to S3 with empty keyword summary, categories, and response time
		a.logToS3(userID, username, userQuestion, keywords, "", "", "", isRateLimited)
		return fmt.Errorf("user rate limited")
	}

	a.UsageCache.AddUsage(userID)

	// Extract keywords from userQuestion
	keywords := utils.ExtractKeywords(userQuestion)

	// Determine keyword summary and categories
	keywordSummary := strings.Join(keywords, ", ")
	categories := utils.DetermineCategories(keywords)

	// Maintain conversation context
	conversationKey := fmt.Sprintf("user_%d", userID)
	var messages []types.OpenAIMessage
	if history, exists := a.ConversationContexts.Get(conversationKey); exists {
		if err := json.Unmarshal([]byte(history), &messages); err != nil {
			log.Printf("Failed to unmarshal conversation history: %v", err)
			messages = []types.OpenAIMessage{
				{Role: "system", Content: "You are a helpful assistant specialized in fishing techniques and knowledge."},
			}
		}
	} else {
		// Initialize with system prompt
		messages = []types.OpenAIMessage{
			{Role: "system", Content: "You are a helpful assistant specialized in fishing techniques and knowledge."},
		}
	}

	// Append the new user message
	messages = append(messages, types.OpenAIMessage{Role: "user", Content: userQuestion})

	// Query Knowledge Base first
	var knowledgeResponse string
	var kbEntry *types.KnowledgeEntryResponse
	if a.KnowledgeBaseActive && a.KnowledgeBaseClient != nil && !a.isKnowledgeBaseDown {
		bodyOfWater, fishSpecies, waterType, category := utils.IdentifyTaxonomyCategories(userQuestion)
		entries, err := a.KnowledgeBaseClient.GetKnowledgeEntries(context.Background(), types.QueryParameters{
			BodyOfWater: bodyOfWater,
			FishSpecies: fishSpecies,
			WaterType:   waterType,
			Category:    category,
			Query:       userQuestion,
		})
		if err != nil {
			log.Printf("Knowledge Base query failed: %v", err)
			a.isKnowledgeBaseDown = true // Mark KB as down
			// Fallback to OpenAI if Knowledge Base fails
			responseText, err := a.APIHandler.QueryOpenAIWithMessages(messages)
			if err != nil {
				log.Printf("OpenAI query failed after Knowledge Base failure: %v", err)
				return err
			}

			responseTime := 0 // Response time not measured for fallback
			finalMessage := a.PrepareFinalMessage(responseText, nil)

			// Append assistant's response to messages
			messages = append(messages, types.OpenAIMessage{Role: "assistant", Content: responseText})

			// Update conversation context
			messagesJSON, _ := json.Marshal(messages)
			a.ConversationContexts.Set(conversationKey, string(messagesJSON))

			if err := a.SendMessage(chatID, finalMessage, messageID); err != nil {
				log.Printf("Failed to send OpenAI fallback message to Telegram: %v", err)
				return err
			}

			// Log the interaction in S3 with empty response time
			a.logToS3(userID, username, userQuestion, keywords, keywordSummary, categories, fmt.Sprintf("%d ms", responseTime), isRateLimited)
			return nil
		}

		if len(entries) > 0 {
			// Assuming the first entry is the most relevant
			kbEntry = &types.KnowledgeEntryResponse{
				ID:                entries[0].ID,
				KBNumber:          entries[0].KBNumber,
				BodyOfWater:       entries[0].BodyOfWater,
				FishSpecies:       entries[0].FishSpecies,
				WaterType:         entries[0].WaterType,
				QuestionTemplate:  entries[0].QuestionTemplate,
				Answer:            entries[0].Answer,
				Category:          entries[0].Category,
				SubCategory:       entries[0].SubCategory,
				HelpfulRatings:    entries[0].HelpfulRatings,
				NotHelpfulRatings: entries[0].NotHelpfulRatings,
			}

			knowledgeResponse = fmt.Sprintf("- **%s**: %s\n", kbEntry.QuestionTemplate, kbEntry.Answer)

			// Append assistant's response to messages
			messages = append(messages, types.OpenAIMessage{Role: "assistant", Content: knowledgeResponse})

			// Send the Knowledge Base response with KB details
			finalMessage := a.PrepareFinalMessage(knowledgeResponse, kbEntry)
			if err := a.SendMessage(chatID, finalMessage, messageID); err != nil {
				log.Printf("Failed to send Knowledge Base message to Telegram: %v", err)
				return err
			}

			// Update conversation context
			messagesJSON, _ := json.Marshal(messages)
			a.ConversationContexts.Set(conversationKey, string(messagesJSON))

			// Log the interaction in S3 with empty response time
			a.logToS3(userID, username, userQuestion, keywords, keywordSummary, categories, "", isRateLimited)
			return nil
		}
	}

	// Fallback to OpenAI if Knowledge Base is inactive, down, or no response
	startTime := time.Now()

	responseText, err := a.APIHandler.QueryOpenAIWithMessages(messages)
	if err != nil {
		log.Printf("OpenAI query failed: %v", err)
		return err
	}

	responseTime := time.Since(startTime).Milliseconds()
	finalMessage := a.PrepareFinalMessage(responseText, nil)

	// Append assistant's response to messages
	messages = append(messages, types.OpenAIMessage{Role: "assistant", Content: responseText})

	// Update conversation context
	messagesJSON, _ := json.Marshal(messages)
	a.ConversationContexts.Set(conversationKey, string(messagesJSON))

	if err := a.SendMessage(chatID, finalMessage, messageID); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
		return err
	}

	// Log the interaction in S3 with keyword summary, categories, and response time
	a.logToS3(userID, username, userQuestion, keywords, keywordSummary, categories, fmt.Sprintf("%d ms", responseTime), isRateLimited)
	return nil
}

// HandleCommand processes Telegram commands such as /learn, /rate, and /help.
func (a *App) HandleCommand(message *types.TelegramMessage, userID int, username string) (string, error) {
	commandParts := strings.SplitN(message.Text, " ", 2)
	command := commandParts[0]

	switch command {
	case "/learn", "/learn@ReelTalkBot": // Added handling for /learn@ReelTalkBot
		// Check if the knowledge base feature is active
		if !a.KnowledgeBaseActive {
			msg := "Knowledge base training is currently disabled."
			a.SendMessage(message.Chat.ID, msg, message.MessageID)
			return "", nil
		}

		// Check if the user is authorized
		if _, ok := a.NoLimitUsers[userID]; !ok {
			msg := "You are not authorized to train the knowledge base."
			a.SendMessage(message.Chat.ID, msg, message.MessageID)
			return "", nil
		}

		// Extract training data from the message
		if len(commandParts) < 2 {
			msg := "Please provide the training data.\nUsage: /learn [Category]: [SubCategory]: [Your Information]\n\nExample: /learn Techniques: Fly Fishing: Information about choosing the right fly fishing gear."
			a.SendMessage(message.Chat.ID, msg, message.MessageID)
			return "", nil
		}
		trainingData := commandParts[1]

		// Validate and parse training data
		category, subCategory, information, err := a.parseTrainingData(trainingData)
		if err != nil {
			msg := fmt.Sprintf("Invalid training data format: %v\n\nUsage: /learn [Category]: [SubCategory]: [Your Information]\n\nExample: /learn Gear Selection: Fly Fishing: Information about choosing the right fly fishing gear.", err)
			a.SendMessage(message.Chat.ID, msg, message.MessageID)
			return "", nil
		}

		// Send training data to the knowledge base microservice
		err = a.sendTrainingData(category, subCategory, information)
		if err != nil {
			log.Printf("Failed to send training data: %v", err)
			msg := "Failed to train the knowledge base. Please ensure your data is correctly formatted."
			a.SendMessage(message.Chat.ID, msg, message.MessageID)
			return "", nil
		}

		msg := fmt.Sprintf("Training data received and is being processed under category: %s, subcategory: %s.", category, subCategory)
		a.SendMessage(message.Chat.ID, msg, message.MessageID)
		return "", nil

	case "/rate":
		// Handle rating of KB articles
		if len(commandParts) < 2 {
			msg := "Please provide the KB number and your rating.\nUsage: /rate [KB Number] [Helpful/Not Helpful]\n\nExample: /rate 123 Helpful"
			a.SendMessage(message.Chat.ID, msg, message.MessageID)
			return "", nil
		}
		ratingData := commandParts[1]
		parts := strings.SplitN(ratingData, " ", 2)
		if len(parts) < 2 {
			msg := "Invalid rating format.\nUsage: /rate [KB Number] [Helpful/Not Helpful]"
			a.SendMessage(message.Chat.ID, msg, message.MessageID)
			return "", nil
		}
		kbNumberStr := parts[0]
		rating := strings.ToLower(strings.TrimSpace(parts[1]))
		kbNumber, err := strconv.Atoi(kbNumberStr)
		if err != nil {
			msg := "KB Number must be a valid integer."
			a.SendMessage(message.Chat.ID, msg, message.MessageID)
			return "", nil
		}
		if rating != "helpful" && rating != "not helpful" {
			msg := "Rating must be either 'Helpful' or 'Not Helpful'."
			a.SendMessage(message.Chat.ID, msg, message.MessageID)
			return "", nil
		}

		// Update the KB entry with the rating
		err = a.KnowledgeBaseClient.UpdateKnowledgeEntryRating(kbNumber, strings.Title(rating))
		if err != nil {
			log.Printf("Failed to update KB entry rating: %v", err)
			msg := "Failed to update your rating. Please try again later."
			a.SendMessage(message.Chat.ID, msg, message.MessageID)
			return "", nil
		}

		msg := "Thank you for your feedback!"
		a.SendMessage(message.Chat.ID, msg, message.MessageID)
		return "", nil

	case "/help", "/help@ReelTalkBot": // Added handling for /help@ReelTalkBot
		// Handle /help command to provide detailed usage instructions and example prompts
		helpMessage := "**ReelTalkBot Help**\n\n" +
			"Welcome to ReelTalkBot! Here's how you can use this bot effectively for your fishing research:\n\n" +
			"1. **/learn [Category]: [SubCategory]: [Your Information]**\n" +
			"   - Train the bot's Knowledge Base with new information.\n" +
			"   - **Example:** `/learn Techniques: Fly Fishing: Information about choosing the right fly fishing gear.`\n\n" +
			"2. **/rate [KB Number] [Helpful/Not Helpful]**\n" +
			"   - Provide feedback on Knowledge Base articles to help improve accuracy.\n" +
			"   - **Example:** `/rate 123 Helpful`\n\n" +
			"3. **Effective AI Prompts:**\n" +
			"   - Use well-structured prompts to get detailed and accurate responses.\n\n" +
			"   **Really Good Prompts:**\n" +
			"- \"How do I fish a live shrimp on a free line near mangroves in the Indian River Lagoon. What are some the advantages and disadvantages?\"\n" +
			"- \"What are the rules according to DEC for Upper Fly Zone in Altmar. Please list regulations with link to DEC website\"\n" +
			"- \"What considerations should I make when choosing nymph size and color when fishing small rivers in the Appalachian Mountains? I will be fishing specifically for rainbow trout\"\n\n" +
			"   **Medium Quality Prompts:**\n" +
			"- \"How do I freeline a live shrimp for redfish?\"\n" +
			"- \"What are some of the regulations for Salmon River in NY?\"\n" +
			"- \"What should throw when nymphing for rainbow trout?\"\n\n" +
			"   **Poor Prompts:**\n" +
			"- \"How do I fish shrimp?\"\n" +
			"- \"What are the rules for fishing in NY?\"\n" +
			"- \"What nymph color should I pick?\"\n\n" +
			"*Click on the buttons below to use these example prompts:*"

		// Define example prompts with concise callback_data identifiers
		examplePrompts := []struct {
			Label      string
			Prompt     string
			CallbackID string
		}{
			{"Excellent Prompt - How do I fish free lined shrimp in the Indian River Lagoon", "How do I fish a live shrimp on a free line near mangroves in the Indian River Lagoon. What are some the advantages and disadvantages?", "prompt_1"},
			{"Excellent Prompt - Give me regulations for Altmar fly fishing area on the Salmon River", "What are the rules according to DEC for Upper Fly Zone in Altmar. Please list regulations with link to DEC website", "prompt_2"},
			{"Excellent Prompt - What size and color nymph should I use for rainbow trout in Applachian Mountains", "What considerations should I make when choosing nymph size and color when fishing small rivers in the Appalachian Mountains? I will be fishing specifically for rainbow trout", "prompt_3"},
		}

		// Populate promptMap with callback_id to prompt mapping
		for _, prompt := range examplePrompts {
			a.promptMap[prompt.CallbackID] = prompt.Prompt
		}

		// Construct inline keyboard buttons with concise callback_data
		var inlineKeyboard [][]map[string]string
		for _, prompt := range examplePrompts {
			button := map[string]string{
				"text":          prompt.Label,
				"callback_data": prompt.CallbackID, // Use concise identifier
			}
			inlineKeyboard = append(inlineKeyboard, []map[string]string{button})
		}

		// Marshal the inline keyboard to JSON
		keyboard := map[string]interface{}{
			"inline_keyboard": inlineKeyboard,
		}
		keyboardJSON, err := json.Marshal(keyboard)
		if err != nil {
			log.Printf("Failed to marshal inline keyboard: %v", err)
			a.SendMessage(message.Chat.ID, "Failed to create help menu.", message.MessageID)
			return "", nil
		}

		// Append buttons to the help message
		helpMessage += "\n\n"

		// Send the help message with inline buttons
		if err := a.SendMessageWithKeyboard(message.Chat.ID, helpMessage, message.MessageID, string(keyboardJSON)); err != nil {
			log.Printf("Failed to send help message: %v", err)
			return "", nil
		}

		return "", nil

	default:
		msg := "Unknown command."
		a.SendMessage(message.Chat.ID, msg, message.MessageID)
		return "", nil
	}
}

// SendMessage sends a plain text message to a Telegram chat without any keyboard.
func (a *App) SendMessage(chatID int64, text string, replyToMessageID int) error {
	return a.sendMessage(chatID, text, replyToMessageID)
}

// SendMessageWithKeyboard sends a message with an inline keyboard to a Telegram chat.
func (a *App) SendMessageWithKeyboard(chatID int64, text string, replyToMessageID int, keyboard string) error {
	return a.sendMessageWithKeyboard(chatID, text, replyToMessageID, keyboard)
}

// GetBotUsername returns the bot's username.
func (a *App) GetBotUsername() string {
	return a.BotUsername
}

// HandleCallbackQuery handles callback queries from inline keyboard buttons.
func (a *App) HandleCallbackQuery(callbackQuery *types.TelegramCallbackQuery) error {
	data := callbackQuery.Data
	chatID := callbackQuery.Message.Chat.ID
	messageID := callbackQuery.Message.MessageID

	// Retrieve the corresponding prompt using callback_data identifier
	prompt, exists := a.promptMap[data]
	if !exists {
		log.Printf("Received unknown callback_data: %s", data)
		// Optionally, send a message indicating the action is not recognized
		a.SendMessage(chatID, "Sorry, I didn't recognize that action.", messageID)
		return fmt.Errorf("unknown callback_data: %s", data)
	}

	// Acknowledge the callback to remove the loading state
	a.acknowledgeCallback(callbackQuery.ID)

	// Process the prompt as if the user sent it as a message
	userID := callbackQuery.From.ID
	username := callbackQuery.From.Username

	err := a.ProcessMessage(chatID, userID, username, prompt, messageID)
	if err != nil {
		log.Printf("Failed to process callback query: %v", err)
		return err
	}

	return nil
}

// acknowledgeCallback sends an acknowledgment to Telegram to remove the loading state on the button.
func (a *App) acknowledgeCallback(callbackID string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", a.TelegramToken)
	payload := map[string]string{
		"callback_query_id": callbackID,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal callback acknowledgment: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("Failed to create callback acknowledgment request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		log.Printf("Failed to send callback acknowledgment: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Callback acknowledgment returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

// parseTrainingData validates and extracts the category, subcategory, and information from training data.
func (a *App) parseTrainingData(data string) (string, string, string, error) {
	// Expected format: [Category]: [SubCategory]: [Training Information]
	// Example: "Gear Selection: Fly Fishing: Information about choosing the right fly fishing gear."
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("training data should be in the format 'Category: SubCategory: Information'")
	}
	category := strings.TrimSpace(parts[0])
	subCategory := strings.TrimSpace(parts[1])
	information := strings.TrimSpace(parts[2])

	if category == "" || subCategory == "" || information == "" {
		return "", "", "", fmt.Errorf("category, subcategory, and information must be provided")
	}

	// Optionally, further validation can be added here

	return category, subCategory, information, nil
}

// sendTrainingData sends training data to the knowledge base microservice.
func (a *App) sendTrainingData(category, subCategory, information string) error {
	// Define the knowledge base microservice endpoint
	trainingEndpoint := a.KnowledgeBaseURL
	if trainingEndpoint == "" {
		return fmt.Errorf("KNOWLEDGE_BASE_TRAIN_ENDPOINT not set")
	}

	// Prepare the payload
	payload := map[string]string{
		"body_of_water":     "General",
		"fish_species":      "General",
		"water_type":        "General",
		"question_template": category + ": " + subCategory,
		"answer":            information,
		"category":          category,
		"sub_category":      subCategory,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal training data: %w", err)
	}

	// Create a new request with API Key
	req, err := http.NewRequest("POST", trainingEndpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create training request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", a.KnowledgeBaseAPIKey)

	// Use context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req = req.WithContext(ctx)

	resp, err := a.HTTPClient.Do(req)
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

// sendMessage sends a plain text message to a Telegram chat without any keyboard.
func (a *App) sendMessage(chatID int64, text string, replyToMessageID int) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", a.TelegramToken)
	payload := map[string]interface{}{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,
		"parse_mode":               "Markdown",
	}

	if replyToMessageID != 0 {
		payload["reply_to_message_id"] = replyToMessageID
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s - %s", resp.Status, string(bodyBytes))
	}

	return nil
}

// sendMessageWithKeyboard sends a message with an inline keyboard to a Telegram chat.
func (a *App) sendMessageWithKeyboard(chatID int64, text string, replyToMessageID int, keyboard string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", a.TelegramToken)
	payload := map[string]interface{}{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,
		"parse_mode":               "Markdown",
		"reply_markup":             keyboard,
	}

	if replyToMessageID != 0 {
		payload["reply_to_message_id"] = replyToMessageID
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s - %s", resp.Status, string(bodyBytes))
	}

	return nil
}

// logToS3 logs user interactions to an S3 bucket with details about rate limiting and usage.
// Added columns for keyword summary, categories, response time, and ratings.
func (a *App) logToS3(userID int, username, userPrompt string, keywords []string, keywordSummary, categories, responseTime string, isRateLimited bool) {
	a.logMutex.Lock()
	defer a.logMutex.Unlock()

	// Prepare the record with new fields
	record := []string{
		fmt.Sprintf("%d", userID),
		username,
		userPrompt,
		strings.Join(keywords, " "), // Concatenate keywords
		keywordSummary,
		categories,
		responseTime,
		fmt.Sprintf("Rate limited: %t", isRateLimited),
	}

	// Define S3 bucket and object key
	bucketName := a.S3BucketName
	objectKey := "logs/telegram_logs.csv"

	// Download the existing CSV from S3
	resp, err := a.S3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})

	var existingData [][]string
	if err == nil {
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil && len(bodyBytes) > 0 {
			reader := csv.NewReader(bytes.NewReader(bodyBytes))
			existingData, err = reader.ReadAll()
			if err != nil {
				log.Printf("Failed to parse existing CSV: %v", err)
				existingData = [][]string{}
			}
		}
	} else {
		log.Printf("Failed to get existing CSV from S3: %v. A new CSV will be created.", err)
	}

	// If the CSV is empty, add headers
	if len(existingData) == 0 {
		headers := []string{
			"userID",
			"username",
			"prompt",
			"keywords",
			"keyword_summary",
			"categories",
			"response_time",
			"is_rate_limited",
		}
		existingData = append(existingData, headers)
	}

	// Append the new record
	existingData = append(existingData, record)

	// Write all records to a buffer
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.WriteAll(existingData); err != nil {
		log.Printf("Failed to write CSV data to buffer: %v", err)
		return
	}

	// Upload the updated CSV back to S3
	_, err = a.S3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(buf.Bytes()),
	})

	if err != nil {
		log.Printf("Failed to upload updated CSV to S3: %v", err)
	} else {
		log.Printf("Successfully appended log data to S3 CSV")
	}
}

// HealthCheck verifies if the Knowledge Base is reachable.
func (a *App) HealthCheck() {
	if !a.KnowledgeBaseActive || a.KnowledgeBaseClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt to fetch a known KB entry or perform a lightweight request
	_, err := a.KnowledgeBaseClient.GetKnowledgeEntries(ctx, types.QueryParameters{
		Query: "health_check",
	})

	if err != nil {
		if !a.isKnowledgeBaseDown {
			log.Printf("Knowledge Base is down: %v", err)
		}
		a.isKnowledgeBaseDown = true
	} else {
		if a.isKnowledgeBaseDown {
			log.Println("Knowledge Base is back online.")
		}
		a.isKnowledgeBaseDown = false
	}
}

// StartHealthCheckRoutine starts a goroutine to periodically check the Knowledge Base's health.
func (a *App) StartHealthCheckRoutine(interval time.Duration) {
	go func() {
		for {
			a.HealthCheck()
			time.Sleep(interval)
		}
	}()
}

// HandleUpdate processes incoming Telegram updates (messages and callback queries).
func (a *App) HandleUpdate(update *types.TelegramUpdate) {
	if update.CallbackQuery != nil {
		// Handle callback queries
		err := a.HandleCallbackQuery(update.CallbackQuery)
		if err != nil {
			log.Printf("Error handling callback query: %v", err)
		}
		return
	}

	// Delegate message processing to TelegramHandler
	response, err := a.TelegramHandler.HandleTelegramMessage(update)
	if err != nil {
		log.Printf("Error handling Telegram message: %v", err)
	}

	// Optionally, send a response back if needed
	if response != "" && update.Message != nil {
		a.SendMessage(update.Message.Chat.ID, response, update.Message.MessageID)
	}
}

// PrepareFinalMessage formats the response message from OpenAI or Knowledge Base for sending to Telegram.
// Now includes KB number, category, and taxonomy information if available, and appends a quick "Need Help?" link.
func (a *App) PrepareFinalMessage(responseText string, kbEntry *types.KnowledgeEntryResponse) string {
	// Append KB number, category, and taxonomy information if available
	finalMessage := responseText
	if kbEntry != nil {
		finalMessage += fmt.Sprintf("\n\n**KB Number:** %d\n**Category:** %s\n**Taxonomy:** %s",
			kbEntry.KBNumber, kbEntry.Category, kbEntry.SubCategory)
	}

	// Append quick help link
	finalMessage += "\n\nNeed Help? Type /help to see how to use this bot effectively."

	return finalMessage // Example return; modify as needed.
}

// ProcessFacebookMessage processes a user's message from Facebook Messenger.
func (a *App) ProcessFacebookMessage(senderID, messageText string) (string, error) {
	// Since Facebook Messenger does not have a concept of chat IDs like Telegram, we can use senderID
	userID, err := strconv.Atoi(senderID)
	if err != nil {
		log.Printf("Invalid sender ID: %v", err)
		return "", fmt.Errorf("invalid sender ID")
	}
	username := "FacebookUser" // Placeholder, as Facebook does not provide username in messages

	// Reuse the existing ProcessMessage method with chatID set to 0
	err = a.ProcessMessage(0, userID, username, messageText, 0)
	if err != nil {
		return "", err
	}

	// Since responses are sent directly in ProcessMessage, we can return an empty string
	return "", nil
}

// HandleFacebookCommand processes commands from Facebook Messenger users.
func (a *App) HandleFacebookCommand(senderID, messageText string) (string, error) {
	userID, err := strconv.Atoi(senderID)
	if err != nil {
		log.Printf("Invalid sender ID: %v", err)
		return "", fmt.Errorf("invalid sender ID")
	}
	username := "FacebookUser"

	// Create a dummy TelegramMessage to reuse HandleCommand
	message := &types.TelegramMessage{
		Text: messageText,
		Chat: types.TelegramChat{
			ID: 0, // Not applicable for Facebook Messenger
		},
	}

	response, err := a.HandleCommand(message, userID, username)
	if err != nil {
		return "", err
	}

	return response, nil
}
