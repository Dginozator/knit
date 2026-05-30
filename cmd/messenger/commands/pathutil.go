// Package commands provides shared utilities for CLI commands.
package commands

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// expandStoragePath expands ~ and $HOME to the user's home directory.
func expandStoragePath(path string) string {
	if path == "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	// Replace $HOME
	path = strings.ReplaceAll(path, "$HOME", home)

	// Replace ~/  and ~\
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		path = filepath.Join(home, path[2:])
	} else if path == "~" {
		path = home
	}

	return filepath.Clean(path)
}

// defaultStoragePath returns the resolved storage path from config,
// falling back to "keys" relative to the working directory.
func defaultStoragePath() string {
	raw := expandStoragePath(viper.GetString("storage.path"))
	if raw == "" {
		return "keys"
	}
	return raw
}
