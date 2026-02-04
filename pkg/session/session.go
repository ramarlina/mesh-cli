// Package session manages authentication session storage.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ramarlina/mesh-cli/pkg/models"
)

var (
	mu             sync.RWMutex
	globalSess     *Session
	sessionPath    string
	lastConfigDir  string
)

// Session represents an authenticated user session.
type Session struct {
	Token     string       `json:"token"`
	User      *models.User `json:"user"`
	ExpiresAt *time.Time   `json:"expires_at,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

func getSessionDir() (string, error) {
	// Check if MSH_CONFIG_DIR is set
	if configDir := os.Getenv("MSH_CONFIG_DIR"); configDir != "" {
		if err := os.MkdirAll(configDir, 0700); err != nil {
			return "", fmt.Errorf("create config directory: %w", err)
		}
		return configDir, nil
	}

	// Fall back to ~/.msh
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	mshDir := filepath.Join(homeDir, ".msh")
	if err := os.MkdirAll(mshDir, 0700); err != nil {
		return "", fmt.Errorf("create .msh directory: %w", err)
	}

	return mshDir, nil
}

// Load reads the session from disk.
func Load() (*Session, error) {
	mu.Lock()
	defer mu.Unlock()

	mshDir, err := getSessionDir()
	if err != nil {
		return nil, err
	}

	// Clear cached session if config directory changed
	if lastConfigDir != "" && lastConfigDir != mshDir {
		globalSess = nil
	}
	lastConfigDir = mshDir

	if globalSess != nil {
		return globalSess, nil
	}

	sessionPath = filepath.Join(mshDir, "session.json")

	// Check if session file exists
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no active session")
	}

	// Load existing session
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("read session file: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}

	// Check if session is expired
	if sess.ExpiresAt != nil && time.Now().After(*sess.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	globalSess = &sess
	return globalSess, nil
}

// Save persists the session to disk.
func Save(sess *Session) error {
	mu.Lock()
	defer mu.Unlock()

	mshDir, err := getSessionDir()
	if err != nil {
		return err
	}

	sessionPath = filepath.Join(mshDir, "session.json")

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	if err := os.WriteFile(sessionPath, data, 0600); err != nil {
		return fmt.Errorf("write session file: %w", err)
	}

	globalSess = sess
	return nil
}

// Clear removes the session from disk and memory.
func Clear() error {
	mu.Lock()
	defer mu.Unlock()

	mshDir, err := getSessionDir()
	if err != nil {
		return err
	}

	sessionPath = filepath.Join(mshDir, "session.json")

	// Remove file if it exists
	if _, err := os.Stat(sessionPath); err == nil {
		if err := os.Remove(sessionPath); err != nil {
			return fmt.Errorf("remove session file: %w", err)
		}
	}

	globalSess = nil
	return nil
}

// IsAuthenticated checks if there's an active, non-expired session.
func IsAuthenticated() bool {
	mu.RLock()
	defer mu.RUnlock()

	if globalSess == nil {
		return false
	}

	if globalSess.ExpiresAt != nil && time.Now().After(*globalSess.ExpiresAt) {
		return false
	}

	return true
}

// GetToken returns the current session token, or empty string if not authenticated.
func GetToken() string {
	mu.RLock()
	defer mu.RUnlock()

	if globalSess == nil {
		return ""
	}

	return globalSess.Token
}

// GetUser returns the current authenticated user, or nil if not authenticated.
func GetUser() *models.User {
	mu.RLock()
	defer mu.RUnlock()

	if globalSess == nil {
		return nil
	}

	return globalSess.User
}
