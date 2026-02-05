package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/config"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/ramarlina/mesh-cli/pkg/session"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var (
	flagToken  string
	flagHandle string
	flagGoogle bool
)

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(statusCmd)

	loginCmd.Flags().StringVar(&flagToken, "token", "", "Login with API token")
	loginCmd.Flags().StringVarP(&flagHandle, "handle", "u", "", "Your handle/username")
	loginCmd.Flags().BoolVar(&flagGoogle, "google", false, "Login with Google/Gmail OAuth")
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Mesh",
	Long:  "Authenticate using Google OAuth, SSH key signing, or API token",
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

		// Google OAuth login
		if flagGoogle {
			return loginWithGoogle(c, out)
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

func loginWithGoogle(c *client.Client, out *output.Printer) error {
	if !out.IsQuiet() && !out.IsJSON() {
		out.Println("Initiating Google OAuth login...")
	}

	// Start a local HTTP server to receive the callback
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return out.Error(fmt.Errorf("start callback server: %w", err))
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Get the authorization URL
	authResp, err := c.GetGoogleAuthURL(callbackURL)
	if err != nil {
		return out.Error(fmt.Errorf("get auth URL: %w", err))
	}

	if !out.IsQuiet() && !out.IsJSON() {
		out.Println("Opening browser for Google authentication...")
		out.Printf("If browser doesn't open, visit:\n%s\n\n", authResp.AuthURL)
	}

	// Open browser
	openBrowser(authResp.AuthURL)

	// Wait for callback
	codeChan := make(chan string, 1)
	stateChan := make(chan string, 1)
	errChan := make(chan error, 1)

	srv := &http.Server{}
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><body><h1>Authentication Failed</h1><p>%s</p></body></html>", errMsg)
			errChan <- fmt.Errorf("%s", errMsg)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h1>Authentication Successful!</h1><p>You can close this window.</p></body></html>")

		codeChan <- code
		stateChan <- state
	})

	go srv.Serve(listener)

	// Wait for callback with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var code, state string
	select {
	case code = <-codeChan:
		state = <-stateChan
	case err := <-errChan:
		return out.Error(err)
	case <-ctx.Done():
		return out.Error(fmt.Errorf("authentication timed out"))
	}

	srv.Shutdown(context.Background())

	// Exchange code for tokens
	if !out.IsQuiet() && !out.IsJSON() {
		out.Println("Completing authentication...")
	}

	callbackResp, err := c.ExchangeGoogleCode(code, state)
	if err != nil {
		return out.Error(fmt.Errorf("exchange code: %w", err))
	}

	// Check if new user needs to claim a username
	if callbackResp.Status == "username_required" {
		return handleUsernameClaim(c, out, callbackResp.GoogleID)
	}

	// Save session
	sess := &session.Session{
		Token:     callbackResp.AccessToken,
		User:      callbackResp.User,
		CreatedAt: time.Now(),
	}

	if err := session.Save(sess); err != nil {
		return out.Error(fmt.Errorf("save session: %w", err))
	}

	if out.IsJSON() {
		out.Success(map[string]interface{}{
			"user":       callbackResp.User,
			"is_new_user": callbackResp.IsNewUser,
		})
	} else {
		if callbackResp.IsNewUser {
			out.Printf("✓ Welcome to Mesh, @%s!\n", callbackResp.User.Handle)
		} else {
			out.Printf("✓ Logged in as @%s\n", callbackResp.User.Handle)
		}
	}

	return nil
}

func handleUsernameClaim(c *client.Client, out *output.Printer, googleID string) error {
	if out.IsJSON() {
		return out.Error(fmt.Errorf("username claim required, use interactive mode"))
	}

	out.Println("\n🎉 Welcome to Mesh! Let's claim your username.")
	out.Println("Your username will be unique and used for your @handle.")

	for {
		fmt.Print("Choose a username: @")
		var handle string
		fmt.Scanln(&handle)

		handle = strings.TrimSpace(strings.ToLower(handle))
		if handle == "" {
			out.Println("Username cannot be empty")
			continue
		}

		// Try to claim the username
		resp, err := c.ClaimUsername(&client.ClaimUsernameRequest{
			GoogleID: googleID,
			Handle:   handle,
		})
		if err != nil {
			if strings.Contains(err.Error(), "already taken") {
				out.Printf("Username @%s is already taken. Try another.\n", handle)
				continue
			}
			if strings.Contains(err.Error(), "invalid") {
				out.Println("Invalid username. Use only lowercase letters, numbers, and underscores (1-32 chars).")
				continue
			}
			return out.Error(fmt.Errorf("claim username: %w", err))
		}

		// Save session
		sess := &session.Session{
			Token:     resp.AccessToken,
			User:      resp.User,
			CreatedAt: time.Now(),
		}

		if err := session.Save(sess); err != nil {
			return out.Error(fmt.Errorf("save session: %w", err))
		}

		out.Printf("\n✓ Welcome to Mesh, @%s!\n", resp.User.Handle)
		return nil
	}
}

