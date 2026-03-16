# Day 2 Design Specification: The Broker (Multi-Topic Management)

## 1. Objective

Introduce a `Broker` component that acts as the central registry and manager for multiple `Topic` instances. The goal is to allow clients to create and look up isolated topics by name, ensuring thread-safe access to the internal topic routing table.

## 2. Constraints & Scope

- **Still In-Memory:** No persistence or disk I/O yet.
- **Still No Network:** No HTTP or REST APIs (that is reserved for Day 3).
- **Two-Tier Architecture:** The system now has two layers of state: the `Broker` (manages topics) and the `Topic` (manages messages). You must carefully separate their responsibilities and their locks.
- **Explicit Creation:** Topics must be explicitly created before they can be used. Do not auto-create a topic if a client tries to append to a non-existent one.

## 3. Data Models

### 3.1. Broker

Represents the central message queue server.

**Attributes:**

- `Topics` (Map): A dictionary/map associating a `String` (topic name) with a pointer to a `Topic` object (`map[string]*Topic`).
- `Mutex` (Read/Write Mutual Exclusion Lock): Required to synchronize concurrent reads and writes to the `Topics` map. *(Note: A standard `Mutex` works, but `RWMutex` is highly recommended — topic lookups will happen far more frequently than topic creation.)*

## 4. Expected Behaviors (API Contract)

### 4.1. Initialization

- **Action:** Create a new `Broker` instance.
- **Input:** None.
- **Output:** An initialized `Broker` object with an empty `Topics` map.

### 4.2. Create Topic

- **Action:** Register a new topic in the broker.
- **Input:** `TopicName` (String).
- **Process:**
  1. Acquire the Write Lock on the Broker.
  2. Check if a topic with `TopicName` already exists in the map.
  3. If it exists, release the lock and return a predefined error (e.g., `ErrTopicAlreadyExists`).
  4. If it does not exist, initialize a new `Topic` (reusing Day 1 logic) and add it to the map.
  5. Release the lock.
- **Output:** A pointer to the newly created `Topic`, and an error (`nil` on success).

### 4.3. Get Topic

- **Action:** Retrieve an existing topic by its name.
- **Input:** `TopicName` (String).
- **Process:**
  1. Acquire the Read Lock on the Broker.
  2. Look up `TopicName` in the map.
  3. Release the lock.
  4. If found, return the `Topic` pointer. If not found, return a predefined error (e.g., `ErrTopicNotFound`).
- **Output:** A pointer to the `Topic`, and an error (`nil` on success).

## 5. Acceptance Criteria (Testing Requirements)

Your `broker_test.go` must verify the following scenarios:

- **Successful Creation:** Creating a topic named `"orders"` succeeds and returns a valid topic pointer.
- **Duplicate Prevention:** Attempting to create `"orders"` a second time returns an error and does not overwrite the existing topic.
- **Successful Retrieval:** Calling `GetTopic("orders")` returns the exact same topic instance that was created.
- **Not Found Handling:** Calling `GetTopic("payments")` before it is created returns an error.
- **End-to-End Flow:** Create a `Broker` → Create a `Topic` → Get the `Topic` → Append a message → Read the message back.
- **Concurrency Safety:** Spawning 50 goroutines each calling `CreateTopic` with the same name simultaneously should result in exactly 1 success and 49 errors. `go test -race` must pass cleanly.
