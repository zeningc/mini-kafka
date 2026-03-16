package minikafka

import (
	"errors"
	"sync"
)

type Topic struct	{
	name string
	messages []Message
	nextOffset int64
	mu sync.RWMutex
}

func NewTopic(name string) *Topic {
	return &Topic{name: name, messages: make([]Message, 0)}
}

func (t *Topic) Append(value string) (int64, error)	{
	if value == "" {
		return -1, errors.New("message is empty")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	nextOffSet := t.nextOffset
	msg := Message{Value: value, Offset: nextOffSet}
	t.messages = append(t.messages, msg)
	t.nextOffset++
	return nextOffSet, nil
}

func (t *Topic) Name() string {
    return t.name
}


func (t *Topic) ReadFrom(offset int64, max int) []Message	{
	t.mu.RLock()
	defer t.mu.RUnlock()
	start := int(offset)
	if start < 0 {
		start = 0
	}
	if start >= len(t.messages) {
		return []Message{}
	}
	end := start + max
	if end > len(t.messages) {
		end = len(t.messages)
	}
	result := make([]Message, end-start)
	copy(result, t.messages[start:end])
	return result
}
