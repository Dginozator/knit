package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"golang.org/x/crypto/argon2"
)

// DeriveKey derives a 32-byte key from password and salt using Argon2id.
func DeriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, 1, 64*1024, 4, 32)
}

// GenerateSalt generates a cryptographically random salt of the given size.
func GenerateSalt(size int) ([]byte, error) {
	salt := make([]byte, size)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// KeystoreErrors
var (
	ErrKeyStoreLocked         = errors.New("keystore is locked")
	ErrInvalidPassword        = errors.New("invalid password")
	ErrKeyNotFound            = errors.New("key not found")
	ErrKeyExists              = errors.New("key already exists")
	ErrInvalidKeystore        = errors.New("invalid keystore format")
	ErrSerializationFailed    = errors.New("serialization failed")
)

// Keystore provides encrypted storage for cryptographic keys.
type Keystore struct {
	path       string
	entries    map[string]*KeystoreEntry
	passphrase []byte
	unlocked   bool
}

// KeystoreEntry represents a single encrypted key entry.
type KeystoreEntry struct {
	Name       string `json:"name"`
	KeyType    string `json:"type"` // "identity", "signing", "age"
	Version    int    `json:"version"`
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// StoredIdentity represents the format stored in the keystore.
type StoredIdentity struct {
	Version      int    `json:"version"`
	AgeIdentity  string `json:"age_identity"` // age private key string
	SignSeed     []byte `json:"sign_seed"`
}

// NewKeystore creates a new keystore at the given path.
func NewKeystore(path string) (*Keystore, error) {
	ks := &Keystore{
		path:    path,
		entries: make(map[string]*KeystoreEntry),
	}
	return ks, nil
}

// Unlock unlocks the keystore with the given passphrase.
func (ks *Keystore) Unlock(passphrase string) error {
	if ks.unlocked {
		return nil
	}

	ks.passphrase = []byte(passphrase)
	ks.unlocked = true
	return nil
}

// Lock locks the keystore and clears sensitive data.
func (ks *Keystore) Lock() {
	ks.passphrase = nil
	ks.unlocked = false
	ks.entries = make(map[string]*KeystoreEntry)
}

// IsUnlocked returns whether the keystore is unlocked.
func (ks *Keystore) IsUnlocked() bool {
	return ks.unlocked
}

// GenerateIdentity generates a new identity and stores it in the keystore.
func (ks *Keystore) GenerateIdentity(name, password string) (*Identity, error) {
	if !ks.unlocked {
		return nil, ErrKeyStoreLocked
	}

	// Check if key already exists
	if _, err := ks.GetIdentity(name); err == nil {
		return nil, fmt.Errorf("%w: %s", ErrKeyExists, name)
	}

	// Generate new identity
	identity, err := NewIdentity()
	if err != nil {
		return nil, err
	}

	// Store identity
	if err := ks.storeIdentity(name, password, identity); err != nil {
		return nil, err
	}

	return identity, nil
}

// GetIdentity retrieves an identity from the keystore.
func (ks *Keystore) GetIdentity(name string) (*Identity, error) {
	if !ks.unlocked {
		return nil, ErrKeyStoreLocked
	}

	// Try to find in entries first
	if entry, ok := ks.entries[name]; ok {
		return ks.unmarshalIdentity(entry)
	}

	// Try to load from file
	entry, err := ks.loadEntry(name)
	if err != nil {
		return nil, err
	}

	ks.entries[name] = entry
	return ks.unmarshalIdentity(entry)
}

// ListIdentities lists all identities in the keystore.
func (ks *Keystore) ListIdentities() ([]string, error) {
	entries, err := os.ReadDir(ks.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read keystore: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			names = append(names, name)
		}
	}
	return names, nil
}

// DeleteIdentity removes an identity from the keystore.
func (ks *Keystore) DeleteIdentity(name string) error {
	if !ks.unlocked {
		return ErrKeyStoreLocked
	}

	path := filepath.Join(ks.path, name+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete identity: %w", err)
	}

	delete(ks.entries, name)
	return nil
}

// storeIdentity stores an identity in the keystore.
func (ks *Keystore) storeIdentity(name, password string, identity *Identity) error {
	// Get sign seed
	signSeed := identity.signIdentity.Seed()

	// Get the age private key as string
	ageIdentityStr, err := identity.GetAgePrivateKey()
	if err != nil {
		return fmt.Errorf("failed to get age private key: %w", err)
	}

	// Create stored identity
	stored := StoredIdentity{
		Version:     1,
		AgeIdentity: ageIdentityStr,
		SignSeed:    signSeed,
	}

	// Serialize
	data, err := json.Marshal(stored)
	if err != nil {
		return ErrSerializationFailed
	}

	// Encrypt - use the keystore's own passphrase for encryption
	// (password param is accepted for API compatibility but keystore uses its own passphrase)
	encryptPassword := password
	if len(ks.passphrase) > 0 {
		encryptPassword = string(ks.passphrase)
	}
	dataStr := string(data)
	entry, err := ks.encryptEntry(name, "identity", dataStr, encryptPassword)
	if err != nil {
		return err
	}

	// Save to file
	if err := ks.saveEntry(entry); err != nil {
		return err
	}

	ks.entries[name] = entry
	return nil
}

// unmarshalIdentity unmarshals an identity from a keystore entry.
func (ks *Keystore) unmarshalIdentity(entry *KeystoreEntry) (*Identity, error) {
	// Decrypt
	data, err := ks.decryptEntry(entry)
	if err != nil {
		return nil, err
	}

	// Unmarshal - data is a string, convert to bytes
	dataBytes := []byte(data)

	// Unmarshal
	var stored StoredIdentity
	if err := json.Unmarshal(dataBytes, &stored); err != nil {
		return nil, ErrInvalidKeystore
	}

	// Reconstruct identity
	signKey := ed25519.NewKeyFromSeed(stored.SignSeed)

	// Parse age identity
	ageIdentities, err := age.ParseIdentities(strings.NewReader(stored.AgeIdentity))
	if err != nil {
		return nil, fmt.Errorf("failed to parse age identity: %w", err)
	}

	return &Identity{
		ageIdentities: ageIdentities,
		signIdentity:  signKey,
	}, nil
}

// encryptEntry encrypts data and creates a keystore entry.
func (ks *Keystore) encryptEntry(name, keyType, data, password string) (*KeystoreEntry, error) {
	// Generate salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using argon2id
	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Create cipher
	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, []byte(data), nil)

	return &KeystoreEntry{
		Name:       name,
		KeyType:    keyType,
		Version:    1,
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}, nil
}

