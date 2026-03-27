package broker

import (
	"errors"
	"sync"
	"github.com/zeningc/mini-kafka/storage"
)

type Topic struct	{
	name string
	messages []Message
	nextOffset int64
	logStore *storage.LogStore
	mu sync.RWMutex
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

	return &Topic{name: name, messages: messages, logStore: logStore, nextOffset: int64(len(messages))}, nil
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
	return nextOffSet, nil
}

func (t *Topic) Name() string {
    return t.name
}


func (t *Topic) ReadFrom(offset int64, max int64) []Message	{
	t.mu.RLock()
	defer t.mu.RUnlock()
	start := int64(offset)
	if start < 0 {
		start = 0
	}
	length := int64(len(t.messages))
	if start >= length {
		return []Message{}
	}
	end := start + max
	if end > length {
		end = length
	}
	result := make([]Message, end-start)
	copy(result, t.messages[start:end])
	return result
}
