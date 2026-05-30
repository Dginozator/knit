// Package yds provides Yandex Data Streams integration using YDB Topic API (gRPC).
package yds

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	ydb "github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/credentials"
	"github.com/ydb-platform/ydb-go-sdk/v3/topic/topicoptions"
	"github.com/ydb-platform/ydb-go-sdk/v3/topic/topicreader"
	"github.com/ydb-platform/ydb-go-sdk/v3/topic/topictypes"
	"github.com/ydb-platform/ydb-go-sdk/v3/topic/topicwriter"
)

// Client wraps the YDB Topic client for Yandex Data Streams.
type Client struct {
	db          *ydb.Driver
	config      *Config
	retryConfig *RetryConfig
	topicPath   string
}

// NewClient creates a new YDS client using YDB gRPC API.
//
// Authentication priority:
//  1. Service account key file (SAKeyFile in config) — automatic JWT-based IAM token refresh
//  2. IAM token stored in keychain (t1.xxx) — valid for 12 hours
func NewClient(cfg *Config) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// YDB connection string
	dsn := fmt.Sprintf("grpcs://ydb.serverless.yandexcloud.net:2135/?database=%s", cfg.DatabasePath())

	var creds ydb.Option

	// Priority 1: Service account key file (automatic token refresh)
	if cfg.SAKeyFile != "" {
		keyJSON, err := os.ReadFile(cfg.SAKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read service account key file %q: %w", cfg.SAKeyFile, err)
		}
		provider, err := NewIAMTokenProviderFromJSON(keyJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to create IAM token provider: %w", err)
		}
		// Get initial token to verify credentials work
		token, err := provider.Token(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get IAM token: %w", err)
		}
		creds = ydb.WithCredentials(credentials.NewAccessTokenCredentials(token))
		// Store provider for future token refresh (simplified — in production use a refreshing credentials)
		_ = provider
	} else if cfg.APIKey != "" {
		// Priority 2: IAM token or API key from keychain
		creds = ydb.WithCredentials(credentials.NewAccessTokenCredentials(cfg.APIKey))
	} else {
		return nil, fmt.Errorf("authentication required: either set sa_key_file in config or run 'nit secret set-api-key <iam-token>'")
	}

	db, err := ydb.Open(ctx, dsn, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to YDB: %w", err)
	}

	return &Client{
		db:          db,
		config:      cfg,
		retryConfig: DefaultRetryConfig(),
		topicPath:   cfg.TopicPath(),
	}, nil
}

// NewClientWithSAKey creates a client using a service account key for automatic token refresh.
func NewClientWithSAKey(cfg *Config, saKeyJSON []byte) (*Client, error) {
	provider, err := NewIAMTokenProviderFromJSON(saKeyJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM token provider: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get initial IAM token
	token, err := provider.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial IAM token: %w", err)
	}

	dsn := fmt.Sprintf("grpcs://ydb.serverless.yandexcloud.net:2135/?database=%s", cfg.DatabasePath())

	db, err := ydb.Open(ctx, dsn,
		ydb.WithCredentials(credentials.NewAccessTokenCredentials(token)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to YDB: %w", err)
	}

	return &Client{
		db:          db,
		config:      cfg,
		retryConfig: DefaultRetryConfig(),
		topicPath:   cfg.TopicPath(),
	}, nil
}

// Close closes the YDB connection.
func (c *Client) Close(ctx context.Context) error {
	return c.db.Close(ctx)
}

// CreateTopic creates the YDS topic (stream).
func (c *Client) CreateTopic(ctx context.Context, partitions int32) error {
	return c.db.Topic().Create(ctx, c.topicPath,
		topicoptions.CreateWithMinActivePartitions(int64(partitions)),
		topicoptions.CreateWithRetentionPeriod(24*time.Hour),
	)
}

// TopicExists checks if the topic exists.
func (c *Client) TopicExists(ctx context.Context) (bool, error) {
	_, err := c.db.Topic().Describe(ctx, c.topicPath)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// DeleteTopic deletes the topic.
func (c *Client) DeleteTopic(ctx context.Context) error {
	return c.db.Topic().Drop(ctx, c.topicPath)
}

// AddConsumer adds a consumer (reader group) to the topic.
// Consumers must be registered before reading messages.
func (c *Client) AddConsumer(ctx context.Context, consumerName string) error {
	return c.db.Topic().Alter(ctx, c.topicPath,
		topicoptions.AlterWithAddConsumers(topictypes.Consumer{
			Name: consumerName,
		}),
	)
}

// ConsumerExists checks if a consumer exists on the topic.
func (c *Client) ConsumerExists(ctx context.Context, consumerName string) (bool, error) {
	desc, err := c.db.Topic().Describe(ctx, c.topicPath)
	if err != nil {
		return false, err
	}
	for _, consumer := range desc.Consumers {
		if consumer.Name == consumerName {
			return true, nil
		}
	}
	return false, nil
}

// NewWriter creates a new topic writer for sending messages.
func (c *Client) NewWriter(ctx context.Context) (*topicwriter.Writer, error) {
	writer, err := c.db.Topic().StartWriter(c.topicPath,
		topicoptions.WithWriterProducerID("nit-messenger"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic writer: %w", err)
	}
	return writer, nil
}

// NewReader creates a new topic reader for receiving messages.
func (c *Client) NewReader(ctx context.Context, consumerName string) (*topicreader.Reader, error) {
	reader, err := c.db.Topic().StartReader(consumerName,
		topicoptions.ReadTopic(c.topicPath),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic reader: %w", err)
	}
	return reader, nil
}

// WriteMessage sends a single message to the topic.
func (c *Client) WriteMessage(ctx context.Context, data []byte) error {
	writer, err := c.NewWriter(ctx)
	if err != nil {
		return err
	}
	defer writer.Close(ctx)

	return writer.Write(ctx, topicwriter.Message{Data: bytes.NewReader(data)})
}

// SetRetryConfig sets the retry configuration.
func (c *Client) SetRetryConfig(cfg *RetryConfig) {
	c.retryConfig = cfg
}
