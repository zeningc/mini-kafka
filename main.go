package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zeningc/mini-kafka/api"
	"github.com/zeningc/mini-kafka/broker"
)

func main() {
	b := broker.NewBroker()

	server := api.NewServer(b)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /topics/{name}", server.HandleCreateTopic)
	mux.HandleFunc("POST /topics/{name}/messages", server.HandleProduce)
	mux.HandleFunc("GET /topics/{name}/messages", server.HandleConsume)

	// 1. Explicitly define the HTTP server
    srv := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }

    // 2. Start the server in a separate goroutine
    go func() {
        log.Println("Starting mini-kafka server on :8080")
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server crashed: %v\n", err)
        }
    }()

    // 3. Create a channel to listen for OS signals
    quit := make(chan os.Signal, 1)
    // Catch Ctrl+C (SIGINT) and Docker/K8s shutdown (SIGTERM)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

    // 4. Block the main goroutine until a signal is received
    <-quit
    log.Println("Shutdown signal received, shutting down gracefully...")

    // 5. Give the server time to finish active requests
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("HTTP server forced to shutdown: %v", err)
    }

    // 6. Clean up core domain resources (close all files)
    log.Println("Closing topic log files...")
    b.Close()

    log.Println("mini-kafka exited cleanly. Bye!")
}