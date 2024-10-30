package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	// Load secrets from Azure Key Vault
	secrets, err := FetchSecrets() // Use FetchSecrets to load secrets
	if err != nil {
		log.Fatalf("Failed to load secrets: %v", err)
	}

	// Define HTTP handler for the Azure Function
	http.HandleFunc("/api/FishingBotFunction", func(w http.ResponseWriter, r *http.Request) {
		// Parse the request body
		var reqBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Process the Telegram message using the loaded secrets
		response, err := HandleTelegramMessage(reqBody, secrets) // Call HandleTelegramMessage
		if err != nil {
			http.Error(w, "Error processing message", http.StatusInternalServerError)
			return
		}

		// Write the response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})

	// Start server on port 8080
	port := "8080"
	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
