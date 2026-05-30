// Package contacts manages the contact book — storing other users' public keys.
package contacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Contact represents a contact with their public keys.
type Contact struct {
	Name   string `json:"name"`
	AgeKey string `json:"age_key"` // age1... public key for encryption
}

// Book manages the list of contacts for a specific identity.
type Book struct {
	path string
}

// NewBook creates a contacts book stored at the given base path.
// Contacts are stored in <storagePath>/../contacts/ (global, shared)
func NewBook(storagePath string) *Book {
	return &Book{path: filepath.Join(storagePath, "..", "contacts")}
}

// NewBookForIdentity creates a personal contacts book for a specific identity.
// Contacts are stored in <storagePath>/../contacts/<identity>/
func NewBookForIdentity(storagePath, identityName string) *Book {
	return &Book{path: filepath.Join(storagePath, "..", "contacts", identityName)}
}

// Add adds or updates a contact.
func (b *Book) Add(name, ageKey string) error {
	if err := os.MkdirAll(b.path, 0700); err != nil {
		return fmt.Errorf("failed to create contacts directory: %w", err)
	}

	contact := &Contact{
		Name:   name,
		AgeKey: ageKey,
	}

	data, err := json.MarshalIndent(contact, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(b.path, name+".json")
	return os.WriteFile(path, data, 0600)
}

// Get retrieves a contact by name.
func (b *Book) Get(name string) (*Contact, error) {
	path := filepath.Join(b.path, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("contact %q not found\n\nAdd them first:\n  nit contacts add %s <age-pubkey> --identity <your-identity>", name, name)
		}
		return nil, err
	}

	var contact Contact
	if err := json.Unmarshal(data, &contact); err != nil {
		return nil, err
	}
	return &contact, nil
}

// List lists all contacts.
func (b *Book) List() ([]*Contact, error) {
	entries, err := os.ReadDir(b.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Contact{}, nil
		}
		return nil, err
	}

	var contacts []*Contact
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			contact, err := b.Get(name)
			if err == nil {
				contacts = append(contacts, contact)
			}
		}
	}
	return contacts, nil
}

// Delete removes a contact.
func (b *Book) Delete(name string) error {
	path := filepath.Join(b.path, name+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("contact %q not found", name)
		}
		return err
	}
	return nil
}
