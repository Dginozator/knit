// Package messenger provides the messenger service interface.
package messenger

import (
	"context"

	"nit/pkg/message"
)

// Messenger defines the interface for the E2EE messenger service.
type Messenger interface {
	// Send sends an encrypted message to a recipient.
	Send(ctx context.Context, to string, content []byte) (*message.Envelope, error)
	
	// Receive retrieves new messages for the current user.
	Receive(ctx context.Context) ([]*message.PlainEnvelope, error)
	
	// Decrypt decrypts a received envelope into a readable message.
	Decrypt(ctx context.Context, envelope *message.Envelope) (*message.PlainEnvelope, error)
	
	// ListContacts lists all contacts.
	ListContacts(ctx context.Context) ([]*message.Contact, error)
	
	// AddContact adds a new contact.
	AddContact(ctx context.Context, contact *message.Contact) error
	
	// RemoveContact removes a contact.
	RemoveContact(ctx context.Context, id string) error
	
	// GetIdentity returns the current user's identity.
	GetIdentity() (UserIdentity, error)
	
	// SyncKeys synchronizes encryption keys with the server.
	SyncKeys(ctx context.Context) error
}

// UserIdentity represents the current user's identity.
type UserIdentity interface {
	// UserID returns the user's ID.
	UserID() string
	
	// PublicKey returns the user's public key for encryption.
	PublicKey() []byte
	
	// Sign signs data with the user's private key.
	Sign(data []byte) ([]byte, error)
	
	// Verify verifies a signature.
	Verify(data, signature []byte) bool
}

// MessageService handles message operations.
type MessageService interface {
	// SendText sends a text message.
	SendText(ctx context.Context, to, text string) error
	
	// SendFile sends a file message.
	SendFile(ctx context.Context, to string, filename string, data []byte) error
	
	// GetMessages retrieves messages with pagination.
	GetMessages(ctx context.Context, before string, limit int) ([]*message.Message, error)
	
	// MarkAsRead marks a message as read.
	MarkAsRead(ctx context.Context, messageID string) error
	
	// DeleteMessage deletes a message.
	DeleteMessage(ctx context.Context, messageID string) error
}

// ContactService handles contact operations.
type ContactService interface {
	// List returns all contacts.
	List(ctx context.Context) ([]*message.Contact, error)
	
	// Get returns a contact by ID.
	Get(ctx context.Context, id string) (*message.Contact, error)
	
	// Add adds a new contact.
	Add(ctx context.Context, contact *message.Contact) error
	
	// Update updates a contact.
	Update(ctx context.Context, contact *message.Contact) error
	
	// Remove removes a contact.
	Remove(ctx context.Context, id string) error
}

// KeyService handles key operations.
type KeyService interface {
	// GenerateKey generates a new key pair.
	GenerateKey(ctx context.Context) error
	
	// ExportKey exports the user's public key.
	ExportKey(ctx context.Context) ([]byte, error)
	
	// ImportKey imports a contact's public key.
	ImportKey(ctx context.Context, id string, key []byte) error
	
	// VerifyKey verifies a contact's key.
	VerifyKey(ctx context.Context, id string) error
}

// StreamService handles Yandex Data Streams operations.
type StreamService interface {
	// EnsureStream ensures the stream exists.
	EnsureStream(ctx context.Context) error
	
	// GetStreamInfo returns stream information.
	GetStreamInfo(ctx context.Context) (*StreamInfo, error)
	
	// Subscribe subscribes to message updates.
	Subscribe(ctx context.Context, handler func(*message.Envelope)) error
	
	// Unsubscribe unsubscribes from message updates.
	Unsubscribe(ctx context.Context) error
}

// StreamInfo contains information about the stream.
type StreamInfo struct {
	Name              string
	Status            string
	ShardCount        int
	RetentionPeriodHours int
}

// ReceiptService handles delivery receipts.
type ReceiptService interface {
	// SendReceipt sends a delivery receipt.
	SendReceipt(ctx context.Context, receipt *message.DeliveryReceipt) error
	
	// GetReceipt gets a delivery receipt.
	GetReceipt(ctx context.Context, messageID string) (*message.DeliveryReceipt, error)
}
