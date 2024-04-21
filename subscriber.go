package rivulet

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/mr-joshcrane/rivulet/store"
)

// Subscriber is a consumer of messages. It expects to receive messages
// from its [Receiver] and save them to its [Store].
type Subscriber struct {
	receiver Receiver
	Store    store.Store
}

// Receiver is a mechanism for receiving messages.
type Receiver interface {
	Receive(context.Context) ([]Message, error)
}

// InMemoryReceiver is a Receiver that receives messages from an InMemoryTransport
type InMemoryReceiver struct {
	messages <-chan Message
}

// Receive blocks until a message is available or the context is done.
// It then returns all messages received up to that point.
// Ideally signal the context when you're done receiving messages, rather than
// closing the channel.
func (r *InMemoryReceiver) Receive(ctx context.Context) ([]Message, error) {
	var messages []Message
	for {
		select {
		case <-ctx.Done():
			return messages, nil
		case msg, ok := <-r.messages:
			if !ok {
				return messages, nil
			}
			messages = append(messages, msg)
		}
	}
}

// Receive receives messages from the [Receiver] and saves them to the [Store].
// It returns an error if there was a problem saving the messages.
func (s *Subscriber) Receive(ctx context.Context) error {
	messages, err := s.receiver.Receive(ctx)
	if err != nil {
		return err
	}
	var convertedMessages []store.Message
	for _, msg := range messages {
		convertedMessages = append(convertedMessages, store.Message{
			Publisher: msg.Publisher,
			Order:     msg.Order,
			Content:   msg.Content,
		})
	}
	err = s.Store.Save(convertedMessages)
	if err != nil {
		return err
	}
	return nil
}

// EventBridgeReceiver
type EventBridgeReceiver struct {
	event events.EventBridgeEvent
}

func NewEventBridgeSubscriber(event events.EventBridgeEvent, store store.Store) *Subscriber {
	return &Subscriber{
		receiver: &EventBridgeReceiver{
			event: event,
		},
		Store: store,
	}
}
