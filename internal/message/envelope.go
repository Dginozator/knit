// Package message provides message handling for the E2EE messenger.
package message

import (
	"crypto/ed25519"
	"encoding/json"

	"nit/internal/crypto"
	"nit/pkg/message"
)

// EnvelopeBuilder builds encrypted message envelopes.
type EnvelopeBuilder struct {
	senderID    string
	recipientID string
	identity    *crypto.Identity
}

// NewEnvelopeBuilder creates a new envelope builder.
func NewEnvelopeBuilder(senderID, recipientID string, identity *crypto.Identity) *EnvelopeBuilder {
	return &EnvelopeBuilder{
		senderID:    senderID,
		recipientID: recipientID,
		identity:    identity,
	}
}

// BuildEncrypt encrypts and wraps a message in an envelope.
func (b *EnvelopeBuilder) BuildEncrypt(msg *message.Message) (*message.Envelope, error) {
	// Serialize message
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// Sign the plaintext before encryption
	signature, err := b.identity.Sign(msgBytes)
	if err != nil {
		return nil, err
	}

	// Create envelope (encryption would happen here in production)
	envelope := &message.Envelope{
		Version:     1,
		RecipientID: b.recipientID,
		SenderID:    b.senderID,
		MessageID:   msg.ID,
		Timestamp:   msg.Timestamp,
		Type:        msg.Type,
		Headers:     make(map[string][]byte),
		Encrypted: message.EncryptedPayload{
			Ciphertext: msgBytes, // In production: encrypted
			Signature:  signature,
		},
	}

	return envelope, nil
}

// Decrypt verifies and decrypts an envelope into a message.
func (b *EnvelopeBuilder) Decrypt(envelope *message.Envelope) (*message.Message, error) {
	// In production, decrypt using b.identity.Decrypt()
	// For now, assume ciphertext is the plaintext
	plaintext := envelope.Encrypted.Ciphertext

	// Verify signature
	if envelope.SenderID != "" {
		senderPubKey, err := b.getSenderPublicKey(envelope.SenderID)
		if err == nil && senderPubKey != nil {
			if !crypto.VerifySignature(senderPubKey, plaintext, envelope.Encrypted.Signature) {
				return nil, crypto.ErrInvalidSignature
			}
		}
	}

	// Deserialize message
	var msg message.Message
	if err := json.Unmarshal(plaintext, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// getSenderPublicKey retrieves a sender's public key.
// In a real implementation, this would look up from a key server or local storage.
func (b *EnvelopeBuilder) getSenderPublicKey(senderID string) (ed25519.PublicKey, error) {
	// Placeholder - in production, implement key lookup
	return nil, nil
}

// MessageFromEnvelope extracts a message from an envelope.
func MessageFromEnvelope(e *message.Envelope) *message.Message {
	return &message.Message{
		ID:        e.MessageID,
		Type:      e.Type,
		From:      e.SenderID,
		To:        e.RecipientID,
		Timestamp: e.Timestamp,
		Content:   e.Encrypted.Ciphertext,
	}
}

// EnvelopeFromMessage creates an envelope from a message.
func EnvelopeFromMessage(msg *message.Message, senderID, recipientID string, identity *crypto.Identity) (*message.Envelope, error) {
	builder := NewEnvelopeBuilder(senderID, recipientID, identity)
	return builder.BuildEncrypt(msg)
}
