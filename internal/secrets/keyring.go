// Package secrets provides secure storage for sensitive credentials
// using the OS native keychain (Windows Credential Manager, macOS Keychain, Linux Secret Service).
package secrets

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the keyring service name used for all entries.
	ServiceName = "nit-messenger"

	// KeyAPIKey is the keyring key name for the YDS API key.
	KeyAPIKey = "yds-api-key"
)

// ErrNotFound is returned when a secret is not found in the keyring.
var ErrNotFound = errors.New("secret not found in keyring")

// SetAPIKey stores the Yandex Data Streams API key securely in the OS keychain.
func SetAPIKey(apiKey string) error {
	if apiKey == "" {
		return errors.New("API key cannot be empty")
	}
	if err := keyring.Set(ServiceName, KeyAPIKey, apiKey); err != nil {
		return fmt.Errorf("failed to store API key in keyring: %w", err)
	}
	return nil
}

// GetAPIKey retrieves the Yandex Data Streams API key from the OS keychain.
// Returns ErrNotFound if the key is not stored.
func GetAPIKey() (string, error) {
	key, err := keyring.Get(ServiceName, KeyAPIKey)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to retrieve API key from keyring: %w", err)
	}
	return key, nil
}

// DeleteAPIKey removes the API key from the OS keychain.
func DeleteAPIKey() error {
	if err := keyring.Delete(ServiceName, KeyAPIKey); err != nil {
		if err == keyring.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to delete API key from keyring: %w", err)
	}
	return nil
}

// Set stores an arbitrary secret by name.
func Set(name, value string) error {
	if err := keyring.Set(ServiceName, name, value); err != nil {
		return fmt.Errorf("failed to store secret %q: %w", name, err)
	}
	return nil
}

// Get retrieves an arbitrary secret by name.
func Get(name string) (string, error) {
	val, err := keyring.Get(ServiceName, name)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to retrieve secret %q: %w", name, err)
	}
	return val, nil
}

// Delete removes an arbitrary secret by name.
func Delete(name string) error {
	if err := keyring.Delete(ServiceName, name); err != nil {
		if err == keyring.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to delete secret %q: %w", name, err)
	}
	return nil
}
