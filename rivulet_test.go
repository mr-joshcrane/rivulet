package rivulet_test

import (
	"strings"
	"testing"

	"github.com/mr-joshcrane/rivulet"
)

func TestProducer_PublishShouldPropagateDataToStore(t *testing.T) {
	t.Parallel()
	producer, store, cleanup := helperNewProducerWithBackingStore(t, "name")
	defer cleanup()
	dataStream := helperDataStream("line1\nline2\nline3\n")
	producer.Publish(dataStream)
	got := store.Read()
	want := "line1\nline2\nline3\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func helperNewProducerWithBackingStore(t *testing.T, name string) (*rivulet.Producer, *rivulet.Store, func()) {
	t.Helper()
	store := rivulet.NewStore()
	producer := rivulet.NewProducer(name, rivulet.WithStore(store))
	return producer, store, func() {
		producer.Close()
		store.Close()
	}
}

func helperDataStream(data string) chan (string) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for _, line := range strings.Split(data, "\n") {
			ch <- line
		}
	}()
	return ch
}
