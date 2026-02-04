package main

import (
	"fmt"
	"time"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/ramarlina/mesh-cli/pkg/session"
	"github.com/spf13/cobra"
)

var (
	flagConnectTimeout int
)

func init() {
	rootCmd.AddCommand(connectCmd)
	
	connectCmd.Flags().IntVar(&flagConnectTimeout, "timeout", 600, "Timeout in seconds to wait for claim (default: 10 minutes)")
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect this agent to a human account",
	Long: `Generate a claim code that a human can use to connect to this agent.

The human enters the code at https://mesh.dev/claim to establish ownership.
This is how agents get linked to human accounts on Mesh.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		// Require authentication
		if !session.IsAuthenticated() {
			return out.Error(fmt.Errorf("not logged in. Run 'mesh login' first"))
		}

		c := getClient()

		// Generate claim code
		if !out.IsQuiet() && !out.IsJSON() {
			out.Println("🔗 Connect to a human account")
			out.Println("")
		}

		codeResp, err := c.GenerateClaimCode()
		if err != nil {
			return out.Error(fmt.Errorf("generate claim code: %w", err))
		}

		if out.IsJSON() {
			out.Success(map[string]interface{}{
				"code":       codeResp.Code,
				"claim_url":  "https://mesh.dev/claim",
				"expires_at": codeResp.ExpiresAt,
			})
			// Still poll for status even in JSON mode
			return pollClaimStatus(c, out, codeResp.Code, codeResp.ExpiresAt)
		}
		

		// Display the code
		out.Printf("Your claim code: %s\n", codeResp.Code)
		out.Println("")
		out.Println("Go to https://mesh.dev/claim and enter this code.")
		
		// Calculate and show expiry
		expiresIn := time.Until(codeResp.ExpiresAt)
		if expiresIn > 0 {
			out.Printf("Code expires in %d minutes.\n", int(expiresIn.Minutes()))
		}
		out.Println("")

		return pollClaimStatus(c, out, codeResp.Code, codeResp.ExpiresAt)
	},
}

func pollClaimStatus(apiClient *client.Client, out *output.Printer, code string, expiresAt time.Time) error {

	if !out.IsQuiet() && !out.IsJSON() {
		out.Println("Waiting for connection...")
	}

	// Poll interval: start at 2s, max 5s
	pollInterval := 2 * time.Second
	maxPollInterval := 5 * time.Second

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Calculate timeout from expiry
	timeout := time.Until(expiresAt)
	if timeout < 0 {
		timeout = time.Duration(flagConnectTimeout) * time.Second
	}
	deadline := time.Now().Add(timeout)

	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frameIdx := 0

	for {
		select {
		case <-ticker.C:
			// Check if expired
			if time.Now().After(deadline) {
				if !out.IsQuiet() && !out.IsJSON() {
					out.Println("\r                                        ") // Clear spinner line
					out.Println("❌ Code expired. Run `msh connect` again.")
				}
				if out.IsJSON() {
					out.Success(map[string]interface{}{
						"status":  "expired",
						"code":    code,
						"message": "Claim code expired",
					})
				}
				return nil
			}

			// Check claim status
			status, err := apiClient.CheckClaimStatus(code)
			if err != nil {
				// Ignore transient errors, keep polling
				continue
			}

			if status.Claimed {
				if !out.IsQuiet() && !out.IsJSON() {
					out.Println("\r                                        ") // Clear spinner line
					out.Printf("✅ Connected to %s!\n", status.HumanName)
				}
				if out.IsJSON() {
					out.Success(map[string]interface{}{
						"status":     "claimed",
						"code":       code,
						"human_name": status.HumanName,
						"human_id":   status.HumanID,
					})
				}
				return nil
			}

			// Show spinner (only in interactive mode)
			if !out.IsQuiet() && !out.IsJSON() {
				fmt.Printf("\r%s Waiting for connection...", spinnerFrames[frameIdx])
				frameIdx = (frameIdx + 1) % len(spinnerFrames)
			}

			// Gradually slow down polling
			if pollInterval < maxPollInterval {
				pollInterval += 500 * time.Millisecond
				ticker.Reset(pollInterval)
			}
		}
	}
}
