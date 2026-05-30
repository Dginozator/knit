package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"nit/internal/crypto"
	"nit/internal/poller"
	"nit/internal/yds"
	pkgmessage "nit/pkg/message"

	"filippo.io/age"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	receiveIdentity string
	receiveLimit    int
)

// receiveCmd represents the receive command.
var receiveCmd = &cobra.Command{
	Use:   "receive",
	Short: "Receive and decrypt your messages",
	Long:  `Receive and decrypt messages addressed to you from the stream.`,
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

		// Get storage path
		storePath := defaultStoragePath()

		store, err := crypto.NewKeystore(storePath)
		if err != nil {
			return fmt.Errorf("failed to create keystore: %w", err)
		}

		identityName := receiveIdentity
		if identityName == "" {
			identityName = viper.GetString("identity.name")
		}
		if identityName == "" {
			return fmt.Errorf("identity name not specified")
		}

		// Unlock keystore
		fmt.Print("Enter password: ")
		password := readPassword()
		if err := store.Unlock(password); err != nil {
			return fmt.Errorf("failed to unlock keystore: %w", err)
		}

		identity, err := store.GetIdentity(identityName)
		if err != nil {
			return fmt.Errorf("failed to get identity: %w", err)
		}

		// Consumer name based on identity
		consumerName := "nit-consumer-" + identityName

		// Auto-create consumer if it doesn't exist
		exists, err := client.ConsumerExists(cmd.Context(), consumerName)
		if err != nil {
			return fmt.Errorf("failed to check consumer: %w", err)
		}
		if !exists {
			fmt.Printf("Creating consumer '%s'...\n", consumerName)
			if err := client.AddConsumer(cmd.Context(), consumerName); err != nil {
				return fmt.Errorf("failed to create consumer: %w", err)
			}
			fmt.Printf("Consumer created.\n")
		}

		// Get own age identity for decryption
		ageIdentity, err := identity.GetAgePrivateKey()
		if err != nil {
			return fmt.Errorf("failed to get age private key: %w", err)
		}

		// Parse age identity for decryption
		ageIdentities, err := age.ParseIdentities(strings.NewReader(ageIdentity))
		if err != nil {
			return fmt.Errorf("failed to parse age identity: %w", err)
		}

		// Create JSON codec for the poller
		codec := &jsonCodec{}
		p := poller.NewPoller(client, codec).WithConsumerName(consumerName)

		// Create handler — decrypt and show only messages for us
		handler := func(envelope *pkgmessage.Envelope) error {
			// Filter: only show messages addressed to me
			if envelope.RecipientID != identityName {
				return nil // not for me — skip silently
			}

			// Decrypt the ciphertext using our age private key
			plaintext, err := decryptAgeMessage(envelope.Encrypted.Ciphertext, ageIdentities)
			if err != nil {
				// Could not decrypt — not our message or corrupted
				return nil
			}

			// Parse the decrypted message
			var msg pkgmessage.Message
			if err := json.Unmarshal(plaintext, &msg); err != nil {
				return nil // skip malformed
			}

			// Show the message
			fmt.Fprintf(os.Stdout, "[%s] %s → %s: %s\n",
				msg.Timestamp.Format(time.RFC3339),
				msg.From,
				msg.To,
				string(msg.Content))

			return nil
		}

		// Start polling
		fmt.Printf("Listening for messages as '%s'...\n", identityName)
		if err := p.Start(cmd.Context(), handler); err != nil {
			return fmt.Errorf("failed to start poller: %w", err)
		}

		// Wait for interrupt or timeout
		select {
		case <-cmd.Context().Done():
		case <-time.After(30 * time.Second):
		}

		p.Stop()
		return nil
	},
}

// jsonCodec implements the poller.Codec interface.
type jsonCodec struct{}

func (c *jsonCodec) Decode(data []byte) (*pkgmessage.Envelope, error) {
	var envelope pkgmessage.Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}

// decryptAgeMessage decrypts age-encrypted ciphertext using the given identities.
func decryptAgeMessage(ciphertext []byte, identities []age.Identity) ([]byte, error) {
	r, err := age.Decrypt(bytes.NewReader(ciphertext), identities...)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func init() {
	RootCmd.AddCommand(receiveCmd)
	receiveCmd.Flags().StringVarP(&receiveIdentity, "identity", "i", "", "identity name to use")
	receiveCmd.Flags().IntVarP(&receiveLimit, "limit", "l", 100, "max messages to receive")
}
