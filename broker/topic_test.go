package broker

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func newTestTopic(t *testing.T, name string) *Topic {
	t.Helper()
	t.Chdir(t.TempDir())
	topic, err := NewTopic(name)
	if err != nil {
		t.Fatalf("NewTopic: %v", err)
	}
	return topic
}

func TestNewTopic(t *testing.T) {
	topic := newTestTopic(t, "test-topic")
	if topic == nil {
		t.Fatal("NewTopic returned nil")
	}
	if topic.name != "test-topic" {
		t.Errorf("expected name %q, got %q", "test-topic", topic.name)
	}
	if topic.nextOffset != 0 {
		t.Errorf("expected nextOffset 0, got %d", topic.nextOffset)
	}
}

func TestAppend_ReturnsIncrementingOffsets(t *testing.T) {
	topic := newTestTopic(t, "t")
	for i := 0; i < 3; i++ {
		offset, err := topic.Append(fmt.Sprintf("msg-%d", i))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if offset != int64(i) {
			t.Errorf("expected offset %d, got %d", i, offset)
		}
	}
}

func TestAppend_EmptyValue_ReturnsError(t *testing.T) {
	topic := newTestTopic(t, "t")
	offset, err := topic.Append("")
	if err == nil {
		t.Error("expected error for empty value, got nil")
	}
	if offset != -1 {
		t.Errorf("expected offset -1 on error, got %d", offset)
	}
}

func TestReadFrom_BasicRead(t *testing.T) {
	topic := newTestTopic(t, "t")
	topic.Append("a") // offset 0
	topic.Append("b") // offset 1
	topic.Append("c") // offset 2

	msgs := topic.ReadFrom(0, 10, 0)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	expected := []string{"a", "b", "c"}
	for i, m := range msgs {
		if m.Value != expected[i] || m.Offset != int64(i) {
			t.Errorf("msg[%d]: expected {%s, %d}, got {%s, %d}", i, expected[i], i, m.Value, m.Offset)
		}
	}
}

func TestReadFrom_PartialRead(t *testing.T) {
	topic := newTestTopic(t, "t")
	topic.Append("a") // offset 0
	topic.Append("b") // offset 1
	topic.Append("c") // offset 2

	msgs := topic.ReadFrom(1, 1, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Value != "b" || msgs[0].Offset != 1 {
		t.Errorf("expected {b, 1}, got {%s, %d}", msgs[0].Value, msgs[0].Offset)
	}
}

func TestReadFrom_OutOfBounds(t *testing.T) {
	topic := newTestTopic(t, "t")
	topic.Append("a")

	msgs := topic.ReadFrom(100, 10, 0)
	if len(msgs) != 0 {
		t.Errorf("expected empty slice, got %d messages", len(msgs))
	}
}

func TestAppend_ConcurrentSafety(t *testing.T) {
	topic := newTestTopic(t, "t")
	const goroutines = 100

	var wg sync.WaitGroup
	offsets := make(chan int64, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			offset, err := topic.Append(fmt.Sprintf("msg-%d", i))
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			offsets <- offset
		}(i)
	}

	wg.Wait()
	close(offsets)

	seen := make(map[int64]bool)
	for o := range offsets {
		if seen[o] {
			t.Errorf("duplicate offset %d", o)
		}
		seen[o] = true
	}
	if len(seen) != goroutines {
		t.Errorf("expected %d unique offsets, got %d", goroutines, len(seen))
	}
	if topic.nextOffset != goroutines {
		t.Errorf("expected nextOffset %d, got %d", goroutines, topic.nextOffset)
	}
}

// TestReadFrom_WaitUnblocksOnAppend verifies that a consumer blocked on a future
// offset is woken immediately when a producer appends.
func TestReadFrom_WaitUnblocksOnAppend(t *testing.T) {
	topic := newTestTopic(t, "t")

	result := make(chan []Message, 1)
	go func() {
		// offset 0 doesn't exist yet — should block until Append fires
		result <- topic.ReadFrom(0, 1, 5*time.Second)
	}()

	// Give the goroutine time to enter the wait
	time.Sleep(20 * time.Millisecond)

	topic.Append("hello")

	select {
	case msgs := <-result:
		if len(msgs) != 1 || msgs[0].Value != "hello" {
			t.Errorf("expected [{hello 0}], got %+v", msgs)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ReadFrom did not unblock after Append")
	}
}

// TestReadFrom_WaitTimeout verifies that a consumer returns an empty slice
// when no message arrives within the timeout window.
func TestReadFrom_WaitTimeout(t *testing.T) {
	topic := newTestTopic(t, "t")

	start := time.Now()
	msgs := topic.ReadFrom(0, 10, 100*time.Millisecond)
	elapsed := time.Since(start)

	if len(msgs) != 0 {
		t.Errorf("expected empty slice on timeout, got %+v", msgs)
	}
	if elapsed < 100*time.Millisecond {
		t.Errorf("returned too early: %v", elapsed)
	}
}

// TestReadFrom_WaitBroadcast verifies that multiple blocked consumers are all
// woken when a single Append fires.
func TestReadFrom_WaitBroadcast(t *testing.T) {
	topic := newTestTopic(t, "t")
	const consumers = 5

	results := make(chan []Message, consumers)
	for range consumers {
		go func() {
			results <- topic.ReadFrom(0, 1, 5*time.Second)
		}()
	}

	time.Sleep(20 * time.Millisecond)
	topic.Append("broadcast")

	for range consumers {
		select {
		case msgs := <-results:
			if len(msgs) != 1 || msgs[0].Value != "broadcast" {
				t.Errorf("consumer got unexpected messages: %+v", msgs)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("a consumer did not unblock after Append")
		}
	}
}
