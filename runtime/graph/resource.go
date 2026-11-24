package graph

type Resource interface {
	ID() string
	Close() error
}
