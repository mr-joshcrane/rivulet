package rivulet

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
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

type EventBridgeClient interface {
	PutEvents(ctx context.Context, events *eventbridge.PutEventsInput) (*eventbridge.PutEventsOutput, error)
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
	if resp.FailedEntryCount != 0 {
		if len(resp.Entries) == 1 {
			return fmt.Errorf("failed to send message: %s", *resp.Entries[0].ErrorMessage)
		}
	}
	return err
}
