package rivulet

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go/aws"
)

type Store interface {
	Receive()
	Write(data string) error
	Read() string
	Register(chan (string))
}

type Publisher struct {
	name    string
	output  chan (string)
	counter atomic.Int64
}

type PublisherOptions func(*Publisher)

func WithStore(s Store) PublisherOptions {
	c := make(chan (string))
	s.Register(c)
	return func(p *Publisher) {
		p.output = c
	}
}

func NewPublisher(name string, options ...PublisherOptions) *Publisher {
	p := &Publisher{
		name: name,
	}
	for _, option := range options {
		option(p)
	}
	return p
}

func (p *Publisher) Publish(s ...string) error {
	for _, data := range s {
		p.counter.Add(1)
		p.output <- data
	}
	return nil
}

func (p *Publisher) Close() {
	close(p.output)
}

type EventBridgePublisher struct {
	eventBridge *eventbridge.Client
}

func NewEventBridgePublisher() *EventBridgePublisher {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}
	return &EventBridgePublisher{
		eventBridge: eventbridge.NewFromConfig(cfg),
	}
}

func (p *EventBridgePublisher) Publish(s string) error {
	detail, err := json.Marshal(struct {
		Message string `json:"message"`
	}{
		Message: s,
	})
	if err != nil {
		panic(err)
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
	resp, err := p.eventBridge.PutEvents(context.Background(), putEventsInput)
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
