package rivulet_test

import (
	"context"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mr-joshcrane/rivulet"
)

func TestPublisher_CanPublish(t *testing.T) {
	tCases := []struct {
		description string
		input       any
		want        []string
	}{
		{
			description: "publishes a single line",
			input:       "a line",
			want:        []string{"a line"},
		},
		{
			description: "publishes multiple variadic lines",
			input:       []string{"line1", "line2", "line3"},
			want:        []string{"line1", "line2", "line3"},
		},
	}
	for _, tc := range tCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			messages := []rivulet.Message{}
			p := rivulet.NewPublisher(tc.description, rivulet.WithInMemoryTransport(&messages))
			str, more := func() (string, []string) {
				switch v := tc.input.(type) {
				case string:
					return v, nil
				case []string:
					return v[0], v[1:]
				}
				t.Fatalf("unexpected type %T", tc.input)
				return "", nil
			}()

			err := p.Publish(str, more...)
			if err != nil {
				t.Errorf("got %v, want nil", err)
			}
			got := orderedMessages(messages)
			if !cmp.Equal(got, tc.want) {
				t.Errorf(cmp.Diff(got, tc.want))
			}
		})
	}
}

func TestPublisher_KeepsTrackOfNumberOfMessagesPublished(t *testing.T) {
	t.Parallel()
	got := []rivulet.Message{}
	p := rivulet.NewPublisher(t.Name(), rivulet.WithInMemoryTransport(&got))
	if p.Counter() != 0 {
		t.Errorf("publisher should have published 0 messages, got %d", p.Counter())
	}
	for range 100 {
		err := p.Publish("a line")
		if err != nil {
			t.Errorf("got %v, want nil", err)
		}
	}
	if p.Counter() != 100 {
		t.Errorf("publisher should have published 100 messages, got %d", p.Counter())
	}
	if len(got) != 100 {
		t.Errorf("transport should have 100 messages, got %d", len(got))
	}
}

func TestPublisher_CanDifferentiateMessagesFromDifferentPublishers(t *testing.T) {
	t.Parallel()
	messages := []rivulet.Message{}
	p1 := rivulet.NewPublisher("p1", rivulet.WithInMemoryTransport(&messages))
	p2 := rivulet.NewPublisher("p2", rivulet.WithInMemoryTransport(&messages))
	err := p1.Publish("p1 line")
	if err != nil {
		t.Errorf("got %v, want nil", err)
	}
	for range 100 {
		err := p2.Publish("p2 line")
		if err != nil {
			t.Errorf("got %v, want nil", err)
		}
		err = p1.Publish("p1 line")
		if err != nil {
			t.Errorf("got %v, want nil", err)
		}
	}
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

//
// func TestTransport_NetworkTransport(t *testing.T) {
// 	t.Parallel()
// 	buf := new(bytes.Buffer)
// 	server := NewServer(func(w http.ResponseWriter, r *http.Request) {
// 		buf.ReadFrom(r.Body)
// 	})
// 	defer server.Close()
//
// 	addr := server.Listener.Addr().String()
// 	addr := server.Listener.Addr().Port
//
// 	p := rivulet.NewPublisher("test", rivulet.WithNetworkTransport(server.URL))
// }

func TestTransport_EventBridgeTransport_RealClientSatsfiesInterface(t *testing.T) {
	t.Parallel()
	cfg := aws.NewConfig()
	eb := eventbridge.NewFromConfig(*cfg)
	_ = rivulet.NewPublisher("test", rivulet.WithEventBridgeTransport(eb))
}

func TestTransport_EventBridgeTransport(t *testing.T) {
	t.Parallel()
	client := &MockEventBridgeClient{}
	p := rivulet.NewPublisher("p1", rivulet.WithEventBridgeTransport(client))
	err := p.Publish("first line", "second line")
	if err != nil {
		t.Errorf("got %v, want nil", err)
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

func TestTransport_HTTPTransport(t *testing.T) {
	t.Skip("not yet implemented")
}

func orderedMessages(messages []rivulet.Message) []string {
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Order < messages[j].Order
	})
	var result []string
	for _, m := range messages {
		result = append(result, m.Content)
	}
	return result
}

func groupByPublisher(messages []rivulet.Message) map[string][]string {
	result := make(map[string][]string)
	for _, m := range messages {
		result[m.Publisher] = append(result[m.Publisher], m.Content)
	}
	return result
}

type MockEventBridgeClient struct {
	Input []*eventbridge.PutEventsInput
}

func (c *MockEventBridgeClient) PutEvents(ctx context.Context, input *eventbridge.PutEventsInput, opts ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
	c.Input = append(c.Input, input)
	return &eventbridge.PutEventsOutput{}, nil
}
func helperPutEventsInput(detail string) *eventbridge.PutEventsInput {
	return &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				Detail:     aws.String(detail),
				DetailType: aws.String("rivulet"),
				Source:     aws.String("rivulet"),
			},
		},
	}
}
