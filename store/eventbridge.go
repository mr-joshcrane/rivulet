package store

type EventBridgeStore struct {
}

func NewEventBridgeStore() *EventBridgeStore {
	return &EventBridgeStore{}
}

func (s *EventBridgeStore) Receive()          {}
func (s *EventBridgeStore) Write(data string) {}
func (s *EventBridgeStore) Read() string {
	return "eventbridge"
}
func (s *EventBridgeStore) Register(chan string) {}
