// Package mockyds provides a mock Yandex Data Streams server for testing.
package mockyds

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
)

// Server is a mock Kinesis/YDS server for testing.
type Server struct {
	mu      sync.RWMutex
	streams map[string]*Stream
	port    int
}

// Stream represents a mock Kinesis stream.
type Stream struct {
	Name          string
	Status       string // "CREATING", "ACTIVE", "DELETING"
	ShardCount    int
	Shards       map[string]*Shard
	Records      []*Record
	Subscriptions map[string]chan *Record
	mu           sync.RWMutex
}

// Shard represents a mock Kinesis shard.
type Shard struct {
	ShardID string
	Parent  string
}

// Record represents a mock Kinesis record.
type Record struct {
	SequenceNumber string
	PartitionKey  string
	Data          []byte
	ShardID       string
	Timestamp     time.Time
}

// NewServer creates a new mock server.
func NewServer(port int) *Server {
	return &Server{
		streams: make(map[string]*Stream),
		port:    port,
	}
}

// CreateStream creates a mock stream.
func (s *Server) CreateStream(name string, shardCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.streams[name]; exists {
		return fmt.Errorf("stream already exists: %s", name)
	}

	stream := &Stream{
		Name:       name,
		Status:     "ACTIVE",
		ShardCount: shardCount,
		Shards:    make(map[string]*Shard),
		Records:   make([]*Record, 0),
	}

	// Create shards
	for i := 0; i < shardCount; i++ {
		shardID := fmt.Sprintf("shardId-000000%02d", i)
		stream.Shards[shardID] = &Shard{
			ShardID: shardID,
		}
	}

	s.streams[name] = stream
	return nil
}

// DeleteStream deletes a mock stream.
func (s *Server) DeleteStream(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, exists := s.streams[name]
	if !exists {
		return fmt.Errorf("stream not found: %s", name)
	}

	stream.Status = "DELETING"
	delete(s.streams, name)
	return nil
}

// DescribeStream describes a stream.
func (s *Server) DescribeStream(name string) (*Stream, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[name]
	if !exists {
		return nil, fmt.Errorf("stream not found: %s", name)
	}

	return stream, nil
}

// PutRecord puts a record to the stream.
func (s *Server) PutRecord(streamName, partitionKey string, data []byte) (*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, exists := s.streams[streamName]
	if !exists {
		return nil, fmt.Errorf("stream not found: %s", streamName)
	}

	seqNum := fmt.Sprintf("%d", len(stream.Records)+1)
	record := &Record{
		SequenceNumber: seqNum,
		PartitionKey:   partitionKey,
		Data:           data,
		ShardID:        "shardId-00000000",
		Timestamp:      time.Now(),
	}

	stream.Records = append(stream.Records, record)

	// Notify subscribers
	for _, ch := range stream.Subscriptions {
		select {
		case ch <- record:
		default:
		}
	}

	return record, nil
}

// GetRecords gets records from a shard iterator.
func (s *Server) GetRecords(streamName, iterator string, limit int64) ([]*Record, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, exists := s.streams[streamName]
	if !exists {
		return nil, "", fmt.Errorf("stream not found: %s", streamName)
	}

	// Simple iterator: just return all records
	var records []*Record
	for i, r := range stream.Records {
		if int64(i) < limit {
			records = append(records, r)
		}
	}

	nextIterator := "next-" + iterator
	return records, nextIterator, nil
}

// GetShardIterator gets a shard iterator.
func (s *Server) GetShardIterator(streamName, shardID, iteratorType string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[streamName]
	if !exists {
		return "", fmt.Errorf("stream not found: %s", streamName)
	}

	if _, exists := stream.Shards[shardID]; !exists {
		return "", fmt.Errorf("shard not found: %s", shardID)
	}

	return "iterator-" + shardID, nil
}

// ListShards lists all shards in a stream.
func (s *Server) ListShards(streamName string) ([]*Shard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[streamName]
	if !exists {
		return nil, fmt.Errorf("stream not found: %s", streamName)
	}

	shards := make([]*Shard, 0, len(stream.Shards))
	for _, shard := range stream.Shards {
		shards = append(shards, shard)
	}

	return shards, nil
}

