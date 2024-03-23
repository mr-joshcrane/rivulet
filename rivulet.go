package rivulet

import (
	"sync/atomic"
)

type Publisher struct {
	name      string
	transport Transport
	counter   atomic.Int64
}

type Message struct {
	Publisher string
	Order     int
	Content   string
}

type PublisherOptions func(*Publisher)

func NewPublisher(name string, options ...PublisherOptions) *Publisher {
	p := &Publisher{
		name: name,
	}
	for _, option := range options {
		option(p)
	}
	return p
}

func (p *Publisher) Counter() int64 {
	return p.counter.Load()
}

func (p *Publisher) Publish(str string) error {
	p.counter.Add(1)
	m := Message{
		Publisher: p.name,
		Order:     int(p.counter.Load()),
		Content:   str,
	}
	return p.transport.Publish(m)
}
