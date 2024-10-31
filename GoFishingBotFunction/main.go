package main

import (
	"log"
	"net/http"
)

func main() {
	// Load secrets from Azure Key Vault
	app, err := NewApp()
	if err != nil {
		log.Fatalf("Failed to set up app: %v", err)
	}

	// Define HTTP handler for the Azure Function
	http.HandleFunc("POST /api/FishingBotFunction", app.Handler)

	// Start server on port 8080
	port := "8080"
	log.Printf("Starting server on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
