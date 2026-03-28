package broker

import (
	"errors"
	"sync"
	"time"

	"github.com/zeningc/mini-kafka/storage"
)

type Topic struct	{
	name string
	messages []Message
	nextOffset int64
	logStore *storage.LogStore
	mu sync.RWMutex
	notifyCh chan struct {}
}

func NewTopic(name string) (*Topic, error) {
	logStore, err := storage.NewLogStore(name)
	if err != nil	{
		return nil, err
	}
	messages, err := logStore.LoadAll()
	if err != nil	{
		return nil, err
	}
	return &Topic{
		name: name,
		messages: messages,
		logStore: logStore,
		nextOffset: int64(len(messages)),
		notifyCh: make(chan struct{}),
	}, nil
}

func (t *Topic) Append(value string) (int64, error)	{
	if value == "" {
		return -1, errors.New("message is empty")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	nextOffSet := t.nextOffset
	msg := Message{Value: value, Offset: nextOffSet}
	if err := t.logStore.Append(msg); err != nil	{
		return -1, err
	}
	t.messages = append(t.messages, msg)
	t.nextOffset++
	close(t.notifyCh)
	t.notifyCh = make(chan struct{})
	return nextOffSet, nil
}

func (t *Topic) Name() string {
    return t.name
}


func (t *Topic) ReadFrom(offset int64, max int64, timeout time.Duration) []Message {
	if offset < 0 {
		offset = 0
	}

	t.mu.RLock()

	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		for offset >= t.nextOffset {
			ch := t.notifyCh // capture while holding lock
			t.mu.RUnlock()
			select {
			case <-ch:
				t.mu.RLock() // re-acquire and loop back to recheck
			case <-timer.C:
				return []Message{} // lock not held — safe to return directly
			}
		}
		// lock is held here; fall through to read
	} else {
		if offset >= t.nextOffset {
			t.mu.RUnlock()
			return []Message{}
		}
	}

	length := int64(len(t.messages))
	end := offset + max
	if end > length {
		end = length
	}
	result := make([]Message, end-offset)
	copy(result, t.messages[offset:end])
	t.mu.RUnlock()
	return result
}

func (t *Topic) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.logStore.Close()
}
