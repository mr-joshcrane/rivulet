package rivulet_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/rivulet"
)

func TestProducer_PublishShouldPropagateDataToStore(t *testing.T) {
	t.Parallel()
	producer, store, cleanup := helperNewProducerWithBackingStore(t, "name")
	defer cleanup()
	dataStream := helperDataStream("line1\nline2\nline3")
	err := producer.Publish(dataStream)
	if err != nil {
		t.Errorf("got %v, want nil", err)
	}
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
	go store.Receive()
	return producer, store, func() {
		producer.Close()
		store.Close()
	}
}

func helperDataStream(data string) chan (string) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for _, character := range data {
			ch <- string(character)
		}
	}()
	return ch
}
