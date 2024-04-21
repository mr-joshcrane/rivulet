package rivulet

import (
	"sync/atomic"

	"github.com/mr-joshcrane/rivulet/store"
)

// Publishers are producers of messages.
// They deliver a [Message] to a [Receiver] via a [Transport].
type Publisher struct {
	name      string
	Transport Transport
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
// Pass them to [NewMemoryPublisher] at construction time.
type PublisherOptions func(*Publisher)

// NewMemoryPublisher creates a new [Publisher] with the given name and options.
// By default, the [Publisher] uses an in-memory [Transport].
func NewMemoryPublisher(name string, options ...PublisherOptions) (*Publisher, *Subscriber) {
	memoryTransport := NewMemoryTransport()
	subscriber := &Subscriber{
		receiver: memoryTransport.GetReceiver(),
		Store:    store.NewMemoryStore(),
	}
	publisher := &Publisher{
		name:      name,
		counter:   atomic.Int64{},
		Transport: memoryTransport,
	}
	for _, option := range options {
		option(publisher)
	}

	return publisher, subscriber
}

func NewEventBridgePublisher(name string, eventBridge EventBridgeClient, opts ...EventBridgeTransportOptions) *Publisher {
	eventBridgeTransport := &EventBridgeTransport{
		EventBridge:  eventBridge,
		detailType:   "rivulet",
		source:       "rivulet",
		eventBusName: "default",
		transform:    DefaultTransform,
	}
	for _, opts := range opts {
		opts(eventBridgeTransport)
	}
	publisher := &Publisher{
		name:      name,
		counter:   atomic.Int64{},
		Transport: eventBridgeTransport,
	}

	return publisher
}

// Counter allows a way for ordering in case transports
// are not guaranteed to deliver in order.
func (p *Publisher) Counter() int64 {
	return p.counter.Load()
}

// Publish sends a message via a [Transport].
// A Publisher is responsible for various metadata about the message.
func (p *Publisher) Publish(str string) error {
	p.counter.Add(1)
	m := Message{
		Publisher: p.name,
		Order:     int(p.counter.Load()),
		Content:   str,
	}
	return p.Transport.Publish(m)
}
