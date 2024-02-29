package rivulet_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/coverage"
	"github.com/mr-joshcrane/rivulet"
	"github.com/mr-joshcrane/rivulet/store"
)

func TestMain(m *testing.M) {
	os.Exit(coverage.ExtendCoverage(m, "rivulet"))
}

func TestProducer_PublishShouldPropagateDataToStore(t *testing.T) {
	t.Parallel()
	cases := []struct {
		description string
		store       rivulet.Store
	}{
		{
			description: "file store",
			store: func() rivulet.Store {
				dir := fmt.Sprintf(t.TempDir(), "test.txt")
				s, _ := store.NewFileStore(dir)
				return s
			}(),
		},
		{
			description: "memory store",
			store:       store.NewMemoryStore(),
		},
		// {
		// 	description: "eventbridge store",
		// 	store:       store.NewEventBridgeStore(),
		// },
	}
	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			store := tc.store
			publisher, wait := helperNewProducerWithBackingStore(t, store)
			err := publisher.Publish("line1\n", "line2\n", "line3")
			if err != nil {
				t.Errorf("got %v, want nil", err)
			}
			err = wait()
			if err != nil {
				t.Error(err)
			}
			got := store.Read()
			want := "line1\nline2\nline3"
			if got != want {
				t.Errorf(cmp.Diff(got, want))
			}
		})
	}
}

func helperNewProducerWithBackingStore(t *testing.T, s rivulet.Store) (*rivulet.Publisher, func() error) {
	t.Helper()
	producer := rivulet.NewPublisher("test", rivulet.WithStore(s))
	done := make(chan bool)
	go func() {
		s.Receive()
		done <- true
	}()
	return producer, func() error {
		producer.Close()
		select {
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timeout out waiting for store close: test %s", t.Name())
		}
	}
}
