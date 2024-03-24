package rivulet

import (
	"sync/atomic"
)

// Publishers are producers of messages.
// They deliver a [Message] to a [Receiver] via a [Transport].
type Publisher struct {
	name      string
	transport Transport
	counter   atomic.Int64
}

// Message is a unit of data that can be published by a [Publisher].
// Message contains the name of the [Publisher] that published it,
// the order in which it was published, and the content of the message.
type Message struct {
	Publisher string
	Order     int
	Content   string
}

// PublisherOptions are functional options for configuring a [Publisher].
// Pass them to [NewPublisher] at construction time.
type PublisherOptions func(*Publisher)

// NewPublisher creates a new [Publisher] with the given name and options.

func NewPublisher(name string, options ...PublisherOptions) *Publisher {
	p := &Publisher{
		name:      name,
		counter:   atomic.Int64{},
		transport: &InMemoryTransport{},
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
