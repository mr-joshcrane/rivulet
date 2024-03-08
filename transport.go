package rivulet

import (
	"encoding/json"
	"os"
)

type Transport interface {
	Publish(Message) error
}

func WithInMemoryTransport(messages *[]Message) PublisherOptions {
	return func(p *Publisher) {
		p.transport = &TestTransport{messages: messages}
	}
}

type TestTransport struct {
	messages *[]Message
}

func (t *TestTransport) Publish(m Message) error {
	*t.messages = append(*t.messages, m)
	return nil
}

func WithFileTransport(file *os.File) PublisherOptions {
	return func(p *Publisher) {
		p.transport = &FileTransport{f: file}
	}
}

type FileTransport struct {
	f *os.File
}

func (t *FileTransport) Publish(m Message) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = t.f.Write(data)
	return err
}
