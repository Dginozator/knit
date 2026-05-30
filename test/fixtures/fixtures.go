// Package fixtures provides test fixtures for the messenger tests.
package fixtures

import (
	"nit/internal/crypto"
	"nit/pkg/message"
	"time"
)

// TestIdentity creates a test identity.
func TestIdentity() (*crypto.Identity, error) {
	return crypto.NewIdentity()
}

// TestMessage creates a test message.
func TestMessage() *message.Message {
	return &message.Message{
		ID:        "test-msg-001",
		Type:      message.MessageTypeText,
		From:      "alice",
		To:        "bob",
		Timestamp: time.Now().UTC(),
		Content:   []byte("Hello, World!"),
	}
}

// TestEnvelope creates a test envelope.
func TestEnvelope(senderID, recipientID string) *message.Envelope {
	return &message.Envelope{
		Version:     1,
		RecipientID: recipientID,
		SenderID:    senderID,
		MessageID:   "test-msg-001",
		Timestamp:   time.Now().UTC(),
		Type:        message.MessageTypeText,
		Headers:     make(map[string][]byte),
		Encrypted: message.EncryptedPayload{
			Ciphertext: []byte("encrypted content"),
			Signature:  []byte("signature"),
		},
	}
}

// TestContact creates a test contact.
func TestContact() *message.Contact {
	return &message.Contact{
		ID:          "bob",
		DisplayName: "Bob",
		PublicKey:   []byte("public-key"),
		AddedAt:     time.Now().UTC(),
		TrustLevel:  message.TrustUnverified,
	}
}

// SampleMessages returns sample messages for testing.
func SampleMessages() []*message.Message {
	return []*message.Message{
		{
			ID:        "msg-001",
			Type:      message.MessageTypeText,
			From:      "alice",
			To:        "bob",
			Timestamp: time.Now().UTC().Add(-1 * time.Hour),
			Content:   []byte("Hello Bob!"),
		},
		{
			ID:        "msg-002",
			Type:      message.MessageTypeText,
			From:      "bob",
			To:        "alice",
			Timestamp: time.Now().UTC().Add(-30 * time.Minute),
			Content:   []byte("Hi Alice!"),
		},
		{
			ID:        "msg-003",
			Type:      message.MessageTypeText,
			From:      "alice",
			To:        "bob",
			Timestamp: time.Now().UTC(),
			Content:   []byte("How are you?"),
		},
	}
}
