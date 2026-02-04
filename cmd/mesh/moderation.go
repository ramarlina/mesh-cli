package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/context"
	"github.com/spf13/cobra"
)

var hideCmd = &cobra.Command{
	Use:   "hide <p_id|this>",
	Short: "Hide a post",
	Long:  "Hide a post from your feed",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		id, _, err := context.ResolveTarget(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		err = c.HidePost(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "hidden", "post": id})
		} else if !flagQuiet {
			out.Printf("✓ Hidden: %s\n", id)
		}
	},
}

var unhideCmd = &cobra.Command{
	Use:   "unhide <p_id>",
	Short: "Unhide a post",
	Long:  "Restore visibility of a hidden post",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		err := c.UnhidePost(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "unhidden", "post": id})
		} else if !flagQuiet {
			out.Printf("✓ Unhidden: %s\n", id)
		}
	},
}

var reportCmd = &cobra.Command{
	Use:   "report <p_id|@user|this> --reason <reason>",
	Short: "Report content or user",
	Long:  "Submit a report for moderation review",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		reason, _ := cmd.Flags().GetString("reason")
		if reason == "" {
			fmt.Fprintf(os.Stderr, "error: --reason is required\n")
			os.Exit(1)
		}

		note, _ := cmd.Flags().GetString("note")

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		var targetType, targetID string

		if strings.HasPrefix(target, "@") {
			targetType = "user"
			targetID = strings.TrimPrefix(target, "@")
		} else {
			id, _, err := context.ResolveTarget(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			targetType = "post"
			targetID = id
		}

		req := &client.ReportRequest{
			TargetType: targetType,
			TargetID:   targetID,
			Reason:     reason,
			Note:       note,
		}

		err := c.Report(req)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]interface{}{
				"status":      "reported",
				"target_type": targetType,
				"target_id":   targetID,
			})
		} else if !flagQuiet {
			out.Printf("✓ Report submitted for %s: %s\n", targetType, targetID)
		}
	},
}

func init() {
	rootCmd.AddCommand(hideCmd)
	rootCmd.AddCommand(unhideCmd)
	rootCmd.AddCommand(reportCmd)

	reportCmd.Flags().String("reason", "", "Reason (spam|abuse|harassment|illegal|other)")
	reportCmd.Flags().String("note", "", "Additional notes")
}
