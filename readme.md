# Mini-Kafka

A toy message queue built in Go, "inspired" by Apache Kafka. This project is a hands-on exercise for learning Go — progressively building up from an in-memory topic to a multi-topic broker with an HTTP API.

## Architecture

| Layer | Description |
|---|---|
| `broker` | Core logic — `Topic` (append-only message log) and `Broker` (multi-topic registry) |
| `api` | HTTP handlers that expose the broker over a REST API |

## API

### Create Topic
```
POST /topics/{name}
```
- `201 Created` on success
- `409 Conflict` if the topic already exists

### Produce Message
```
POST /topics/{name}/messages
Content-Type: application/json

{"value": "your message here"}
```
- `201 Created` with `{"offset": <n>}` on success
- `404 Not Found` if the topic does not exist
- `400 Bad Request` if the JSON payload is malformed

### Consume Messages
```
GET /topics/{name}/messages?offset=<start>&max=<limit>
```
- `200 OK` with `{"messages": [...]}` on success
- `404 Not Found` if the topic does not exist
- `400 Bad Request` if `offset` or `max` are missing, non-integer, or negative

## Run

```bash
go run main.go
# Server starts on :8080
```

## Example

```bash
# Create a topic
curl -X POST http://localhost:8080/topics/sensor-data

# Produce messages
curl -X POST http://localhost:8080/topics/sensor-data/messages \
  -H "Content-Type: application/json" \
  -d '{"value": "temperature: 22"}'

# Consume messages
curl "http://localhost:8080/topics/sensor-data/messages?offset=0&max=10"
```

## Test

```bash
go test -race ./...
```
