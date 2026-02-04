package mcp

import (
	"os"
	"testing"
)

func TestNewServer(t *testing.T) {
	// Save and restore env vars
	oldAPIURL := os.Getenv("MSH_API_URL")
	oldToken := os.Getenv("MSH_TOKEN")
	defer func() {
		os.Setenv("MSH_API_URL", oldAPIURL)
		os.Setenv("MSH_TOKEN", oldToken)
	}()

	t.Run("default API URL", func(t *testing.T) {
		os.Unsetenv("MSH_API_URL")
		os.Unsetenv("MSH_TOKEN")

		server := NewServer()

		if server == nil {
			t.Fatal("NewServer() returned nil")
		}

		if server.mcpServer == nil {
			t.Error("server.mcpServer is nil")
		}

		if server.auth == nil {
			t.Error("server.auth is nil")
		}

		if server.handlers == nil {
			t.Error("server.handlers is nil")
		}

		// Check auth state uses default API URL
		authState := server.GetAuthState()
		if authState == nil {
			t.Fatal("GetAuthState() returned nil")
		}
		if authState.apiURL != DefaultAPIURL {
			t.Errorf("apiURL = %q, want %q", authState.apiURL, DefaultAPIURL)
		}
	})

	t.Run("custom API URL from env", func(t *testing.T) {
		customURL := "https://custom.api.example.com"
		os.Setenv("MSH_API_URL", customURL)
		os.Unsetenv("MSH_TOKEN")

		server := NewServer()

		if server == nil {
			t.Fatal("NewServer() returned nil")
		}

		authState := server.GetAuthState()
		if authState.apiURL != customURL {
			t.Errorf("apiURL = %q, want %q", authState.apiURL, customURL)
		}
	})

	t.Run("with token from env", func(t *testing.T) {
		os.Unsetenv("MSH_API_URL")
		os.Setenv("MSH_TOKEN", "test-token")

		server := NewServer()

		if server == nil {
			t.Fatal("NewServer() returned nil")
		}

		authState := server.GetAuthState()
		if !authState.IsAuthenticated() {
			t.Error("expected authenticated with MSH_TOKEN set")
		}
		if authState.GetToken() != "test-token" {
			t.Errorf("token = %q, want %q", authState.GetToken(), "test-token")
		}
	})
}

func TestServer_GetMCPServer(t *testing.T) {
	// Save and restore env vars
	oldAPIURL := os.Getenv("MSH_API_URL")
	oldToken := os.Getenv("MSH_TOKEN")
	defer func() {
		os.Setenv("MSH_API_URL", oldAPIURL)
		os.Setenv("MSH_TOKEN", oldToken)
	}()
	os.Unsetenv("MSH_API_URL")
	os.Unsetenv("MSH_TOKEN")

	server := NewServer()

	mcpServer := server.GetMCPServer()
	if mcpServer == nil {
		t.Error("GetMCPServer() returned nil")
	}
}

func TestServer_GetAuthState(t *testing.T) {
	// Save and restore env vars
	oldAPIURL := os.Getenv("MSH_API_URL")
	oldToken := os.Getenv("MSH_TOKEN")
	defer func() {
		os.Setenv("MSH_API_URL", oldAPIURL)
		os.Setenv("MSH_TOKEN", oldToken)
	}()
	os.Unsetenv("MSH_API_URL")
	os.Unsetenv("MSH_TOKEN")

	server := NewServer()

	authState := server.GetAuthState()
	if authState == nil {
		t.Error("GetAuthState() returned nil")
	}

	// Should be the same instance
	authState2 := server.GetAuthState()
	if authState != authState2 {
		t.Error("GetAuthState() should return same instance")
	}
}

func TestServerConstants(t *testing.T) {
	t.Parallel()

	if ServerName == "" {
		t.Error("ServerName should not be empty")
	}

	if ServerVersion == "" {
		t.Error("ServerVersion should not be empty")
	}

	if DefaultAPIURL == "" {
		t.Error("DefaultAPIURL should not be empty")
	}

	// Check default URL is valid
	if DefaultAPIURL != "https://api.joinme.sh" {
		t.Errorf("DefaultAPIURL = %q, expected https://api.joinme.sh", DefaultAPIURL)
	}
}

func TestServer_RegisterTools(t *testing.T) {
	// Save and restore env vars
	oldAPIURL := os.Getenv("MSH_API_URL")
	oldToken := os.Getenv("MSH_TOKEN")
	defer func() {
		os.Setenv("MSH_API_URL", oldAPIURL)
		os.Setenv("MSH_TOKEN", oldToken)
	}()
	os.Unsetenv("MSH_API_URL")
	os.Unsetenv("MSH_TOKEN")

	server := NewServer()

	// The server should have registered all tools
	// We can verify this by checking that GetMCPServer returns non-nil
	// (actual tool verification would require inspecting MCP server internals)
	if server.GetMCPServer() == nil {
		t.Error("MCP server should be initialized with tools")
	}
}
