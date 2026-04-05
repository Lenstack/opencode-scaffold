package core

type Storage interface {
	Put(namespace, key string, value any) error
	Get(namespace, key string, dest any) error
	Delete(namespace, key string) error
	Has(namespace, key string) bool
	Iterate(namespace string, fn func(key string, value []byte) error) error
	Count(namespace string) (int, error)
	PutWithTTL(namespace, key string, value any, ttl interface{}) error
	GetWithTTL(namespace, key string, dest any) error
	PruneExpired(namespace string) (int, error)
}

var ErrNotFound = &NotFoundError{}

type NotFoundError struct{}

func (e *NotFoundError) Error() string { return "not found" }
