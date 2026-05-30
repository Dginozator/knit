// Package commands provides CLI commands for the messenger.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

// RootCmd is the root command.
var RootCmd = &cobra.Command{
	Use:   "nit",
	Short: "E2EE Messenger with Yandex Data Streams",
	Long: `A secure, end-to-end encrypted messenger using Yandex Data Streams.
Supports age encryption and ed25519 digital signatures.`,
}

// Execute executes the root command.
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config/default.yaml)")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("default")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("$HOME/.nit")
		viper.AddConfigPath(".")
	}

	// Environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("NIT")

	// Defaults
	viper.SetDefault("yds.endpoint", "yds.serverless.yandexcloud.net")
	viper.SetDefault("yds.folder_id", "")
	viper.SetDefault("yds.stream_name", "messenger")
	viper.SetDefault("yds.region", "ru-central1")
	// Set storage path default to actual home directory
	if home, err := os.UserHomeDir(); err == nil {
		viper.SetDefault("storage.path", filepath.Join(home, ".nit", "keys"))
	} else {
		viper.SetDefault("storage.path", "$HOME/.nit/keys")
	}
	viper.SetDefault("polling.interval", "1s")
	viper.SetDefault("polling.batch_size", 10)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		}
	}
}
