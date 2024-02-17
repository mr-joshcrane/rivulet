package rivulet

import (
	"bytes"
)

type Store struct {
	input chan (string)
	buf   bytes.Buffer
}

func (s *Store) Receive() {
	for {
		data, ok := <-s.input
		if !ok {
			s.Write(data)
			return
		}
		s.Write(data)
	}
}

type Producer struct {
	name   string
	output chan (string)
}

type ProducerOptions func(*Producer)

func WithStore(s *Store) ProducerOptions {
	c := make(chan (string))
	s.input = c
	return func(p *Producer) {
		p.output = c
	}
}

func NewProducer(name string, options ...ProducerOptions) *Producer {
	p := &Producer{
		name: name,
	}
	for _, option := range options {
		option(p)
	}
	return p
}

func (p *Producer) Publish(s ...string) error {
	for _, data := range s {
		p.output <- data
	}
	return nil
}

func (p *Producer) Close() {
	close(p.output)
}

func NewStore() *Store {
	return &Store{
		buf: bytes.Buffer{},
	}
}

func (s *Store) Write(data string) {
	s.buf.WriteString(data)
}

func (s *Store) Read() string {
	return s.buf.String()
}
