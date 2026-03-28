# Day 7 Design Specification: Graceful Shutdown & Polish

## 1. Objective

Implement OS signal handling so the server can shut down gracefully. Ensure all in-flight HTTP requests finish processing, all buffered data is flushed to disk, and all open file descriptors are safely closed. Finally, document the project so others can run it.

## 2. Constraints & Scope

- **Signals:** Intercept `SIGINT` (`Ctrl+C`) and `SIGTERM` (standard termination signal from Docker or Kubernetes).
- **Zero Data Loss:** If a producer is in the middle of writing a message when the shutdown signal arrives, the server must wait for that write to complete before exiting.
- **No File Leaks:** Every `*.log` file opened by a `Topic` must be explicitly closed.

## 3. Architectural Updates

### 3.1. The Domain Layer (Cleanup Methods)

Add teardown logic to your core components.

- **`Topic.Close()`:** Add a method that closes the underlying `os.File` in your storage layer.
- **`Broker.Close()`:** Add a method that iterates through the `Topics` map and calls `Close()` on every topic.

### 3.2. The Transport Layer (HTTP Server Shutdown)

Up until now you've likely been using `http.ListenAndServe(":8080", mux)`. This function blocks forever until the process is killed.

You must switch to explicitly creating an `http.Server` struct so you can call its `Shutdown()` method.

## 4. The Graceful Shutdown Pattern (Go Idiom)

Here is the standard Go pattern for trapping OS signals and shutting down cleanly. Put this inside your `main.go`:

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    "mini-kafka/api"
    "mini-kafka/broker"
)

func main() {
    b := broker.NewBroker()
    serverAPI := api.NewServer(b)

    mux := http.NewServeMux()
    // ... attach your routes here ...

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
```

## 5. Documentation (`README.md`)

Create a `README.md` at the root of your project containing:

- **Title & Description:** "A minimalist, single-node message queue written in Go."
- **Features:** Highlight what it does — in-memory routing, disk persistence via JSONL, long-polling/blocking pull.
- **Getting Started:**
  - Run: `go run main.go`
  - Test: `go test -race ./...`
- **API Usage Examples:** Copy-pasteable `curl` commands for:
  - Creating a topic
  - Producing a message
  - Consuming a message (with and without `wait`)

## 6. Acceptance Criteria

- **The `Ctrl+C` Test:** Start the server, press `Ctrl+C`. Your custom log messages should print in order and the process should exit with code `0`.
- **File Integrity:** After a clean shutdown, verify that all `.log` files in `data/` are intact and not corrupted.
- **The Final Race Test:** Run `go test -race ./...` one last time to confirm that adding the `Close()` methods didn't introduce any lock contentions or data races.