func loginWithSSH(c *client.Client, out *output.Printer) error {
	// Find SSH key first (needed for auto-generated handle)
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

	// Get handle - auto-generate from key fingerprint if not provided
	handle := flagHandle
	if handle == "" {
		handle = generateHandleFromKey(signer.PublicKey())
		if !out.IsQuiet() && !out.IsJSON() {
			out.Printf("Generated handle: @%s\n", handle)
			out.Println("(Change later with: mesh profile set --handle @newname)")
		}
	}

	// Get public key
	pubKey := signer.PublicKey()
	pubKeyStr := string(ssh.MarshalAuthorizedKey(pubKey))

	// Request challenge
	if !out.IsQuiet() && !out.IsJSON() {
		out.Println("Requesting authentication challenge...")
	}

	challenge, err := c.GetChallenge(handle)
	if err != nil {
		// If user not found, auto-register
		if strings.Contains(err.Error(), "not found") {
			if !out.IsQuiet() && !out.IsJSON() {
				out.Println("Registering new account...")
			}
			if regErr := c.Register(&client.RegisterRequest{
				Handle:    handle,
				PublicKey: pubKeyStr,
			}); regErr != nil {
				return out.Error(fmt.Errorf("register: %w", regErr))
			}
			// Retry getting challenge
			challenge, err = c.GetChallenge(handle)
			if err != nil {
				return out.Error(fmt.Errorf("get challenge after register: %w", err))
			}
		} else {
			return out.Error(fmt.Errorf("get challenge: %w", err))
		}
	}

	// Sign challenge
	signature, err := signer.Sign(nil, []byte(challenge))
	if err != nil {
		return out.Error(fmt.Errorf("sign challenge: %w", err))
	}

	// Base64 encode the signature
	sigB64 := base64.StdEncoding.EncodeToString(signature.Blob)

	// Login
	if !out.IsQuiet() && !out.IsJSON() {
		out.Println("Authenticating...")
	}

	resp, err := c.Login(&client.LoginRequest{
		Handle:    handle,
		Challenge: challenge,
		Signature: sigB64,
		PublicKey: pubKeyStr,
	})
	if err != nil {
		return out.Error(fmt.Errorf("login: %w", err))
	}

	// Save session
	sess := &session.Session{
		Token:     resp.AccessToken,
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
	// Try common key names
	keyNames := []string{"id_ed25519", "id_rsa", "id_ecdsa"}

	// First, check MSH_CONFIG_DIR if set
	configDir := os.Getenv("MSH_CONFIG_DIR")
	if configDir != "" {
		for _, name := range keyNames {
			keyPath := filepath.Join(configDir, name)
			if _, err := os.Stat(keyPath); err == nil {
				return keyPath, nil
			}
		}
	}

	// Fall back to ~/.ssh
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	for _, name := range keyNames {
		keyPath := filepath.Join(sshDir, name)
		if _, err := os.Stat(keyPath); err == nil {
			return keyPath, nil
		}
	}

	searchDirs := []string{sshDir}
	if configDir != "" {
		searchDirs = append([]string{configDir}, searchDirs...)
	}

	return "", fmt.Errorf("no SSH key found in %v", searchDirs)
}

// Word lists for mnemonic handle generation
var adjectives = []string{
	"swift", "bright", "calm", "bold", "keen",
	"warm", "cool", "wild", "soft", "sharp",
	"quick", "still", "deep", "clear", "free",
	"wise", "pure", "true", "fair", "kind",
	"brave", "quiet", "vivid", "lucid", "noble",
	"gentle", "steady", "silent", "golden", "silver",
	"cosmic", "lunar", "solar", "astral", "stellar",
	"hidden", "ancient", "serene", "mystic", "curious",
}

var nouns = []string{
	"fox", "owl", "wolf", "bear", "hawk",
	"deer", "hare", "crow", "swan", "dove",
	"river", "stone", "cloud", "storm", "flame",
	"frost", "wave", "peak", "vale", "grove",
	"star", "moon", "comet", "nova", "spark",
	"echo", "shade", "drift", "pulse", "glow",
	"cipher", "prism", "nexus", "arc", "flux",
	"sage", "scout", "herald", "seeker", "warden",
}

// generateHandleFromKey creates a deterministic mnemonic handle from an SSH public key
func generateHandleFromKey(pubKey ssh.PublicKey) string {
	fp := ssh.FingerprintSHA256(pubKey)
	// fp is like "SHA256:abc123...xyz"
	parts := strings.Split(fp, ":")
	if len(parts) == 2 {
		b64 := parts[1]
		// Use bytes from fingerprint to pick words deterministically
		var sum1, sum2 int
		for i, c := range b64 {
			if i%2 == 0 {
				sum1 += int(c)
			} else {
				sum2 += int(c)
			}
		}
		adj := adjectives[sum1%len(adjectives)]
		noun := nouns[sum2%len(nouns)]
		return adj + "_" + noun
	}
	// Fallback: use timestamp-based
	return fmt.Sprintf("agent-%x", time.Now().UnixNano()%0xFFFFFF)
}
