package commands

import (
	"fmt"
	"os"

	"nit/internal/crypto"

	"github.com/spf13/cobra"
)

var (
	keygenName     string
	keygenPassword string
	keygenExport   string
)

// keygenCmd represents the keygen command.
var keygenCmd = &cobra.Command{
	Use:   "keygen [name]",
	Short: "Generate a new identity key pair",
	Long: `Generate a new cryptographic identity with encryption and signing keys.
The identity is stored securely using AES-256-GCM encryption.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Get password if not provided
		password := keygenPassword
		if password == "" {
			fmt.Print("Enter password: ")
			password = readPassword()
		}

		// Create keystore
		storePath := defaultStoragePath()

		store, err := crypto.NewKeystore(storePath)
		if err != nil {
			return fmt.Errorf("failed to create keystore: %w", err)
		}

		// Unlock keystore
		if err := store.Unlock(password); err != nil {
			return fmt.Errorf("failed to unlock keystore: %w", err)
		}

		// Generate identity
		identity, err := store.GenerateIdentity(name, password)
		if err != nil {
			return fmt.Errorf("failed to generate identity: %w", err)
		}

		// Print public key
		pubKey := identity.SignPublicKey()
		fmt.Printf("Identity '%s' generated successfully.\n", name)
		fmt.Printf("Public key (base64): %s\n", encodeBase64(pubKey))

		// Export if requested
		if keygenExport != "" {
			exportData, err := store.ExportIdentity(name, keygenExport)
			if err != nil {
				return fmt.Errorf("failed to export identity: %w", err)
			}
			fmt.Printf("Exported to: %s\n", keygenExport)
			os.WriteFile(keygenExport, exportData, 0600)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(keygenCmd)
	keygenCmd.Flags().StringVarP(&keygenPassword, "password", "p", "", "password for encryption")
	keygenCmd.Flags().StringVarP(&keygenExport, "export", "e", "", "export identity to file")
}

// readPassword reads a password from stdin.
func readPassword() string {
	var password string
	fmt.Scanln(&password)
	return password
}

func encodeBase64(data []byte) string {
	// Simple base64 encoding
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	result := make([]byte, (len(data)+2)/3*4)
	for i, j := 0, 0; i < len(data); i, j = i+3, j+4 {
		var v int
		switch len(data) - i {
		case 3:
			v = int(data[i])<<16 | int(data[i+1])<<8 | int(data[i+2])
		case 2:
			v = int(data[i])<<16 | int(data[i+1])<<8
		case 1:
			v = int(data[i])<<16
		}
		result[j] = alphabet[v>>18&0x3F]
		result[j+1] = alphabet[v>>12&0x3F]
		if len(data)-i > 1 {
			result[j+2] = alphabet[v>>6&0x3F]
		}
		if len(data)-i > 2 {
			result[j+3] = alphabet[v&0x3F]
		}
	}
	// Padding
	switch len(data) % 3 {
	case 1:
		result[len(result)-1] = '='
		result[len(result)-2] = '='
	case 2:
		result[len(result)-1] = '='
	}
	return string(result)
}
