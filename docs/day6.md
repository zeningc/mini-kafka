# Day 6 Design Specification: Blocking Pull (Long Polling)

## 1. Objective

Implement a mechanism that allows consumer requests to "wait" for new messages if they request an offset that has not been written yet. When a producer appends a new message, all waiting consumers should be immediately awakened to read the new data.

## 2. Constraints & Scope

- **Graceful Timeouts:** The client must be able to specify a maximum wait time (e.g., `5s`). If no message arrives in that window, the server should return an empty response rather than hanging forever.
- **No Busy-Waiting:** The server must not use an infinite `for` loop with `time.Sleep()` to check for new messages. You must use Go's concurrency primitives to pause the goroutine efficiently.
- **Broadcasting:** One published message might fulfill the requests of multiple waiting consumers simultaneously. Your wake-up mechanism must be able to notify everyone at once.

## 3. The "Boss Fight": Concurrency Design

This is the hardest concurrency challenge in the 7-day plan. You have two main architectural choices for how to implement the waiting logic inside your `Topic` struct.

### Option A: `sync.Cond` (The Classic Threading Way)

Add a `*sync.Cond` to your `Topic`.

- **Setup:** `t.cond = sync.NewCond(&t.mu)` — binds directly to your existing `Mutex`.
- **Waiter:** The consumer locks the mutex, checks if `offset >= t.nextOffset`, and if so calls `t.cond.Wait()`. This pauses the consumer and temporarily unlocks the mutex so producers can still write.
- **Waker:** When a producer appends a message, it calls `t.cond.Broadcast()` to wake up all waiting consumers.
- **The Catch (Timeout Problem):** `sync.Cond` has no built-in timeout. Implementing the `wait=5s` HTTP timeout is notoriously difficult because you cannot forcefully interrupt a `cond.Wait()`.

### Option B: Channel Notification (The "Go" Way)

A more idiomatic Go approach — use channels to signal that new data is available.

- **Setup:** Add a channel to your `Topic`: `notifyCh chan struct{}`.
- **Waiter:** The consumer checks the offset. If the message isn't there, it releases the lock and waits on:
  ```go
  select {
  case <-t.notifyCh:
      // new data arrived
  case <-time.After(timeout):
      // timed out
  }
  ```
- **Waker:** When a producer appends, it closes the current channel (broadcasting to all listeners) and immediately creates a new one for future waiters:
  ```go
  close(t.notifyCh)
  t.notifyCh = make(chan struct{})
  ```

**Recommendation:** If you want to strictly follow the original prompt, try `sync.Cond` first and ignore the timeout (let it block until a message arrives). If you want production-grade Go, use Option B — the `select` statement trivially solves the HTTP timeout requirement.

## 4. Expected Behaviors (API Contract)

### 4.1. HTTP Layer Update (Consume Endpoint)

**Route:** `GET /topics/{name}/messages?offset={start}&max={limit}&wait={timeout}`

- **New Parameter:** Parse the `wait` query parameter using `time.ParseDuration` so clients can send human-readable strings like `?wait=5s` or `?wait=500ms`.
- **Action:** Pass this timeout duration down to your broker and topic layers.
- **Success (Data Arrives):** Return `200 OK` with the JSON messages exactly as in Day 3.
- **Success (Timeout Reached):** Return `200 OK` with an empty JSON array `{"messages": []}`. A timeout is a valid business scenario — do not return `404` or `500`.

### 4.2. Core Domain Update (`Topic.ReadFrom`)

Upgrade your read method signature (or create a new one):

```go
func (t *Topic) ReadFromWait(offset int64, max int, timeout time.Duration) []Message
```

**Process:**
1. Check if the requested offset is already available. If yes, return the messages immediately.
2. If not, wait for a notification (via `Cond` or channel).
3. If the timeout expires before a notification arrives, return an empty slice.
4. If a notification arrives, loop back and re-check — another consumer may have changed state in the meantime (though in an append-only log this is less risky).

## 5. Acceptance Criteria

You will need two terminal windows to test this manually.

**Terminal 1 — The Consumer:**
```sh
curl "http://localhost:8080/topics/events/messages?offset=100&max=10&wait=10s"
```
The terminal should hang and not return immediately.

**Terminal 2 — The Producer** (while Terminal 1 is still hanging):
```sh
curl -X POST http://localhost:8080/topics/events/messages \
  -d '{"value": "triggered!"}'
```

**Validation:**
- The instant you press Enter in Terminal 2, Terminal 1 should unblock and print the JSON response containing the `"triggered!"` message.
- If you run Terminal 1 again and send nothing from Terminal 2, Terminal 1 should gracefully return an empty array after exactly 10 seconds.
- Run `go test -race ./...` to confirm that introducing channels/condition variables didn't break the Day 1 lock integrity.
