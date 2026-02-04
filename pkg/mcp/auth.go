// Package mcp provides an MCP server implementation for Mesh.
package mcp

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/models"
	"golang.org/x/crypto/ssh"
)

// AuthState manages in-memory authentication state for the MCP server.
// This is separate from the CLI's disk-based session to support
// stateless MCP operation.
type AuthState struct {
	mu       sync.RWMutex
	token    string
	user     *models.User
	apiURL   string
	client   *client.Client
	meshbotToken string
}

// NewAuthState creates a new authentication state manager.
func NewAuthState(apiURL string) *AuthState {
	state := &AuthState{
		apiURL: apiURL,
		meshbotToken: os.Getenv("MSH_MESHBOT_TOKEN"),
	}

	// Check for pre-configured token from environment
	if token := os.Getenv("MSH_TOKEN"); token != "" {
		state.token = token
		state.client = client.New(apiURL, client.WithToken(token))
	} else {
		state.client = client.New(apiURL)
	}

	return state
}

// IsAuthenticated returns true if there is a valid token.
func (a *AuthState) IsAuthenticated() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.token != ""
}

// GetToken returns the current authentication token.
func (a *AuthState) GetToken() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.token
}

// GetUser returns the current authenticated user.
func (a *AuthState) GetUser() *models.User {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.user
}

// GetClient returns an API client with current authentication.
func (a *AuthState) GetClient() *client.Client {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}

// GetMeshbotClient returns an API client authenticated as meshbot.
func (a *AuthState) GetMeshbotClient() (*client.Client, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.meshbotToken == "" {
		return nil, fmt.Errorf("MSH_MESHBOT_TOKEN not configured")
	}

	return client.New(a.apiURL, client.WithToken(a.meshbotToken)), nil
}

// SetAuth updates the authentication state.
func (a *AuthState) SetAuth(token string, user *models.User) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.token = token
	a.user = user
	a.client = client.New(a.apiURL, client.WithToken(token))
}

// Clear removes the authentication state.
func (a *AuthState) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.token = ""
	a.user = nil
	a.client = client.New(a.apiURL)
}

// Login performs SSH key-based authentication.
func (a *AuthState) Login(handle, keyPath string) error {
	// Normalize handle
	handle = strings.TrimPrefix(handle, "@")
	if handle == "" {
		return fmt.Errorf("handle is required")
	}

	// Find SSH key
	actualKeyPath, err := a.findSSHKey(keyPath)
	if err != nil {
		return fmt.Errorf("find SSH key: %w", err)
	}

	// Read private key
	keyData, err := os.ReadFile(actualKeyPath)
	if err != nil {
		return fmt.Errorf("read key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return fmt.Errorf("parse key: %w", err)
	}

	// Get public key
	pubKey := signer.PublicKey()
	pubKeyStr := string(ssh.MarshalAuthorizedKey(pubKey))

	// Request challenge
	c := client.New(a.apiURL)
	challenge, err := c.GetChallenge(handle)
	if err != nil {
		return fmt.Errorf("get challenge: %w", err)
	}

	// Sign challenge
	signature, err := signer.Sign(nil, []byte(challenge))
	if err != nil {
		return fmt.Errorf("sign challenge: %w", err)
	}

	// Base64 encode the signature
	sigB64 := base64.StdEncoding.EncodeToString(signature.Blob)

	// Verify and get token
	resp, err := c.Login(&client.LoginRequest{
		Handle:    handle,
		Challenge: challenge,
		Signature: sigB64,
		PublicKey: pubKeyStr,
	})
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	// Store authentication state
	a.SetAuth(resp.AccessToken, resp.User)

	return nil
}

// findSSHKey locates an SSH private key, using the provided path or searching
// default locations.
func (a *AuthState) findSSHKey(keyPath string) (string, error) {
	// If a specific path is provided, validate it
	if keyPath != "" {
		resolved, err := a.validateKeyPath(keyPath)
		if err != nil {
			return "", err
		}
		return resolved, nil
	}

	// Search for keys in default locations
	keyNames := []string{"id_ed25519", "id_rsa", "id_ecdsa"}

	// Check MSH_CONFIG_DIR first
	if configDir := os.Getenv("MSH_CONFIG_DIR"); configDir != "" {
		for _, name := range keyNames {
			kp := filepath.Join(configDir, name)
			if _, err := os.Stat(kp); err == nil {
				return kp, nil
			}
		}
	}

	// Fall back to ~/.ssh
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	for _, name := range keyNames {
		kp := filepath.Join(sshDir, name)
		if _, err := os.Stat(kp); err == nil {
			return kp, nil
		}
	}

	searchDirs := []string{sshDir}
	if configDir := os.Getenv("MSH_CONFIG_DIR"); configDir != "" {
		searchDirs = append([]string{configDir}, searchDirs...)
	}

	return "", fmt.Errorf("no SSH key found in %v", searchDirs)
}

// validateKeyPath ensures the key path is within allowed directories and exists.
func (a *AuthState) validateKeyPath(keyPath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	allowedDirs := []string{filepath.Join(homeDir, ".ssh")}
	if configDir := os.Getenv("MSH_CONFIG_DIR"); configDir != "" {
		allowedDirs = append(allowedDirs, configDir)
	}

	// Resolve the path
	resolved, err := filepath.Abs(keyPath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	resolved = filepath.Clean(resolved)

	// Check if path is within allowed directories
	allowed := false
	for _, dir := range allowedDirs {
		absDir, _ := filepath.Abs(dir)
		if strings.HasPrefix(resolved, absDir) {
			allowed = true
			break
		}
	}

	if !allowed {
		return "", fmt.Errorf("invalid key path: must be within ~/.ssh or MSH_CONFIG_DIR")
	}

	// Check if file exists
	if _, err := os.Stat(resolved); err != nil {
		return "", fmt.Errorf("SSH key not found: %s", keyPath)
	}

	return resolved, nil
}
