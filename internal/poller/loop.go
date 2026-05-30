// Package poller provides message polling functionality using YDB Topic API.
package poller

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"nit/internal/yds"
	pkgmessage "nit/pkg/message"

	"github.com/ydb-platform/ydb-go-sdk/v3/topic/topicreader"
)

// Codec is the interface for encoding/decoding envelopes.
type Codec interface {
	Decode([]byte) (*pkgmessage.Envelope, error)
}

// MessageHandler is a function that handles incoming messages.
type MessageHandler func(*pkgmessage.Envelope) error

// Poller polls Yandex Data Streams for new messages using YDB Topic reader.
type Poller struct {
	client       *yds.Client
	codec        Codec
	consumerName string
	reader       *topicreader.Reader
	mu           sync.Mutex
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// NewPoller creates a new poller.
func NewPoller(client *yds.Client, codec Codec) *Poller {
	return &Poller{
		client:       client,
		codec:        codec,
		consumerName: "nit-consumer",
		stopCh:       make(chan struct{}),
	}
}

// WithConsumerName sets the consumer name for the reader.
func (p *Poller) WithConsumerName(name string) *Poller {
	p.consumerName = name
	return p
}

// Start begins polling for messages.
func (p *Poller) Start(ctx context.Context, handler MessageHandler) error {
	reader, err := p.client.NewReader(ctx, p.consumerName)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}

	p.mu.Lock()
	p.reader = reader
	p.mu.Unlock()

	p.wg.Add(1)
	go p.readLoop(ctx, handler)

	return nil
}

// Stop stops the poller.
func (p *Poller) Stop() {
	close(p.stopCh)
	p.wg.Wait()

	p.mu.Lock()
	if p.reader != nil {
		p.reader.Close(context.Background())
	}
	p.mu.Unlock()
}

// readLoop continuously reads messages from the topic.
func (p *Poller) readLoop(ctx context.Context, handler MessageHandler) {
	defer p.wg.Done()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		// Read next message with timeout
		readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		msg, err := p.reader.ReadMessage(readCtx)
		cancel()

		if err != nil {
			// Timeout or context cancelled — continue polling
			continue
		}

		// Read message data — Message implements io.Reader directly
		data, err := io.ReadAll(msg)
		if err != nil || len(data) == 0 {
			p.reader.Commit(ctx, msg)
			continue
		}

		// Decode envelope
		envelope, err := p.codec.Decode(data)
		if err != nil {
			// Skip malformed messages
			p.reader.Commit(ctx, msg)
			continue
		}

		// Handle message
		if err := handler(envelope); err != nil {
			// Log error but continue
			_ = err
		}

		// Commit offset
		p.reader.Commit(ctx, msg)
	}
}

// PollerConfig holds poller configuration.
type PollerConfig struct {
	Interval     time.Duration
	BatchSize    int
	NumWorkers   int
	SaveInterval time.Duration
	ConsumerName string
}

// DefaultPollerConfig returns default poller configuration.
func DefaultPollerConfig() *PollerConfig {
	return &PollerConfig{
		Interval:     1 * time.Second,
		BatchSize:    10,
		NumWorkers:   1,
		SaveInterval: 30 * time.Second,
		ConsumerName: "nit-consumer",
	}
}
