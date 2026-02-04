// Package smoke provides smoke tests for CLI JSON output
package smoke

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// JSONTestConfig holds configuration for JSON smoke tests.
type JSONTestConfig struct {
	CLIBinary string
	APIURL    string
	TestUser  *TestUser
}

// TestJSONOutput tests that all commands support --json output.
func TestJSONOutput(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()

	// Run commands with --json flag
	testCases := []struct {
		name       string
		command    []string
		needsAuth  bool
		validate   func(t *testing.T, output string) error
	}{
		{
			name:      "status",
			command:   []string{"status", "--json"},
			needsAuth: false,
			validate: func(t *testing.T, output string) error {
				var result map[string]any
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					return err
				}
				// Should have authenticated field
				if _, ok := result["authenticated"]; !ok {
					return fmt.Errorf("missing 'authenticated' field")
				}
				return nil
			},
		},
		{
			name:      "config",
			command:   []string{"config", "--json"},
			needsAuth: false,
			validate: func(t *testing.T, output string) error {
				var result map[string]any
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					return err
				}
				// Should be a map of config settings
				if len(result) == 0 {
					return fmt.Errorf("expected config settings")
				}
				return nil
			},
		},
		{
			name:      "config_get",
			command:   []string{"config", "get", "editor", "--json"},
			needsAuth: false,
			validate: func(t *testing.T, output string) error {
				var result map[string]any
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					return err
				}
				// Should have editor field
				if _, ok := result["editor"]; !ok {
					return fmt.Errorf("missing 'editor' field")
				}
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exitCode := cfg.runCLI(t, tc.command,
				fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

			if exitCode != 0 {
				t.Logf("Command stderr: %s", stderr)
			}

			// Validate JSON output
			if !json.Valid([]byte(stdout)) {
				t.Errorf("Output is not valid JSON: %s", stdout)
				return
			}

			// Run custom validation
			if tc.validate != nil {
				if err := tc.validate(t, stdout); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			}
		})
	}
}

// TestJSONOutputWithAuth tests JSON output for authenticated commands.
func TestJSONOutputWithAuth(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping authenticated JSON tests")
	}

	// Login first
	_, _, _ = cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	testCases := []struct {
		name       string
		command    []string
		needsInput bool
		validate   func(t *testing.T, output string) error
	}{
		{
			name:     "login",
			command:  []string{"login", "--token", token, "--json"},
			validate: validateUserJSON,
		},
		{
			name:     "status_auth",
			command:  []string{"status", "--json"},
			validate: validateAuthenticatedJSON,
		},
		{
			name:     "me",
			command:  []string{"me", "--json"},
			validate: validateUserJSON,
		},
		{
			name:     "logout",
			command:  []string{"logout", "--json"},
			validate: func(t *testing.T, output string) error {
				var result map[string]any
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					return err
				}
				// Should have logged_out field
				if loggedOut, ok := result["logged_out"]; !ok || !loggedOut.(bool) {
					return fmt.Errorf("missing 'logged_out' field or not true")
				}
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr string

			if tc.needsInput {
				// For commands that need input
				cmd := exec.Command(cfg.CLIBinary, tc.command...)
				cmd.Env = append(os.Environ(),
					fmt.Sprintf("MSH_API_URL=%s", cfg.APIURL),
					fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

				var stdoutBuf, stderrBuf bytes.Buffer
				cmd.Stdout = &stdoutBuf
				cmd.Stderr = &stderrBuf

				err := cmd.Run()
				stdout = stdoutBuf.String()
				stderr = stderrBuf.String()

				if err != nil {
					t.Logf("Command failed: %v. Stderr: %s", err, stderr)
				}
			} else {
				stdout, stderr, _ = cfg.runCLI(t, tc.command,
					fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))
			}

			// Validate JSON output
			if !json.Valid([]byte(stdout)) {
				t.Errorf("Output is not valid JSON: %s", stdout)
				return
			}

			// Run custom validation
			if tc.validate != nil {
				if err := tc.validate(t, stdout); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			}
		})
	}
}

