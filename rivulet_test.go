package rivulet_test

import (
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/rivulet"
)

func TestProducer_PublishShouldPropagateDataToStore(t *testing.T) {
	t.Parallel()
	producer, store, wait := helperNewProducerWithBackingStore(t, "name")
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
}

func helperNewProducerWithBackingStore(t *testing.T, name string) (*rivulet.Producer, *rivulet.Store, func()) {
	t.Helper()
	store := rivulet.NewStore()
	producer := rivulet.NewProducer(name, rivulet.WithStore(store))
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		store.Receive()
		wg.Done()
	}()
	return producer, store, func() {
		producer.Close()
		wg.Wait()
	}
}
