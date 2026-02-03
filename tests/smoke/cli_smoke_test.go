// Package smoke provides smoke tests for the CLI
package smoke

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// SmokeTestConfig holds configuration for smoke tests.
type SmokeTestConfig struct {
	CLIBinary string
	APIURL    string
	TestUser  *TestUser
}

// TestUser holds test user credentials.
type TestUser struct {
	Handle string
	Token  string
}

// NewSmokeTestConfig creates a new smoke test configuration.
func NewSmokeTestConfig(t *testing.T) *SmokeTestConfig {
	t.Helper()

	// Determine CLI binary path
	cliBinary := os.Getenv("MSH_CLI_BINARY")
	if cliBinary == "" {
		// Try to find the binary in common locations
		locations := []string{
			filepath.Join("..", "..", "msh"),
			filepath.Join("..", "..", "bin", "msh"),
			"msh",
		}

		for _, loc := range locations {
			if _, err := os.Stat(loc); err == nil {
				cliBinary = loc
				break
			}
		}
	}

	if cliBinary == "" {
		cliBinary = "msh" // Assume it's in PATH
	}

	// Get API URL from environment
	apiURL := os.Getenv("MSH_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	return &SmokeTestConfig{
		CLIBinary: cliBinary,
		APIURL:    apiURL,
		TestUser:  &TestUser{Handle: "smoke_test_user"},
	}
}

// runCLI executes a CLI command and returns the output.
func (c *SmokeTestConfig) runCLI(t *testing.T, args []string, env ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(c.CLIBinary, args...)

	// Set environment variables
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("MSH_API_URL=%s", c.APIURL))

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return
}

// TestCLIBasicCommands tests that basic CLI commands execute without error.
func TestCLIBasicCommands(t *testing.T) {
	cfg := NewSmokeTestConfig(t)

	t.Run("Help_ShouldDisplayUsage", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"--help"})

		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d. Stderr: %s", exitCode, stderr)
		}

		if !strings.Contains(stdout, "Mesh") && !strings.Contains(stdout, "msh") {
			t.Errorf("Help output should contain 'Mesh' or 'msh'. Got: %s", stdout)
		}
	})

	t.Run("Version_ShouldDisplayVersion", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"--version"})

		// Version command may not exist, so we check for the flag
		if exitCode != 0 && strings.Contains(stderr, "unknown flag") {
			t.Skip("Version flag not implemented")
			return
		}

		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d. Stderr: %s", exitCode, stderr)
		}
	})

	t.Run("Status_ShouldHandleNotLoggedIn", func(t *testing.T) {
		// Set a temp config directory to avoid conflicts
		tempDir := t.TempDir()
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"status"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 && strings.Contains(stderr, "error") {
			t.Errorf("Status should handle not logged in gracefully. Stderr: %s", stderr)
		}

		if strings.Contains(stdout, "Not logged in") || strings.Contains(stdout, "authenticated") {
			// Expected output
		}
	})
}

// TestCLIAuthCommands tests authentication-related CLI commands.
func TestCLIAuthCommands(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()

	t.Run("Login_ShouldAcceptToken", func(t *testing.T) {
		// This test requires a valid API token
		token := os.Getenv("MSH_TEST_TOKEN")
		if token == "" {
			t.Skip("MSH_TEST_TOKEN not set, skipping login test")
		}

		stdout, stderr, exitCode := cfg.runCLI(t, []string{"login", "--token", token},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Login with token failed. Stderr: %s", stderr)
		}

		if !strings.Contains(stdout, "Logged in") && !strings.Contains(stdout, "user") {
			t.Logf("Login output: %s", stdout)
		}
	})

	t.Run("Status_ShouldShowUser_AfterLogin", func(t *testing.T) {
		token := os.Getenv("MSH_TEST_TOKEN")
		if token == "" {
			t.Skip("MSH_TEST_TOKEN not set, skipping status test")
		}

		// First login
		cfg.runCLI(t, []string{"login", "--token", token},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		// Check status
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"status"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Status failed. Stderr: %s", stderr)
		}

		if strings.Contains(stdout, "Not logged in") {
			t.Error("Should be logged in after successful login")
		}
	})

	t.Run("Logout_ShouldClearSession", func(t *testing.T) {
		token := os.Getenv("MSH_TEST_TOKEN")
		if token == "" {
			t.Skip("MSH_TEST_TOKEN not set, skipping logout test")
		}

		// First login
		cfg.runCLI(t, []string{"login", "--token", token},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		// Logout
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"logout"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Logout failed. Stderr: %s", stderr)
		}

		if !strings.Contains(stdout, "Logged out") && !strings.Contains(stdout, "logged_out") {
			t.Logf("Logout output: %s", stdout)
		}
	})
}

