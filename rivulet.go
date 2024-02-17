package rivulet

type Store struct {
}

type Producer struct {
}

func WithStore(s *Store) func(*Producer) {
	return func(p *Producer) {
	}
}

func NewProducer(name string, options ...func(*Producer)) *Producer {
	return &Producer{}
}

func (p *Producer) Publish(dataStream chan (string)) error {
	return nil
}

func (p *Producer) Close() {
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Read() string {
	return ""
}

func (s *Store) Close() {
}
