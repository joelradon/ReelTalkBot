// cmd/main.go

package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"ReelTalkBot-Go/internal/app"
	"ReelTalkBot-Go/internal/telegram"
	"ReelTalkBot-Go/internal/types"
)

func main() {
	appInstance := app.NewApp()
	telegramHandler := telegram.NewTelegramHandler(appInstance)

	// Start server with mainHandler function for Telegram webhook
	srv := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mainHandler(w, r, telegramHandler)
		}),
	}

	// Start the server in a new goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed:", err)
		}
	}()

	log.Println("Server started on port 8080")

	// Graceful shutdown on interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

// mainHandler processes incoming HTTP requests for the Telegram bot webhook
func mainHandler(w http.ResponseWriter, r *http.Request, th *telegram.TelegramHandler) {
	// Log the raw body of the incoming request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	log.Printf("Received raw update payload: %s", string(body))

	// Parse the JSON payload into an Update struct
	var update types.Update
	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("Error unmarshalling update: %v", err)
		http.Error(w, "cannot parse body", http.StatusBadRequest)
		return
	}

	// Pass the parsed update to HandleTelegramMessage
	result, err := th.HandleTelegramMessage(&update)
	if err != nil {
		log.Printf("Error handling message: %v", err)
		http.Error(w, "error processing message", http.StatusInternalServerError)
		return
	}

	log.Printf("Result: %s", result)
	w.WriteHeader(http.StatusOK)
}
