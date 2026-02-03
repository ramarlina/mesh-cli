package main

import (
	"fmt"
	"os"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/config"
	"github.com/ramarlina/mesh-cli/pkg/session"
	"github.com/spf13/cobra"
)

var (
	flagKeyName string
)

func init() {
	rootCmd.AddCommand(keysCmd)

	keysCmd.AddCommand(keysAddCmd)
	keysCmd.AddCommand(keysLsCmd)
	keysCmd.AddCommand(keysRmCmd)

	keysAddCmd.Flags().StringVar(&flagKeyName, "name", "", "Display name for the key")
}

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage SSH keys",
	Long:  "Register and manage SSH public keys for authentication",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var keysAddCmd = &cobra.Command{
	Use:   "add <path-to-pubkey>",
	Short: "Register SSH public key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		// Must be authenticated
		token := session.GetToken()
		if token == "" {
			return out.Error(fmt.Errorf("not authenticated: run 'msh login' first"))
		}

		keyPath := args[0]

		// Read public key
		pubKeyData, err := os.ReadFile(keyPath)
		if err != nil {
			return out.Error(fmt.Errorf("read key: %w", err))
		}

		c := client.New(config.GetAPIUrl(), client.WithToken(token))

		key, err := c.AddSSHKey(&client.AddSSHKeyRequest{
			PublicKey: string(pubKeyData),
			Name:      flagKeyName,
		})
		if err != nil {
			return out.Error(fmt.Errorf("add key: %w", err))
		}

		if out.IsJSON() {
			return out.Success(key)
		}

		out.Printf("✓ Key added: %s\n", key.Fingerprint)
		if key.Name != "" {
			out.Printf("  Name: %s\n", key.Name)
		}

		return nil
	},
}

var keysLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List registered SSH keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		// Must be authenticated
		token := session.GetToken()
		if token == "" {
			return out.Error(fmt.Errorf("not authenticated: run 'msh login' first"))
		}

		c := client.New(config.GetAPIUrl(), client.WithToken(token))

		keys, err := c.ListSSHKeys()
		if err != nil {
			return out.Error(fmt.Errorf("list keys: %w", err))
		}

		if out.IsJSON() {
			return out.Success(keys)
		}

		if len(keys) == 0 {
			out.Println("No SSH keys registered")
			return nil
		}

		headers := []string{"Fingerprint", "Name", "Created"}
		rows := [][]string{}

		for _, key := range keys {
			name := key.Name
			if name == "" {
				name = "-"
			}
			rows = append(rows, []string{
				key.Fingerprint,
				name,
				key.CreatedAt.Format("2006-01-02"),
			})
		}

		return out.Table(headers, rows)
	},
}

var keysRmCmd = &cobra.Command{
	Use:   "rm <fingerprint>",
	Short: "Remove SSH key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		// Must be authenticated
		token := session.GetToken()
		if token == "" {
			return out.Error(fmt.Errorf("not authenticated: run 'msh login' first"))
		}

		fingerprint := args[0]

		if !flagYes && !out.IsJSON() {
			fmt.Printf("Remove SSH key %s? (y/N): ", fingerprint)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				return nil
			}
		}

		c := client.New(config.GetAPIUrl(), client.WithToken(token))

		if err := c.DeleteSSHKey(fingerprint); err != nil {
			return out.Error(fmt.Errorf("remove key: %w", err))
		}

		if out.IsJSON() {
			return out.Success(map[string]bool{"removed": true})
		}

		out.Printf("✓ Key removed: %s\n", fingerprint)
		return nil
	},
}