// Subscribe subscribes to records on a stream.
func (s *Server) Subscribe(streamName string) (chan *Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, exists := s.streams[streamName]
	if !exists {
		return nil, fmt.Errorf("stream not found: %s", streamName)
	}

	ch := make(chan *Record, 100)
	stream.Subscriptions["sub-"+fmt.Sprintf("%d", time.Now().UnixNano())] = ch

	return ch, nil
}

// MockKinesisClient is a mock Kinesis client that uses the Server.
type MockKinesisClient struct {
	server *Server
	stream string
}

// NewMockKinesisClient creates a new mock Kinesis client.
func NewMockKinesisClient(server *Server, stream string) *MockKinesisClient {
	return &MockKinesisClient{
		server: server,
		stream: stream,
	}
}

// PutRecord mocks PutRecord API.
func (m *MockKinesisClient) PutRecord(ctx context.Context, params *kinesis.PutRecordInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordOutput, error) {
	record, err := m.server.PutRecord(m.stream, aws.ToString(params.PartitionKey), params.Data)
	if err != nil {
		return nil, err
	}

	return &kinesis.PutRecordOutput{
		ShardId:        aws.String(record.ShardID),
		SequenceNumber: aws.String(record.SequenceNumber),
	}, nil
}

// GetRecords mocks GetRecords API.
func (m *MockKinesisClient) GetRecords(ctx context.Context, params *kinesis.GetRecordsInput, optFns ...func(*kinesis.Options)) (*kinesis.GetRecordsOutput, error) {
	var limit int64
	if params.Limit != nil {
		limit = int64(*params.Limit)
	}
	records, nextIter, err := m.server.GetRecords(m.stream, aws.ToString(params.ShardIterator), limit)
	if err != nil {
		return nil, err
	}

	kinesisRecords := make([]types.Record, len(records))
	for i, r := range records {
		kinesisRecords[i] = types.Record{
			SequenceNumber: aws.String(r.SequenceNumber),
			PartitionKey:   aws.String(r.PartitionKey),
			Data:           r.Data,
		}
	}

	return &kinesis.GetRecordsOutput{
		Records:          kinesisRecords,
		NextShardIterator: aws.String(nextIter),
	}, nil
}

// GetShardIterator mocks GetShardIterator API.
func (m *MockKinesisClient) GetShardIterator(ctx context.Context, params *kinesis.GetShardIteratorInput, optFns ...func(*kinesis.Options)) (*kinesis.GetShardIteratorOutput, error) {
	iter, err := m.server.GetShardIterator(m.stream, aws.ToString(params.ShardId), string(params.ShardIteratorType))
	if err != nil {
		return nil, err
	}

	return &kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String(iter),
	}, nil
}

// DescribeStream mocks DescribeStream API.
func (m *MockKinesisClient) DescribeStream(ctx context.Context, params *kinesis.DescribeStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.DescribeStreamOutput, error) {
	stream, err := m.server.DescribeStream(m.stream)
	if err != nil {
		return nil, err
	}

	return &kinesis.DescribeStreamOutput{
		StreamDescription: &types.StreamDescription{
			StreamName:   aws.String(stream.Name),
			StreamStatus: types.StreamStatus(stream.Status),
		},
	}, nil
}

// ListShards mocks ListShards API.
func (m *MockKinesisClient) ListShards(ctx context.Context, params *kinesis.ListShardsInput, optFns ...func(*kinesis.Options)) (*kinesis.ListShardsOutput, error) {
	shards, err := m.server.ListShards(m.stream)
	if err != nil {
		return nil, err
	}

	kinesisShards := make([]types.Shard, len(shards))
	for i, s := range shards {
		kinesisShards[i] = types.Shard{
			ShardId: aws.String(s.ShardID),
		}
	}

	return &kinesis.ListShardsOutput{
		Shards: kinesisShards,
	}, nil
}