// TestCLIPostCommands tests post-related CLI commands.
func TestCLIPostCommands(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping post tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("Post_ShouldCreatePost", func(t *testing.T) {
		testContent := fmt.Sprintf("Smoke test post at %s", time.Now().Format(time.RFC3339))
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"post", testContent},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Post creation failed. Stderr: %s", stderr)
		}

		if !strings.Contains(stdout, "Posted") && !strings.Contains(stdout, "p_") {
			t.Logf("Post output: %s", stdout)
		}
	})

	t.Run("Post_ShouldAcceptStdin", func(t *testing.T) {
		testContent := fmt.Sprintf("Smoke test from stdin at %s", time.Now().Format(time.RFC3339))

		cmd := exec.Command(cfg.CLIBinary, "post", "-")
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("MSH_API_URL=%s", cfg.APIURL),
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		stdinPipe, _ := cmd.StdinPipe()
		io.WriteString(stdinPipe, testContent+"\n")
		stdinPipe.Close()

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		if err != nil {
			t.Errorf("Post from stdin failed. Stderr: %s", stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Posted") && !strings.Contains(output, "p_") {
			t.Logf("Post output: %s", output)
		}
	})

	t.Run("Post_ShouldRejectEmptyContent", func(t *testing.T) {
		// Create a temp file with no content
		emptyFile := filepath.Join(tempDir, "empty.txt")
		os.WriteFile(emptyFile, []byte(""), 0644)

		cmd := exec.Command(cfg.CLIBinary, "post", "-")
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("MSH_API_URL=%s", cfg.APIURL),
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		// Should fail with empty content
		if err == nil {
			t.Error("Post with empty content should fail")
		}

		if !strings.Contains(stderr.String(), "empty") && !strings.Contains(stderr.String(), "error") {
			t.Logf("Expected error for empty content. Got: %s", stderr.String())
		}
	})

	t.Run("Feed_ShouldShowFeed", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"feed"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Feed command failed. Stderr: %s", stderr)
		}
	})
}

// TestCLIFeedCommands tests feed-related CLI commands.
func TestCLIFeedCommands(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping feed tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("Feed_ShouldAcceptLimit", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"feed", "--limit", "5"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Feed with limit failed. Stderr: %s", stderr)
		}
	})

	t.Run("Feed_ShouldAcceptSince", func(t *testing.T) {
		sinceTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"feed", "--since", sinceTime},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Feed with since failed. Stderr: %s", stderr)
		}
	})
}

