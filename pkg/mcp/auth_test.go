package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ramarlina/mesh-cli/pkg/models"
)

func TestNewAuthState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		apiURL          string
		envToken        string
		envMeshbotToken string
		wantAuth        bool
		wantMeshbot     bool
	}{
		{
			name:     "basic initialization",
			apiURL:   "https://api.mesh.dev",
			wantAuth: false,
		},
		{
			name:     "with MSH_TOKEN env",
			apiURL:   "https://api.mesh.dev",
			envToken: "test-token-123",
			wantAuth: true,
		},
		{
			name:            "with MSH_MESHBOT_TOKEN env",
			apiURL:          "https://api.mesh.dev",
			envMeshbotToken: "meshbot-token-456",
			wantAuth:        false,
			wantMeshbot:     true,
		},
		{
			name:            "with both tokens",
			apiURL:          "https://api.mesh.dev",
			envToken:        "test-token-123",
			envMeshbotToken: "meshbot-token-456",
			wantAuth:        true,
			wantMeshbot:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			oldToken := os.Getenv("MSH_TOKEN")
			oldMeshbot := os.Getenv("MSH_MESHBOT_TOKEN")
			defer func() {
				os.Setenv("MSH_TOKEN", oldToken)
				os.Setenv("MSH_MESHBOT_TOKEN", oldMeshbot)
			}()

			// Set test env vars
			if tt.envToken != "" {
				os.Setenv("MSH_TOKEN", tt.envToken)
			} else {
				os.Unsetenv("MSH_TOKEN")
			}
			if tt.envMeshbotToken != "" {
				os.Setenv("MSH_MESHBOT_TOKEN", tt.envMeshbotToken)
			} else {
				os.Unsetenv("MSH_MESHBOT_TOKEN")
			}

			state := NewAuthState(tt.apiURL)

			if state == nil {
				t.Fatal("NewAuthState returned nil")
			}

			if state.apiURL != tt.apiURL {
				t.Errorf("apiURL = %q, want %q", state.apiURL, tt.apiURL)
			}

			if got := state.IsAuthenticated(); got != tt.wantAuth {
				t.Errorf("IsAuthenticated() = %v, want %v", got, tt.wantAuth)
			}

			if state.GetClient() == nil {
				t.Error("GetClient() returned nil")
			}

			// Test meshbot client
			meshbotClient, err := state.GetMeshbotClient()
			if tt.wantMeshbot {
				if err != nil {
					t.Errorf("GetMeshbotClient() error = %v, want nil", err)
				}
				if meshbotClient == nil {
					t.Error("GetMeshbotClient() returned nil client")
				}
			} else {
				if err == nil {
					t.Error("GetMeshbotClient() expected error, got nil")
				}
			}
		})
	}
}

func TestAuthState_IsAuthenticated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{"empty token", "", false},
		{"valid token", "some-token", true},
		{"whitespace token", "  ", true}, // Note: whitespace is considered valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &AuthState{token: tt.token}
			if got := state.IsAuthenticated(); got != tt.expected {
				t.Errorf("IsAuthenticated() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAuthState_GetToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"valid token", "test-token-abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &AuthState{token: tt.token}
			if got := state.GetToken(); got != tt.token {
				t.Errorf("GetToken() = %q, want %q", got, tt.token)
			}
		})
	}
}

