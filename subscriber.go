package rivulet

import (
	"context"
	"fmt"

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
	var convertedMessages []store.Message
	for _, msg := range messages {
		convertedMessages = append(convertedMessages, store.Message{
			Publisher: msg.Publisher,
			Order:     msg.Order,
			Content:   msg.Content,
		})
	}
	err := s.store.Save(convertedMessages)
	if err != nil {
		return err
	}
	return nil
}

func (s *Subscriber) Messages() ([]Message, error) {
	messages := s.store.Messages()
	fmt.Println(messages)
	var m []Message
	for _, msg := range messages {
		m = append(m, Message{
			Publisher: msg.Publisher,
			Order:     msg.Order,
			Content:   msg.Content,
		})
	}
	return m, nil
}
