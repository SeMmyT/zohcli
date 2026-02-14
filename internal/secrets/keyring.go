package secrets

import (
	"fmt"

	"github.com/99designs/keyring"
	"github.com/adrg/xdg"
)

// KeyringStore implements the Store interface using the OS keyring.
type KeyringStore struct {
	ring keyring.Keyring
}

// NewKeyringStore creates a new keyring-backed credential store.
// Returns an error if the keyring is unavailable on this platform.
func NewKeyringStore() (*KeyringStore, error) {
	cfg := keyring.Config{
		ServiceName:              "zoh",
		KeychainTrustApplication: true, // macOS: don't prompt every access
		FileDir:                  xdg.DataHome + "/zoh/keyring",
		FilePasswordFunc:         keyring.TerminalPrompt,
	}

	ring, err := keyring.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	return &KeyringStore{ring: ring}, nil
}

// Get retrieves a credential by key from the keyring.
func (s *KeyringStore) Get(key string) (string, error) {
	item, err := s.ring.Get(key)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("keyring get failed: %w", err)
	}
	return string(item.Data), nil
}

// Set stores a credential in the keyring.
func (s *KeyringStore) Set(key, value string) error {
	item := keyring.Item{
		Key:  key,
		Data: []byte(value),
	}
	if err := s.ring.Set(item); err != nil {
		return fmt.Errorf("keyring set failed: %w", err)
	}
	return nil
}

// Delete removes a credential from the keyring.
func (s *KeyringStore) Delete(key string) error {
	if err := s.ring.Remove(key); err != nil {
		if err == keyring.ErrKeyNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("keyring delete failed: %w", err)
	}
	return nil
}

// List returns all credential keys stored in the keyring.
func (s *KeyringStore) List() ([]string, error) {
	keys, err := s.ring.Keys()
	if err != nil {
		return nil, fmt.Errorf("keyring list failed: %w", err)
	}
	return keys, nil
}
