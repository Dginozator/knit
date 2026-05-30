// Package crypto provides end-to-end encryption using age and digital signatures using ed25519.
package crypto

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
)

// Common errors
var (
	ErrInvalidKey       = errors.New("invalid key")
	ErrDecryptionFailed = errors.New("decryption failed")
	ErrSigningFailed    = errors.New("signing failed")
)

// Identity represents a user's cryptographic identity containing both
// encryption (age) and signing (ed25519) keys.
type Identity struct {
	ageIdentities []age.Identity
	signIdentity  ed25519.PrivateKey
}

// Recipient represents a recipient's public keys for encryption.
type Recipient struct {
	ageRecipient age.Recipient
	signKey      ed25519.PublicKey
}

// NewIdentity creates a new cryptographic identity with fresh keys.
func NewIdentity() (*Identity, error) {
	// Generate ed25519 signing key
	_, signKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signing key: %w", err)
	}

	// Generate age X25519 identity
	ageIdentity, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("failed to generate age identity: %w", err)
	}

	return &Identity{
		ageIdentities: []age.Identity{ageIdentity},
		signIdentity:  signKey,
	}, nil
}

// NewIdentityFromRecipients creates an identity from an age private key string.
func NewIdentityFromRecipients(identityStr string) (*Identity, error) {
	identities, err := age.ParseIdentities(strings.NewReader(identityStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse age identity: %w", err)
	}

	// Generate ed25519 signing key
	_, signKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signing key: %w", err)
	}

	return &Identity{
		ageIdentities: identities,
		signIdentity:  signKey,
	}, nil
}

// ParseAgeRecipient parses an age public key string into a Recipient.
func ParseAgeRecipient(recipientStr string) (*Recipient, error) {
	recipients, err := age.ParseRecipients(strings.NewReader(recipientStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse age recipient: %w", err)
	}

	if len(recipients) == 0 {
		return nil, errors.New("no recipients found")
	}

	return &Recipient{
		ageRecipient: recipients[0],
		signKey:      nil,
	}, nil
}

// GetAgeRecipient returns the age Recipient (public key) for this identity.
func (id *Identity) GetAgeRecipient() (age.Recipient, error) {
	if len(id.ageIdentities) == 0 {
		return nil, ErrInvalidKey
	}
	// X25519Identity implements both Identity and Recipient
	x25519Identity, ok := id.ageIdentities[0].(*age.X25519Identity)
	if !ok {
		return nil, fmt.Errorf("identity is not an X25519Identity")
	}
	return x25519Identity.Recipient(), nil
}

// GetAgePublicKey returns the age public key as a string (for sharing with others).
func (id *Identity) GetAgePublicKey() (string, error) {
	recipient, err := id.GetAgeRecipient()
	if err != nil {
		return "", err
	}
	x25519Recipient, ok := recipient.(*age.X25519Recipient)
	if !ok {
		return "", fmt.Errorf("recipient is not an X25519Recipient")
	}
	return x25519Recipient.String(), nil
}

// GetAgePrivateKey returns the age private key as a string (for storage).
func (id *Identity) GetAgePrivateKey() (string, error) {
	if len(id.ageIdentities) == 0 {
		return "", ErrInvalidKey
	}
	x25519Identity, ok := id.ageIdentities[0].(*age.X25519Identity)
	if !ok {
		return "", fmt.Errorf("identity is not an X25519Identity")
	}
	return x25519Identity.String(), nil
}

// Sign signs a message using the identity's ed25519 private key.
func (id *Identity) Sign(message []byte) ([]byte, error) {
	if id.signIdentity == nil {
		return nil, ErrInvalidKey
	}
	return ed25519.Sign(id.signIdentity, message), nil
}

// SignPublicKey returns the public key for signature verification.
func (id *Identity) SignPublicKey() ed25519.PublicKey {
	if id.signIdentity == nil {
		return nil
	}
	return id.signIdentity.Public().(ed25519.PublicKey)
}

// RecipientFromPublicKey creates a Recipient from public key components.
func RecipientFromPublicKey(ageRecipient age.Recipient, signKey ed25519.PublicKey) *Recipient {
	return &Recipient{
		ageRecipient: ageRecipient,
		signKey:      signKey,
	}
}

// Encrypt encrypts a message for the given recipients using age.
func Encrypt(message []byte, recipients []*Recipient) ([]byte, error) {
	if len(recipients) == 0 {
		return nil, errors.New("at least one recipient required")
	}

	// Create age recipients
	ageRecipients := make([]age.Recipient, len(recipients))
	for i, r := range recipients {
		ageRecipients[i] = r.ageRecipient
	}

	// Encrypt using age
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, ageRecipients...)
	if err != nil {
		return nil, fmt.Errorf("failed to create age encryptor: %w", err)
	}

	if _, err := w.Write(message); err != nil {
		return nil, fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close encryptor: %w", err)
	}

	return buf.Bytes(), nil
}

// EncryptForIdentity encrypts a message for an identity's public key.
func EncryptForIdentity(message []byte, recipientIdentity *Identity) ([]byte, error) {
	recipient, err := recipientIdentity.GetAgeRecipient()
	if err != nil {
		return nil, err
	}
	wrapper := &Recipient{ageRecipient: recipient}
	return Encrypt(message, []*Recipient{wrapper})
}

// Decrypt decrypts an age-encrypted message using the identity.
func (id *Identity) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(id.ageIdentities) == 0 {
		return nil, ErrInvalidKey
	}

	r, err := age.Decrypt(bytes.NewReader(ciphertext), id.ageIdentities...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	plaintext, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read decrypted data: %w", err)
	}

	return plaintext, nil
}

// VerifySignature verifies an ed25519 signature.
func VerifySignature(pubKey ed25519.PublicKey, message, signature []byte) bool {
	return ed25519.Verify(pubKey, message, signature)
}

// GenerateNonce generates a random nonce for encryption.
func GenerateNonce(size int) ([]byte, error) {
	nonce := make([]byte, size)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}

// encodeBase64Std encodes bytes to standard base64.
func encodeBase64Std(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
