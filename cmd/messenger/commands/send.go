package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"nit/internal/contacts"
	"nit/internal/crypto"
	"nit/internal/yds"
	pkgmessage "nit/pkg/message"

	"filippo.io/age"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	sendRecipient string
	sendIdentity  string
)

// sendCmd represents the send command.
var sendCmd = &cobra.Command{
	Use:   "send [recipient] [message]",
	Short: "Send an encrypted message",
	Long: `Send an end-to-end encrypted message to a recipient.

The message is encrypted with the recipient's age public key — only they can decrypt it.
You must have the recipient in your contacts first:
  nit contacts add bob age1xyz...`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		recipient := args[0]
		msgText := args[1]

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

		// Get sender identity
		store, err := crypto.NewKeystore(storePath)
		if err != nil {
			return fmt.Errorf("failed to create keystore: %w", err)
		}

		identityName := sendIdentity
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

		// Look up recipient's public key from personal contacts
		book := contacts.NewBookForIdentity(storePath, identityName)
		contact, err := book.Get(recipient)
		if err != nil {
			return fmt.Errorf("%w\n\nFirst add %s as a contact:\n  nit pubkey %s  (run on %s's device)\n  nit contacts add %s <their-age-key> --identity %s",
				err, recipient, recipient, recipient, recipient, identityName)
		}

		// Parse recipient's age public key
		recipientAge, err := age.ParseX25519Recipient(contact.AgeKey)
		if err != nil {
			return fmt.Errorf("invalid age key for contact %s: %w", recipient, err)
		}

		// Create message
		msg := &pkgmessage.Message{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			Type:      pkgmessage.MessageTypeText,
			From:      identityName,
			To:        recipient,
			Timestamp: time.Now().UTC(),
			Content:   []byte(msgText),
		}

		// Sign the message content with sender's ed25519 key
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to serialize message: %w", err)
		}

		signature, err := identity.Sign(msgBytes)
		if err != nil {
			return fmt.Errorf("failed to sign message: %w", err)
		}

		// Encrypt the message content with recipient's age public key
		var encBuf bytes.Buffer
		w, err := age.Encrypt(&encBuf, recipientAge)
		if err != nil {
			return fmt.Errorf("failed to create encryptor: %w", err)
		}
		if _, err := w.Write(msgBytes); err != nil {
			return fmt.Errorf("failed to encrypt message: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("failed to finalize encryption: %w", err)
		}

		// Build envelope with encrypted payload
		envelope := &pkgmessage.Envelope{
			Version:     1,
			RecipientID: recipient,
			SenderID:    identityName,
			MessageID:   msg.ID,
			Timestamp:   msg.Timestamp,
			Type:        msg.Type,
			Headers:     make(map[string][]byte),
			Encrypted: pkgmessage.EncryptedPayload{
				Ciphertext: encBuf.Bytes(), // age-encrypted
				Signature:  signature,       // ed25519 signature of plaintext
			},
		}

		// Encode to JSON
		data, err := json.Marshal(envelope)
		if err != nil {
			return fmt.Errorf("failed to encode envelope: %w", err)
		}

		// Send to YDB topic
		if err := client.WriteMessage(cmd.Context(), data); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		fmt.Printf("✅ Message sent to '%s'\n", recipient)
		fmt.Printf("   ID: %s\n", envelope.MessageID)
		fmt.Printf("   Encrypted with: %s's age key\n", recipient)

		return nil
	},
}

func init() {
	RootCmd.AddCommand(sendCmd)
	sendCmd.Flags().StringVarP(&sendIdentity, "identity", "i", "", "identity name to use")
}
