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
}

func NewApp() *App {
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
	}

	if app.BotUsername == "" {
		log.Println("Warning: BOT_USERNAME environment variable is missing. The bot will not respond to mentions.")
	} else {
		log.Printf("Bot username is set to: %s", app.BotUsername)
	}

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

func (a *App) ProcessMessage(chatID int64, userID int, userQuestion string, messageID int) error {
	// Rate limit check
	noLimitUsers := strings.Split(os.Getenv("NO_LIMIT_USERS"), ",")
	isNoLimitUser := false
	for _, id := range noLimitUsers {
		if id == strconv.Itoa(userID) {
			isNoLimitUser = true
			break
		}
	}

	if !isNoLimitUser && !a.UsageCache.CanUserChat(userID) {
		limitMsg := "Thanks for using ReelTalkBot. We restrict to 10 messages per 10 minutes to keep costs low and allow everyone to use the tool."
		if err := a.sendMessage(chatID, limitMsg, messageID); err != nil {
			log.Printf("Failed to send rate limit message to Telegram: %v", err)
		}
		return fmt.Errorf("user rate limited")
	}

	a.UsageCache.AddUsage(userID)

	// Prepare messages for OpenAI
	messages := []OpenAIMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: userQuestion},
	}

	startTime := time.Now()

	responseText, err := a.QueryOpenAIWithMessagesSimple(messages)
	if err != nil {
		log.Printf("OpenAI query failed: %v", err)
		return err
	}

	responseTime := time.Since(startTime)
	finalMessage := a.PrepareFinalMessageDetailed(responseText)

	if err := a.sendMessage(chatID, finalMessage, messageID); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
		return err
	}

	a.logToS3(userID, userQuestion, responseTime)
	return nil
}

// QueryOpenAIWithMessagesSimple sends a request to OpenAI with given messages and returns only the response text
func (a *App) QueryOpenAIWithMessagesSimple(messages []OpenAIMessage) (string, error) {
	fullEndpoint := fmt.Sprintf("%s/chat/completions", a.OpenAIEndpoint)

	query := OpenAIQuery{
		Model:       "gpt-4o-mini", // Specify the model to use
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   500,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAI query: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", fullEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.OpenAIKey)

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request to OpenAI: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result OpenAIResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}

	if len(result.Choices) > 0 {
		content := result.Choices[0].Message.Content
		if len(content) > 4096 { // Ensure content fits Telegram's max message length
			content = a.SummarizeToLength(content, 4096)
		}
		return content, nil
	}

	return "", fmt.Errorf("no choices returned in OpenAI response")
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

// SummarizeToLength trims the text to the specified maximum length
func (a *App) SummarizeToLength(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength]
}

func (a *App) logToS3(userID int, userPrompt string, responseTime time.Duration) {
	record := []string{
		fmt.Sprintf("%d", userID),
		userPrompt,
		fmt.Sprintf("%d", responseTime.Milliseconds()),
	}

	var csvData [][]string
	csvData = append(csvData, record)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.WriteAll(csvData); err != nil {
		log.Printf("Failed to write CSV data: %v", err)
		return
	}

	_, err := a.S3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(a.S3BucketName),
		Key:    aws.String("logs/telegram_logs.csv"),
		Body:   bytes.NewReader(buf.Bytes()),
	})

	if err != nil {
		log.Printf("Failed to upload log to S3: %v", err)
	} else {
		log.Printf("Successfully logged data to S3")
	}
}
