package message

import (
	"encoding/json"
	"testing"
	"time"

	pkgmessage "nit/pkg/message"
)

// newTestMessage creates a test message
func newTestMessage(from, to string, content []byte) *pkgmessage.Message {
	return &pkgmessage.Message{
		ID:        "test-id-001",
		Type:      pkgmessage.MessageTypeText,
		From:      from,
		To:        to,
		Timestamp: time.Now().UTC(),
		Content:   content,
	}
}

// newTestEnvelope creates a test envelope
func newTestEnvelope(recipientID, senderID string, msg *pkgmessage.Message) *pkgmessage.Envelope {
	return &pkgmessage.Envelope{
		Version:     1,
		RecipientID: recipientID,
		SenderID:    senderID,
		MessageID:   msg.ID,
		Timestamp:   msg.Timestamp,
		Type:        msg.Type,
		Headers:     make(map[string][]byte),
	}
}

func TestJSONCodecEncode(t *testing.T) {
	codec := NewJSONCodec()

	msg := newTestMessage("alice", "bob", []byte("Hello"))
	env := newTestEnvelope("bob", "alice", msg)
	env.MessageID = "test-001"

	// Encode
	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Encoded data should not be empty")
	}

	// Decode
	env2, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	if env2.MessageID != env.MessageID {
		t.Errorf("MessageID mismatch: got %s, want %s", env2.MessageID, env.MessageID)
	}

	if env2.SenderID != env.SenderID {
		t.Errorf("SenderID mismatch: got %s, want %s", env2.SenderID, env.SenderID)
	}

	if env2.RecipientID != env.RecipientID {
		t.Errorf("RecipientID mismatch: got %s, want %s", env2.RecipientID, env.RecipientID)
	}
}

func TestJSONCodecDecode_Invalid(t *testing.T) {
	codec := NewJSONCodec()

	_, err := codec.Decode([]byte("invalid json"))
	if err == nil {
		t.Error("Expected error decoding invalid JSON")
	}
}

func TestBase64Codec(t *testing.T) {
	base := NewJSONCodec()
	codec := NewBase64Codec(base)

	msg := newTestMessage("alice", "bob", []byte("Hello base64"))
	env := newTestEnvelope("bob", "alice", msg)
	env.MessageID = "base64-test"

	// Encode
	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	// Decode
	env2, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode() failed: %v", err)
	}

	if env2.MessageID != env.MessageID {
		t.Errorf("MessageID mismatch: got %s, want %s", env2.MessageID, env.MessageID)
	}
}

func TestMarshal(t *testing.T) {
	msg := newTestMessage("alice", "bob", []byte("Test"))
	env := newTestEnvelope("bob", "alice", msg)
	env.MessageID = "marshal-test"

	// Marshal
	data, err := Marshal(env)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	// Unmarshal
	env2, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if env2.MessageID != env.MessageID {
		t.Errorf("MessageID mismatch: got %s, want %s", env2.MessageID, env.MessageID)
	}
}

func TestMarshalToBase64(t *testing.T) {
	msg := newTestMessage("alice", "bob", []byte("Hello base64"))
	env := newTestEnvelope("bob", "alice", msg)
	env.MessageID = "b64-test"

	s, err := MarshalToBase64(env)
	if err != nil {
		t.Fatalf("MarshalToBase64() failed: %v", err)
	}

	if len(s) == 0 {
		t.Error("Base64 string should not be empty")
	}

	env2, err := UnmarshalFromBase64(s)
	if err != nil {
		t.Fatalf("UnmarshalFromBase64() failed: %v", err)
	}

	if env2.MessageID != env.MessageID {
		t.Errorf("MessageID mismatch: got %s, want %s", env2.MessageID, env.MessageID)
	}
}

func TestEnvelopeJSONRoundTrip(t *testing.T) {
	msg := newTestMessage("alice", "bob", []byte("Test"))
	env := newTestEnvelope("bob", "alice", msg)
	env.MessageID = "roundtrip-001"
	env.Encrypted = pkgmessage.EncryptedPayload{
		Ciphertext: []byte("encrypted data"),
		Signature:  []byte("signature"),
	}

	// Serialize
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Deserialize
	var env2 pkgmessage.Envelope
	err = json.Unmarshal(data, &env2)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if env2.MessageID != env.MessageID {
		t.Errorf("MessageID mismatch: got %s, want %s", env2.MessageID, env.MessageID)
	}
}
