package secrets

import "errors"

// Store is the interface for credential storage
// Implementations will be added in Plan 02 (keyring and file)
type Store interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
	List() ([]string, error)
}

// ErrNotFound is returned when a key is not found in the store
var ErrNotFound = errors.New("key not found")

// ServiceName is the service identifier for keyring storage
const ServiceName = "zoh"
