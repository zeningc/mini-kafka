package broker

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewBroker(t *testing.T) {
	b := NewBroker()
	if b == nil {
		t.Fatal("NewBroker returned nil")
	}
	if b.topics == nil {
		t.Error("expected topics map to be initialized, got nil")
	}
}

func TestAddTopic_Successful(t *testing.T) {
	t.Chdir(t.TempDir())
	b := NewBroker()
	topic, err := b.AddTopic("orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic == nil {
		t.Fatal("expected non-nil topic pointer")
	}
	if topic.Name() != "orders" {
		t.Errorf("expected topic name %q, got %q", "orders", topic.Name())
	}
}

func TestAddTopic_DuplicatePrevention(t *testing.T) {
	t.Chdir(t.TempDir())
	b := NewBroker()
	first, err := b.AddTopic("orders")
	if err != nil {
		t.Fatalf("unexpected error on first create: %v", err)
	}

	second, err := b.AddTopic("orders")
	if err == nil {
		t.Error("expected error on duplicate create, got nil")
	}
	if second != nil {
		t.Error("expected nil topic pointer on duplicate create")
	}

	// original topic must not be overwritten
	got, _ := b.GetTopic("orders")
	if got != first {
		t.Error("duplicate creation overwrote the original topic")
	}
}

func TestGetTopic_Successful(t *testing.T) {
	t.Chdir(t.TempDir())
	b := NewBroker()
	created, _ := b.AddTopic("orders")

	got, err := b.GetTopic("orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != created {
		t.Error("GetTopic returned a different instance than the one created")
	}
}

func TestGetTopic_NotFound(t *testing.T) {
	b := NewBroker()
	topic, err := b.GetTopic("payments")
	if err == nil {
		t.Error("expected error for non-existent topic, got nil")
	}
	if topic != nil {
		t.Error("expected nil topic pointer for non-existent topic")
	}
}

func TestBroker_EndToEnd(t *testing.T) {
	t.Chdir(t.TempDir())
	b := NewBroker()

	_, err := b.AddTopic("events")
	if err != nil {
		t.Fatalf("AddTopic: %v", err)
	}

	topic, err := b.GetTopic("events")
	if err != nil {
		t.Fatalf("GetTopic: %v", err)
	}

	offset, err := topic.Append("hello")
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	msgs := topic.ReadFrom(offset, 1, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Value != "hello" || msgs[0].Offset != offset {
		t.Errorf("expected {hello, %d}, got {%s, %d}", offset, msgs[0].Value, msgs[0].Offset)
	}
}

func TestBroker_Close(t *testing.T) {
	t.Chdir(t.TempDir())
	b := NewBroker()

	names := []string{"alpha", "beta", "gamma"}
	for _, name := range names {
		_, err := b.AddTopic(name)
		if err != nil {
			t.Fatalf("AddTopic(%s): %v", name, err)
		}
	}

	b.Close()

	// After Close, every topic's file is shut — Append must return an error.
	for _, name := range names {
		topic, _ := b.GetTopic(name)
		_, err := topic.Append("post-close")
		if err == nil {
			t.Errorf("topic %q: expected error after Broker.Close, got nil", name)
		}
	}
}

// TestBroker_Close_ConcurrentAppends verifies that Close doesn't race with
// appends happening on multiple topics simultaneously.
func TestBroker_Close_ConcurrentAppends(t *testing.T) {
	t.Chdir(t.TempDir())
	b := NewBroker()

	for _, name := range []string{"t1", "t2", "t3"} {
		b.AddTopic(name)
	}

	var wg sync.WaitGroup
	for _, name := range []string{"t1", "t2", "t3"} {
		topic, _ := b.GetTopic(name)
		for i := range 10 {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				topic.Append(fmt.Sprintf("msg-%d", i)) // may succeed or fail — must not race
			}(i)
		}
	}

	b.Close()
	wg.Wait()
}

func TestAddTopic_ConcurrentSafety(t *testing.T) {
	t.Chdir(t.TempDir())
	b := NewBroker()
	const goroutines = 50

	var wg sync.WaitGroup
	results := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := b.AddTopic("shared-topic")
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	successes, failures := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}

	if successes != 1 {
		t.Errorf("expected exactly 1 successful creation, got %d", successes)
	}
	if failures != goroutines-1 {
		t.Errorf("expected %d errors, got %d", goroutines-1, failures)
	}
}
