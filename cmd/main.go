// cmd/main.go

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"ReelTalkBot-Go/facebook" // Added import for Facebook handling
	"ReelTalkBot-Go/internal/app"
	"ReelTalkBot-Go/internal/types"
)

func main() {
	botApp := app.NewApp()

	// Load environment variables for Facebook Messenger
	fbVerifyToken := os.Getenv("FB_VERIFY_TOKEN")
	fbPageAccessToken := os.Getenv("FB_PAGE_ACCESS_TOKEN")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Determine if the request is from Telegram or Facebook Messenger
		if r.URL.Query().Get("hub.mode") != "" {
			// Handle Facebook Messenger webhook verification and events
			facebook.HandleFacebookRequest(w, r, fbVerifyToken, fbPageAccessToken, botApp)
			return
		}

		// Handle Telegram updates
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		var update types.TelegramUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			log.Printf("Failed to decode update: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		go botApp.HandleUpdate(&update)

		w.WriteHeader(http.StatusOK)
	})

	port := ":8080"
	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
