package store

import "bytes"

type MemoryStore struct {
	input chan (string)
	buf   bytes.Buffer
}

func (s *MemoryStore) Receive() {
	for {
		data, ok := <-s.input
		if !ok {
			return
		}
		s.Write(data)
	}
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		buf: bytes.Buffer{},
	}
}

func (s *MemoryStore) Write(data string) {
	s.buf.WriteString(data)
}

func (s *MemoryStore) Read() string {
	return s.buf.String()
}

func (s *MemoryStore) Register(c chan (string)) {
	s.input = c
}