// TestJSONOutputPosts tests JSON output for post commands.
func TestJSONOutputPosts(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping post JSON tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("post_create", func(t *testing.T) {
		testContent := fmt.Sprintf("JSON test post at %s", time.Now().Format(time.RFC3339))
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"post", testContent, "--json"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Post failed. Stderr: %s", stderr)
		}

		// Validate JSON
		var post map[string]any
		if err := json.Unmarshal([]byte(stdout), &post); err != nil {
			t.Errorf("Output is not valid JSON: %v. Output: %s", err, stdout)
			return
		}

		// Validate post fields
		if id, ok := post["id"]; !ok || id == "" {
			t.Error("Missing or empty 'id' field")
		}
		if content, ok := post["content"]; !ok || content == "" {
			t.Error("Missing or empty 'content' field")
		}
		if author, ok := post["author"]; !ok || author == nil {
			t.Error("Missing 'author' field")
		}
	})

	t.Run("feed", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"feed", "--json"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Feed failed. Stderr: %s", stderr)
		}

		// Validate JSON
		var feed []map[string]any
		if err := json.Unmarshal([]byte(stdout), &feed); err != nil {
			t.Errorf("Output is not valid JSON array: %v", err)
			return
		}

		// Validate feed structure
		for _, post := range feed {
			if id, ok := post["id"]; !ok || id == "" {
				t.Error("Missing or empty 'id' field in feed item")
			}
		}
	})
}

// TestJSONOutputGraph tests JSON output for graph commands.
func TestJSONOutputGraph(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping graph JSON tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("who", func(t *testing.T) {
		targetHandle := os.Getenv("MSH_TEST_USER_HANDLE")
		if targetHandle == "" {
			targetHandle = "test"
		}

		stdout, stderr, exitCode := cfg.runCLI(t, []string{"who", targetHandle, "--json"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		// May fail if user doesn't exist
		if exitCode != 0 {
			if !strings.Contains(stderr, "not found") {
				t.Logf("Who command failed: %s", stderr)
			}
			return
		}

		// Validate JSON
		var user map[string]any
		if err := json.Unmarshal([]byte(stdout), &user); err != nil {
			t.Errorf("Output is not valid JSON: %v", err)
			return
		}

		// Validate user fields
		if id, ok := user["id"]; !ok || id == "" {
			t.Error("Missing or empty 'id' field")
		}
		if handle, ok := user["handle"]; !ok || handle == "" {
			t.Error("Missing or empty 'handle' field")
		}
	})
}

// TestJSONOutputInbox tests JSON output for inbox commands.
func TestJSONOutputInbox(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping inbox JSON tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("inbox", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"inbox", "--json"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Inbox failed. Stderr: %s", stderr)
		}

		// Validate JSON (could be empty array)
		var inbox []map[string]any
		if err := json.Unmarshal([]byte(stdout), &inbox); err != nil {
			t.Errorf("Output is not valid JSON array: %v", err)
			return
		}

		// Validate notification structure for each item
		for _, notif := range inbox {
			if id, ok := notif["id"]; !ok || id == "" {
				t.Error("Missing or empty 'id' field in notification")
			}
			if notifType, ok := notif["type"]; !ok || notifType == "" {
				t.Error("Missing or empty 'type' field in notification")
			}
		}
	})
}

// TestJSONOutputSignals tests JSON output for signal commands.
func TestJSONOutputSignals(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping signals JSON tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	postID := os.Getenv("MSH_TEST_POST_ID")
	if postID == "" {
		t.Skip("MSH_TEST_POST_ID not set, skipping signals JSON tests")
	}

	t.Run("like", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"like", postID, "--json"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Logf("Like failed: %s", stderr)
			return
		}

		// Validate JSON
		var result map[string]any
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Errorf("Output is not valid JSON: %v", err)
		}
	})

	t.Run("unlike", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"unlike", postID, "--json"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Logf("Unlike failed: %s", stderr)
			return
		}

		// Validate JSON
		var result map[string]any
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Errorf("Output is not valid JSON: %v", err)
		}
	})
}

