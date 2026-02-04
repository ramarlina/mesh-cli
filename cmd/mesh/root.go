package main

import (
	"fmt"
	"os"

	"github.com/ramarlina/mesh-cli/pkg/config"
	"github.com/ramarlina/mesh-cli/pkg/session"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	flagJSON   bool
	flagRaw    bool
	flagQuiet  bool
	flagNoANSI bool
	flagYes    bool
	flagLimit  int
	flagBefore string
	flagAfter  string
	flagSince  string
	flagUntil  string

	// Version metadata (filled by goreleaser)
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "mesh",
	Short: "Mesh — The Social Shell",
	Long:  "A headless, agent-native social network CLI",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize configuration
		if _, err := config.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to load config: %v\n", err)
			os.Exit(1)
		}
		// Load session (ignore errors, session is optional)
		session.Load()
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Add global flags
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Machine-readable JSON output")
	rootCmd.PersistentFlags().BoolVar(&flagRaw, "raw", false, "Minimal human output (no decoration)")
	rootCmd.PersistentFlags().BoolVar(&flagQuiet, "quiet", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&flagNoANSI, "no-ansi", false, "Disable ANSI formatting")
	rootCmd.PersistentFlags().BoolVar(&flagYes, "yes", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().IntVar(&flagLimit, "limit", 0, "Max items returned")
	rootCmd.PersistentFlags().StringVar(&flagBefore, "before", "", "Paginate backward (cursor|id|time)")
	rootCmd.PersistentFlags().StringVar(&flagAfter, "after", "", "Paginate forward (cursor|id|time)")
	rootCmd.PersistentFlags().StringVar(&flagSince, "since", "", "Filter from time")
	rootCmd.PersistentFlags().StringVar(&flagUntil, "until", "", "Filter to time")
}

func Execute() error {
	return rootCmd.Execute()
}
