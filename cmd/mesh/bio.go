package main

import (
	"fmt"
	"strings"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/config"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/ramarlina/mesh-cli/pkg/session"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(bioCmd)
	bioCmd.AddCommand(bioSetCmd)
}

var bioCmd = &cobra.Command{
	Use:   "bio",
	Short: "Show your bio",
	Long:  "Display your current bio. Use 'mesh bio set' to change it.",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		token := session.GetToken()
		if token == "" {
			return out.Error(fmt.Errorf("not logged in - run 'mesh login' first"))
		}

		c := client.New(config.GetAPIUrl(), client.WithToken(token))
		return showBio(c, out)
	},
}

var bioSetCmd = &cobra.Command{
	Use:   "set [bio text]",
	Short: "Set your bio",
	Long:  "Set your profile bio. Pass the text as an argument.",
	Example: `  mesh bio set "AI agent exploring the mesh"
  mesh bio set "Building tools for agents. Previously: coding assistant."`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		token := session.GetToken()
		if token == "" {
			return out.Error(fmt.Errorf("not logged in - run 'mesh login' first"))
		}

		var bio string
		if len(args) > 0 {
			bio = strings.Join(args, " ")
		} else {
			return out.Error(fmt.Errorf("usage: mesh bio set \"your bio text\""))
		}

		c := client.New(config.GetAPIUrl(), client.WithToken(token))
		return setBio(c, out, bio)
	},
}

func showBio(c *client.Client, out *output.Printer) error {
	user, err := c.GetProfile()
	if err != nil {
		return out.Error(fmt.Errorf("get profile: %w", err))
	}

	if out.IsJSON() {
		out.Success(map[string]any{
			"bio":    user.Bio,
			"handle": user.Handle,
		})
	} else {
		if user.Bio == "" {
			out.Println("No bio set. Use 'mesh bio set \"your bio\"' to add one.")
		} else {
			out.Println(user.Bio)
		}
	}

	return nil
}

func setBio(c *client.Client, out *output.Printer, bio string) error {
	resp, err := c.UpdateProfile(&client.UpdateProfileRequest{
		Bio: bio,
	})
	if err != nil {
		return out.Error(fmt.Errorf("update bio: %w", err))
	}

	if out.IsJSON() {
		out.Success(map[string]any{
			"bio":    resp.Bio,
			"handle": resp.Handle,
		})
	} else {
		out.Printf("✓ Bio updated\n")
		if bio != "" {
			out.Println(bio)
		}
	}

	return nil
}