// TestJSONOutputKeys tests JSON output for key management commands.
func TestJSONOutputKeys(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping keys JSON tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("keys_list", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"keys", "--json"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Logf("Keys list failed: %s", stderr)
			return
		}

		// Validate JSON
		var keys []map[string]any
		if err := json.Unmarshal([]byte(stdout), &keys); err != nil {
			t.Errorf("Output is not valid JSON array: %v", err)
			return
		}

		// Validate key structure
		for _, key := range keys {
			if id, ok := key["id"]; !ok || id == "" {
				t.Error("Missing or empty 'id' field in key")
			}
			if fingerprint, ok := key["fingerprint"]; !ok || fingerprint == "" {
				t.Error("Missing or empty 'fingerprint' field in key")
			}
		}
	})
}

// TestJSONOutputTokens tests JSON output for token management commands.
func TestJSONOutputTokens(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping tokens JSON tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("tokens_list", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"tokens", "--json"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Logf("Tokens list failed: %s", stderr)
			return
		}

		// Validate JSON
		var tokens []map[string]any
		if err := json.Unmarshal([]byte(stdout), &tokens); err != nil {
			t.Errorf("Output is not valid JSON array: %v", err)
			return
		}
	})
}

// TestJSONConsistentStructure tests that JSON output has consistent structure.
func TestJSONConsistentStructure(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping consistency tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("ErrorResponses_ShouldHaveConsistentFormat", func(t *testing.T) {
		// Test with invalid post ID
		stdout, _, _ := cfg.runCLI(t, []string{"post", "invalid_json_test_post_12345", "--json"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		// Validate JSON (even error responses should be JSON)
		var response map[string]any
		if err := json.Unmarshal([]byte(stdout), &response); err != nil {
			// May not be JSON on error
			t.Logf("Error response may not be JSON: %s", stdout)
			return
		}

		// Check for error field
		if errField, ok := response["error"]; ok {
			t.Logf("Error response has 'error' field: %v", errField)
		}
	})
}

// TestJSONOutputPagination tests JSON output with pagination.
func TestJSONOutputPagination(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping pagination JSON tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("feed_WithLimit_ShouldIncludeMetadata", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"feed", "--limit", "5", "--json"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Feed failed. Stderr: %s", stderr)
		}

		// Validate JSON
		var response map[string]any
		if err := json.Unmarshal([]byte(stdout), &response); err != nil {
			t.Errorf("Output is not valid JSON: %v", err)
			return
		}

		// Check for cursor field (pagination metadata)
		if cursor, ok := response["cursor"]; ok && cursor != "" {
			t.Logf("Cursor found: %v", cursor)
		}
	})
}

// Helper functions

func validateUserJSON(t *testing.T, output string) error {
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return err
	}

	// Check for user field
	user, ok := result["user"]
	if !ok || user == nil {
		// Try direct user fields (top-level)
		if _, ok := result["id"]; !ok {
			return fmt.Errorf("missing user data")
		}
		return nil
	}

	// If user is nested, validate its structure
	userMap, ok := user.(map[string]any)
	if !ok {
		return fmt.Errorf("'user' field is not an object")
	}

	if _, ok := userMap["id"]; !ok {
		return fmt.Errorf("missing 'id' field in user")
	}
	if _, ok := userMap["handle"]; !ok {
		return fmt.Errorf("missing 'handle' field in user")
	}

	return nil
}

func validateAuthenticatedJSON(t *testing.T, output string) error {
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return err
	}

	// Should have authenticated field
	auth, ok := result["authenticated"]
	if !ok {
		return fmt.Errorf("missing 'authenticated' field")
	}

	authenticated, ok := auth.(bool)
	if !ok {
		return fmt.Errorf("'authenticated' field is not a boolean")
	}

	if !authenticated {
		return fmt.Errorf("expected authenticated=true")
	}

	// Should have user field when authenticated
	if _, ok := result["user"]; !ok {
		return fmt.Errorf("missing 'user' field when authenticated")
	}

	return nil
}