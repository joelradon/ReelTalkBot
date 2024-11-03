// cmd/main.go

package main

import (
	"encoding/json"
	"log"
	"net/http"

	"ReelTalkBot-Go/internal/app"
	"ReelTalkBot-Go/internal/types"
)

func main() {
	botApp := app.NewApp()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		var update types.TelegramUpdate // Changed from types.Update to types.TelegramUpdate
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
