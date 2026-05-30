package message

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"nit/pkg/message"
)

// Codec handles message encoding and decoding.
type Codec interface {
	Encode(*message.Envelope) ([]byte, error)
	Decode([]byte) (*message.Envelope, error)
}

// JSONCodec encodes/decodes messages as JSON.
type JSONCodec struct{}

// NewJSONCodec creates a new JSON codec.
func NewJSONCodec() *JSONCodec {
	return &JSONCodec{}
}

// Encode encodes an envelope to JSON.
func (c *JSONCodec) Encode(e *message.Envelope) ([]byte, error) {
	return json.Marshal(e)
}

// Decode decodes JSON to an envelope.
func (c *JSONCodec) Decode(data []byte) (*message.Envelope, error) {
	var envelope message.Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to decode envelope: %w", err)
	}
	return &envelope, nil
}

// BinaryCodec encodes/decodes messages in a compact binary format.
type BinaryCodec struct{}

// NewBinaryCodec creates a new binary codec.
func NewBinaryCodec() *BinaryCodec {
	return &BinaryCodec{}
}

// Encode encodes an envelope to binary format.
func (c *BinaryCodec) Encode(e *message.Envelope) ([]byte, error) {
	// Use JSON for simplicity - implement binary encoding for production
	return json.Marshal(e)
}

// Decode decodes binary to an envelope.
func (c *BinaryCodec) Decode(data []byte) (*message.Envelope, error) {
	var envelope message.Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to decode envelope: %w", err)
	}
	return &envelope, nil
}

// Base64Codec wraps a codec with base64 encoding for transport.
type Base64Codec struct {
	codec Codec
}

// NewBase64Codec creates a new base64 codec wrapping another codec.
func NewBase64Codec(codec Codec) *Base64Codec {
	return &Base64Codec{codec: codec}
}

// Encode encodes then base64 encodes the envelope.
func (c *Base64Codec) Encode(e *message.Envelope) ([]byte, error) {
	data, err := c.codec.Encode(e)
	if err != nil {
		return nil, err
	}
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encoded, data)
	return encoded, nil
}

// Decode decodes base64 then decodes the envelope.
func (c *Base64Codec) Decode(data []byte) (*message.Envelope, error) {
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(decoded, data)
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode: %w", err)
	}
	return c.codec.Decode(decoded[:n])
}

// Marshal marshals an envelope to JSON bytes.
func Marshal(e *message.Envelope) ([]byte, error) {
	return json.Marshal(e)
}

// Unmarshal unmarshals JSON bytes to an envelope.
func Unmarshal(data []byte) (*message.Envelope, error) {
	var envelope message.Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal envelope: %w", err)
	}
	return &envelope, nil
}

// MarshalToBase64 marshals an envelope to base64 encoded string.
func MarshalToBase64(e *message.Envelope) (string, error) {
	data, err := Marshal(e)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// UnmarshalFromBase64 unmarshals a base64 encoded string to an envelope.
func UnmarshalFromBase64(s string) (*message.Envelope, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	return Unmarshal(data)
}
