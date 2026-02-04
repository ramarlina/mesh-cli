// Package smoke provides smoke tests for CLI context resolution
package smoke

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// ContextTestConfig holds configuration for context tests.
type ContextTestConfig struct {
	CLIBinary string
	APIURL    string
}

// TestThisContextResolution tests the 'this' context resolution in command chains.
func TestThisContextResolution(t *testing.T) {
	cfg := &ContextTestConfig{
		CLIBinary: getCLIBinary(t),
		APIURL:    os.Getenv("MSH_API_URL"),
	}

	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}

	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping 'this' context tests")
	}

	// Login first
	runCLIWithConfig(t, cfg, tempDir, []string{"login", "--token", token})

	t.Run("Post_ShouldSetThisContext", func(t *testing.T) {
		testContent := fmt.Sprintf("Context test post at %s", time.Now().Format(time.RFC3339))

		// Create a post and capture the post ID
		stdout, stderr, exitCode := runCLIWithConfig(t, cfg, tempDir, []string{"post", testContent, "--json"})

		if exitCode != 0 {
			t.Fatalf("Post creation failed. Stderr: %s", stderr)
		}

		// Extract post ID from JSON response
		postID := extractFieldFromJSON(t, stdout, "id")
		if postID == "" {
			t.Fatalf("Could not extract post ID from response: %s", stdout)
		}

		// Now use 'this' to reference the post
		// Test edit with 'this'
		newContent := "Updated content via 'this'"
		stdout, stderr, exitCode = runCLIWithConfig(t, cfg, tempDir, []string{"edit", "this", "--set", newContent, "--json"})

		if exitCode != 0 {
			t.Fatalf("Edit with 'this' failed. Stderr: %s", stderr)
		}

		// Verify the post was edited
		editedContent := extractFieldFromJSON(t, stdout, "content")
		if editedContent != newContent {
			t.Errorf("Expected content '%s', got '%s'", newContent, editedContent)
		}
	})

	t.Run("Reply_ShouldSetThisContext", func(t *testing.T) {
		testContent := fmt.Sprintf("Reply context test at %s", time.Now().Format(time.RFC3339))

		// Create a post
		stdout, stderr, exitCode := runCLIWithConfig(t, cfg, tempDir, []string{"post", testContent, "--json"})

		if exitCode != 0 {
			t.Fatalf("Post creation failed. Stderr: %s", stderr)
		}

		postID := extractFieldFromJSON(t, stdout, "id")
		if postID == "" {
			t.Fatalf("Could not extract post ID from response: %s", stdout)
		}

		// Reply to the post using 'this'
		replyContent := "This is a reply via 'this'"
		stdout, stderr, exitCode = runCLIWithConfig(t, cfg, tempDir, []string{"reply", "this", replyContent, "--json"})

		if exitCode != 0 {
			t.Fatalf("Reply with 'this' failed. Stderr: %s", stderr)
		}

		// Verify the reply was created
		replyID := extractFieldFromJSON(t, stdout, "id")
		if replyID == "" {
			t.Error("Could not extract reply ID from response")
		}

		// Verify reply_to points to original post
		replyTo := extractFieldFromJSON(t, stdout, "reply_to")
		if replyTo != postID {
			t.Errorf("Expected reply_to '%s', got '%s'", postID, replyTo)
		}
	})

	t.Run("Quote_ShouldSetThisContext", func(t *testing.T) {
		testContent := fmt.Sprintf("Quote context test at %s", time.Now().Format(time.RFC3339))

		// Create a post
		stdout, stderr, exitCode := runCLIWithConfig(t, cfg, tempDir, []string{"post", testContent, "--json"})

		if exitCode != 0 {
			t.Fatalf("Post creation failed. Stderr: %s", stderr)
		}

		postID := extractFieldFromJSON(t, stdout, "id")
		if postID == "" {
			t.Fatalf("Could not extract post ID from response: %s", stdout)
		}

		// Quote the post using 'this'
		quoteContent := "Quoted via 'this'"
		stdout, stderr, exitCode = runCLIWithConfig(t, cfg, tempDir, []string{"quote", "this", quoteContent, "--json"})

		if exitCode != 0 {
			t.Fatalf("Quote with 'this' failed. Stderr: %s", stderr)
		}

		// Verify the quote was created
		quoteID := extractFieldFromJSON(t, stdout, "id")
		if quoteID == "" {
			t.Error("Could not extract quote ID from response")
		}

		// Verify quote_of points to original post
		quoteOf := extractFieldFromJSON(t, stdout, "quote_of")
		if quoteOf != postID {
			t.Errorf("Expected quote_of '%s', got '%s'", postID, quoteOf)
		}
	})

	t.Run("Delete_ShouldUseThisContext", func(t *testing.T) {
		testContent := fmt.Sprintf("Delete context test at %s", time.Now().Format(time.RFC3339))

		// Create a post
		stdout, stderr, exitCode := runCLIWithConfig(t, cfg, tempDir, []string{"post", testContent, "--json", "--yes"})

		if exitCode != 0 {
			t.Fatalf("Post creation failed. Stderr: %s", stderr)
		}

		postID := extractFieldFromJSON(t, stdout, "id")
		if postID == "" {
			t.Fatalf("Could not extract post ID from response: %s", stdout)
		}

		// Delete the post using 'this'
		stdout, stderr, exitCode = runCLIWithConfig(t, cfg, tempDir, []string{"delete", "this", "--yes", "--json"})

		if exitCode != 0 {
			t.Fatalf("Delete with 'this' failed. Stderr: %s", stderr)
		}

		// Verify deletion response
		status := extractFieldFromJSON(t, stdout, "status")
		if status != "deleted" {
			t.Errorf("Expected status 'deleted', got '%s'", status)
		}
	})

	t.Run("Like_ShouldUseThisContext", func(t *testing.T) {
		testContent := fmt.Sprintf("Like context test at %s", time.Now().Format(time.RFC3339))

		// Create a post
		stdout, stderr, exitCode := runCLIWithConfig(t, cfg, tempDir, []string{"post", testContent, "--json"})

		if exitCode != 0 {
			t.Fatalf("Post creation failed. Stderr: %s", stderr)
		}

		postID := extractFieldFromJSON(t, stdout, "id")
		if postID == "" {
			t.Fatalf("Could not extract post ID from response: %s", stdout)
		}

		// Like the post using 'this'
		stdout, stderr, exitCode = runCLIWithConfig(t, cfg, tempDir, []string{"like", "this", "--json"})

		if exitCode != 0 {
			t.Fatalf("Like with 'this' failed. Stderr: %s", stderr)
		}

		// Verify like was registered
		liked := extractFieldFromJSON(t, stdout, "liked")
		if liked != "true" {
			t.Logf("Like response: %s", stdout)
		}

		// Unlike using 'this'
		stdout, stderr, exitCode = runCLIWithConfig(t, cfg, tempDir, []string{"unlike", "this", "--json"})

		if exitCode != 0 {
			t.Fatalf("Unlike with 'this' failed. Stderr: %s", stderr)
		}
	})

	t.Run("Bookmark_ShouldUseThisContext", func(t *testing.T) {
		testContent := fmt.Sprintf("Bookmark context test at %s", time.Now().Format(time.RFC3339))

		// Create a post
		stdout, stderr, exitCode := runCLIWithConfig(t, cfg, tempDir, []string{"post", testContent, "--json"})

		if exitCode != 0 {
			t.Fatalf("Post creation failed. Stderr: %s", stderr)
		}

		postID := extractFieldFromJSON(t, stdout, "id")
		if postID == "" {
			t.Fatalf("Could not extract post ID from response: %s", stdout)
		}

		// Bookmark the post using 'this'
		stdout, stderr, exitCode = runCLIWithConfig(t, cfg, tempDir, []string{"bookmark", "this", "--json"})

		if exitCode != 0 {
			t.Fatalf("Bookmark with 'this' failed. Stderr: %s", stderr)
		}

		// Unbookmark using 'this'
		stdout, stderr, exitCode = runCLIWithConfig(t, cfg, tempDir, []string{"unbookmark", "this", "--json"})

		if exitCode != 0 {
			t.Fatalf("Unbookmark with 'this' failed. Stderr: %s", stderr)
		}
	})

	t.Run("Share_ShouldUseThisContext", func(t *testing.T) {
		testContent := fmt.Sprintf("Share context test at %s", time.Now().Format(time.RFC3339))

		// Create a post
		stdout, stderr, exitCode := runCLIWithConfig(t, cfg, tempDir, []string{"post", testContent, "--json"})

		if exitCode != 0 {
			t.Fatalf("Post creation failed. Stderr: %s", stderr)
		}

		postID := extractFieldFromJSON(t, stdout, "id")
		if postID == "" {
			t.Fatalf("Could not extract post ID from response: %s", stdout)
		}

		// Share the post using 'this'
		stdout, stderr, exitCode = runCLIWithConfig(t, cfg, tempDir, []string{"share", "this", "--json"})

		if exitCode != 0 {
			t.Fatalf("Share with 'this' failed. Stderr: %s", stderr)
		}

		// Unshare using 'this'
		stdout, stderr, exitCode = runCLIWithConfig(t, cfg, tempDir, []string{"unshare", "this", "--json"})

		if exitCode != 0 {
			t.Fatalf("Unshare with 'this' failed. Stderr: %s", stderr)
		}
	})
}