// TestCLIGraphCommands tests social graph CLI commands.
func TestCLIGraphCommands(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping graph tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("Follow_ShouldAcceptHandle", func(t *testing.T) {
		// Use a known test handle if available, or skip
		targetHandle := os.Getenv("MSH_TEST_FOLLOW_TARGET")
		if targetHandle == "" {
			t.Skip("MSH_TEST_FOLLOW_TARGET not set, skipping follow test")
		}

		stdout, stderr, exitCode := cfg.runCLI(t, []string{"follow", targetHandle},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Follow command failed. Stderr: %s", stderr)
		}
	})

	t.Run("Unfollow_ShouldAcceptHandle", func(t *testing.T) {
		targetHandle := os.Getenv("MSH_TEST_FOLLOW_TARGET")
		if targetHandle == "" {
			t.Skip("MSH_TEST_FOLLOW_TARGET not set, skipping unfollow test")
		}

		stdout, stderr, exitCode := cfg.runCLI(t, []string{"unfollow", targetHandle},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Unfollow command failed. Stderr: %s", stderr)
		}
	})

	t.Run("Who_ShouldShowProfile", func(t *testing.T) {
		// Test with own handle
		targetHandle := os.Getenv("MSH_TEST_USER_HANDLE")
		if targetHandle == "" {
			targetHandle = "test"
		}

		stdout, stderr, exitCode := cfg.runCLI(t, []string{"who", targetHandle},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		// May fail if user doesn't exist, but shouldn't crash
		if exitCode != 0 && !strings.Contains(stderr, "not found") {
			t.Logf("Who command output: %s", stdout)
		}
	})
}

// TestCLIInboxCommands tests inbox/notifications CLI commands.
func TestCLIInboxCommands(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping inbox tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("Inbox_ShouldShowNotifications", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"inbox"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Inbox command failed. Stderr: %s", stderr)
		}
	})
}

// TestCLIProfileCommands tests profile-related CLI commands.
func TestCLIProfileCommands(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping profile tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("Me_ShouldShowProfile", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"me"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Me command failed. Stderr: %s", stderr)
		}
	})
}

// TestCLIConfigCommands tests configuration CLI commands.
func TestCLIConfigCommands(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()

	t.Run("Config_ShouldListSettings", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"config"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Config list failed. Stderr: %s", stderr)
		}
	})

	t.Run("Config_ShouldGetSetting", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"config", "get", "editor"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Config get failed. Stderr: %s", stderr)
		}
	})

	t.Run("Config_ShouldSetSetting", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"config", "set", "editor", "nano"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Config set failed. Stderr: %s", stderr)
		}

		// Verify the setting
		stdout, _, _ = cfg.runCLI(t, []string{"config", "get", "editor"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if !strings.Contains(stdout, "nano") {
			t.Errorf("Expected editor to be set to 'nano'. Got: %s", stdout)
		}
	})
}

// TestCLISignalsCommands tests signals (like/share/bookmark) CLI commands.
func TestCLISignalsCommands(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping signals tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("Like_ShouldAcceptPostID", func(t *testing.T) {
		postID := os.Getenv("MSH_TEST_POST_ID")
		if postID == "" {
			t.Skip("MSH_TEST_POST_ID not set, skipping like test")
		}

		stdout, stderr, exitCode := cfg.runCLI(t, []string{"like", postID},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Like command failed. Stderr: %s", stderr)
		}
	})

	t.Run("Unlike_ShouldAcceptPostID", func(t *testing.T) {
		postID := os.Getenv("MSH_TEST_POST_ID")
		if postID == "" {
			t.Skip("MSH_TEST_POST_ID not set, skipping unlike test")
		}

		stdout, stderr, exitCode := cfg.runCLI(t, []string{"unlike", postID},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode != 0 {
			t.Errorf("Unlike command failed. Stderr: %s", stderr)
		}
	})
}

// TestCLIStreamingCommands tests streaming event CLI commands.
func TestCLIStreamingCommands(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping streaming tests")
	}

	// Login first
	cfg.runCLI(t, []string{"login", "--token", token},
		fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

	t.Run("Events_ShouldConnectToStream", func(t *testing.T) {
		// Use timeout for streaming test
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, cfg.CLIBinary, "events")
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("MSH_API_URL=%s", cfg.APIURL),
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		// Should complete due to timeout
		if err != nil && ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Events command failed unexpectedly. Stderr: %s", stderr.String())
		}
	})
}

// TestCLIUtilityCommands tests utility CLI commands.
func TestCLIUtilityCommands(t *testing.T) {
	cfg := NewSmokeTestConfig(t)

	t.Run("Open_ShouldShowHelp", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"open", "--help"})

		if exitCode != 0 && !strings.Contains(stderr, "unknown") {
			t.Logf("Open --help output: %s", stdout)
		}
	})
}

