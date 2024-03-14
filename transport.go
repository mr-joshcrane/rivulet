package rivulet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type Transport interface {
	Publish(Message) error
}

// InMemoryTransport is a Transport that deals with messages within process

type InMemoryTransport struct {
	messages chan Message
}

func WithTransport(t Transport) PublisherOptions {
	return func(p *Publisher) {
		p.transport = t
	}
}

func NewMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{messages: make(chan Message, 1_000)}
}

func (t *InMemoryTransport) Publish(m Message) error {
	t.messages <- m
	return nil
}

func (t *InMemoryTransport) GetReceiver() InMemoryReceiver {
	return InMemoryReceiver{messages: t.messages}
}

// NetworkTransport is a Transport that ships messages over the NetworkTransport

type NetworkTransport struct {
	endpoint string
}

func WithNetworkTransport(endpoint string) PublisherOptions {
	endpoint = fmt.Sprintf("http://%s", endpoint)
	return func(p *Publisher) {
		p.transport = &NetworkTransport{endpoint: endpoint}
	}
}

func (t *NetworkTransport) Publish(m Message) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(data)
	req, err := http.NewRequest("POST", t.endpoint, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// EventBridgeTransport is a Transport that ships messages via AWS EventBridge

type EventBridgeClient interface {
	PutEvents(ctx context.Context, events *eventbridge.PutEventsInput, opts ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

type EventBridgeTransport struct {
	EventBridge EventBridgeClient
	detailType  string
	source      string
	eventBusName string
}

func WithEventBridgeTransport(eventBridge EventBridgeClient, detailType, source, eventBusName string) PublisherOptions {
	return func(p *Publisher) {
		p.transport = &EventBridgeTransport{
			EventBridge: eventBridge,
			detailType:  detailType,
			source:      source,
			eventBusName: eventBusName,
		}
	}
}

func (t *EventBridgeTransport) Publish(message Message) error {
	detail, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if t.eventBusName == "" {
		t.eventBusName = "default"
	}
	putEventsInput := &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				Detail:     aws.String(string(detail)),
				DetailType: aws.String(t.detailType),
				Source:     aws.String(t.source),
				EventBusName: aws.String(t.eventBusName),
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
