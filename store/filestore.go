package store

import (
	"io"
	"os"
	"sync"
)

type FileStore struct {
	input chan (string)
	mu    *sync.RWMutex
	file  *os.File
	err   error
}

func (s *FileStore) Receive() {
	for {
		data, ok := <-s.input
		if !ok {
			return
		}
		s.Write(data)
	}
}

func NewFileStore(filename string) (*FileStore, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return &FileStore{
		file: file,
		mu:   &sync.RWMutex{},
	}, nil
}

func (s *FileStore) Write(data string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.file.WriteString(data)
	if err != nil {
		s.err = err
		s.file.Close()
	}
}

func (s *FileStore) Read() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, err := s.file.Seek(0, 0)
	if err != nil {
		s.err = err
		return ""
	}
	data, err := io.ReadAll(s.file)
	if err != nil {
		s.err = err
		return ""
	}
	_, err = s.file.Seek(0, 2)
	if err != nil {
		s.err = err
		return ""
	}
	return string(data)
}

func (s *FileStore) Register(c chan (string)) {
	s.input = c
}

func (s *FileStore) Close() error {
	s.file.Close()
	return nil
}
