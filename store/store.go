package store

type Store interface {
	// Will be a []Message
	Save(interface{}) error
}
