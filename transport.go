package rivulet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type Transport interface {
	Publish(Message) error
}

// InMemoryTransport is a Transport that deals with messages within process

type InMemoryTransport struct {
	messages *[]Message
}

func WithInMemoryTransport(messages *[]Message) PublisherOptions {
	return func(p *Publisher) {
		p.transport = &InMemoryTransport{messages: messages}
	}
}
func (t *InMemoryTransport) Publish(m Message) error {
	*t.messages = append(*t.messages, m)
	return nil
}

// NetworkTransport is a Transport that ships messages over the NetworkTransport

type NetworkTransport struct {
	endpoint string
	port     int
}

func WithNetworkTransport(endpoint string, port int) PublisherOptions {
	return func(p *Publisher) {
		p.transport = &NetworkTransport{endpoint: endpoint, port: port}
	}
}

func (t *NetworkTransport) Publish(m Message) error {
	return nil
}

// EventBridgeTransport is a Transport that ships messages via AWS EventBridge

type EventBridgeClient interface {
	PutEvents(ctx context.Context, events *eventbridge.PutEventsInput, opts ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

type EventBridgeTransport struct {
	EventBridge EventBridgeClient
}

func WithEventBridgeTransport(eventBridge EventBridgeClient) PublisherOptions {
	return func(p *Publisher) {
		p.transport = &EventBridgeTransport{EventBridge: eventBridge}
	}
}
func (t *EventBridgeTransport) Publish(message Message) error {
	detail, err := json.Marshal(message)
	if err != nil {
		return err
	}
	putEventsInput := &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				Detail:     aws.String(string(detail)),
				DetailType: aws.String("rivulet"),
				Source:     aws.String("rivulet"),
			},
		},
	}
	resp, err := t.EventBridge.PutEvents(context.Background(), putEventsInput)
	if err != nil {
		return err
	}
	if resp.FailedEntryCount > 0 {
		return fmt.Errorf("failed to publish events: %v", resp)
	}
	return nil
}
