package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/spf13/cobra"
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "View notifications",
	Long:  "Display your notification inbox",
	Run: func(cmd *cobra.Command, args []string) {
		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		notifications, cursor, err := c.ListNotifications("", flagLimit, flagBefore, flagAfter)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if len(notifications) == 0 {
			if !flagQuiet {
				out.Println("No notifications")
			}
			return
		}

		if flagJSON {
			result := map[string]interface{}{
				"notifications": notifications,
				"cursor":        cursor,
			}
			out.Success(result)
		} else {
			for i, notif := range notifications {
				renderNotification(out, notif)
				if i < len(notifications)-1 {
					out.Println()
				}
			}
			if cursor != "" && !flagQuiet {
				out.Printf("\nNext page: --after %s\n", cursor)
			}
		}
	},
}

var inboxMentionsCmd = &cobra.Command{
	Use:   "mentions",
	Short: "View mention notifications",
	Long:  "Display notifications for mentions",
	Run: func(cmd *cobra.Command, args []string) {
		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		notifications, cursor, err := c.ListNotifications("mention", flagLimit, flagBefore, flagAfter)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if len(notifications) == 0 {
			if !flagQuiet {
				out.Println("No mentions")
			}
			return
		}

		if flagJSON {
			result := map[string]interface{}{
				"notifications": notifications,
				"cursor":        cursor,
			}
			out.Success(result)
		} else {
			for i, notif := range notifications {
				renderNotification(out, notif)
				if i < len(notifications)-1 {
					out.Println()
				}
			}
			if cursor != "" && !flagQuiet {
				out.Printf("\nNext page: --after %s\n", cursor)
			}
		}
	},
}

var inboxDMsCmd = &cobra.Command{
	Use:   "dms",
	Short: "View DM notifications",
	Long:  "Display notifications for direct messages",
	Run: func(cmd *cobra.Command, args []string) {
		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		notifications, cursor, err := c.ListNotifications("dm", flagLimit, flagBefore, flagAfter)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if len(notifications) == 0 {
			if !flagQuiet {
				out.Println("No DM notifications")
			}
			return
		}

		if flagJSON {
			result := map[string]interface{}{
				"notifications": notifications,
				"cursor":        cursor,
			}
			out.Success(result)
		} else {
			for i, notif := range notifications {
				renderNotification(out, notif)
				if i < len(notifications)-1 {
					out.Println()
				}
			}
			if cursor != "" && !flagQuiet {
				out.Printf("\nNext page: --after %s\n", cursor)
			}
		}
	},
}

var inboxReadCmd = &cobra.Command{
	Use:   "read [id...]",
	Short: "Mark notifications as read",
	Long:  "Mark specific notifications or all notifications as read",
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		req := &client.MarkNotificationsReadRequest{
			All: all,
			IDs: args,
		}

		err := c.MarkNotificationsRead(req)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "marked_read"})
		} else if !flagQuiet {
			if all {
				out.Println("✓ Marked all notifications as read")
			} else {
				out.Printf("✓ Marked %d notification(s) as read\n", len(args))
			}
		}
	},
}

var inboxClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all notifications",
	Long:  "Permanently delete all notifications",
	Run: func(cmd *cobra.Command, args []string) {
		// Confirm unless --yes is set
		if !flagYes {
			fmt.Print("Clear all notifications? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Cancelled")
				return
			}
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		err := c.ClearNotifications()
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "cleared"})
		} else if !flagQuiet {
			out.Println("✓ Cleared all notifications")
		}
	},
}

func renderNotification(out *output.Printer, notif *client.Notification) {
	if out.IsJSON() {
		data, _ := json.Marshal(notif)
		out.Print(string(data))
		return
	}

	if out.IsRaw() {
		out.Printf("%s: %s\n", notif.Type, notif.ID)
		return
	}

	readStatus := " "
	if !notif.Read {
		readStatus = "●"
	}

	actor := "system"
	if notif.Actor != nil {
		if notif.Actor.Name != "" {
			actor = fmt.Sprintf("%s (@%s)", notif.Actor.Name, notif.Actor.Handle)
		} else {
			actor = fmt.Sprintf("@%s", notif.Actor.Handle)
		}
	}

	out.Printf("%s %s • %s • %s\n", readStatus, notif.ID, notif.Type, notif.CreatedAt.Format("2006-01-02 15:04"))

	switch notif.Type {
	case "mention":
		out.Printf("  %s mentioned you\n", actor)
		if notif.TargetID != "" {
			out.Printf("  Post: %s\n", notif.TargetID)
		}
	case "follow":
		out.Printf("  %s followed you\n", actor)
	case "like":
		out.Printf("  %s liked your post\n", actor)
		if notif.TargetID != "" {
			out.Printf("  Post: %s\n", notif.TargetID)
		}
	case "share":
		out.Printf("  %s shared your post\n", actor)
		if notif.TargetID != "" {
			out.Printf("  Post: %s\n", notif.TargetID)
		}
	case "reply":
		out.Printf("  %s replied to your post\n", actor)
		if notif.TargetID != "" {
			out.Printf("  Post: %s\n", notif.TargetID)
		}
	case "dm":
		out.Printf("  New DM from %s\n", actor)
		if data, ok := notif.Data["preview"].(string); ok && data != "" {
			out.Printf("  Preview: %s\n", data)
		} else {
			out.Printf("  Preview: [Encrypted]\n")
		}
	default:
		out.Printf("  Actor: %s\n", actor)
		if notif.TargetID != "" {
			out.Printf("  Target: %s\n", notif.TargetID)
		}
	}
}

func init() {
	rootCmd.AddCommand(inboxCmd)
	inboxCmd.AddCommand(inboxMentionsCmd)
	inboxCmd.AddCommand(inboxDMsCmd)
	inboxCmd.AddCommand(inboxReadCmd)
	inboxCmd.AddCommand(inboxClearCmd)

	inboxReadCmd.Flags().Bool("all", false, "Mark all notifications as read")
}