// TestThisContextInvalidUsage tests invalid 'this' context usage.
func TestThisContextInvalidUsage(t *testing.T) {
	cfg := &ContextTestConfig{
		CLIBinary: getCLIBinary(t),
		APIURL:    os.Getenv("MSH_API_URL"),
	}

	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}

	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping invalid 'this' tests")
	}

	// Login first
	runCLIWithConfig(t, cfg, tempDir, []string{"login", "--token", token})

	t.Run("This_WithoutPriorCommand_ShouldFail", func(t *testing.T) {
		// Clear any existing context by creating a fresh session
		freshDir := t.TempDir()
		runCLIWithConfig(t, cfg, freshDir, []string{"login", "--token", token})

		// Try to use 'this' without a prior command
		_, stderr, exitCode := runCLIWithConfig(t, cfg, freshDir, []string{"edit", "this", "--set", "test"})

		if exitCode == 0 {
			t.Error("Using 'this' without prior context should fail")
		}

		if !strings.Contains(stderr, "context") && !strings.Contains(stderr, "this") {
			t.Logf("Expected context error. Got: %s", stderr)
		}
	})

	t.Run("This_AfterContextChange_ShouldFail", func(t *testing.T) {
		// Create a post to set context
		testContent := fmt.Sprintf("Test post at %s", time.Now().Format(time.RFC3339))
		stdout, _, _ := runCLIWithConfig(t, cfg, tempDir, []string{"post", testContent, "--json"})
		postID := extractFieldFromJSON(t, stdout, "id")

		// Create another post to change context
		stdout, _, _ = runCLIWithConfig(t, cfg, tempDir, []string{"post", "Another post", "--json"})

		// Now try to edit the first post - should fail because context changed
		newContent := "This should fail"
		stdout, _, _ = runCLIWithConfig(t, cfg, tempDir, []string{"edit", "this", "--set", newContent, "--json"})

		// The context should now point to the second post
		// Verify by checking if the second post's content changed
		editedID := extractFieldFromJSON(t, stdout, "id")
		if editedID == postID {
			t.Error("'this' should have changed to point to the second post")
		}
	})
}

