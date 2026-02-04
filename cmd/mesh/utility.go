package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/config"
	"github.com/ramarlina/mesh-cli/pkg/context"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/ramarlina/mesh-cli/pkg/session"
	"github.com/spf13/cobra"
)

var idCmd = &cobra.Command{
	Use:   "id",
	Short: "Print current context object ID",
	Long:  "Print the ID of the last rendered object from context",
	Run: func(cmd *cobra.Command, args []string) {
		id, err := context.GetID()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(id)
	},
}

var openCmd = &cobra.Command{
	Use:   "open [id|@handle|this]",
	Short: "Open canonical URL in browser",
	Long:  "Open the canonical URL for a post, asset, or user profile",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := "this"
		if len(args) > 0 {
			target = args[0]
		}

		id, _, err := context.ResolveTarget(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		url := buildCanonicalURL(id)

		// If --raw flag is set, just print the URL
		if flagRaw {
			fmt.Println(url)
			return
		}

		// Otherwise, open in browser
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to open browser: %v\n", err)
			fmt.Fprintf(os.Stderr, "URL: %s\n", url)
			os.Exit(1)
		}

		if !flagQuiet {
			fmt.Printf("Opened: %s\n", url)
		}
	},
}

var resolveCmd = &cobra.Command{
	Use:   "resolve <id|@handle>",
	Short: "Resolve identifier to full object",
	Long:  "Fetch and display full object data for a post, asset, or user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		// cfg, _ := config.Load()
		c := getClient()

		out := getOutputPrinter()

		// Determine what type of ID this is
		if strings.HasPrefix(target, "@") {
			// User handle
			user, err := c.GetUser(strings.TrimPrefix(target, "@"))
			if err != nil {
				out.Error(err)
				os.Exit(1)
			}
			out.Success(user)
		} else if strings.HasPrefix(target, "p_") {
			// Post ID
			post, err := c.GetPost(target)
			if err != nil {
				out.Error(err)
				os.Exit(1)
			}
			out.Success(post)
		} else if strings.HasPrefix(target, "as_") {
			// Asset ID
			asset, err := c.GetAsset(target)
			if err != nil {
				out.Error(err)
				os.Exit(1)
			}
			out.Success(asset)
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown identifier type: %s\n", target)
			os.Exit(1)
		}
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose installation",
	Long:  "Check CLI installation, configuration, and connectivity",
	Run: func(cmd *cobra.Command, args []string) {
		out := getOutputPrinter()

		if flagJSON {
			runDoctorJSON(out)
		} else {
			runDoctorHuman(out)
		}
	},
}

func buildCanonicalURL(id string) string {
	if strings.HasPrefix(id, "@") {
		return fmt.Sprintf("https://joinm.sh/%s", id)
	} else if strings.HasPrefix(id, "p_") {
		return fmt.Sprintf("https://joinm.sh/p/%s", id)
	} else if strings.HasPrefix(id, "as_") {
		return fmt.Sprintf("https://cdn.joinm.sh/%s", id)
	}
	return fmt.Sprintf("https://joinm.sh/%s", id)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

func runDoctorHuman(out *output.Printer) {
	out.Println("Mesh CLI Diagnostics")
	out.Println("====================")
	out.Println()

	// Check config
	out.Printf("Configuration:\n")
	cfg, err := config.Load()
	if err != nil {
		out.Printf("  ✗ Config: %v\n", err)
	} else {
		out.Printf("  ✓ Config loaded\n")
		out.Printf("    API URL: %s\n", cfg.APIUrl)
	}
	out.Println()

	// Check authentication
	out.Printf("Authentication:\n")
	token := session.GetToken()
	if token != "" {
		out.Printf("  ✓ Token present\n")

		// Try to verify token
		c := getClient()
		user, err := c.GetStatus()
		if err != nil {
			out.Printf("  ✗ Token validation failed: %v\n", err)
		} else {
			out.Printf("  ✓ Authenticated as: @%s (%s)\n", user.Handle, user.Name)
		}
	} else {
		out.Printf("  ✗ Not authenticated (run 'mesh login')\n")
	}
	out.Println()

	// Check connectivity
	out.Printf("Connectivity:\n")
	apiURL := config.GetAPIUrl()
	c := client.New(apiURL)
	err = c.Health()
	if err != nil {
		out.Printf("  ✗ Cannot reach server: %v\n", err)
	} else {
		out.Printf("  ✓ Server reachable at %s\n", apiURL)
	}
	out.Println()

	// Check context
	out.Printf("Context:\n")
	id, typ, err := context.Get()
	if err != nil {
		out.Printf("  ⓘ No active context\n")
	} else {
		out.Printf("  ✓ Current context: %s (%s)\n", id, typ)
	}
	out.Println()

	out.Println("✓ Diagnostics complete")
}

func runDoctorJSON(out *output.Printer) {
	result := make(map[string]interface{})

	// Check config
	cfg, err := config.Load()
	if err != nil {
		result["config"] = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	} else {
		result["config"] = map[string]interface{}{
			"status":  "ok",
			"api_url": cfg.APIUrl,
		}
	}

	// Check authentication
	token := session.GetToken()
	if token != "" {
		c := getClient()
		user, err := c.GetStatus()
		if err != nil {
			result["auth"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			result["auth"] = map[string]interface{}{
				"status": "ok",
				"user":   user,
			}
		}
	} else {
		result["auth"] = map[string]interface{}{
			"status": "not_authenticated",
		}
	}

	// Check connectivity
	apiURL := config.GetAPIUrl()
	c := client.New(apiURL)
	err = c.Health()
	if err != nil {
		result["connectivity"] = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	} else {
		result["connectivity"] = map[string]interface{}{
			"status": "ok",
		}
	}

	// Check context
	id, typ, err := context.Get()
	if err != nil {
		result["context"] = map[string]interface{}{
			"status": "none",
		}
	} else {
		result["context"] = map[string]interface{}{
			"status": "ok",
			"id":     id,
			"type":   typ,
		}
	}

	out.Success(result)
}

func init() {
	rootCmd.AddCommand(idCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(resolveCmd)
	rootCmd.AddCommand(doctorCmd)
}
