package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/nacl/box"
)

var dmCmd = &cobra.Command{
	Use:   "dm <@user> [text|-]",
	Short: "Send direct message",
	Long:  "Send an end-to-end encrypted direct message to a user",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		recipient := strings.TrimPrefix(args[0], "@")

		var content string
		var err error

		if len(args) > 1 {
			if args[1] == "-" {
				content, err = getStdinInput()
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: failed to read stdin: %v\n", err)
					os.Exit(1)
				}
			} else {
				content = strings.Join(args[1:], " ")
			}
		} else {
			content, err = getStdinInput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: failed to read stdin: %v\n", err)
				os.Exit(1)
			}
		}

		content = strings.TrimSpace(content)
		if content == "" {
			fmt.Fprintf(os.Stderr, "error: message content cannot be empty\n")
			os.Exit(1)
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		// Load or generate DM keys
		privateKey, publicKey, err := loadOrGenerateDMKeys()
		if err != nil {
			out.Error(fmt.Errorf("key management: %w", err))
			os.Exit(1)
		}

		// Get recipient's public key
		recipientKey, err := c.GetDMKey(recipient)
		if err != nil {
			out.Error(fmt.Errorf("failed to get recipient key: %w", err))
			os.Exit(1)
		}

		// Decrypt recipient's public key
		recipientPubKey, err := decodePublicKey(recipientKey.PublicKey)
		if err != nil {
			out.Error(fmt.Errorf("invalid recipient key: %w", err))
			os.Exit(1)
		}

		// Encrypt the message
		encryptedContent, err := encryptMessage(content, privateKey, recipientPubKey)
		if err != nil {
			out.Error(fmt.Errorf("encryption failed: %w", err))
			os.Exit(1)
		}

		// Send the DM
		req := &client.SendDMRequest{
			RecipientHandle: recipient,
			Content:         encryptedContent,
			AssetIDs:        postAttach,
		}

		dm, err := c.SendDM(req)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(dm)
		} else if !flagQuiet {
			out.Printf("✓ Sent DM to @%s: %s\n", recipient, dm.ID)
		}

		// Also ensure our public key is registered
		_ = registerDMKeyIfNeeded(c, publicKey)
	},
}

var dmLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List DM conversations",
	Long:  "List your direct message conversations",
	Run: func(cmd *cobra.Command, args []string) {
		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		dms, cursor, err := c.ListDMs(flagLimit, flagBefore, flagAfter)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if len(dms) == 0 {
			if !flagQuiet {
				out.Println("No DMs")
			}
			return
		}

		// Try to decrypt messages
		_, _, err = loadDMKeys()
		if err != nil {
			// Can't decrypt without keys
			if flagJSON {
				result := map[string]interface{}{
					"dms":    dms,
					"cursor": cursor,
				}
				out.Success(result)
			} else {
				for _, dm := range dms {
					renderDM(out, dm, "[Encrypted]")
				}
				if cursor != "" && !flagQuiet {
					out.Printf("\nNext page: --after %s\n", cursor)
				}
			}
			return
		}

		if flagJSON {
			result := map[string]interface{}{
				"dms":    dms,
				"cursor": cursor,
			}
			out.Success(result)
		} else {
			for _, dm := range dms {
				// Try to decrypt
				decrypted := "[Encrypted]"
				// Note: In a real implementation, we'd need the sender's public key
				// For now, just show encrypted
				renderDM(out, dm, decrypted)
			}
			if cursor != "" && !flagQuiet {
				out.Printf("\nNext page: --after %s\n", cursor)
			}
		}
	},
}

var dmKeyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage DM encryption keys",
	Long:  "Manage your end-to-end encryption keys for direct messages",
}

var dmKeyInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize DM encryption key",
	Long:  "Generate and register a new encryption key pair for DMs",
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		// Check if keys already exist
		if !force {
			if _, _, err := loadDMKeys(); err == nil {
				fmt.Fprintf(os.Stderr, "error: DM keys already exist. Use --force to regenerate.\n")
				fmt.Fprintf(os.Stderr, "Warning: Regenerating keys will make previous DMs unreadable.\n")
				os.Exit(1)
			}
		}

		out := getOutputPrinter()

		// Generate new keys
		publicKey, privateKey, err := box.GenerateKey(rand.Reader)
		if err != nil {
			out.Error(fmt.Errorf("key generation failed: %w", err))
			os.Exit(1)
		}

		// Save private key
		if err := saveDMKeys(privateKey, publicKey); err != nil {
			out.Error(fmt.Errorf("failed to save keys: %w", err))
			os.Exit(1)
		}

		// Register public key with server
		// cfg, _ := config.Load()
		c := getClient()

		pubKeyB64 := base64.StdEncoding.EncodeToString(publicKey[:])
		req := &client.RegisterDMKeyRequest{
			PublicKey: pubKeyB64,
		}

		key, err := c.RegisterDMKey(req)
		if err != nil {
			out.Error(fmt.Errorf("failed to register key: %w", err))
			os.Exit(1)
		}

		if flagJSON {
			out.Success(key)
		} else if !flagQuiet {
			out.Println("✓ DM encryption key initialized")
			out.Printf("  Public key: %s\n", pubKeyB64[:16]+"...")
		}
	},
}

var dmKeyShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show DM public key",
	Long:  "Display your DM encryption public key",
	Run: func(cmd *cobra.Command, args []string) {
		out := getOutputPrinter()

		_, publicKey, err := loadDMKeys()
		if err != nil {
			out.Error(fmt.Errorf("no DM keys found. Run 'mesh dm key init' first"))
			os.Exit(1)
		}

		pubKeyB64 := base64.StdEncoding.EncodeToString(publicKey[:])

		if flagJSON {
			out.Success(map[string]string{"public_key": pubKeyB64})
		} else {
			out.Printf("Public key: %s\n", pubKeyB64)
		}
	},
}

func loadOrGenerateDMKeys() (*[32]byte, *[32]byte, error) {
	privateKey, publicKey, err := loadDMKeys()
	if err == nil {
		return privateKey, publicKey, nil
	}

	// Generate new keys
	publicKey, privateKey, err = box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("key generation: %w", err)
	}

	if err := saveDMKeys(privateKey, publicKey); err != nil {
		return nil, nil, fmt.Errorf("save keys: %w", err)
	}

	return privateKey, publicKey, nil
}

func loadDMKeys() (*[32]byte, *[32]byte, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, fmt.Errorf("get home dir: %w", err)
	}

	keysDir := filepath.Join(homeDir, ".msh", "keys")
	privateKeyPath := filepath.Join(keysDir, "dm_private.key")

	data, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read private key: %w", err)
	}

	var keyData struct {
		PrivateKey string `json:"private_key"`
		PublicKey  string `json:"public_key"`
	}

	if err := json.Unmarshal(data, &keyData); err != nil {
		return nil, nil, fmt.Errorf("parse key data: %w", err)
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(keyData.PrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("decode private key: %w", err)
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(keyData.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("decode public key: %w", err)
	}

	var privateKey [32]byte
	var publicKey [32]byte
	copy(privateKey[:], privateKeyBytes)
	copy(publicKey[:], publicKeyBytes)

	return &privateKey, &publicKey, nil
}

func saveDMKeys(privateKey, publicKey *[32]byte) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	keysDir := filepath.Join(homeDir, ".msh", "keys")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return fmt.Errorf("create keys directory: %w", err)
	}

	privateKeyPath := filepath.Join(keysDir, "dm_private.key")

	keyData := struct {
		PrivateKey string `json:"private_key"`
		PublicKey  string `json:"public_key"`
	}{
		PrivateKey: base64.StdEncoding.EncodeToString(privateKey[:]),
		PublicKey:  base64.StdEncoding.EncodeToString(publicKey[:]),
	}

	data, err := json.MarshalIndent(keyData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal keys: %w", err)
	}

	if err := os.WriteFile(privateKeyPath, data, 0600); err != nil {
		return fmt.Errorf("write keys: %w", err)
	}

	return nil
}

func registerDMKeyIfNeeded(c *client.Client, publicKey *[32]byte) error {
	pubKeyB64 := base64.StdEncoding.EncodeToString(publicKey[:])
	req := &client.RegisterDMKeyRequest{
		PublicKey: pubKeyB64,
	}

	_, err := c.RegisterDMKey(req)
	return err
}

func decodePublicKey(encoded string) (*[32]byte, error) {
	bytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	if len(bytes) != 32 {
		return nil, fmt.Errorf("invalid key length: %d", len(bytes))
	}

	var key [32]byte
	copy(key[:], bytes)
	return &key, nil
}

func encryptMessage(message string, senderPrivateKey, recipientPublicKey *[32]byte) (string, error) {
	// Generate a random nonce
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt the message
	encrypted := box.Seal(nonce[:], []byte(message), &nonce, recipientPublicKey, senderPrivateKey)

	// Encode as base64
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func decryptMessage(encrypted string, recipientPrivateKey, senderPublicKey *[32]byte) (string, error) {
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	if len(data) < 24 {
		return "", fmt.Errorf("invalid encrypted message")
	}

	// Extract nonce
	var nonce [24]byte
	copy(nonce[:], data[:24])

	// Decrypt
	decrypted, ok := box.Open(nil, data[24:], &nonce, senderPublicKey, recipientPrivateKey)
	if !ok {
		return "", fmt.Errorf("decryption failed")
	}

	return string(decrypted), nil
}

func renderDM(out *output.Printer, dm *client.DM, decryptedContent string) {
	if out.IsJSON() {
		data, _ := json.Marshal(dm)
		out.Print(string(data))
		return
	}

	if out.IsRaw() {
		out.Printf("%s: %s\n", dm.ID, decryptedContent)
		return
	}

	direction := "→"
	if dm.SenderID != "" {
		// Determine if sent or received based on current user
		// For simplicity, showing as is
		out.Printf("%s %s • %s\n", direction, dm.ID, dm.CreatedAt.Format("2006-01-02 15:04"))
	}

	out.Printf("  %s\n", decryptedContent)

	if len(dm.AssetIDs) > 0 {
		out.Printf("  Attachments: %d\n", len(dm.AssetIDs))
	}
}

func init() {
	rootCmd.AddCommand(dmCmd)

	dmCmd.AddCommand(dmLsCmd)
	dmCmd.AddCommand(dmKeyCmd)

	dmKeyCmd.AddCommand(dmKeyInitCmd)
	dmKeyCmd.AddCommand(dmKeyShowCmd)

	dmCmd.Flags().StringSliceVar(&postAttach, "attach", []string{}, "Attach asset (path or as_id)")
	dmKeyInitCmd.Flags().Bool("force", false, "Force regenerate keys (makes old DMs unreadable)")
}
