package commands

import (
	"fmt"

	"nit/internal/yds"

	"github.com/spf13/cobra"
)

var (
	initShardCount int
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the messenger (create topic/stream)",
	Long:  `Initialize the messenger by creating the Yandex Data Streams topic.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := yds.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create client
		client, err := yds.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer client.Close(cmd.Context())

		// Check if topic exists
		exists, err := client.TopicExists(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to check topic: %w", err)
		}

		if exists {
			fmt.Printf("Topic '%s' already exists.\n", cfg.TopicPath())
			return nil
		}

		// Create topic
		shardCount := int32(initShardCount)
		if shardCount == 0 {
			shardCount = 1
		}

		fmt.Printf("Creating topic '%s' with %d partitions...\n", cfg.TopicPath(), shardCount)
		if err := client.CreateTopic(cmd.Context(), shardCount); err != nil {
			return fmt.Errorf("failed to create topic: %w", err)
		}

		fmt.Printf("Topic '%s' created successfully.\n", cfg.TopicPath())
		return nil
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
	initCmd.Flags().IntVarP(&initShardCount, "shards", "s", 1, "number of partitions")
}
