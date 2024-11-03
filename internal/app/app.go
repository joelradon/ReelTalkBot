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
	"ReelTalkBot-Go/internal/knowledgebase"
	"ReelTalkBot-Go/internal/types"
	"ReelTalkBot-Go/internal/usage"
	"ReelTalkBot-Go/internal/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

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
	logMutex             sync.Mutex                      // Mutex to ensure thread-safe logging
	KnowledgeBaseURL     string                          // URL of the Knowledge Base API
	KnowledgeBaseAPIKey  string                          // API Key for authenticating with Knowledge Base
	ConversationContexts *conversation.ConversationCache // Cache for maintaining conversation contexts
	KnowledgeBaseClient  *knowledgebase.KnowledgeBaseClient
	APIHandler           *api.APIHandler // Added APIHandler
}

// NewApp initializes the App with configurations from environment variables.
func NewApp() *App {
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
		TelegramToken:  os.Getenv("TELEGRAM_TOKEN"),
		OpenAIKey:      os.Getenv("OPENAI_KEY"),
		OpenAIEndpoint: os.Getenv("OPENAI_ENDPOINT"),
		BotUsername:    os.Getenv("BOT_USERNAME"),
		Cache:          cache.NewCache(),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		RateLimiter:          rate.NewLimiter(rate.Every(time.Second), 5),
		S3BucketName:         os.Getenv("BUCKET_NAME"),
		S3Endpoint:           os.Getenv("AWS_ENDPOINT_URL_S3"),
		S3Region:             os.Getenv("AWS_REGION"),
		S3Client:             s3Client,
		UsageCache:           usage.NewUsageCache(),
		NoLimitUsers:         noLimitUsers,
		KnowledgeBaseActive:  knowledgeBaseActive,
		KnowledgeBaseURL:     os.Getenv("KNOWLEDGE_BASE_TRAIN_ENDPOINT"),
		KnowledgeBaseAPIKey:  os.Getenv("API_KEY"),
		ConversationContexts: conversation.NewConversationCache(),
		APIHandler:           apiHandler, // Initialize APIHandler
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

// PrepareFinalMessage formats the response message from OpenAI for sending to Telegram.
func (a *App) PrepareFinalMessage(responseText string) string {
	// Implement your message preparation logic here.
	return responseText // Example return; modify as needed.
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
		if err := a.sendMessage(chatID, limitMsg, messageID); err != nil {
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
	if a.KnowledgeBaseActive && a.KnowledgeBaseClient != nil {
		bodyOfWater, fishSpecies, waterType, category := utils.IdentifyTaxonomyCategories(userQuestion)
		entries, err := a.KnowledgeBaseClient.GetKnowledgeEntries(types.QueryParameters{
			BodyOfWater: bodyOfWater,
			FishSpecies: fishSpecies,
			WaterType:   waterType,
			Category:    category,
			Query:       userQuestion,
		})
		if err != nil {
			log.Printf("Knowledge Base query failed: %v", err)
			// Fallback to OpenAI if Knowledge Base fails
			responseText, err := a.APIHandler.QueryOpenAIWithMessages(messages)
			if err != nil {
				log.Printf("OpenAI query failed after Knowledge Base failure: %v", err)
				return err
			}

			responseTime := 0 // Response time not measured for fallback
			finalMessage := a.PrepareFinalMessage(responseText)

			// Append assistant's response to messages
			messages = append(messages, types.OpenAIMessage{Role: "assistant", Content: responseText})

			// Update conversation context
			messagesJSON, _ := json.Marshal(messages)
			a.ConversationContexts.Set(conversationKey, string(messagesJSON))

			if err := a.sendMessage(chatID, finalMessage, messageID); err != nil {
				log.Printf("Failed to send OpenAI fallback message to Telegram: %v", err)
				return err
			}

			// Log the interaction in S3 with empty response time
			a.logToS3(userID, username, userQuestion, keywords, keywordSummary, categories, fmt.Sprintf("%d ms", responseTime), isRateLimited)
			return nil
		}

		if len(entries) > 0 {
			// Build knowledge response
			var responseBuilder strings.Builder
			currentCategory := ""
			for _, entry := range entries {
				if entry.Category != currentCategory {
					currentCategory = entry.Category
					responseBuilder.WriteString(fmt.Sprintf("\n### %s\n", currentCategory))
				}
				responseBuilder.WriteString(fmt.Sprintf("- **%s**: %s\n", entry.QuestionTemplate, entry.Answer))
			}
			knowledgeResponse = responseBuilder.String()

			// Append assistant's response to messages
			messages = append(messages, types.OpenAIMessage{Role: "assistant", Content: knowledgeResponse})

			// Send the Knowledge Base response
			if err := a.sendMessage(chatID, knowledgeResponse, messageID); err != nil {
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

	// Fallback to OpenAI if Knowledge Base is inactive or no response
	startTime := time.Now()

	responseText, err := a.APIHandler.QueryOpenAIWithMessages(messages)
	if err != nil {
		log.Printf("OpenAI query failed: %v", err)
		return err
	}

	responseTime := time.Since(startTime).Milliseconds()
	finalMessage := a.PrepareFinalMessage(responseText)

	// Append assistant's response to messages
	messages = append(messages, types.OpenAIMessage{Role: "assistant", Content: responseText})

	// Update conversation context
	messagesJSON, _ := json.Marshal(messages)
	a.ConversationContexts.Set(conversationKey, string(messagesJSON))

	if err := a.sendMessage(chatID, finalMessage, messageID); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
		return err
	}

	// Log the interaction in S3 with keyword summary, categories, and response time
	a.logToS3(userID, username, userQuestion, keywords, keywordSummary, categories, fmt.Sprintf("%d ms", responseTime), isRateLimited)
	return nil
}

// HandleCommand processes Telegram commands such as /learn.
func (a *App) HandleCommand(message *types.TelegramMessage, userID int, username string) (string, error) {
	command := strings.SplitN(message.Text, " ", 2)[0]

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
		parts := strings.SplitN(message.Text, " ", 2)
		if len(parts) < 2 {
			msg := "Please provide the training data.\nUsage: /learn [Category]: [SubCategory]: [Your Information]\n\nExample: /learn Gear Selection: Fly Fishing: Information about choosing the right fly fishing gear."
			a.sendMessage(message.Chat.ID, msg, message.MessageID)
			return "Incomplete training command", nil
		}
		trainingData := parts[1]

		// Validate and parse training data
		category, err := a.parseTrainingData(trainingData)
		if err != nil {
			msg := fmt.Sprintf("Invalid training data format: %v\n\nUsage: /learn [Category]: [SubCategory]: [Your Information]\n\nExample: /learn Gear Selection: Fly Fishing: Information about choosing the right fly fishing gear.", err)
			a.sendMessage(message.Chat.ID, msg, message.MessageID)
			// Send error message to user if they are in NoLimitUsers
			if _, isNoLimitUser := a.NoLimitUsers[userID]; isNoLimitUser {
				a.sendMessage(message.Chat.ID, "Failed to process your training data. Please ensure it follows the correct format.", message.MessageID)
			}
			return "Invalid training data", err
		}

		// Send training data to the knowledge base microservice
		err = a.sendTrainingData(trainingData)
		if err != nil {
			log.Printf("Failed to send training data: %v", err)
			msg := "Failed to train the knowledge base. Please ensure your data is correctly formatted."
			a.sendMessage(message.Chat.ID, msg, message.MessageID)
			// Send error message to user if they are in NoLimitUsers
			if _, isNoLimitUser := a.NoLimitUsers[userID]; isNoLimitUser {
				a.sendMessage(message.Chat.ID, "An error occurred while processing your training data.", message.MessageID)
			}
			return "Training failed", err
		}

		msg := fmt.Sprintf("Training data received and is being processed under category: %s.", category)
		a.sendMessage(message.Chat.ID, msg, message.MessageID)
		return "Training command processed", nil

	default:
		msg := "Unknown command."
		a.sendMessage(message.Chat.ID, msg, message.MessageID)
		return "Unknown command", nil
	}
}

// parseTrainingData validates and extracts the category from training data.
func (a *App) parseTrainingData(data string) (string, error) {
	// Expected format: [Category]: [SubCategory]: [Training Information]
	// Example: "Gear Selection: Fly Fishing: Information about choosing the right fly fishing gear."
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 3 {
		return "", fmt.Errorf("training data should be in the format 'Category: SubCategory: Information'")
	}
	category := strings.TrimSpace(parts[0])
	subCategory := strings.TrimSpace(parts[1])
	information := strings.TrimSpace(parts[2])

	if category == "" || subCategory == "" || information == "" {
		return "", fmt.Errorf("category, subcategory, and information must be provided")
	}

	// Optionally, further validation can be added here

	return category, nil
}

// sendTrainingData sends training data to the knowledge base microservice.
func (a *App) sendTrainingData(data string) error {
	// Define the knowledge base microservice endpoint
	trainingEndpoint := a.KnowledgeBaseURL
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

// sendMessage sends a plain text message to a Telegram chat
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

// SummarizeToLength trims the text to the specified maximum length
func (a *App) SummarizeToLength(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength]
}

// logToS3 logs user interactions to an S3 bucket with details about rate limiting and usage.
// Added columns for keyword summary and categories.
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