func TestAuthState_GetUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		user *models.User
	}{
		{"nil user", nil},
		{
			"valid user",
			&models.User{
				ID:     "user-123",
				Handle: "testuser",
				Name:   "Test User",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &AuthState{user: tt.user}
			got := state.GetUser()

			if tt.user == nil {
				if got != nil {
					t.Errorf("GetUser() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("GetUser() returned nil, want user")
			}

			if got.ID != tt.user.ID {
				t.Errorf("GetUser().ID = %q, want %q", got.ID, tt.user.ID)
			}
			if got.Handle != tt.user.Handle {
				t.Errorf("GetUser().Handle = %q, want %q", got.Handle, tt.user.Handle)
			}
		})
	}
}

func TestAuthState_SetAuth(t *testing.T) {
	t.Parallel()

	// Clear env to avoid interference
	oldToken := os.Getenv("MSH_TOKEN")
	defer os.Setenv("MSH_TOKEN", oldToken)
	os.Unsetenv("MSH_TOKEN")

	state := NewAuthState("https://api.mesh.dev")

	// Initially not authenticated
	if state.IsAuthenticated() {
		t.Error("expected not authenticated initially")
	}

	// Set auth
	user := &models.User{
		ID:     "user-456",
		Handle: "newuser",
		Name:   "New User",
	}
	state.SetAuth("new-token-789", user)

	// Check authenticated
	if !state.IsAuthenticated() {
		t.Error("expected authenticated after SetAuth")
	}

	if got := state.GetToken(); got != "new-token-789" {
		t.Errorf("GetToken() = %q, want %q", got, "new-token-789")
	}

	gotUser := state.GetUser()
	if gotUser == nil {
		t.Fatal("GetUser() returned nil")
	}
	if gotUser.Handle != "newuser" {
		t.Errorf("GetUser().Handle = %q, want %q", gotUser.Handle, "newuser")
	}

	// Client should be updated with new token
	if state.GetClient() == nil {
		t.Error("GetClient() returned nil after SetAuth")
	}
}

func TestAuthState_Clear(t *testing.T) {
	t.Parallel()

	// Clear env to avoid interference
	oldToken := os.Getenv("MSH_TOKEN")
	defer os.Setenv("MSH_TOKEN", oldToken)
	os.Unsetenv("MSH_TOKEN")

	state := NewAuthState("https://api.mesh.dev")

	// Set auth first
	user := &models.User{
		ID:     "user-789",
		Handle: "clearme",
	}
	state.SetAuth("token-to-clear", user)

	if !state.IsAuthenticated() {
		t.Fatal("expected authenticated before clear")
	}

	// Clear
	state.Clear()

	// Check cleared
	if state.IsAuthenticated() {
		t.Error("expected not authenticated after Clear")
	}

	if got := state.GetToken(); got != "" {
		t.Errorf("GetToken() = %q, want empty", got)
	}

	if got := state.GetUser(); got != nil {
		t.Errorf("GetUser() = %v, want nil", got)
	}

	// Client should still be available (unauthenticated)
	if state.GetClient() == nil {
		t.Error("GetClient() returned nil after Clear")
	}
}

func TestAuthState_findSSHKey(t *testing.T) {
	// Create temp directory for test keys
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("failed to create temp .ssh dir: %v", err)
	}

	// Create test key files
	testKeyContent := []byte("test key content")
	ed25519Key := filepath.Join(sshDir, "id_ed25519")
	rsaKey := filepath.Join(sshDir, "id_rsa")

	if err := os.WriteFile(ed25519Key, testKeyContent, 0600); err != nil {
		t.Fatalf("failed to write test ed25519 key: %v", err)
	}
	if err := os.WriteFile(rsaKey, testKeyContent, 0600); err != nil {
		t.Fatalf("failed to write test rsa key: %v", err)
	}

	// Save and restore HOME
	oldHome := os.Getenv("HOME")
	oldConfigDir := os.Getenv("MSH_CONFIG_DIR")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("MSH_CONFIG_DIR", oldConfigDir)
	}()
	os.Setenv("HOME", tempDir)
	os.Unsetenv("MSH_CONFIG_DIR")

	state := &AuthState{}

	t.Run("finds ed25519 key in default location", func(t *testing.T) {
		path, err := state.findSSHKey("")
		if err != nil {
			t.Fatalf("findSSHKey() error = %v", err)
		}
		if path != ed25519Key {
			t.Errorf("findSSHKey() = %q, want %q", path, ed25519Key)
		}
	})

	t.Run("finds key when only rsa exists", func(t *testing.T) {
		// Remove ed25519 key
		if err := os.Remove(ed25519Key); err != nil {
			t.Fatalf("failed to remove ed25519 key: %v", err)
		}

		path, err := state.findSSHKey("")
		if err != nil {
			t.Fatalf("findSSHKey() error = %v", err)
		}
		if path != rsaKey {
			t.Errorf("findSSHKey() = %q, want %q", path, rsaKey)
		}

		// Restore ed25519 key for other tests
		if err := os.WriteFile(ed25519Key, testKeyContent, 0600); err != nil {
			t.Fatalf("failed to restore ed25519 key: %v", err)
		}
	})

	t.Run("uses specific path when provided", func(t *testing.T) {
		path, err := state.findSSHKey(rsaKey)
		if err != nil {
			t.Fatalf("findSSHKey() error = %v", err)
		}
		if path != rsaKey {
			t.Errorf("findSSHKey() = %q, want %q", path, rsaKey)
		}
	})

	t.Run("checks MSH_CONFIG_DIR first", func(t *testing.T) {
		configDir := filepath.Join(tempDir, "mesh-config")
		if err := os.MkdirAll(configDir, 0700); err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}
		configKey := filepath.Join(configDir, "id_ed25519")
		if err := os.WriteFile(configKey, testKeyContent, 0600); err != nil {
			t.Fatalf("failed to write config key: %v", err)
		}

		os.Setenv("MSH_CONFIG_DIR", configDir)
		defer os.Unsetenv("MSH_CONFIG_DIR")

		path, err := state.findSSHKey("")
		if err != nil {
			t.Fatalf("findSSHKey() error = %v", err)
		}
		if path != configKey {
			t.Errorf("findSSHKey() = %q, want %q", path, configKey)
		}
	})

	t.Run("returns error when no key found", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		if err := os.MkdirAll(filepath.Join(emptyDir, ".ssh"), 0700); err != nil {
			t.Fatalf("failed to create empty dir: %v", err)
		}

		os.Setenv("HOME", emptyDir)
		os.Unsetenv("MSH_CONFIG_DIR")

		_, err := state.findSSHKey("")
		if err == nil {
			t.Error("findSSHKey() expected error for missing key, got nil")
		}
	})
}

