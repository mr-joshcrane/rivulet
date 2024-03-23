package rivulet

import (
	"context"

	"github.com/mr-joshcrane/rivulet/store"
)

type Subscriber struct {
	receiver Receiver
	store    store.Store
}

type Receiver interface {
	Receive(context.Context) []Message
}

type InMemoryReceiver struct {
	messages <-chan Message
}

func (r *InMemoryReceiver) Receive(ctx context.Context) []Message {
	var messages []Message
	for {
		select {
		case <-ctx.Done():
			return messages
		case msg, ok := <-r.messages:
			if !ok {
				return messages
			}
			messages = append(messages, msg)
		}
	}
}

func (s *Subscriber) Receive(ctx context.Context) error {
	messages := s.receiver.Receive(ctx)
	return s.store.Save(messages)
}
