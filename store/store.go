package store

type Message struct {
	Publisher string
	Order     int
	Content   string
}
type Store interface {
	// Will be a []Message
	Save([]Message) error
	Messages(string) []Message
}
