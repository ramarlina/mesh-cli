package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/config"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/ramarlina/mesh-cli/pkg/session"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var (
	flagToken string
)

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(statusCmd)

	loginCmd.Flags().StringVar(&flagToken, "token", "", "Login with API token")
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Mesh",
	Long:  "Authenticate using SSH key signing or API token",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		// Check if already logged in
		if session.IsAuthenticated() {
			user := session.GetUser()
			if user != nil {
				if !flagYes && !out.IsJSON() {
					fmt.Printf("Already logged in as @%s\n", user.Handle)
					fmt.Print("Logout first? (y/N): ")
					var response string
					fmt.Scanln(&response)
					if response != "y" && response != "Y" {
						return nil
					}
					if err := session.Clear(); err != nil {
						return fmt.Errorf("logout: %w", err)
					}
				}
			}
		}

		apiURL := config.GetAPIUrl()
		c := client.New(apiURL)

		// Token-based login
		if flagToken != "" {
			return loginWithToken(c, out, flagToken)
		}

		// SSH key signing login
		return loginWithSSH(c, out)
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "End current session",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		if err := session.Clear(); err != nil {
			return out.Error(err)
		}

		if !out.IsJSON() {
			out.Println("Logged out successfully")
		} else {
			out.Success(map[string]bool{"logged_out": true})
		}

		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		sess, err := session.Load()
		if err != nil {
			if out.IsJSON() {
				out.Success(map[string]interface{}{
					"authenticated": false,
				})
				return nil
			}
			out.Println("Not logged in")
			return nil
		}

		if out.IsJSON() {
			out.Success(map[string]interface{}{
				"authenticated": true,
				"user":          sess.User,
				"expires_at":    sess.ExpiresAt,
			})
			return nil
		}

		out.Printf("Logged in as @%s\n", sess.User.Handle)
		if sess.User.Name != "" {
			out.Printf("Name: %s\n", sess.User.Name)
		}
		out.Printf("User ID: %s\n", sess.User.ID)
		if sess.ExpiresAt != nil {
			out.Printf("Session expires: %s\n", sess.ExpiresAt.Format(time.RFC3339))
		}

		return nil
	},
}

func loginWithToken(c *client.Client, out *output.Printer, token string) error {
	// Create client with token
	c = client.New(config.GetAPIUrl(), client.WithToken(token))

	// Verify token by getting status
	user, err := c.GetStatus()
	if err != nil {
		return out.Error(fmt.Errorf("invalid token: %w", err))
	}

	// Save session
	sess := &session.Session{
		Token:     token,
		User:      user,
		CreatedAt: time.Now(),
	}

	if err := session.Save(sess); err != nil {
		return out.Error(fmt.Errorf("save session: %w", err))
	}

	if out.IsJSON() {
		out.Success(map[string]interface{}{
			"user": user,
		})
	} else {
		out.Printf("Logged in as @%s\n", user.Handle)
	}

	return nil
}

func loginWithSSH(c *client.Client, out *output.Printer) error {
	// Find SSH key
	keyPath, err := findSSHKey()
	if err != nil {
		return out.Error(fmt.Errorf("find SSH key: %w", err))
	}

	if !out.IsQuiet() && !out.IsJSON() {
		out.Printf("Using SSH key: %s\n", keyPath)
	}

	// Read private key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return out.Error(fmt.Errorf("read key: %w", err))
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return out.Error(fmt.Errorf("parse key: %w", err))
	}

	// Get public key
	pubKey := signer.PublicKey()
	pubKeyStr := string(ssh.MarshalAuthorizedKey(pubKey))

	// Request challenge
	if !out.IsQuiet() && !out.IsJSON() {
		out.Println("Requesting authentication challenge...")
	}

	challenge, err := c.GetChallenge()
	if err != nil {
		return out.Error(fmt.Errorf("get challenge: %w", err))
	}

	// Sign challenge
	signature, err := signer.Sign(nil, []byte(challenge))
	if err != nil {
		return out.Error(fmt.Errorf("sign challenge: %w", err))
	}

	// Login
	if !out.IsQuiet() && !out.IsJSON() {
		out.Println("Authenticating...")
	}

	resp, err := c.Login(&client.LoginRequest{
		Challenge: challenge,
		Signature: string(signature.Blob),
		PublicKey: pubKeyStr,
	})
	if err != nil {
		return out.Error(fmt.Errorf("login: %w", err))
	}

	// Save session
	sess := &session.Session{
		Token:     resp.Token,
		User:      resp.User,
		CreatedAt: time.Now(),
	}

	if err := session.Save(sess); err != nil {
		return out.Error(fmt.Errorf("save session: %w", err))
	}

	if out.IsJSON() {
		out.Success(map[string]interface{}{
			"user": resp.User,
		})
	} else {
		out.Printf("✓ Logged in as @%s\n", resp.User.Handle)
	}

	return nil
}

func findSSHKey() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Try common key names
	keyNames := []string{"id_ed25519", "id_rsa", "id_ecdsa"}

	for _, name := range keyNames {
		keyPath := filepath.Join(sshDir, name)
		if _, err := os.Stat(keyPath); err == nil {
			return keyPath, nil
		}
	}

	return "", fmt.Errorf("no SSH key found in %s", sshDir)
}
