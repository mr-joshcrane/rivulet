package rivulet_test

import (
	"context"
	"encoding/json"
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

func TestPublisher_KeepsTrackOfNumberOfMessagesPublished(t *testing.T) {
	t.Parallel()
	transport := rivulet.NewMemoryTransport()
	reciever := transport.GetReceiver()
	p := rivulet.NewPublisher(t.Name(), rivulet.WithTransport(transport))
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
	p1 := rivulet.NewPublisher("p1", rivulet.WithTransport(transport))
	p2 := rivulet.NewPublisher("p2", rivulet.WithTransport(transport))
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

	p := rivulet.NewPublisher("test", rivulet.WithNetworkTransport(server.URL))
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
	_ = rivulet.NewPublisher("test", rivulet.WithEventBridgeTransport(eb))
}

func TestTransport_EventBridgeTransport(t *testing.T) {
	t.Parallel()
	client := &DummyEventBridge{}
	p := rivulet.NewPublisher("p1", rivulet.WithEventBridgeTransport(client))
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
	p := rivulet.NewPublisher("test", rivulet.WithNetworkTransport("httttp://badurl"))
	err := p.Publish("a line")
	if err == nil {
		t.Errorf("got nil, want error")
	}
}

func TestNetworkTransport_FailsGracefullyWithBadHTTPResponseCode(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(nil)
	defer server.Close()
	p := rivulet.NewPublisher("test", rivulet.WithNetworkTransport(server.URL))
	err := p.Publish("a line")
	if err == nil {
		t.Errorf("got nil, want error")
	}
}

func TestEventBridgeTransport_DetectsFailedPublishAttempts(t *testing.T) {
	t.Parallel()
	client := &BrokenEventBridge{}
	p := rivulet.NewPublisher("p1", rivulet.WithEventBridgeTransport(client))
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
				Detail:     aws.String(detail),
				DetailType: aws.String("rivulet"),
				Source:     aws.String("rivulet"),
				EventBusName: aws.String("default"),
			},
		},
	}
}
