// internal/app.go

package internal

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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

// App represents the application with all necessary configurations and clients
type App struct {
	TelegramToken  string
	OpenAIKey      string
	OpenAIEndpoint string
	BotUsername    string
	Cache          *Cache
	HTTPClient     *http.Client
	RateLimiter    *rate.Limiter
	S3BucketName   string
	S3Endpoint     string
	S3Region       string
	S3Client       *s3.S3
	UsageCache     *UsageCache
	AIServiceURL   string // URL of the Python AI microservice
}

// NewApp initializes and returns a new App instance
func NewApp() *App {
	// Load environment variables from .env file, if available
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found. Proceeding with environment variables.")
	}

	app := &App{
		TelegramToken:  os.Getenv("TELEGRAM_TOKEN"),
		OpenAIKey:      os.Getenv("OPENAI_KEY"),
		OpenAIEndpoint: os.Getenv("OPENAI_ENDPOINT"),
		BotUsername:    os.Getenv("BOT_USERNAME"),
		Cache:          NewCache(),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		RateLimiter:  rate.NewLimiter(rate.Every(time.Second), 5),
		S3BucketName: os.Getenv("BUCKET_NAME"),
		S3Endpoint:   os.Getenv("AWS_ENDPOINT_URL_S3"),
		S3Region:     os.Getenv("AWS_REGION"),
		UsageCache:   NewUsageCache(),
		AIServiceURL: os.Getenv("AI_SERVICE_URL"), // Initialize AIServiceURL
	}

	// Warn if BotUsername is not set
	if app.BotUsername == "" {
		log.Println("Warning: BOT_USERNAME environment variable is missing. The bot will not respond to mentions.")
	} else {
		log.Printf("Bot username is set to: %s", app.BotUsername)
	}

	// Initialize AWS S3 client
	sess, err := session.NewSession(&aws.Config{
		Region:   aws.String(app.S3Region),
		Endpoint: aws.String(app.S3Endpoint),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	app.S3Client = s3.New(sess)
	return app
}

// ProcessMessage handles the entire flow of processing a Telegram message
func (a *App) ProcessMessage(chatID int64, userID int, username, userQuestion string, messageID int) error {
	// Rate limit check
	noLimitUsers := strings.Split(os.Getenv("NO_LIMIT_USERS"), ",")
	isNoLimitUser := false
	for _, id := range noLimitUsers {
		if id == strconv.Itoa(userID) {
			isNoLimitUser = true
			break
		}
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

		// Log the attempt to S3
		a.logToS3(userID, username, userQuestion, 0, isRateLimited)
		return fmt.Errorf("user rate limited")
	}

	// Track usage
	a.UsageCache.AddUsage(userID)

	// Step 1: Extract Keywords using AI microservice
	keywords, err := a.ExtractKeywords(userID, userQuestion)
	if err != nil {
		log.Printf("Error extracting keywords: %v", err)
		// Notify user about the error
		a.notifyUserError(chatID, "Sorry, I couldn't process your request at the moment.")
		return err
	}

	log.Printf("Extracted keywords: %v", keywords)

	// Step 2: Generate Response using AI microservice
	// Format keywords appropriately (e.g., space-separated or comma-separated)
	keywordsStr := strings.Join(keywords, " ")
	response, err := a.GenerateResponse(userID, keywordsStr)
	if err != nil {
		log.Printf("Error generating response: %v", err)
		a.notifyUserError(chatID, "Sorry, I couldn't generate a response at the moment.")
		return err
	}

	log.Printf("Generated response: %s", response)

	// Summarize the response if it's too long
	finalMessage := a.PrepareFinalMessageDetailed(response)

	// Send the response back to the user
	if err := a.sendMessage(chatID, finalMessage, messageID); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
		return err
	}

	// Log the interaction in S3, tracking if the user is rate-limited
	isRateLimited = !a.UsageCache.CanUserChat(userID)
	a.logToS3(userID, username, userQuestion, 0, isRateLimited) // Assuming responseTime is not tracked here
	return nil
}

// notifyUserError sends an error message to the user
func (a *App) notifyUserError(chatID int64, errorMessage string) {
	err := a.sendMessage(chatID, errorMessage, 0)
	if err != nil {
		log.Printf("Failed to send error message to user: %v", err)
	}
}

// PrepareFinalMessageDetailed ensures the message fits within Telegram's character limits
func (a *App) PrepareFinalMessageDetailed(responseText string) string {
	maxLength := 4096
	if len(responseText) > maxLength {
		responseText = a.SummarizeToLength(responseText, maxLength)
	}
	return responseText
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

// SummarizeToLength trims the text to the specified maximum length.
func (a *App) SummarizeToLength(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength]
}

// logToS3 logs user interactions to an S3 bucket with details about rate limiting and usage
func (a *App) logToS3(userID int, username, userPrompt string, responseTime time.Duration, isRateLimited bool) {
	// Retrieve usage count for the last 10 minutes
	queryCount := len(a.UsageCache.filterRecentMessages(userID))

	// Prepare the record with new fields
	record := []string{
		fmt.Sprintf("%d", userID),
		username,
		userPrompt,
		fmt.Sprintf("%d ms", responseTime.Milliseconds()),
		fmt.Sprintf("%d queries in last 10 mins", queryCount),
		fmt.Sprintf("Rate limited: %t", isRateLimited),
	}

	// Define S3 bucket and object key
	bucketName := a.S3BucketName
	objectKey := "logs/telegram_logs.csv"

	// Initialize buffer for writing CSV data
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(record); err != nil {
		log.Printf("Failed to write CSV data: %v", err)
		return
	}
	w.Flush()

	// Upload the CSV log to S3
	_, err := a.S3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(buf.Bytes()),
	})

	if err != nil {
		log.Printf("Failed to upload log to S3: %v", err)
	} else {
		log.Printf("Successfully logged data to S3")
	}
}
