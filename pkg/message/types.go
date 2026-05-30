// Package message provides core message types for the E2EE messenger.
package message

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"time"

	"nit/internal/crypto"
)

// MessageType represents the type of message.
type MessageType uint8

const (
	MessageTypeText        MessageType = 1
	MessageTypeFile        MessageType = 2
	MessageTypeKeyExchange MessageType = 3
	MessageTypeAck         MessageType = 4
	MessageTypeDeleted     MessageType = 5
)

// Message represents a message in the E2EE messenger.
type Message struct {
	ID          string       `json:"id"`
	Type        MessageType  `json:"type"`
	From        string       `json:"from"`
	To          string       `json:"to"`
	Timestamp   time.Time    `json:"timestamp"`
	Content     []byte       `json:"content"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Metadata    Metadata     `json:"metadata,omitempty"`
}

// Attachment represents a file attachment.
type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int    `json:"size"`
	Checksum []byte `json:"checksum"`
}

// Metadata contains additional message metadata.
type Metadata struct {
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Priority    uint8       `json:"priority,omitempty"`
	References  []string   `json:"references,omitempty"`
	CustomField string     `json:"custom,omitempty"`
}

// Envelope represents the outer wrapper of an encrypted message.
type Envelope struct {
	Version     int              `json:"v"`
	RecipientID string           `json:"rid"`
	SenderID    string           `json:"sid"`
	MessageID   string           `json:"mid"`
	Timestamp   time.Time        `json:"ts"`
	Type        MessageType      `json:"type"`
	Headers     map[string][]byte `json:"headers"`
	Encrypted   EncryptedPayload `json:"encrypted"`
}

// EncryptedPayload contains the encrypted content and signature.
type EncryptedPayload struct {
	// EncryptedKey is the message key encrypted for the recipient
	EncryptedKey []byte `json:"ek"`
	// Nonce is the nonce used for encryption
	Nonce []byte `json:"n"`
	// Ciphertext is the encrypted message content
	Ciphertext []byte `json:"ct"`
	// Signature is the ed25519 signature of the plaintext
	Signature []byte `json:"sig"`
}

// SignedMessage represents an unencrypted message with signature.
type SignedMessage struct {
	Message   Message  `json:"msg"`
	Signature []byte   `json:"sig"`
	Signer    []byte   `json:"signer"`
}

// PlainEnvelope represents an unencrypted message envelope for local storage.
type PlainEnvelope struct {
	ID          string       `json:"id"`
	Type        MessageType  `json:"type"`
	From        string       `json:"from"`
	To          string       `json:"to"`
	Timestamp   time.Time    `json:"timestamp"`
	Content     []byte       `json:"content,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Metadata    Metadata     `json:"metadata,omitempty"`
	DecryptedAt time.Time    `json:"decrypted_at"`
}

// DeliveryStatus represents the delivery status of a message.
type DeliveryStatus uint8

const (
	StatusPending   DeliveryStatus = 0
	StatusSent      DeliveryStatus = 1
	StatusDelivered DeliveryStatus = 2
	StatusRead      DeliveryStatus = 3
	StatusFailed    DeliveryStatus = 4
	StatusDeleted   DeliveryStatus = 5
)

// DeliveryReceipt represents a delivery receipt.
type DeliveryReceipt struct {
	MessageID string         `json:"message_id"`
	Status    DeliveryStatus `json:"status"`
	Timestamp time.Time      `json:"timestamp"`
}

// KeyExchangeMessage represents a key exchange message.
type KeyExchangeMessage struct {
	RecipientID string    `json:"rid"`
	IdentityKey []byte    `json:"ik"` // ed25519 public key
	SessionID   string    `json:"sid"`
	Timestamp   time.Time `json:"ts"`
}

// Contact represents a user's contact.
type Contact struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"name"`
	PublicKey   []byte     `json:"pubkey"`
	IdentityKey []byte     `json:"identity_key"`
	AddedAt     time.Time  `json:"added_at"`
	LastSeenAt  *time.Time `json:"last_seen,omitempty"`
	TrustLevel  TrustLevel `json:"trust"`
}

// TrustLevel represents the trust level of a contact.
type TrustLevel uint8

const (
	TrustUnverified TrustLevel = 0
	TrustVerified   TrustLevel = 1
	TrustBlocked    TrustLevel = 2
)

// NewMessage creates a new message.
func NewMessage(from, to string, content []byte) *Message {
	return &Message{
		ID:        generateID(),
		Type:      MessageTypeText,
		From:      from,
		To:        to,
		Timestamp: time.Now().UTC(),
		Content:   content,
	}
}

// NewEnvelope creates a new envelope for a message.
func NewEnvelope(recipientID, senderID string, msg *Message) *Envelope {
	return &Envelope{
		Version:     1,
		RecipientID: recipientID,
		SenderID:    senderID,
		MessageID:   msg.ID,
		Timestamp:   msg.Timestamp,
		Type:        msg.Type,
		Headers:     make(map[string][]byte),
	}
}

// ToSignedMessage converts a message to a signed message.
func (m *Message) ToSignedMessage(signer *crypto.Signer) (*SignedMessage, error) {
	// Serialize message
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	// Sign
	sig, err := signer.Sign(data)
	if err != nil {
		return nil, err
	}

	return &SignedMessage{
		Message:   *m,
		Signature: sig,
		Signer:    signer.PublicKey(),
	}, nil
}

// Verify verifies a signed message.
func (sm *SignedMessage) Verify(pubKey []byte) bool {
	// Serialize message
	data, err := json.Marshal(sm.Message)
	if err != nil {
		return false
	}

	pubKeyObj, err := crypto.ParsePublicKey(pubKey)
	if err != nil {
		return false
	}
	return ed25519.Verify(pubKeyObj, data, sm.Signature)
}

// ToPlainEnvelope converts an envelope to a plain envelope after decryption.
func (e *Envelope) ToPlainEnvelope(decrypted *Message) *PlainEnvelope {
	return &PlainEnvelope{
		ID:          e.MessageID,
		Type:        e.Type,
		From:        e.SenderID,
		To:          e.RecipientID,
		Timestamp:   e.Timestamp,
		Content:     decrypted.Content,
		Attachments: decrypted.Attachments,
		Metadata:    decrypted.Metadata,
		DecryptedAt: time.Now().UTC(),
	}
}

// generateID generates a unique message ID.
// In production, use UUID or similar.
func generateID() string {
	b := make([]byte, 16)
	nonce, err := crypto.GenerateNonce(16)
	if err != nil {
		// Fallback to simple implementation
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	copy(b, nonce)
	return fmt.Sprintf("%x", b)
}
