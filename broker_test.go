package minikafka

import (
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

	msgs := topic.ReadFrom(offset, 1)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Value != "hello" || msgs[0].Offset != offset {
		t.Errorf("expected {hello, %d}, got {%s, %d}", offset, msgs[0].Value, msgs[0].Offset)
	}
}

func TestAddTopic_ConcurrentSafety(t *testing.T) {
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
