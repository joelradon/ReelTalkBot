package main

import (
	"ReelTalkBot-Go/internal"
	"log"
	"net/http"
)

func main() {
	app, err := internal.NewApp()
	if err != nil {
		log.Fatalf("Failed to set up app: %v", err)
	}

	http.HandleFunc("/api/FishingBotFunction", app.Handler)

	port := "8080"
	log.Printf("Starting server on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
