package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/config"
	"github.com/ramarlina/mesh-cli/pkg/models"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/ramarlina/mesh-cli/pkg/session"
	"github.com/spf13/cobra"
)

var (
	flagEditor bool
)

func init() {
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(whoisCmd)

	profileCmd.AddCommand(profileEditCmd)

	profileEditCmd.Flags().BoolVar(&flagEditor, "editor", false, "Open in $EDITOR")
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show your profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		// Must be authenticated
		token := session.GetToken()
		if token == "" {
			return out.Error(fmt.Errorf("not authenticated: run 'mesh login' first"))
		}

		c := client.New(config.GetAPIUrl(), client.WithToken(token))

		user, err := c.GetProfile()
		if err != nil {
			return out.Error(fmt.Errorf("get profile: %w", err))
		}

		if out.IsJSON() {
			return out.Success(user)
		}

		return printUser(out, user)
	},
}

var profileEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit your profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		// Must be authenticated
		token := session.GetToken()
		if token == "" {
			return out.Error(fmt.Errorf("not authenticated: run 'mesh login' first"))
		}

		c := client.New(config.GetAPIUrl(), client.WithToken(token))

		// Get current profile
		user, err := c.GetProfile()
		if err != nil {
			return out.Error(fmt.Errorf("get profile: %w", err))
		}

		if flagEditor {
			// TODO: Implement editor-based editing
			return out.Error(fmt.Errorf("--editor not yet implemented"))
		}

		// Interactive wizard
		reader := bufio.NewReader(os.Stdin)

		fmt.Printf("Name [%s]: ", user.Name)
		nameInput, _ := reader.ReadString('\n')
		nameInput = strings.TrimSpace(nameInput)

		fmt.Printf("Bio [%s]: ", user.Bio)
		bioInput, _ := reader.ReadString('\n')
		bioInput = strings.TrimSpace(bioInput)

		// Prepare update request
		req := &client.UpdateProfileRequest{}

		if nameInput != "" && nameInput != user.Name {
			req.Name = nameInput
		}

		if bioInput != "" && bioInput != user.Bio {
			req.Bio = bioInput
		}

		// If nothing changed, return
		if req.Name == "" && req.Bio == "" {
			out.Println("No changes made")
			return nil
		}

		// Update profile
		updatedUser, err := c.UpdateProfile(req)
		if err != nil {
			return out.Error(fmt.Errorf("update profile: %w", err))
		}

		if out.IsJSON() {
			return out.Success(updatedUser)
		}

		out.Println("✓ Profile updated")
		return printUser(out, updatedUser)
	},
}

var whoisCmd = &cobra.Command{
	Use:   "whois <@user|email>",
	Short: "View user profile by username or email",
	Long:  "Look up a user profile by @username or email address",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		// Must be authenticated
		token := session.GetToken()
		if token == "" {
			return out.Error(fmt.Errorf("not authenticated: run 'mesh login' first"))
		}

		identifier := args[0]
		// Remove @ prefix if present (for handles)
		if strings.HasPrefix(identifier, "@") {
			identifier = strings.TrimPrefix(identifier, "@")
		}

		c := client.New(config.GetAPIUrl(), client.WithToken(token))

		user, err := c.GetUser(identifier)
		if err != nil {
			return out.Error(fmt.Errorf("get user: %w", err))
		}

		if out.IsJSON() {
			return out.Success(user)
		}

		return printUser(out, user)
	},
}

func printUser(out *output.Printer, user *models.User) error {
	if out.IsRaw() {
		out.Printf("@%s\n", user.Handle)
		return nil
	}

	out.Printf("@%s\n", user.Handle)
	if user.Name != "" {
		out.Printf("Name: %s\n", user.Name)
	}
	if user.Bio != "" {
		out.Printf("Bio: %s\n", user.Bio)
	}
	out.Printf("ID: %s\n", user.ID)
	out.Printf("Joined: %s\n", user.CreatedAt.Format("2006-01-02"))

	return nil
}
