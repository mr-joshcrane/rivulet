package rivulet_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mr-joshcrane/rivulet"
)

func TestPublisher_NewPublisherByDefaultCreatesAMemoryPublisherAndSubscriber(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1000)
	defer cancel()
	p, s := rivulet.NewMemoryPublisher(t.Name())
	go func() {
		err := s.Receive(ctx)
		if err != nil {
			fmt.Println(err)
		}
	}()
	err := p.Publish("a line")
	if err != nil {
		t.Fatalf("got %v, want nil", err)
	}
	time.Sleep(time.Millisecond * 2000)
	messages, err := s.Messages()
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Fatalf("wanted 1 message, got %d", len(messages))
	}
	message := messages[0]
	if message.Content != "a line" {
		t.Errorf("wanted %q, got %q", "a line", message.Content)
	}
	if message.Publisher != t.Name() {
		t.Errorf("wanted %q, got %q", t.Name(), message.Publisher)
	}
	if message.Order != 1 {
		t.Errorf("wanted 1, got %d", message.Order)
	}

}

func TestPublisher_KeepsTrackOfNumberOfMessagesPublished(t *testing.T) {
	t.Parallel()
	transport := rivulet.NewMemoryTransport()
	reciever := transport.GetReceiver()
	p, _ := rivulet.NewMemoryPublisher(t.Name(), rivulet.WithTransport(transport))
	if p.Counter() != 0 {
		t.Errorf("publisher should have published 0 messages, got %d", p.Counter())
	}
	for i := 0; i < 100; i++ {
		err := p.Publish("a line")
		if err != nil {
			t.Errorf("got %v, want nil", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*20)
	defer cancel()
	got := reciever.Receive(ctx)
	if p.Counter() != 100 {
		t.Errorf("publisher should have published 100 messages, got %d", p.Counter())
	}
	if len(got) != 100 {
		t.Errorf("transport should have 100 messages, got %d", len(got))
	}
}

func TestPublisher_CanDifferentiateMessagesFromDifferentPublishers(t *testing.T) {
	t.Parallel()
	transport := rivulet.NewMemoryTransport()
	receiver := transport.GetReceiver()
	p1, _ := rivulet.NewMemoryPublisher("p1", rivulet.WithTransport(transport))
	p2, _ := rivulet.NewMemoryPublisher("p2", rivulet.WithTransport(transport))
	err := p1.Publish("p1 line")
	if err != nil {
		t.Errorf("got %v, want nil", err)
	}
	for i := 0; i < 100; i++ {
		err := p2.Publish("p2 line")
		if err != nil {
			t.Errorf("got %v, want nil", err)
		}
		err = p1.Publish("p1 line")
		if err != nil {
			t.Errorf("got %v, want nil", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*20)
	defer cancel()
	messages := receiver.Receive(ctx)
	publishers := groupByPublisher(messages)
	if len(publishers) != 2 {
		t.Errorf("transport should have 2 publishers, got %d", len(publishers))
	}
	var keys []string
	for k := range publishers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if !cmp.Equal(keys, []string{"p1", "p2"}) {
		t.Errorf(cmp.Diff(keys, []string{"p1", "p2"}))
	}
	if len(publishers["p1"]) != 101 {
		t.Errorf("transport should have 101 p1 messages, got %d", len(publishers["p1"]))
	}
	if len(publishers["p2"]) != 100 {
		t.Errorf("transport should have 100 p2 messages, got %d", len(publishers["p2"]))
	}
}

func TestTransport_NetworkTransport(t *testing.T) {
	t.Parallel()
	got := []rivulet.Message{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message rivulet.Message
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, message)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	p, _ := rivulet.NewMemoryPublisher("test", rivulet.WithNetworkTransport(server.URL))
	err := p.Publish("first line")
	if err != nil {
		t.Fatal(err)
	}
	err = p.Publish("second line")
	if err != nil {
		t.Fatal(err)
	}
	want := []rivulet.Message{
		{Publisher: "test", Order: 1, Content: "first line"},
		{Publisher: "test", Order: 2, Content: "second line"},
	}
	if !cmp.Equal(want, got) {
		t.Fatalf(cmp.Diff(want, got))
	}
}

func TestTransport_EventBridgeTransport_RealClientSatsfiesInterface(t *testing.T) {
	t.Parallel()
	cfg := aws.NewConfig()
	eb := eventbridge.NewFromConfig(*cfg)
	_, _ = rivulet.NewMemoryPublisher("test", rivulet.WithEventBridgeTransport(eb))
}

func TestTransport_EventBridgeTransport(t *testing.T) {
	t.Parallel()
	client := &DummyEventBridge{}
	p, _ := rivulet.NewMemoryPublisher("p1", rivulet.WithEventBridgeTransport(client))
	for _, line := range []string{"first line", "second line"} {
		err := p.Publish(line)
		if err != nil {
			t.Fatal(err)
		}
	}
	want := []*eventbridge.PutEventsInput{
		helperPutEventsInput(`{"Publisher":"p1","Order":1,"Content":"first line"}`),
		helperPutEventsInput(`{"Publisher":"p1","Order":2,"Content":"second line"}`),
	}
	got := client.Input
	ignore := cmpopts.IgnoreUnexported(types.PutEventsRequestEntry{}, eventbridge.PutEventsInput{})
	if !cmp.Equal(got, want, ignore) {
		t.Errorf(cmp.Diff(want, got, ignore))
	}
}

func TestNetworkTransport_FailsGracefullyWithBadURL(t *testing.T) {
	t.Parallel()
	p, _ := rivulet.NewMemoryPublisher("test", rivulet.WithNetworkTransport("httttp://badurl"))
	err := p.Publish("a line")
	if err == nil {
		t.Errorf("got nil, want error")
	}
}

func TestNetworkTransport_FailsGracefullyWithBadHTTPResponseCode(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(nil)
	defer server.Close()
	p, _ := rivulet.NewMemoryPublisher("test", rivulet.WithNetworkTransport(server.URL))
	err := p.Publish("a line")
	if err == nil {
		t.Errorf("got nil, want error")
	}
}

func TestEventBridgeTransport_DetectsFailedPublishAttempts(t *testing.T) {
	t.Parallel()
	client := &BrokenEventBridge{}
	p, _ := rivulet.NewMemoryPublisher("p1", rivulet.WithEventBridgeTransport(client))
	err := p.Publish("a line")
	if err == nil {
		t.Errorf("got nil, want error")
	}

}

func TestEventBridgeTransport_CanSetUpWithDifferentOptions(t *testing.T) {
	t.Parallel()
	client := &DummyEventBridge{}
	p, _ := rivulet.NewMemoryPublisher("p1", rivulet.WithEventBridgeTransport(
		client,
		rivulet.WithDetailType("test"),
		rivulet.WithSource("test"),
		rivulet.WithEventBusName("test"),
	),
	)
	err := p.Publish("a line")
	if err != nil {
		t.Fatal(err)
	}
	got := client.Input
	if len(got) != 1 {
		t.Fatalf("expected 1 input, got %d", len(got))
	}
	if *got[0].Entries[0].DetailType != "test" {
		t.Errorf("expected %q, got %q", "test", *got[0].Entries[0].DetailType)
	}
	if *got[0].Entries[0].Source != "test" {
		t.Errorf("expected %q, got %q", "test", *got[0].Entries[0].Source)
	}
	if *got[0].Entries[0].EventBusName != "test" {
		t.Errorf("expected %q, got %q", "test", *got[0].Entries[0].EventBusName)
	}
}

func TestEventBridgeTransport_CanTakeUserProvidedMessageTransform(t *testing.T) {
	t.Parallel()
	client := &DummyEventBridge{}
	transform := func(m rivulet.Message) (string, error) {
		transformed := struct {
			Original                     rivulet.Message
			SomeArbitraryAdditionalField string
		}{
			Original:                     m,
			SomeArbitraryAdditionalField: "arbitrary",
		}
		data, err := json.Marshal(transformed)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	p, _ := rivulet.NewMemoryPublisher("p1", rivulet.WithEventBridgeTransport(client, rivulet.WithTransform(transform)))
	err := p.Publish("a line")
	if err != nil {
		t.Fatal(err)
	}
	got := client.Input
	if len(got) != 1 {
		t.Fatalf("expected 1 input, got %d", len(got))
	}
	want := `{"Original":{"Publisher":"p1","Order":1,"Content":"a line"},"SomeArbitraryAdditionalField":"arbitrary"}`
	if *got[0].Entries[0].Detail != want {
		t.Errorf("expected %q, got %q", want, *got[0].Entries[0].Detail)
	}
}

func TestEventBridgeTransport_AWellDefinedUserTransformErroringIsHandledByPublish(t *testing.T) {
	t.Parallel()
	client := &DummyEventBridge{}
	transform := func(rivulet.Message) (string, error) {
		return "", fmt.Errorf("a user transform that correctly handles its errors")
	}
	p, _ := rivulet.NewMemoryPublisher("p1", rivulet.WithEventBridgeTransport(client, rivulet.WithTransform(transform)))
	err := p.Publish("a line")
	if err == nil {
		t.Errorf("got nil, want error")
	}
}

func TestEventBridgeTransport_APoorlyDefinedUserTransformIsCaughtByPublish(t *testing.T) {
	t.Parallel()
	client := &DummyEventBridge{}
	transform := func(rivulet.Message) (string, error) {
		return "", nil
	}
	p, _ := rivulet.NewMemoryPublisher("p1", rivulet.WithEventBridgeTransport(client, rivulet.WithTransform(transform)))
	err := p.Publish("a line")
	if err == nil {
		t.Errorf("got nil, want error")
	}
}

func groupByPublisher(messages []rivulet.Message) map[string][]string {
	result := make(map[string][]string)
	for _, m := range messages {
		result[m.Publisher] = append(result[m.Publisher], m.Content)
	}
	return result
}

type DummyEventBridge struct {
	Input []*eventbridge.PutEventsInput
}

func (c *DummyEventBridge) PutEvents(ctx context.Context, input *eventbridge.PutEventsInput, opts ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
	c.Input = append(c.Input, input)
	return &eventbridge.PutEventsOutput{}, nil
}

type BrokenEventBridge struct{}

func (b *BrokenEventBridge) PutEvents(ctx context.Context, input *eventbridge.PutEventsInput, opts ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
	return &eventbridge.PutEventsOutput{
		FailedEntryCount: 1,
		Entries: []types.PutEventsResultEntry{
			{
				ErrorCode:    aws.String("400"),
				ErrorMessage: aws.String("ThrottlingException"),
			},
		},
	}, nil
}

func helperPutEventsInput(detail string) *eventbridge.PutEventsInput {
	return &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				Detail:       aws.String(detail),
				DetailType:   aws.String("rivulet"),
				Source:       aws.String("rivulet"),
				EventBusName: aws.String("default"),
			},
		},
	}
}
