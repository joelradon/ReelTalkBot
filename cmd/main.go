package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"ReelTalkBot-Go/internal"
)

func main() {
	app := internal.NewApp()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(app.Handler),
	}

	// Start the server in a new goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed:", err)
		}
	}()

	log.Println("Server started on port 8080")

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline to wait for ongoing processes to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
