package commands

import (
	"encoding/base64"
	"fmt"

	"nit/internal/contacts"
	"nit/internal/crypto"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var contactsIdentity string

// contactsCmd is the parent command for contact management.
var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Manage contacts (recipients' public keys)",
	Long: `Manage your personal contact book.

Each identity has their own contacts. Always specify --identity.
Ask contacts to share their key with: nit pubkey <name>`,
}

// contactsAddCmd adds a contact.
var contactsAddCmd = &cobra.Command{
	Use:   "add [name] [age-public-key]",
	Short: "Add a contact with their public key",
	Long: `Add a contact's age public key to your personal address book.

1. Ask the recipient to share their key: nit pubkey bob
2. Add them: nit contacts add bob age1xyz... --identity alice`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		ageKey := args[1]

		identity := contactsIdentity
		if identity == "" {
			identity = viper.GetString("identity.name")
		}
		if identity == "" {
			return fmt.Errorf("specify identity with --identity ИМЯ")
		}

		// Validate key format
		if len(ageKey) < 4 || ageKey[:4] != "age1" {
			return fmt.Errorf("invalid age public key — should start with 'age1...'")
		}

		book := contacts.NewBookForIdentity(defaultStoragePath(), identity)
		if err := book.Add(name, ageKey); err != nil {
			return fmt.Errorf("failed to add contact: %w", err)
		}

		fmt.Printf("✅ Contact '%s' added to %s's address book.\n", name, identity)
		fmt.Printf("   Age key: %s...\n", ageKey[:20])
		return nil
	},
}

// contactsListCmd lists all contacts for an identity.
var contactsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List contacts for an identity",
	RunE: func(cmd *cobra.Command, args []string) error {
		identity := contactsIdentity
		if identity == "" {
			identity = viper.GetString("identity.name")
		}
		if identity == "" {
			return fmt.Errorf("specify identity with --identity ИМЯ")
		}

		book := contacts.NewBookForIdentity(defaultStoragePath(), identity)
		list, err := book.List()
		if err != nil {
			return fmt.Errorf("failed to list contacts: %w", err)
		}

		fmt.Printf("Contacts for '%s':\n", identity)
		if len(list) == 0 {
			fmt.Println("  (none) — add with: nit contacts add <name> <age-pubkey> --identity", identity)
			return nil
		}

		fmt.Printf("%-15s %s\n", "NAME", "AGE PUBLIC KEY")
		fmt.Println("──────────────────────────────────────────────────────")
		for _, c := range list {
			keyPreview := c.AgeKey
			if len(keyPreview) > 40 {
				keyPreview = keyPreview[:40] + "..."
			}
			fmt.Printf("%-15s %s\n", c.Name, keyPreview)
		}
		return nil
	},
}

// contactsRemoveCmd removes a contact.
var contactsRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		identity := contactsIdentity
		if identity == "" {
			identity = viper.GetString("identity.name")
		}
		if identity == "" {
			return fmt.Errorf("specify identity with --identity ИМЯ")
		}

		book := contacts.NewBookForIdentity(defaultStoragePath(), identity)
		if err := book.Delete(name); err != nil {
			return fmt.Errorf("failed to remove contact: %w", err)
		}
		fmt.Printf("Contact '%s' removed from %s's address book.\n", name, identity)
		return nil
	},
}

// pubkeyCmd shows your own public key to share with others.
var pubkeyCmd = &cobra.Command{
	Use:   "pubkey [identity]",
	Short: "Show your public key to share with contacts",
	Long: `Show your age public key so others can add you as a contact.

Share this key, then they add you with:
  nit contacts add your-name <key> --identity their-name`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		identityName := args[0]

		store, err := crypto.NewKeystore(defaultStoragePath())
		if err != nil {
			return fmt.Errorf("failed to create keystore: %w", err)
		}

		fmt.Print("Enter password: ")
		password := readPassword()
		if err := store.Unlock(password); err != nil {
			return fmt.Errorf("failed to unlock keystore: %w", err)
		}

		identity, err := store.GetIdentity(identityName)
		if err != nil {
			return fmt.Errorf("failed to get identity: %w", err)
		}

		// Get age public key
		ageKey, err := identity.GetAgePublicKey()
		if err != nil {
			return fmt.Errorf("failed to get age public key: %w", err)
		}

		// Get signing public key
		signPub := identity.SignPublicKey()
		signPubB64 := base64.StdEncoding.EncodeToString(signPub)

		fmt.Printf("\n=== Public key for '%s' ===\n\n", identityName)
		fmt.Printf("Age encryption key (share this):\n  %s\n\n", ageKey)
		fmt.Printf("Signing verification key:\n  %s\n\n", signPubB64)
		fmt.Printf("The other person adds you with:\n")
		fmt.Printf("  nit contacts add %s %s --identity their-name\n", identityName, ageKey)

		return nil
	},
}

func init() {
	RootCmd.AddCommand(contactsCmd)
	contactsCmd.AddCommand(contactsAddCmd)
	contactsCmd.AddCommand(contactsListCmd)
	contactsCmd.AddCommand(contactsRemoveCmd)

	// Add --identity flag to all contacts subcommands
	for _, sub := range []*cobra.Command{contactsAddCmd, contactsListCmd, contactsRemoveCmd} {
		sub.Flags().StringVarP(&contactsIdentity, "identity", "i", "", "your identity name")
	}

	RootCmd.AddCommand(pubkeyCmd)
}
