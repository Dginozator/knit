package crypto

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewKeystore(t *testing.T) {
	path := filepath.Join(os.TempDir(), "keystore-test")
	defer os.RemoveAll(path)

	ks, err := NewKeystore(path)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}

	if ks == nil {
		t.Fatal("NewKeystore() returned nil")
	}

	if ks.IsUnlocked() {
		t.Error("New keystore should be locked")
	}
}

func TestKeystoreUnlock(t *testing.T) {
	path := filepath.Join(os.TempDir(), "keystore-test")
	defer os.RemoveAll(path)

	ks, err := NewKeystore(path)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}

	err = ks.Unlock("test-password")
	if err != nil {
		t.Fatalf("Unlock() failed: %v", err)
	}

	if !ks.IsUnlocked() {
		t.Error("Keystore should be unlocked")
	}
}

func TestKeystoreLock(t *testing.T) {
	path := filepath.Join(os.TempDir(), "keystore-test")
	defer os.RemoveAll(path)

	ks, err := NewKeystore(path)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}

	ks.Unlock("test-password")
	ks.Lock()

	if ks.IsUnlocked() {
		t.Error("Keystore should be locked after Lock()")
	}
}

func TestKeystoreGenerateIdentity(t *testing.T) {
	path := filepath.Join(os.TempDir(), "keystore-test")
	defer os.RemoveAll(path)

	ks, err := NewKeystore(path)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}

	ks.Unlock("test-password")

	identity, err := ks.GenerateIdentity("test", "password")
	if err != nil {
		t.Fatalf("GenerateIdentity() failed: %v", err)
	}

	if identity == nil {
		t.Fatal("GenerateIdentity() returned nil")
	}

	// Verify identity can sign
	msg := []byte("test")
	sig, err := identity.Sign(msg)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	if !VerifySignature(identity.SignPublicKey(), msg, sig) {
		t.Error("Signature verification failed")
	}
}

func TestKeystoreGetIdentity(t *testing.T) {
	path := filepath.Join(os.TempDir(), "keystore-test")
	defer os.RemoveAll(path)

	ks1, err := NewKeystore(path)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}

	ks1.Unlock("test-password")

	// Generate
	identity1, err := ks1.GenerateIdentity("test", "password")
	if err != nil {
		t.Fatalf("GenerateIdentity() failed: %v", err)
	}

	pubKey1 := identity1.SignPublicKey()

	// Create new keystore instance and load
	ks2, err := NewKeystore(path)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}

	ks2.Unlock("test-password")

	identity2, err := ks2.GetIdentity("test")
	if err != nil {
		t.Fatalf("GetIdentity() failed: %v", err)
	}

	pubKey2 := identity2.SignPublicKey()

	// Keys should be the same
	if string(pubKey1) != string(pubKey2) {
		t.Error("Loaded identity doesn't match stored identity")
	}
}

func TestKeystoreListIdentities(t *testing.T) {
	path := filepath.Join(os.TempDir(), "keystore-test")
	defer os.RemoveAll(path)

	ks, err := NewKeystore(path)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}

	ks.Unlock("test-password")

	// Generate multiple identities
	_, err = ks.GenerateIdentity("alice", "password")
	if err != nil {
		t.Fatalf("GenerateIdentity() failed: %v", err)
	}

	_, err = ks.GenerateIdentity("bob", "password")
	if err != nil {
		t.Fatalf("GenerateIdentity() failed: %v", err)
	}

	names, err := ks.ListIdentities()
	if err != nil {
		t.Fatalf("ListIdentities() failed: %v", err)
	}

	if len(names) != 2 {
		t.Errorf("Expected 2 identities, got %d", len(names))
	}
}

func TestKeystoreDeleteIdentity(t *testing.T) {
	path := filepath.Join(os.TempDir(), "keystore-test")
	defer os.RemoveAll(path)

	ks, err := NewKeystore(path)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}

	ks.Unlock("test-password")

	_, err = ks.GenerateIdentity("test", "password")
	if err != nil {
		t.Fatalf("GenerateIdentity() failed: %v", err)
	}

	err = ks.DeleteIdentity("test")
	if err != nil {
		t.Fatalf("DeleteIdentity() failed: %v", err)
	}

	// Should not be able to get deleted identity
	_, err = ks.GetIdentity("test")
	if err == nil {
		t.Error("Expected error getting deleted identity")
	}
}

func TestKeystoreLockedOperations(t *testing.T) {
	path := filepath.Join(os.TempDir(), "keystore-test")
	defer os.RemoveAll(path)

	ks, err := NewKeystore(path)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}

	// Try operations on locked keystore
	_, err = ks.GenerateIdentity("test", "password")
	if err != ErrKeyStoreLocked {
		t.Error("Expected ErrKeyStoreLocked for GenerateIdentity on locked keystore")
	}

	_, err = ks.GetIdentity("test")
	if err != ErrKeyStoreLocked {
		t.Error("Expected ErrKeyStoreLocked for GetIdentity on locked keystore")
	}

	err = ks.DeleteIdentity("test")
	if err != ErrKeyStoreLocked {
		t.Error("Expected ErrKeyStoreLocked for DeleteIdentity on locked keystore")
	}
}

func TestDeriveKey(t *testing.T) {
	password := []byte("test-password")
	salt := []byte("test-salt-16byte")

	key1 := DeriveKey(password, salt)
	if len(key1) != 32 {
		t.Errorf("Expected 32-byte key, got %d", len(key1))
	}

	// Same inputs should produce same key
	key2 := DeriveKey(password, salt)
	if string(key1) != string(key2) {
		t.Error("Same inputs should produce same key")
	}

	// Different salt should produce different key
	key3 := DeriveKey(password, []byte("different-salt!!"))
	if string(key1) == string(key3) {
		t.Error("Different salt should produce different key")
	}
}

func TestGenerateSalt(t *testing.T) {
	salt, err := GenerateSalt(16)
	if err != nil {
		t.Fatalf("GenerateSalt() failed: %v", err)
	}

	if len(salt) != 16 {
		t.Errorf("Expected 16-byte salt, got %d", len(salt))
	}

	// Two salts should be different
	salt2, _ := GenerateSalt(16)
	if string(salt) == string(salt2) {
		t.Error("Two salts should be different")
	}
}