func TestAuthState_validateKeyPath(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	configDir := filepath.Join(tempDir, "mesh-config")
	outsideDir := filepath.Join(tempDir, "outside")

	for _, dir := range []string{sshDir, configDir, outsideDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	// Create test keys
	testKeyContent := []byte("test key content")
	sshKey := filepath.Join(sshDir, "id_ed25519")
	configKey := filepath.Join(configDir, "id_ed25519")
	outsideKey := filepath.Join(outsideDir, "id_ed25519")

	for _, key := range []string{sshKey, configKey, outsideKey} {
		if err := os.WriteFile(key, testKeyContent, 0600); err != nil {
			t.Fatalf("failed to write key %s: %v", key, err)
		}
	}

	// Save and restore env
	oldHome := os.Getenv("HOME")
	oldConfigDir := os.Getenv("MSH_CONFIG_DIR")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("MSH_CONFIG_DIR", oldConfigDir)
	}()
	os.Setenv("HOME", tempDir)

	state := &AuthState{}

	tests := []struct {
		name       string
		keyPath    string
		configDir  string
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid key in .ssh",
			keyPath: sshKey,
			wantErr: false,
		},
		{
			name:      "valid key in MSH_CONFIG_DIR",
			keyPath:   configKey,
			configDir: configDir,
			wantErr:   false,
		},
		{
			name:       "key outside allowed directories",
			keyPath:    outsideKey,
			wantErr:    true,
			errContain: "invalid key path",
		},
		{
			name:       "key outside allowed directories with config set",
			keyPath:    outsideKey,
			configDir:  configDir,
			wantErr:    true,
			errContain: "invalid key path",
		},
		{
			name:       "nonexistent key",
			keyPath:    filepath.Join(sshDir, "nonexistent"),
			wantErr:    true,
			errContain: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.configDir != "" {
				os.Setenv("MSH_CONFIG_DIR", tt.configDir)
			} else {
				os.Unsetenv("MSH_CONFIG_DIR")
			}

			resolved, err := state.validateKeyPath(tt.keyPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateKeyPath() expected error containing %q, got nil", tt.errContain)
				} else if tt.errContain != "" && !contains(err.Error(), tt.errContain) {
					t.Errorf("validateKeyPath() error = %q, want containing %q", err.Error(), tt.errContain)
				}
				return
			}

			if err != nil {
				t.Fatalf("validateKeyPath() unexpected error: %v", err)
			}

			// Check that resolved path is clean and absolute
			if !filepath.IsAbs(resolved) {
				t.Errorf("validateKeyPath() returned non-absolute path: %q", resolved)
			}
			if resolved != filepath.Clean(resolved) {
				t.Errorf("validateKeyPath() returned non-clean path: %q", resolved)
			}
		})
	}
}

func TestAuthState_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	// Clear env to avoid interference
	oldToken := os.Getenv("MSH_TOKEN")
	defer os.Setenv("MSH_TOKEN", oldToken)
	os.Unsetenv("MSH_TOKEN")

	state := NewAuthState("https://api.mesh.dev")

	// Run concurrent reads and writes
	done := make(chan bool)
	iterations := 100

	// Writer goroutine
	go func() {
		for i := 0; i < iterations; i++ {
			user := &models.User{ID: "user", Handle: "test"}
			state.SetAuth("token", user)
			state.Clear()
		}
		done <- true
	}()

	// Reader goroutines
	for i := 0; i < 3; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				_ = state.IsAuthenticated()
				_ = state.GetToken()
				_ = state.GetUser()
				_ = state.GetClient()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
	// If we get here without a data race (when running with -race), the test passes
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