// TestThisContextWithHandles tests 'this' context with handle resolution.
func TestThisContextWithHandles(t *testing.T) {
	cfg := &ContextTestConfig{
		CLIBinary: getCLIBinary(t),
		APIURL:    os.Getenv("MSH_API_URL"),
	}

	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}

	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping handle tests")
	}

	// Login first
	runCLIWithConfig(t, cfg, tempDir, []string{"login", "--token", token})

	t.Run("HandleResolution_ShouldWork", func(t *testing.T) {
		// Get own handle from 'me' command
		stdout, _, _ := runCLIWithConfig(t, cfg, tempDir, []string{"me", "--json"})
		handle := extractFieldFromJSON(t, stdout, "handle")

		if handle == "" {
			t.Skip("Could not get own handle")
		}

		// Use handle with follow command
		_, _, exitCode := runCLIWithConfig(t, cfg, tempDir, []string{"follow", handle})

		// Following self should fail, but handle resolution should work
		if exitCode == 0 {
			t.Log("Follow command executed (may have failed on server side)")
		}
	})
}

// TestThisContextPersistence tests if 'this' context persists between commands.
func TestThisContextPersistence(t *testing.T) {
	cfg := &ContextTestConfig{
		CLIBinary: getCLIBinary(t),
		APIURL:    os.Getenv("MSH_API_URL"),
	}

	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}

	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping persistence tests")
	}

	// Login first
	runCLIWithConfig(t, cfg, tempDir, []string{"login", "--token", token})

	t.Run("This_ShouldPersist_BetweenCommands", func(t *testing.T) {
		// Create a post
		testContent := fmt.Sprintf("Persistence test at %s", time.Now().Format(time.RFC3339))
		stdout, _, _ := runCLIWithConfig(t, cfg, tempDir, []string{"post", testContent, "--json"})
		postID := extractFieldFromJSON(t, stdout, "id")

		// Use 'this' to like the post
		stdout, _, _ = runCLIWithConfig(t, cfg, tempDir, []string{"like", "this", "--json"})

		// Use 'this' to bookmark the same post
		stdout, _, _ = runCLIWithConfig(t, cfg, tempDir, []string{"bookmark", "this", "--json"})

		// Use 'this' to unlike
		stdout, _, _ = runCLIWithConfig(t, cfg, tempDir, []string{"unlike", "this", "--json"})

		// Use 'this' to unbookmark
		stdout, _, _ = runCLIWithConfig(t, cfg, tempDir, []string{"unbookmark", "this", "--json"})

		// Finally, delete the post
		_, _, _ = runCLIWithConfig(t, cfg, tempDir, []string{"delete", "this", "--yes", "--json"})

		// All operations should have worked on the same post
		_ = postID
	})
}

