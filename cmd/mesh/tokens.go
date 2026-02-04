package main

import (
	"fmt"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/config"
	"github.com/ramarlina/mesh-cli/pkg/session"
	"github.com/spf13/cobra"
)

var (
	flagTokenName    string
	flagTokenExpires string
)

func init() {
	rootCmd.AddCommand(tokensCmd)

	tokensCmd.AddCommand(tokensCreateCmd)
	tokensCmd.AddCommand(tokensLsCmd)
	tokensCmd.AddCommand(tokensRevokeCmd)

	tokensCreateCmd.Flags().StringVar(&flagTokenName, "name", "", "Display name for the token (required)")
	tokensCreateCmd.MarkFlagRequired("name")
	tokensCreateCmd.Flags().StringVar(&flagTokenExpires, "expires", "", "Expiration duration (e.g., 1h, 7d, 30d)")
}

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Manage API tokens",
	Long:  "Create and manage long-lived API tokens for integrations",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var tokensCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create API token",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		// Must be authenticated
		token := session.GetToken()
		if token == "" {
			return out.Error(fmt.Errorf("not authenticated: run 'mesh login' first"))
		}

		c := client.New(config.GetAPIUrl(), client.WithToken(token))

		apiToken, err := c.CreateToken(&client.CreateTokenRequest{
			Name:    flagTokenName,
			Expires: flagTokenExpires,
		})
		if err != nil {
			return out.Error(fmt.Errorf("create token: %w", err))
		}

		if out.IsJSON() {
			return out.Success(apiToken)
		}

		out.Printf("✓ Token created: %s\n", apiToken.Name)
		out.Println()
		out.Printf("Token: %s\n", apiToken.Token)
		out.Println()
		out.Println("⚠️  Save this token securely — it won't be shown again!")

		if apiToken.ExpiresAt != nil {
			out.Printf("Expires: %s\n", apiToken.ExpiresAt.Format("2006-01-02 15:04:05"))
		}

		return nil
	},
}

var tokensLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List API tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		// Must be authenticated
		token := session.GetToken()
		if token == "" {
			return out.Error(fmt.Errorf("not authenticated: run 'mesh login' first"))
		}

		c := client.New(config.GetAPIUrl(), client.WithToken(token))

		tokens, err := c.ListTokens()
		if err != nil {
			return out.Error(fmt.Errorf("list tokens: %w", err))
		}

		if out.IsJSON() {
			return out.Success(tokens)
		}

		if len(tokens) == 0 {
			out.Println("No API tokens")
			return nil
		}

		headers := []string{"Prefix", "Name", "Expires", "Created"}
		rows := [][]string{}

		for _, t := range tokens {
			expires := "-"
			if t.ExpiresAt != nil {
				expires = t.ExpiresAt.Format("2006-01-02")
			}

			rows = append(rows, []string{
				t.Prefix,
				t.Name,
				expires,
				t.CreatedAt.Format("2006-01-02"),
			})
		}

		return out.Table(headers, rows)
	},
}

var tokensRevokeCmd = &cobra.Command{
	Use:   "revoke <token_prefix>",
	Short: "Revoke API token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		// Must be authenticated
		token := session.GetToken()
		if token == "" {
			return out.Error(fmt.Errorf("not authenticated: run 'mesh login' first"))
		}

		prefix := args[0]

		if !flagYes && !out.IsJSON() {
			fmt.Printf("Revoke token %s? (y/N): ", prefix)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				return nil
			}
		}

		c := client.New(config.GetAPIUrl(), client.WithToken(token))

		if err := c.RevokeToken(prefix); err != nil {
			return out.Error(fmt.Errorf("revoke token: %w", err))
		}

		if out.IsJSON() {
			return out.Success(map[string]bool{"revoked": true})
		}

		out.Printf("✓ Token revoked: %s\n", prefix)
		return nil
	},
}
