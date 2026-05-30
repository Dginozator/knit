// Package yds provides Yandex Data Streams integration using AWS Kinesis SDK.
package yds

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nit/internal/secrets"

	"github.com/spf13/viper"
)

// expandPath expands ~ to the user's home directory.
// If the literal path exists (e.g. a folder named "~" in the project), it uses that.
func expandPath(path string) string {
	if path == "" {
		return path
	}
	// Check if the literal path exists first
	if _, err := os.Stat(path); err == nil {
		return path
	}
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// Config holds Yandex Data Streams configuration.
type Config struct {
	// Endpoint is the YDS endpoint (without https://)
	Endpoint string
	// FolderID is the Yandex Cloud folder (catalog) ID
	FolderID string
	// StreamName is the name of the YDS stream (without folder prefix)
	StreamName string
	// Region is the Yandex Cloud region
	Region string
	// APIKey is the IAM token or API key for authentication (from keychain)
	APIKey string
	// SAKeyFile is the path to the service account authorized key JSON file.
	// When set, IAM tokens are automatically obtained and refreshed via JWT.
	// Download from: Yandex Cloud Console → IAM → Service Accounts → Authorized keys → Create
	SAKeyFile string
	// MaxRetries is the maximum number of retries for failed operations
	MaxRetries int
	// RetryDelay is the base delay between retries
	RetryDelay time.Duration
	// ReadTimeout is the timeout for read operations
	ReadTimeout time.Duration
	// WriteTimeout is the timeout for write operations
	WriteTimeout time.Duration
	// ShardIteratorTimeout is how long a shard iterator is valid
	ShardIteratorTimeout time.Duration
}

// FullStreamName returns the full stream name in the format required by YDS: "folder_id/stream_name"
func (c *Config) FullStreamName() string {
	if c.FolderID == "" {
		return c.StreamName
	}
	return fmt.Sprintf("%s/%s", c.FolderID, c.StreamName)
}

// DatabasePath returns the YDB database path from the endpoint.
// The endpoint format is: yds.serverless.yandexcloud.net/ru-central1/folder_id/db_id
// The database path is: /ru-central1/folder_id/db_id
func (c *Config) DatabasePath() string {
	// Extract path from endpoint (everything after the hostname)
	endpoint := c.Endpoint
	// Find the first slash after the hostname
	for i, ch := range endpoint {
		if ch == '/' {
			return endpoint[i:]
		}
	}
	// Fallback: construct from folder_id
	return fmt.Sprintf("/ru-central1/%s", c.FolderID)
}

// TopicPath returns the YDB topic path for the stream.
func (c *Config) TopicPath() string {
	return c.StreamName
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Endpoint:             "yds.serverless.yandexcloud.net",
		StreamName:           "messenger",
		Region:               "ru-central1",
		MaxRetries:           3,
		RetryDelay:           100 * time.Millisecond,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         10 * time.Second,
		ShardIteratorTimeout: 5 * time.Minute,
	}
}

// LoadConfig loads configuration from environment and config files.
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("yds.endpoint", "yds.serverless.yandexcloud.net")
	v.SetDefault("yds.folder_id", "")
	v.SetDefault("yds.stream_name", "messenger")
	v.SetDefault("yds.region", "ru-central1")
	v.SetDefault("yds.max_retries", 3)
	v.SetDefault("yds.retry_delay_ms", 100)

	// Environment variables
	v.SetEnvPrefix("NIT")
	v.AutomaticEnv()

	// Read config file if exists
	v.SetConfigName("default")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath("$HOME/.nit")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	cfg := &Config{
		Endpoint:             v.GetString("yds.endpoint"),
		FolderID:             v.GetString("yds.folder_id"),
		StreamName:           v.GetString("yds.stream_name"),
		Region:               v.GetString("yds.region"),
		APIKey:               v.GetString("yds.api_key"),
		SAKeyFile:            expandPath(v.GetString("yds.sa_key_file")),
		MaxRetries:           v.GetInt("yds.max_retries"),
		RetryDelay:           time.Duration(v.GetInt("yds.retry_delay_ms")) * time.Millisecond,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         10 * time.Second,
		ShardIteratorTimeout: 5 * time.Minute,
	}

	// Override with environment variables if set
	if endpoint := os.Getenv("YDS_ENDPOINT"); endpoint != "" {
		cfg.Endpoint = endpoint
	}
	if folderID := os.Getenv("YDS_FOLDER_ID"); folderID != "" {
		cfg.FolderID = folderID
	}
	if streamName := os.Getenv("YDS_STREAM_NAME"); streamName != "" {
		cfg.StreamName = streamName
	}
	if apiKey := os.Getenv("YDS_API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
	}

	// If API key still empty, try OS keyring (most secure option)
	if cfg.APIKey == "" {
		if apiKey, err := secrets.GetAPIKey(); err == nil {
			cfg.APIKey = apiKey
		}
	}

	return cfg, nil
}
