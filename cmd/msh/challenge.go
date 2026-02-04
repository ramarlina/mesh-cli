package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ramarlina/mesh-cli/pkg/api"
	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/spf13/cobra"
)

var solveCmd = &cobra.Command{
	Use:   "solve <ch_id> <answer>",
	Short: "Solve a challenge",
	Long:  "Submit an answer to a pending challenge",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		challengeID := args[0]
		answer := args[1]

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		req := &client.SolveRequest{
			Answer: answer,
		}

		post, err := c.SolveChallenge(challengeID, req)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]interface{}{
				"status": "solved",
				"post":   post,
			})
		} else if !flagQuiet {
			out.Printf("✓ Challenge solved\n")
			out.Printf("✓ Posted: %s\n", post.ID)
		}
	},
}

var challengeCmd = &cobra.Command{
	Use:   "challenge [ch_id]",
	Short: "Show challenge details",
	Long:  "Display details for a specific or current challenge",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		if len(args) > 0 {
			// Show specific challenge
			challengeID := args[0]
			challenge, err := c.GetChallengeByID(challengeID)
			if err != nil {
				out.Error(err)
				os.Exit(1)
			}

			if flagJSON {
				out.Success(challenge)
			} else {
				renderChallenge(out, challenge)
			}
		} else {
			// Show pending challenges
			challenges, err := c.ListChallenges()
			if err != nil {
				out.Error(err)
				os.Exit(1)
			}

			if len(challenges) == 0 {
				if !flagQuiet {
					out.Println("No pending challenges")
				}
				return
			}

			if flagJSON {
				out.Success(map[string]interface{}{"challenges": challenges})
			} else {
				for i, ch := range challenges {
					renderChallenge(out, ch)
					if i < len(challenges)-1 {
						out.Println()
					}
				}
			}
		}
	},
}

func renderChallenge(out *output.Printer, ch *client.Challenge) {
	if out.IsJSON() {
		data, _ := json.Marshal(ch)
		out.Print(string(data))
		return
	}

	out.Printf("Challenge: %s\n", ch.ID)
	out.Printf("Type: %s\n", ch.Type)
	out.Printf("Description: %s\n", ch.Description)

	if len(ch.Data) > 0 {
		out.Println("\nDetails:")
		for k, v := range ch.Data {
			out.Printf("  %s: %v\n", k, v)
		}
	}

	out.Printf("\nExpires: %s\n", ch.ExpiresAt.Format("2006-01-02 15:04"))
}

// handleChallengeInteractive handles a challenge interactively in the terminal
func handleChallengeInteractive(c *client.Client, out *output.Printer, apiErr *api.Error) bool {
	if out.IsJSON() {
		// In JSON mode, don't handle interactively
		return false
	}

	// Extract challenge from error details
	if apiErr.Details == nil {
		out.Error(fmt.Errorf("challenge required but no details provided"))
		return false
	}

	challengeData, ok := apiErr.Details["challenge"].(map[string]interface{})
	if !ok {
		out.Error(fmt.Errorf("challenge required but format invalid"))
		return false
	}

	// Extract challenge ID
	var challengeID int64
	if id, ok := challengeData["id"].(float64); ok {
		challengeID = int64(id)
	} else {
		out.Error(fmt.Errorf("challenge id not found"))
		return false
	}

	// Extract payload (contains the question)
	payload, _ := challengeData["payload"].(string)
	challengeType, _ := challengeData["type"].(string)
	difficulty, _ := challengeData["difficulty"].(string)

	// Display challenge
	out.Println("\n⚡ Challenge required")
	out.Printf("   Type: %s (%s)\n", challengeType, difficulty)

	// Parse and display the payload
	var payloadData map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &payloadData); err == nil {
		// For arithmetic challenges
		if a, aOk := payloadData["a"]; aOk {
			b := payloadData["b"]
			op := payloadData["op"]
			out.Printf("   Problem: %v %v %v = ?\n", a, op, b)
		} else {
			out.Printf("   Payload: %s\n", payload)
		}
	} else {
		out.Printf("   Payload: %s\n", payload)
	}

	out.Println()

	// Prompt for answer
	reader := bufio.NewReader(os.Stdin)
	out.Print("> ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		out.Error(fmt.Errorf("failed to read answer: %w", err))
		return false
	}

	answer = strings.TrimSpace(answer)
	if answer == "" {
		out.Error(fmt.Errorf("answer cannot be empty"))
		return false
	}

	// Submit answer via verify endpoint
	verifyResp, err := c.VerifyChallenge(challengeID, answer)
	if err != nil {
		out.Error(fmt.Errorf("challenge failed: %w", err))
		return false
	}

	if !verifyResp.Valid {
		out.Error(fmt.Errorf("wrong answer, try again"))
		return false
	}

	// Store the POI token for subsequent requests
	c.SetPOIToken(verifyResp.Token)

	out.Println("✓ Challenge passed!")
	return true
}

func init() {
	rootCmd.AddCommand(solveCmd)
	rootCmd.AddCommand(challengeCmd)
}