// decryptEntry decrypts a keystore entry.
func (ks *Keystore) decryptEntry(entry *KeystoreEntry) (string, error) {
	// Derive key using argon2id
	key := argon2.IDKey(ks.passphrase, entry.Salt, 1, 64*1024, 4, 32)

	// Create cipher
	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	data, err := gcm.Open(nil, entry.Nonce, entry.Ciphertext, nil)
	if err != nil {
		return "", ErrInvalidPassword
	}

	return string(data), nil
}

// saveEntry saves a keystore entry to disk.
func (ks *Keystore) saveEntry(entry *KeystoreEntry) error {
	// Ensure directory exists
	if err := os.MkdirAll(ks.path, 0700); err != nil {
		return fmt.Errorf("failed to create keystore directory: %w", err)
	}

	// Marshal entry
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	// Write file
	path := filepath.Join(ks.path, entry.Name+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	return nil
}

// loadEntry loads a keystore entry from disk.
func (ks *Keystore) loadEntry(name string) (*KeystoreEntry, error) {
	path := filepath.Join(ks.path, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to read entry: %w", err)
	}

	var entry KeystoreEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, ErrInvalidKeystore
	}

	return &entry, nil
}

// ExportIdentity exports an identity to an encrypted backup.
func (ks *Keystore) ExportIdentity(name, password string) ([]byte, error) {
	entry, err := ks.loadEntry(name)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entry: %w", err)
	}

	// Re-encrypt with provided password
	newEntry, err := ks.encryptEntry(name, entry.KeyType, string(data), password)
	if err != nil {
		return nil, err
	}

	return json.Marshal(newEntry)
}

// ImportIdentity imports an identity from an encrypted backup.
func (ks *Keystore) ImportIdentity(name, password string, data []byte) error {
	if !ks.unlocked {
		return ErrKeyStoreLocked
	}

	var entry KeystoreEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return ErrInvalidKeystore
	}

	// Decrypt to verify password
	if _, err := ks.decryptEntry(&entry); err != nil {
		return err
	}

	// Re-save with current keystore password
	return ks.saveEntry(&entry)
}
