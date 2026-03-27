package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/zeningc/mini-kafka/model"
)

type LogStore struct {
	file *os.File
}

func openFileByPath(topicName string) (*os.File, error)	{
	err := os.MkdirAll("data", 0755)
	if err != nil {
		return nil, err
	}
	fileName := fmt.Sprintf("data/%s.json", topicName)
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil	{
		return nil, err
	}

	return file, nil
}

func NewLogStore(topicName string) (*LogStore, error)	{
	file, err := openFileByPath(topicName);
	if err != nil	{
		return nil, err
	}
	return &LogStore{file: file}, nil
}

func (logStore *LogStore) Close() error {
    return logStore.file.Close()
}

func (logStore *LogStore) Append(msg model.Message) error {
	encoder := json.NewEncoder(logStore.file)

	if err := encoder.Encode(msg); err != nil {
		return err
	}
	if err := logStore.file.Sync(); err != nil	{
		return err
	}
	return nil
}

func (logStore *LogStore) LoadAll() ([]model.Message, error) {
	scanner := bufio.NewScanner(logStore.file)
	messages := make([]model.Message, 0)
	for scanner.Scan() {
		line := scanner.Bytes()
		var message model.Message
		
		if err := json.Unmarshal(line, &message); err != nil	{
			return nil, err
		}

		messages = append(messages, message)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
