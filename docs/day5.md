# Day 5 Design Specification: Disk Persistence (Append-Only Logs)

Today is a major milestone. By adding disk persistence, your application graduates from a fragile memory structure into a true, stateful data store. When the server crashes or restarts, your messages will survive.

## 1. Objective

Introduce a storage layer that writes every new message to the local file system. Upon starting up or creating a topic, the system must read the existing files from disk to reconstruct the in-memory state (the message slice and `NextOffset`).

## 2. Constraints & Scope

- **One File Per Topic:** Each topic will have its own dedicated log file (e.g., `data/orders.log`, `data/sensor-data.log`).
- **Format:** Use JSON Lines (JSONL). Each line in the file represents exactly one JSON-encoded message. This is vastly easier to parse and debug for V1 than a custom binary format.

  Example file content:
  ```
  {"offset":0,"value":"temperature: 22"}
  {"offset":1,"value":"temperature: 24"}
  ```

- **In-Memory Loading:** For this version, you will still keep all messages in the `Topic.messages` memory slice. When a topic is initialized, read the entire file line-by-line and load it into memory. *(We acknowledge this is a memory bottleneck for huge queues, but it is the right stepping stone.)*
- **Synchronous Writes:** When a client produces a message, the HTTP response should not be sent until the message is successfully written to the file.

## 3. Architectural Design

To maintain the Separation of Concerns established in Day 3, it is highly recommended to create a new package for file operations.

**Proposed Directory Structure:**

```
mini-kafka/
├── data/                  ← (New) Where the actual .log files will live
├── storage/               ← (New) Your persistence package
│   └── log_store.go
├── broker/
│   ├── broker.go
│   └── topic.go           ← Will now use the storage package
...
```

### 3.1. The Storage Interface / Struct

Create a component (e.g., `LogStore`) responsible for file I/O.

- **State:** Holds a pointer to an open `os.File`.
- **Behavior 1 — Append:** Takes a `Message`, serializes it to JSON, appends a newline character (`\n`), and writes it to the end of the file.
- **Behavior 2 — LoadAll:** Reads the file from top to bottom, unmarshals each line back into a `Message` struct, and returns the full slice of messages so the `Topic` can rebuild itself.

## 4. Expected Behaviors (Integration with Broker/Topic)

You will need to modify your `broker` package to wire up the new storage layer.

### 4.1. Topic Initialization (The Recovery Phase)

**Trigger:** When `NewTopic` (or `CreateTopic`) is called.

**Process:**
1. Ensure the `data/` directory exists.
2. Open the file `data/{topic_name}.log`. If it doesn't exist, create it.
3. Call the storage layer's `LoadAll()` method.
4. Populate the `Topic`'s internal message slice with the loaded messages.
5. Set `NextOffset` to `len(messages)` (or the last message's offset + 1).

### 4.2. Topic Append (The Persistence Phase)

**Trigger:** When `topic.Append()` is called.

**Process:**
1. Acquire the lock.
2. Assign the offset and append to the in-memory slice (existing Day 1 logic).
3. **NEW:** Call the storage layer to append the same message to disk.
4. If the disk write fails, return an error and undo the memory append (or panic).
5. Increment `NextOffset`.
6. Release the lock.

## 5. Acceptance Criteria

Your manual testing (via `curl` or Postman) and automated tests must pass this sequence:

**The Restart Test (Crucial):**
1. Start your Go server.
2. `POST` 3 messages to a new topic called `events`.
3. Verify via `GET` that the 3 messages exist.
4. Kill the Go server process (`Ctrl+C`).
5. Check the `data/` directory to verify `events.log` was created and contains 3 lines of JSON.
6. Start the Go server again.
7. Immediately send a `GET` request for the `events` topic.
   - **Requirement:** The 3 messages must be returned perfectly.
8. `POST` a 4th message.
   - **Requirement:** It must be assigned offset `3` (not `0`).

**File Descriptor Leak Prevention:** Ensure that if you ever delete or close a topic, the underlying `os.File` is closed properly.