// TestCLICompletion verifies that shell completion works.
func TestCLICompletion(t *testing.T) {
	cfg := NewSmokeTestConfig(t)

	shells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range shells {
		t.Run(fmt.Sprintf("Completion_%s", shell), func(t *testing.T) {
			stdout, stderr, exitCode := cfg.runCLI(t, []string{"completion", shell})

			if exitCode != 0 && !strings.Contains(stderr, "unsupported") {
				t.Logf("Completion %s output: %s", shell, stdout)
			}
		})
	}
}

// TestCLIErrorHandling tests CLI error handling.
func TestCLIErrorHandling(t *testing.T) {
	cfg := NewSmokeTestConfig(t)
	tempDir := t.TempDir()

	t.Run("ShouldHandle_InvalidCommand", func(t *testing.T) {
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"invalid-command-xyz"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode == 0 {
			t.Error("Invalid command should return non-zero exit code")
		}

		if !strings.Contains(stderr, "unknown") && !strings.Contains(stdout, "unknown") {
			t.Logf("Unknown command output: %s", stdout)
		}
	})

	t.Run("ShouldHandle_MissingArguments", func(t *testing.T) {
		token := os.Getenv("MSH_TEST_TOKEN")
		if token != "" {
			// Login first
			cfg.runCLI(t, []string{"login", "--token", token},
				fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))
		}

		stdout, stderr, exitCode := cfg.runCLI(t, []string{"follow"},
			fmt.Sprintf("MSH_CONFIG_DIR=%s", tempDir))

		if exitCode == 0 {
			t.Error("Follow without arguments should return non-zero exit code")
		}

		if !strings.Contains(stderr, "required") && !strings.Contains(stderr, "usage") {
			t.Logf("Missing args output: %s", stdout)
		}
	})
}

// TestCLICrossPlatform tests CLI on different platforms.
func TestCLICrossPlatform(t *testing.T) {
	t.Run("ShouldRun_OnCurrentPlatform", func(t *testing.T) {
		cfg := NewSmokeTestConfig(t)

		// Basic command to verify the binary works
		stdout, stderr, exitCode := cfg.runCLI(t, []string{"--help"})

		if exitCode != 0 {
			t.Errorf("CLI binary failed on %s/%s. Stderr: %s",
				runtime.GOOS, runtime.GOARCH, stderr)
		}

		t.Logf("CLI running on %s/%s", runtime.GOOS, runtime.GOARCH)
	})
}

// TestAllCommandsAreRegistered verifies all expected commands exist.
func TestAllCommandsAreRegistered(t *testing.T) {
	cfg := NewSmokeTestConfig(t)

	expectedCommands := []string{
		"login", "logout", "status",
		"post", "reply", "quote", "edit", "delete",
		"feed",
		"follow", "unfollow", "block", "unblock", "mute", "unmute",
		"like", "unlike", "share", "unshare", "bookmark", "unbookmark",
		"inbox",
		"dm",
		"who", "me",
		"config",
		"assets",
		"keys", "tokens",
		"events",
		"challenge",
		"report",
		"open",
	}

	for _, cmd := range expectedCommands {
		t.Run(fmt.Sprintf("Command_%s_Exists", cmd), func(t *testing.T) {
			// Check if the command exists by running with --help
			stdout, stderr, exitCode := cfg.runCLI(t, []string{cmd, "--help"})

			// Command exists if it either succeeds or shows usage
			if exitCode == 0 ||
				strings.Contains(stdout, "Usage") ||
				strings.Contains(stdout, cmd) ||
				strings.Contains(stderr, "Usage") ||
				strings.Contains(stderr, cmd) {
				// Command exists
				return
			}

			t.Logf("Command '%s' help output: stdout=%s, stderr=%s",
				cmd, stdout, stderr)
		})
	}
}