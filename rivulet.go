package rivulet

type Store interface {
	Receive()
	Write(data string)
	Read() string
	Register(chan (string))
}

type Producer struct {
	name   string
	output chan (string)
}

type ProducerOptions func(*Producer)

func WithStore(s Store) ProducerOptions {
	c := make(chan (string))
	s.Register(c)
	return func(p *Producer) {
		p.output = c
	}
}

func NewProducer(name string, options ...ProducerOptions) *Producer {
	p := &Producer{
		name: name,
	}
	for _, option := range options {
		option(p)
	}
	return p
}

func (p *Producer) Publish(s ...string) error {
	for _, data := range s {
		p.output <- data
	}
	return nil
}

func (p *Producer) Close() {
	close(p.output)
}
