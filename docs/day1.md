# Day 1 Design Specification: In-Memory Single Topic

## 1. Objective

Establish the foundational data structure for a single message queue (`Topic`) that operates purely in memory. The core goal is to ensure thread-safe operations for appending messages and reading messages sequentially based on an offset.

## 2. Constraints & Scope

- **In-Memory Only:** No disk I/O or file storage is required for this phase.
- **Single Topic:** We are only modeling the behavior of one topic. Do not worry about managing multiple topics or routing yet.
- **No Network Layer:** No HTTP server or REST APIs. All interactions will be tested via Go's testing framework or a simple `main()` function execution.
- **Concurrency:** The structure must be safe for concurrent access (multiple goroutines appending or reading simultaneously).

## 3. Data Models

### 3.1. Message

Represents a single unit of data in the queue.

**Attributes:**

- `Offset` (Integer): A unique, monotonically increasing sequence number assigned by the Topic.
- `Value` (String or Byte Slice): The actual payload of the message. *(Tip: String is easier for Day 1 debugging, but `[]byte` is closer to production reality.)*

### 3.2. Topic

Represents the message queue itself.

**Attributes:**

- `Name` (String): The identifier for the topic.
- `Messages` (List/Slice of Message): The sequential, append-only log of messages.
- `NextOffset` (Integer): The offset that will be assigned to the next incoming message. Starts at `0`.
- `Mutex` (Mutual Exclusion Lock): Required to synchronize access to `Messages` and `NextOffset`.

## 4. Expected Behaviors (API Contract)

### 4.1. Initialization

- **Action:** Create a new `Topic` instance.
- **Input:** Topic Name.
- **Output:** An initialized `Topic` object with an empty message list and `NextOffset` set to `0`.

### 4.2. Append Message

- **Action:** Add a new message to the end of the topic.
- **Input:** The message `Value`.
- **Process:**
  1. Acquire the lock.
  2. Create a new `Message` object using the current `NextOffset`.
  3. Append the `Message` to the `Messages` list.
  4. Increment `NextOffset` by 1.
  5. Release the lock.
- **Output:** Return the assigned `Offset` (and an error if you want to practice Go's standard error handling, though memory appends rarely fail).

### 4.3. Read Messages

- **Action:** Retrieve a batch of messages starting from a specific point.
- **Input:**
  - `StartOffset` (Integer): The offset to begin reading from.
  - `MaxMessages` (Integer): The maximum number of messages to return in this call.
- **Process:**
  1. Acquire the lock (a Read Lock is highly recommended if you use `sync.RWMutex`).
  2. Validate the `StartOffset`. If it is greater than or equal to `NextOffset`, return an empty list (meaning no new messages).
  3. Calculate the end index based on `StartOffset` and `MaxMessages`, ensuring you do not trigger an index out-of-bounds panic.
  4. Extract the slice of messages.
  5. Release the lock.
- **Output:** A list/slice of `Message` objects.

## 5. Acceptance Criteria (Testing Requirements)

Your `topic_test.go` must verify the following scenarios:

- **Sequential Append:** Appending 3 messages should result in them having offsets `0`, `1`, and `2`.
- **Basic Read:** Reading from offset `0` with a max of `10` should return all 3 messages.
- **Partial Read:** Reading from offset `1` with a max of `1` should return exactly the second message (offset `1`).
- **Out-of-Bounds Read:** Reading from an offset that hasn't been written yet (e.g., offset `100`) should safely return an empty list without crashing.
- **Concurrency Safety:** Spawning 100 goroutines that each append 1 message simultaneously should result in a `Topic` with exactly 100 messages, and `NextOffset` should be `100`. Run with `go test -race` to prove there are no data races.
