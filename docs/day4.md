# Day 4 Design Specification: Testing, Hardening & Edge Cases

## 1. Objective

Solidify the codebase by writing comprehensive unit tests across all layers (topic, broker, and api). Identify and fix edge cases (like out-of-bounds reads or malformed requests) to ensure the system degrades gracefully instead of panicking.

## 2. Constraints & Scope

- **No New Features:** Do not add persistence, consumer groups, or anything else.
- **Standard Library Only:** Use Go's native `testing` and `net/http/httptest` packages. No third-party assertion libraries (like testify) for now, so you learn the standard Go testing idioms.
- **Coverage:** Aim to cover the "Happy Path" (everything works) and the "Unhappy Path" (bad user input) for every single exported method and HTTP handler.

## 3. Testing Strategy & Requirements

You will write tests for three distinct layers. Each layer has specific edge cases you must handle in your implementation if you haven't already.

### 3.1. Layer 1: Core Domain (`broker/topic_test.go`)

**Happy Path:**
- Appending 3 messages results in offsets 0, 1, 2.
- Reading from offset 0 with `max=2` returns exactly 2 messages.

**Edge Cases to Fix & Test:**
- **Future Reads:** What happens if `ReadFrom(offset=100)` is called when `nextOffset` is only 5? Requirement: Return an empty slice (`[]Message{}`), do not panic with an index out of bounds.
- **Zero or Negative Max:** What happens if `ReadFrom` is called with `max=0` or `max=-1`? Requirement: Safely return an empty slice.
- **Concurrency:** Write a test that spawns 100 goroutines calling `Append()` simultaneously. Run `go test -race ./broker` to prove your `sync.Mutex` works.

### 3.2. Layer 2: Registry (`broker/broker_test.go`)

**Happy Path:**
- Creating a topic works.
- Getting that same topic by name returns the exact same memory address (pointer).

**Edge Cases to Fix & Test:**
- **Duplicate Creation:** Creating `"orders"` twice must return an error on the second attempt.
- **Not Found:** Getting `"payments"` before it is created must return an error.

### 3.3. Layer 3: Transport (`api/server_test.go`)

This is where you use `net/http/httptest`. You will test your handlers without actually starting a server on port 8080.

**Happy Path (End-to-End):**
1. Simulate a `POST` to create a topic.
2. Simulate a `POST` to produce a message.
3. Simulate a `GET` to consume the message. Verify the JSON response matches what you produced.

**Edge Cases to Fix & Test:**
- **Missing Topic:** `POST` a message to `/topics/does-not-exist/messages` → Expect `404 Not Found`.
- **Malformed JSON:** `POST` a string like `{"value": oops...` → Expect `400 Bad Request`.
- **Bad Query Params:** `GET` with `?offset=abc` or `?max=-5` → Expect `400 Bad Request`.

## 4. Crash Course: Using `httptest` (For Layer 3)

Since we discussed this in the last step, here is the exact pattern you should use in `api/server_test.go` to test your HTTP layer.

The secret sauce is `httptest.NewRecorder()`, which acts as a "fake" browser/client capturing the response, and `httptest.NewRequest()`, which creates a fake incoming request.

## 5. Acceptance Criteria

- Running `go test ./...` in the root directory passes successfully.
- Running `go test -race ./...` shows zero data races.
- You have intentionally tried to break your server via `curl` (sending bad JSON, negative offsets, etc.), and the server responds with clean HTTP error codes instead of crashing the process.
