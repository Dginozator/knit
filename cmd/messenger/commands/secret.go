package commands

import (
	"errors"
	"fmt"

	"nit/internal/secrets"

	"github.com/spf13/cobra"
)

// secretCmd is the parent command for secret management.
var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage secrets stored in the OS keychain",
	Long: `Securely store and manage sensitive credentials (API keys, tokens)
using the native OS keychain:
  - Windows: Credential Manager
  - macOS:   Keychain
  - Linux:   Secret Service (GNOME Keyring / KWallet)`,
}

// secretSetCmd stores a secret in the keychain.
var secretSetCmd = &cobra.Command{
	Use:   "set-api-key [api-key]",
	Short: "Store YDS API key in OS keychain",
	Long: `Store the Yandex Data Streams API key securely in the OS keychain.
After this, you no longer need to set the YDS_API_KEY environment variable.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := args[0]
		if err := secrets.SetAPIKey(apiKey); err != nil {
			return fmt.Errorf("failed to store API key: %w", err)
		}
		fmt.Println("✅ API key stored securely in OS keychain.")
		fmt.Println("   You no longer need to set YDS_API_KEY environment variable.")
		return nil
	},
}

// secretGetCmd retrieves the API key from keychain (masked).
var secretGetCmd = &cobra.Command{
	Use:   "get-api-key",
	Short: "Check if API key is stored in OS keychain",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := secrets.GetAPIKey()
		if err != nil {
			if errors.Is(err, secrets.ErrNotFound) {
				fmt.Println("❌ No API key found in OS keychain.")
				fmt.Println("   Run: nit secret set-api-key AQVN3...")
				return nil
			}
			return fmt.Errorf("failed to retrieve API key: %w", err)
		}
		// Show masked version for confirmation
		masked := maskSecret(apiKey)
		fmt.Printf("✅ API key found in keychain: %s\n", masked)
		return nil
	},
}

// secretDeleteCmd removes the API key from keychain.
var secretDeleteCmd = &cobra.Command{
	Use:   "delete-api-key",
	Short: "Remove API key from OS keychain",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := secrets.DeleteAPIKey(); err != nil {
			if errors.Is(err, secrets.ErrNotFound) {
				fmt.Println("No API key found in keychain (nothing to delete).")
				return nil
			}
			return fmt.Errorf("failed to delete API key: %w", err)
		}
		fmt.Println("✅ API key removed from OS keychain.")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(secretCmd)
	secretCmd.AddCommand(secretSetCmd)
	secretCmd.AddCommand(secretGetCmd)
	secretCmd.AddCommand(secretDeleteCmd)
}

// maskSecret masks all but the first 4 characters of a secret.
func maskSecret(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "..." + "****"
}
