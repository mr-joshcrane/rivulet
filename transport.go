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

// Transport some generic mechanism for delivering messages
type Transport interface {
	Publish(Message) error
}

// WithTransport is a functional option specifying that a [Publisher]
// should use the given [Transport] to deliver messages.
func WithTransport(t Transport) PublisherOptions {
	return func(p *Publisher) {
		p.transport = t
	}
}

// InMemoryTransport is a Transport that deals with messages within process
type InMemoryTransport struct {
	messages chan Message
}

// InMemoryReceiver is a Receiver that receives messages from an InMemoryTransport
// This is primarily useful in testing both this package and any dependent package
func NewMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{messages: make(chan Message, 1_000)}
}

// Publish sends a message to the InMemoryTransport by sending a message on the channel
func (t *InMemoryTransport) Publish(m Message) error {
	t.messages <- m
	return nil
}

// GetReceiver returns a Receiver that can be used to receive messages from the InMemoryTransport
// It's convenient to be able to get the associated [Receiver] from the [Transport]
func (t *InMemoryTransport) GetReceiver() *InMemoryReceiver {
	return &InMemoryReceiver{messages: t.messages}
}

// NetworkTransport is a Transport that ships messages over the Network
type NetworkTransport struct {
	endpoint string
}

// WithNetworkTransport is a functional option specifying that a [Publisher]
// should use the given endpoint to deliver messages over the network
func WithNetworkTransport(endpoint string) PublisherOptions {
	return func(p *Publisher) {
		p.transport = &NetworkTransport{endpoint: endpoint}
	}
}

// Publish sends a message to the NetworkTransport by sending an HTTP POST Request
func (t *NetworkTransport) Publish(m Message) error {
	data, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(data)
	req, err := http.NewRequest("POST", t.endpoint, buf)
	if err != nil {
		panic(err)
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

// Transform is a way to modify a message before sending it to EventBridgeClient
// It can be considered a way to add a pre-processing step to the message
// Useful in the case that you have specific requirements you need to adhere to
// in order for downstream AWS EventBridge Rules to target your event correctly
type Transform func(Message) (string, error)

// EventBridgeTransport is a Transport that ships messages via AWS EventBridge
type EventBridgeTransport struct {
	EventBridge  EventBridgeClient
	detailType   string
	source       string
	eventBusName string
	transform    Transform
}

// EventBridgeTransportOptions are functional options for configuring an EventBridgeTransport
// Such options include setting the DetailType, Source, EventBusName, and
// any custom [Transform] function needed (if any) to modify the message before sending it
type EventBridgeTransportOptions func(*EventBridgeTransport)

var DefaultTransform = func(m Message) (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WithDetailType is a functional option specifying the DetailType of the EventBridgeTransport
func WithDetailType(detailType string) EventBridgeTransportOptions {
	return func(t *EventBridgeTransport) {
		t.detailType = detailType
	}
}

// WithSource is a functional option specifying the Source of the EventBridgeTransport
func WithSource(source string) EventBridgeTransportOptions {
	return func(t *EventBridgeTransport) {
		t.source = source
	}
}

// WithEventBusName is a functional option specifying the EventBusName of the EventBridgeTransport
func WithEventBusName(eventBusName string) EventBridgeTransportOptions {
	return func(t *EventBridgeTransport) {
		t.eventBusName = eventBusName
	}
}

// WithTransform is a functional option specifying the Transform function of the EventBridgeTransport
// It's the users responsibility to ensure that any errors are handled correctly
func WithTransform(transform Transform) EventBridgeTransportOptions {
	return func(t *EventBridgeTransport) {
		t.transform = transform
	}
}

// WithEventBridgeTransport is a functional option specifying that a [Publisher]
// should use an [EventBridgeTransport] to deliver messages
func WithEventBridgeTransport(eventBridge EventBridgeClient, opts ...EventBridgeTransportOptions) PublisherOptions {
	transport := &EventBridgeTransport{
		EventBridge:  eventBridge,
		detailType:   "rivulet",
		source:       "rivulet",
		eventBusName: "default",
		transform:    DefaultTransform,
	}
	for _, opt := range opts {
		opt(transport)
	}
	return func(p *Publisher) {
		p.transport = transport
	}
}

// Publish sends a message to the specified AWS EventBus via the EventBridgeTransport
// Note that the message is transformed before being sent and the default transform
// is to simply marshal the message to JSON. This can be overridden by providing
// a custom transform function via the WithTransform functional option.
// Note that it is assumed that the corresponding Rule in AWS EventBridge is configured
// to match this event and route it to the appropriate target. Successfully delivery
// of the event to the EventBus is no indication that the event will be routed to the target.
func (t *EventBridgeTransport) Publish(message Message) error {
	detail, err := t.transform(message)
	if err != nil {
		return err
	}
	if detail == "" {
		return fmt.Errorf("message transform returned an empty string")
	}
	putEventsInput := &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				Detail:       aws.String(detail),
				DetailType:   aws.String(t.detailType),
				Source:       aws.String(t.source),
				EventBusName: aws.String(t.eventBusName),
			},
		},
	}
	resp, err := t.EventBridge.PutEvents(context.Background(), putEventsInput)
	if err != nil {
		return err
	}
	if resp.FailedEntryCount > 0 {
		if len(resp.Entries) > 0 {
			return fmt.Errorf("failed to publish events:%s, %s",
				*(resp.Entries[0].ErrorCode),
				*(resp.Entries[0].ErrorMessage))
		}
	}
	return nil
}
