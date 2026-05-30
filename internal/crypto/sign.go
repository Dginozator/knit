package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

// SignErrors
var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrSigningKeyNil   = errors.New("signing key is nil")
	ErrPublicKeyNil    = errors.New("public key is nil")
)

// Signer handles digital signatures using ed25519.
type Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// NewSigner creates a new signer with a fresh key pair.
func NewSigner() (*Signer, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}
	return &Signer{
		privateKey: priv,
		publicKey:  pub,
	}, nil
}

// NewSignerFromSeed creates a signer from a seed (deterministic).
func NewSignerFromSeed(seed []byte) (*Signer, error) {
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("seed must be %d bytes, got %d", ed25519.SeedSize, len(seed))
	}
	priv := ed25519.NewKeyFromSeed(seed)
	return &Signer{
		privateKey: priv,
		publicKey:  priv.Public().(ed25519.PublicKey),
	}, nil
}

// Sign signs a message and returns the signature.
func (s *Signer) Sign(message []byte) ([]byte, error) {
	if s.privateKey == nil {
		return nil, ErrSigningKeyNil
	}
	return ed25519.Sign(s.privateKey, message), nil
}

// Verify verifies a signature against a message.
func (s *Signer) Verify(message, signature []byte) bool {
	return ed25519.Verify(s.publicKey, message, signature)
}

// PublicKey returns the public key in raw form.
func (s *Signer) PublicKey() ed25519.PublicKey {
	return s.publicKey
}

// PrivateKey returns the private key (use with caution).
func (s *Signer) PrivateKey() ed25519.PrivateKey {
	return s.privateKey
}

// SignPublicKey returns the public key for distribution.
func (s *Signer) SignPublicKey() []byte {
	return s.publicKey
}

// MarshalPrivateKey marshals the private key to PEM format.
func MarshalPrivateKey(priv ed25519.PrivateKey) ([]byte, error) {
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: privBytes,
	}), nil
}

// MarshalPublicKey marshals the public key to PEM format.
func MarshalPublicKey(pub ed25519.PublicKey) ([]byte, error) {
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PUBLIC KEY",
		Bytes: pubBytes,
	}), nil
}

// MarshalSSHPublicKey marshals the public key in SSH format.
func MarshalSSHPublicKey(pub ed25519.PublicKey) ([]byte, error) {
	return sshEncodeEd25519PublicKey(pub)
}

// sshEncodeEd25519PublicKey encodes an ed25519 public key in SSH format.
func sshEncodeEd25519PublicKey(pub ed25519.PublicKey) ([]byte, error) {
	// SSH key format: string("ssh-ed25519") + string(key) + string(comment)
	// For simplicity, we just return the key without comment
	key := []byte("ssh-ed25519")
	key = append(key, marshalUint32BE(0)...) // Reserved
	key = append(key, marshalUint32BE(uint32(len(pub)))...)
	key = append(key, pub...)
	return key, nil
}

// marshalUint32BE marshals a uint32 in big-endian format.
func marshalUint32BE(v uint32) []byte {
	b := make([]byte, 4)
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
	return b
}

// ParsePrivateKey parses a PEM-encoded private key.
func ParsePrivateKey(data []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}

	// Try PKCS8 format first
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		if ed25519Key, ok := key.(ed25519.PrivateKey); ok {
			return ed25519Key, nil
		}
	}

	// Try raw ed25519 format
	if strings.HasSuffix(block.Type, "PRIVATE KEY") {
		// Try parsing as PKCS8 again with different approach
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		ed25519Key, ok := key.(ed25519.PrivateKey)
		if !ok {
			return nil, errors.New("parsed key is not ed25519")
		}
		return ed25519Key, nil
	}

	return nil, errors.New("failed to parse private key")
}

// ParsePublicKey parses a PEM-encoded public key.
func ParsePublicKey(data []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	ed25519Key, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("not an ed25519 public key")
	}

	return ed25519Key, nil
}

// ParseSSHPublicKey parses an SSH-format public key.
func ParseSSHPublicKey(data []byte) (ed25519.PublicKey, error) {
	// SSH format for ed25519:
	// string("ssh-ed25519") + string(key) + string(comment)
	rest := data

	// Read key type
	keyType, rest := readSSHString(rest)
	if string(keyType) != "ssh-ed25519" {
		return nil, fmt.Errorf("expected ssh-ed25519, got %s", keyType)
	}

	// Read key
	pubKey, _ := readSSHString(rest)
	if len(pubKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid ed25519 key length: %d", len(pubKey))
	}

	return ed25519.PublicKey(pubKey), nil
}

// readSSHString reads an SSH string (length-prefixed).
func readSSHString(data []byte) ([]byte, []byte) {
	if len(data) < 4 {
		return nil, data
	}
	length := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	if int(length) > len(data)-4 {
		return nil, data
	}
	return data[4 : 4+length], data[4+length:]
}

// Verify verifies a signature using the public key.
func Verify(pub ed25519.PublicKey, message, signature []byte) error {
	if pub == nil {
		return ErrPublicKeyNil
	}
	if !ed25519.Verify(pub, message, signature) {
		return ErrInvalidSignature
	}
	return nil
}
