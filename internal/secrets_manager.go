package internal

import (
	"fmt"
	"os"
)

type App struct {
	TelegramToken  string
	CQAKey         string
	CQAEndpoint    string
	OpenAIKey      string
	OpenAIEndpoint string
}

// NewApp initializes and returns the App struct
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
