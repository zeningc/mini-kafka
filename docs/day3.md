# Day 3 Design Specification: HTTP API Layer

## 1. Objective

Build a RESTful HTTP server that exposes the internal `Broker` and `Topic` capabilities over the network. External clients must be able to create topics, publish messages, and consume messages using standard HTTP requests and JSON payloads.

## 2. Constraints & Scope

- **Standard Library Only:** Use Go's native `net/http` and `encoding/json` packages. Avoid third-party web frameworks (like Gin or Echo) to deeply understand standard routing and response handling.
- **String Payloads:** Keep the message `Value` as a `string` to simplify JSON marshaling/unmarshaling in this first iteration.
- **Stateless Handlers:** HTTP handlers must not hold any queue state themselves — they act strictly as a translation layer, delegating to the shared `Broker` instance.
- **Synchronous:** All requests are handled synchronously. Blocking/long-polling for new messages is out of scope.

## 3. Data Models (JSON Schemas)

Define Go struct representations for your JSON requests and responses.

- **`ProduceRequest`** — incoming payload for a new message.
  - `Value` (String)

- **`ProduceResponse`** — confirmation of a successful append.
  - `Offset` (Integer)

- **`ConsumeResponse`** — batch of messages returned to the reader.
  - `Messages` (List of `Message` objects, each with `Offset` and `Value`)

- **`ErrorResponse`** — standard format for API errors *(optional but highly recommended)*.
  - `Error` (String)

## 4. Expected Behaviors (API Contract)

### 4.1. Create Topic

- **Route:** `POST /topics/{name}`
- **Action:** Parse `{name}` from the URL path and instruct the `Broker` to create it.
- **Success:** `HTTP 201 Created` (body can be empty or a simple confirmation).
- **Failure:** `HTTP 409 Conflict` if the topic already exists.

### 4.2. Produce Message

- **Route:** `POST /topics/{name}/messages`
- **Action:** Parse `{name}` from the URL, find the topic, decode the JSON body into a `ProduceRequest`, and append the value.
- **Success:** `HTTP 201 Created` with a JSON `ProduceResponse` containing the new offset.
- **Failure:**
  - `HTTP 404 Not Found` if the topic does not exist.
  - `HTTP 400 Bad Request` if the JSON payload is malformed.

### 4.3. Consume Messages

- **Route:** `GET /topics/{name}/messages?offset={start}&max={limit}`
- **Action:** Parse `{name}` from the URL, parse `offset` and `max` from query parameters, and fetch the slice of messages from the target topic.
- **Success:** `HTTP 200 OK` with a JSON `ConsumeResponse`.
- **Failure:**
  - `HTTP 404 Not Found` if the topic does not exist.
  - `HTTP 400 Bad Request` if `offset` or `max` are missing, not integers, or negative.

## 5. Acceptance Criteria (Testing Requirements)

Test manually via `curl` or with Go's `net/http/httptest` package. Your implementation must pass these scenarios:

- **Routing Accuracy:** A request to a non-existent route (e.g., `GET /unknown`) returns `404 Not Found`.
- **Topic Creation Flow:** `POST /topics/sensor-data` succeeds. A subsequent `POST` to the same URL returns an error status code.
- **Produce Flow:** Sending `{"value": "temperature: 22"}` to a valid topic returns offset `0`. A second message returns offset `1`.
- **Consume Flow:** `GET /topics/sensor-data/messages?offset=0&max=10` returns a JSON array containing the exact messages produced above.
- **Validation:** Omitting `offset` or passing `?offset=abc` returns `400 Bad Request` with a descriptive error message.
