package internal

import (
	"fmt"
	"os"
)

// App struct defines the core application dependencies, including secrets and endpoints
type App struct {
	TelegramToken  string
	CQAKey         string
	CQAEndpoint    string
	OpenAIKey      string
	OpenAIEndpoint string
}

// NewApp initializes and returns a new App struct with required environment variables
func NewApp() (*App, error) {
	tt := os.Getenv("TELEGRAM_TOKEN")
	cKey := os.Getenv("CQA_KEY")
	cEndpoint := os.Getenv("CQA_ENDPOINT")
	oaKey := os.Getenv("OPENAI_KEY")
	oaEndpoint := os.Getenv("OPENAI_ENDPOINT")

	if tt == "" || cKey == "" || cEndpoint == "" || oaKey == "" || oaEndpoint == "" {
		return nil, fmt.Errorf("all environment variables are required")
	}

	return &App{
		TelegramToken:  tt,
		CQAKey:         cKey,
		CQAEndpoint:    cEndpoint,
		OpenAIKey:      oaKey,
		OpenAIEndpoint: oaEndpoint,
	}, nil
}
