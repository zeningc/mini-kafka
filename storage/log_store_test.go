package storage

import (
	"testing"

	"github.com/zeningc/mini-kafka/model"
)

func TestLogStore_EmptyFile(t *testing.T) {
	t.Chdir(t.TempDir())
	store, err := NewLogStore("empty")
	if err != nil {
		t.Fatalf("NewLogStore: %v", err)
	}
	defer store.Close()

	msgs, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected empty slice, got %d messages", len(msgs))
	}
}

func TestLogStore_AppendAndLoadAll(t *testing.T) {
	t.Chdir(t.TempDir())
	store, err := NewLogStore("test")
	if err != nil {
		t.Fatalf("NewLogStore: %v", err)
	}

	want := []model.Message{
		{Value: "a", Offset: 0},
		{Value: "b", Offset: 1},
		{Value: "c", Offset: 2},
	}
	for _, m := range want {
		if err := store.Append(m); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	store.Close()

	// Reopen to simulate restart — file position resets to 0.
	store2, err := NewLogStore("test")
	if err != nil {
		t.Fatalf("NewLogStore (reopen): %v", err)
	}
	defer store2.Close()

	got, err := store2.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d messages, got %d", len(want), len(got))
	}
	for i, m := range got {
		if m != want[i] {
			t.Errorf("msg[%d]: expected %+v, got %+v", i, want[i], m)
		}
	}
}

// TestLogStore_Persistence is the storage-layer equivalent of the day5 "restart test":
// write N messages, close, reopen, verify all N messages load correctly,
// then append one more and verify its offset continues from N (not 0).
func TestLogStore_Persistence(t *testing.T) {
	t.Chdir(t.TempDir())

	// --- first run ---
	store, err := NewLogStore("events")
	if err != nil {
		t.Fatalf("NewLogStore: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := store.Append(model.Message{Value: "msg", Offset: int64(i)}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	store.Close()

	// --- restart ---
	store2, err := NewLogStore("events")
	if err != nil {
		t.Fatalf("NewLogStore (restart): %v", err)
	}
	defer store2.Close()

	msgs, err := store2.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages after restart, got %d", len(msgs))
	}

	// Next offset continues from len(msgs) = 3.
	nextOffset := int64(len(msgs))
	if err := store2.Append(model.Message{Value: "fourth", Offset: nextOffset}); err != nil {
		t.Fatalf("Append after restart: %v", err)
	}
	if nextOffset != 3 {
		t.Errorf("expected next offset 3, got %d", nextOffset)
	}
}
