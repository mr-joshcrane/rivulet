package rivulet_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/rivulet"
	"github.com/mr-joshcrane/rivulet/store"
)

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
	}
	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			store := tc.store
			producer, wait := helperNewProducerWithBackingStore(t, store)
			err := producer.Publish("line1\n", "line2\n", "line3")
			if err != nil {
				t.Errorf("got %v, want nil", err)
			}
			wait()
			got := store.Read()
			want := "line1\nline2\nline3"
			if got != want {
				t.Errorf(cmp.Diff(got, want))
			}
		})
	}

}

func helperNewProducerWithBackingStore(t *testing.T, s rivulet.Store) (*rivulet.Producer, func()) {
	t.Helper()
	producer := rivulet.NewProducer("test", rivulet.WithStore(s))
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		s.Receive()
		wg.Done()
	}()
	return producer, func() {
		producer.Close()
		wg.Wait()
	}
}
