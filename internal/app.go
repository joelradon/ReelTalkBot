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
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
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
}

func NewApp() *App {
	// Load .env file if present
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
	}

	if app.BotUsername == "" {
		log.Println("Warning: BOT_USERNAME environment variable is missing. The bot will not respond to mentions.")
	} else {
		log.Printf("Bot username is set to: %s", app.BotUsername)
	}

	// Initialize AWS session
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

// Handler processes incoming HTTP requests from Telegram
func (a *App) Handler(w http.ResponseWriter, r *http.Request) {
	var update Update
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	response, err := a.HandleTelegramMessage(&update, r)
	if err != nil {
		http.Error(w, "Error processing message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

// SendMessage sends a plain text message to a Telegram chat
func (a *App) SendMessage(chatID int64, text string, replyToMessageID int) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", a.TelegramToken)
	payload := map[string]interface{}{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,       // Disable link previews
		"parse_mode":               "Markdown", // Optional: Change to "HTML" or remove for Plain Text
	}

	if replyToMessageID != 0 {
		payload["reply_to_message_id"] = replyToMessageID
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Use context with timeout
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

	// Confirm that the response from Telegram was successful
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s - %s", resp.Status, string(bodyBytes))
	}

	return nil
}

// SendMessageWithThrottle sends a message with rate limiting
func (a *App) SendMessageWithThrottle(chatID int64, text string, replyToMessageID int) error {
	ctx := context.Background()
	if err := a.RateLimiter.Wait(ctx); err != nil {
		return err
	}
	return a.SendMessage(chatID, text, replyToMessageID)
}

// LogToS3 logs user interactions to an S3 bucket
func (a *App) LogToS3(message *TelegramMessage, userPrompt string, responseTime time.Duration) {
	// Collect data
	userID := message.From.ID
	username := message.From.Username
	prompt := strings.ReplaceAll(userPrompt, "\n", " ") // Remove newlines
	responseTimeMS := responseTime.Milliseconds()

	// Format data as CSV
	record := []string{
		fmt.Sprintf("%d", userID),
		username,
		prompt,
		fmt.Sprintf("%d", responseTimeMS),
	}

	// Define S3 bucket and object key
	bucketName := a.S3BucketName
	objectKey := "logs/telegram_logs.csv"

	// Get existing object content
	existingObj, err := a.S3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})

	var existingData []byte
	if err == nil {
		existingData, _ = io.ReadAll(existingObj.Body)
		existingObj.Body.Close()
	} else {
		log.Printf("S3 GetObject error (may not exist yet): %v", err)
	}

	// Prepare CSV data
	var csvData [][]string

	// If there's existing data, parse it
	if len(existingData) > 0 {
		r := csv.NewReader(bytes.NewReader(existingData))
		existingRecords, err := r.ReadAll()
		if err != nil {
			log.Printf("Failed to read existing CSV data: %v", err)
		} else {
			csvData = existingRecords
		}
	}

	// Append new record
	csvData = append(csvData, record)

	// Write CSV data to buffer
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.WriteAll(csvData); err != nil {
		log.Printf("Failed to write CSV data: %v", err)
		return
	}

	// Upload updated content
	_, err = a.S3Client.PutObject(&s3.PutObjectInput{
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