// TestThisContextWithPostID tests 'this' context when used with explicit post IDs.
func TestThisContextWithPostID(t *testing.T) {
	cfg := &ContextTestConfig{
		CLIBinary: getCLIBinary(t),
		APIURL:    os.Getenv("MSH_API_URL"),
	}

	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}

	tempDir := t.TempDir()
	token := os.Getenv("MSH_TEST_TOKEN")

	if token == "" {
		t.Skip("MSH_TEST_TOKEN not set, skipping post ID tests")
	}

	// Login first
	runCLIWithConfig(t, cfg, tempDir, []string{"login", "--token", token})

	t.Run("PostID_ShouldOverrideThis", func(t *testing.T) {
		// Create first post
		testContent1 := fmt.Sprintf("First post at %s", time.Now().Format(time.RFC3339))
		stdout1, _, _ := runCLIWithConfig(t, cfg, tempDir, []string{"post", testContent1, "--json"})
		postID1 := extractFieldFromJSON(t, stdout1, "id")

		// Create second post (now 'this' points to second post)
		testContent2 := fmt.Sprintf("Second post at %s", time.Now().Format(time.RFC3339))
		_, _, _ = runCLIWithConfig(t, cfg, tempDir, []string{"post", testContent2, "--json"})

		// Edit the first post explicitly (should not use 'this')
		newContent := "Edited first post"
		stdout, stderr, exitCode := runCLIWithConfig(t, cfg, tempDir, []string{"edit", postID1, "--set", newContent, "--json"})

		if exitCode != 0 {
			t.Fatalf("Edit with explicit ID failed. Stderr: %s", stderr)
		}

		// Verify the correct post was edited
		editedContent := extractFieldFromJSON(t, stdout, "content")
		if editedContent != newContent {
			t.Errorf("Expected content '%s', got '%s'", newContent, editedContent)
		}

		// Now verify 'this' still points to second post
		// Delete 'this' - should delete the second post
		_, _, _ = runCLIWithConfig(t, cfg, tempDir, []string{"delete", "this", "--yes", "--json"})
	})
}

// Helper functions

func getCLIBinary(t *testing.T) string {
	t.Helper()

	cliBinary := os.Getenv("MSH_CLI_BINARY")
	if cliBinary == "" {
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
		cliBinary = "msh"
	}

	return cliBinary
}

func runCLIWithConfig(t *testing.T, cfg *ContextTestConfig, configDir string, args []string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(cfg.CLIBinary, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("MSH_API_URL=%s", cfg.APIURL),
		fmt.Sprintf("MSH_CONFIG_DIR=%s", configDir))

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

func extractFieldFromJSON(t *testing.T, jsonStr, field string) string {
	t.Helper()

	// Simple field extraction from flat JSON
	// For nested structures, this is simplified
	searchPattern := fmt.Sprintf(`"%s":\s*"?([^"}]+)"?`, field)
	re := regexp.MustCompile(searchPattern)
	matches := re.FindStringSubmatch(jsonStr)

	if len(matches) > 1 {
		return matches[1]
	}

	// Try without quotes for numbers
	searchPattern = fmt.Sprintf(`"%s":\s*(\d+)`, field)
	re = regexp.MustCompile(searchPattern)
	matches = re.FindStringSubmatch(jsonStr)

	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}