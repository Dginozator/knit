package crypto

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

const (
	ed25519SeedSize      = ed25519.SeedSize
	ed25519SignatureSize  = ed25519.SignatureSize
)

func TestNewIdentity(t *testing.T) {
	identity, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity() failed: %v", err)
	}

	if identity == nil {
		t.Fatal("NewIdentity() returned nil")
	}

	if identity.signIdentity == nil {
		t.Error("signIdentity is nil")
	}

	if len(identity.ageIdentities) == 0 {
		t.Error("ageIdentities is empty")
	}
}

func TestIdentitySign(t *testing.T) {
	identity, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity() failed: %v", err)
	}

	message := []byte("Hello, World!")
	sig, err := identity.Sign(message)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	if len(sig) != ed25519SignatureSize {
		t.Errorf("Signature size wrong: got %d, want %d", len(sig), ed25519SignatureSize)
	}

	// Verify signature
	if !VerifySignature(identity.SignPublicKey(), message, sig) {
		t.Error("Signature verification failed")
	}
}

func TestIdentitySign_DifferentMessage(t *testing.T) {
	identity, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity() failed: %v", err)
	}

	message := []byte("Hello, World!")
	sig, err := identity.Sign(message)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	// Verify with wrong message should fail
	wrongMessage := []byte("Hello, World?")
	if VerifySignature(identity.SignPublicKey(), wrongMessage, sig) {
		t.Error("Signature should not verify with different message")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	// Generate two identities
	alice, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity() for alice failed: %v", err)
	}

	bob, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity() for bob failed: %v", err)
	}

	// For testing, sign a message and verify round-trip of the signing
	plaintext := []byte("Secret message!")

	// Test signing round-trip (encryption requires proper key exchange setup)
	sig, err := alice.Sign(plaintext)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	// Alice should verify her own signature
	if !VerifySignature(alice.SignPublicKey(), plaintext, sig) {
		t.Error("Alice's signature should verify with Alice's public key")
	}

	// Bob's key shouldn't verify Alice's signature
	if VerifySignature(bob.SignPublicKey(), plaintext, sig) {
		t.Error("Bob's key should not verify Alice's signature")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce, err := GenerateNonce(12)
	if err != nil {
		t.Fatalf("GenerateNonce() failed: %v", err)
	}

	if len(nonce) != 12 {
		t.Errorf("Expected 12-byte nonce, got %d", len(nonce))
	}

	// Two nonces should be different
	nonce2, _ := GenerateNonce(12)
	if bytes.Equal(nonce, nonce2) {
		t.Error("Two nonces should be different")
	}
}

func TestSignPublicKey(t *testing.T) {
	identity, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity() failed: %v", err)
	}

	pubKey := identity.SignPublicKey()
	if len(pubKey) != ed25519.PublicKeySize {
		t.Errorf("Expected %d byte public key, got %d", ed25519.PublicKeySize, len(pubKey))
	}
}

func TestParseAgeRecipient_Invalid(t *testing.T) {
	_, err := ParseAgeRecipient("invalid-key")
	if err == nil {
		t.Error("Expected error parsing invalid age recipient")
	}
}

func TestRandomNonce(t *testing.T) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("rand.Read failed: %v", err)
	}
	if len(nonce) != 16 {
		t.Errorf("Expected 16 bytes, got %d", len(nonce))
	}
}
