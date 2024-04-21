package store

import (
	"sync"
)

type MemoryStore struct {
	syncMap sync.Map
}
type Ledger map[int]string

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		syncMap: sync.Map{},
	}
}

func (s *MemoryStore) Save(m []Message) error {
	for _, msg := range m {
		p, ok := s.syncMap.Load(msg.Publisher)
		if !ok {
			s.syncMap.Store(msg.Publisher, Ledger{msg.Order: msg.Content})
		} else {
			p.(Ledger)[msg.Order] = msg.Content
		}
	}
	return nil
}

func (s *MemoryStore) Messages(publisher string) ([]Message, error) {
	var m []Message
	messages, ok := s.syncMap.Load(publisher)
	if !ok {
		return []Message{}, nil
	}
	for k, v := range messages.(Ledger) {
		m = append(m, Message{
			Publisher: publisher,
			Order:     k,
			Content:   v,
		})
	}
	return m, nil
}
