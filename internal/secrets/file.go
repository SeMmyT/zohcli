package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

// FileStore implements the Store interface using an AES-256-GCM encrypted file.
// This is a fallback for environments where OS keyring is unavailable (WSL, headless, Docker).
type FileStore struct {
	path string
	key  []byte
}

// NewFileStore creates a new file-backed credential store.
// If password is empty, uses a machine-specific default (less secure, prints warning).
// Future improvement: use scrypt or argon2 for key derivation instead of sha256.
func NewFileStore(password string) (*FileStore, error) {
	path := filepath.Join(xdg.DataHome, "zoh", "credentials.enc")

	var key []byte
	if password == "" {
		// Machine-specific default (less secure than user-provided password)
		hostname, _ := os.Hostname()
		username := os.Getenv("USER")
		if username == "" {
			username = os.Getenv("USERNAME") // Windows fallback
		}
		machineID := fmt.Sprintf("%s@%s", username, hostname)
		hash := sha256.Sum256([]byte(machineID))
		key = hash[:]
		warnOnce("WARNING: Using machine-specific encryption key. For better security, set a password via ZOH_STORE_PASSWORD env var.")
	} else {
		// Derive key from password using sha256 (simple for v1)
		// TODO: Replace with scrypt or argon2 for better security
		hash := sha256.Sum256([]byte(password))
		key = hash[:]
	}

	// Create parent directory with 0700 permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create credentials directory: %w", err)
	}

	return &FileStore{
		path: path,
		key:  key,
	}, nil
}

// encrypt encrypts plaintext using AES-256-GCM with a random 12-byte nonce.
// The nonce is prepended to the ciphertext.
func (s *FileStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts ciphertext that was encrypted with encrypt().
// Extracts the nonce from the first 12 bytes.
func (s *FileStore) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// readStore decrypts and parses the credential file.
// Returns an empty map if the file doesn't exist.
func (s *FileStore) readStore() (map[string]string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	if len(data) == 0 {
		return make(map[string]string), nil
	}

	plaintext, err := s.decrypt(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	var store map[string]string
	if err := json.Unmarshal(plaintext, &store); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return store, nil
}

// writeStore encrypts and writes the credential map to disk.
func (s *FileStore) writeStore(store map[string]string) error {
	plaintext, err := json.Marshal(store)
	if err != nil {
		return fmt.Errorf("failed to serialize credentials: %w", err)
	}

	ciphertext, err := s.encrypt(plaintext)
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.path, ciphertext, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// Get retrieves a credential by key from the encrypted file.
func (s *FileStore) Get(key string) (string, error) {
	store, err := s.readStore()
	if err != nil {
		return "", err
	}

	value, ok := store[key]
	if !ok {
		return "", ErrNotFound
	}

	return value, nil
}

// Set stores a credential in the encrypted file.
func (s *FileStore) Set(key, value string) error {
	store, err := s.readStore()
	if err != nil {
		return err
	}

	store[key] = value
	return s.writeStore(store)
}

// Delete removes a credential from the encrypted file.
func (s *FileStore) Delete(key string) error {
	store, err := s.readStore()
	if err != nil {
		return err
	}

	if _, ok := store[key]; !ok {
		return ErrNotFound
	}

	delete(store, key)
	return s.writeStore(store)
}

// List returns all credential keys from the encrypted file.
func (s *FileStore) List() ([]string, error) {
	store, err := s.readStore()
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(store))
	for k := range store {
		keys = append(keys, k)
	}

	return keys, nil
}
