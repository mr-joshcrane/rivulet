package store

import "fmt"

type MemoryStore struct {
	messages []Message
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		messages: []Message{},
	}
}

func (s *MemoryStore) Save(m []Message) error {
	s.messages = append(s.messages, m...)
	fmt.Println("messages in Save()", s.messages)
	return nil
}

func (s *MemoryStore) Messages() []Message {
	fmt.Println("messages in Messages()", s.messages)
	return s.messages
}
