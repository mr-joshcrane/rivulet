package rivulet

import (
	"sync/atomic"
)

type Store interface {
	Receive()
	Write(data string) error
	Read() string
	Register(chan (string))
}

type Publisher struct {
	name      string
	transport Transport
	counter   atomic.Int64
}

type Message struct {
	Publisher string
	Order     int
	Content   string
}

type PublisherOptions func(*Publisher)

func NewPublisher(name string, options ...PublisherOptions) *Publisher {
	p := &Publisher{
		name: name,
	}
	for _, option := range options {
		option(p)
	}
	return p
}

func (p *Publisher) Counter() int64 {
	return p.counter.Load()
}

func (p *Publisher) Publish(str string, more ...string) error {
	s := append([]string{str}, more...)
	for _, data := range s {
		p.counter.Add(1)
		m := Message{
			Publisher: p.name,
			Order:     int(p.counter.Load()),
			Content:   data,
		}
		err := p.transport.Publish(m)
		if err != nil {
			return err
		}
	}
	return nil
}

//
// type EventBridgePublisher struct {
// 	eventBridge *eventbridge.Client
// }
//
// func NewEventBridgePublisher() *EventBridgePublisher {
// 	cfg, err := config.LoadDefaultConfig(context.Background())
// 	if err != nil {
// 		panic(err)
// 	}
// 	return &EventBridgePublisher{
// 		eventBridge: eventbridge.NewFromConfig(cfg),
// 	}
// }
//
// func (p *EventBridgePublisher) Publish(s string) error {
// 	detail, err := json.Marshal(struct {
// 		Message string `json:"message"`
// 	}{
// 		Message: s,
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	putEventsInput := &eventbridge.PutEventsInput{
// 		Entries: []types.PutEventsRequestEntry{
// 			{
// 				Detail:     aws.String(string(detail)),
// 				DetailType: aws.String("rivulet"),
// 				Source:     aws.String("rivulet"),
// 			},
// 		},
// 	}
// 	resp, err := p.eventBridge.PutEvents(context.Background(), putEventsInput)
// 	if err != nil {
// 		return err
// 	}
// 	if resp.FailedEntryCount != 0 {
// 		if len(resp.Entries) == 1 {
// 			return fmt.Errorf("failed to send message: %s", *resp.Entries[0].ErrorMessage)
// 		}
// 	}
// 	return err
// }
