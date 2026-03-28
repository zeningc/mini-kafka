package broker

import (
	"fmt"
	"sync"
)

type Broker struct	{
	topics map[string]*Topic
	mu sync.RWMutex
}


func NewBroker() *Broker {
	return &Broker{topics: make(map[string]*Topic)}
}

func (b *Broker) AddTopic(topicName string) (*Topic, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.topics[topicName]; ok {
		return nil, fmt.Errorf("topic %s already exists", topicName)
	}
	t, err := NewTopic(topicName)
	if err != nil	{
		return nil, err
	}
	b.topics[topicName] = t
	return t, nil
}

func (b *Broker) GetTopic(topicName string) (*Topic, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if t, ok := b.topics[topicName]; !ok {
		return nil, fmt.Errorf("topic %s doesn't exist", topicName)
	}	else	{
		return t, nil
	}
}

func (b *Broker) Close()	{
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, v := range b.topics	{
		v.Close()
	}
}