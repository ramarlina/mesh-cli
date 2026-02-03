package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var followCmd = &cobra.Command{
	Use:   "follow <@user>",
	Short: "Follow a user",
	Long:  "Subscribe to a user's posts",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handle := strings.TrimPrefix(args[0], "@")

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		err := c.FollowUser(handle)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "followed", "user": handle})
		} else if !flagQuiet {
			out.Printf("✓ Followed @%s\n", handle)
		}
	},
}

var unfollowCmd = &cobra.Command{
	Use:   "unfollow <@user>",
	Short: "Unfollow a user",
	Long:  "Unsubscribe from a user's posts",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handle := strings.TrimPrefix(args[0], "@")

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		err := c.UnfollowUser(handle)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "unfollowed", "user": handle})
		} else if !flagQuiet {
			out.Printf("✓ Unfollowed @%s\n", handle)
		}
	},
}

var blockCmd = &cobra.Command{
	Use:   "block <@user>",
	Short: "Block a user",
	Long:  "Sever relationship with user and hide their content",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handle := strings.TrimPrefix(args[0], "@")

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		err := c.BlockUser(handle)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "blocked", "user": handle})
		} else if !flagQuiet {
			out.Printf("✓ Blocked @%s\n", handle)
		}
	},
}

var unblockCmd = &cobra.Command{
	Use:   "unblock <@user>",
	Short: "Unblock a user",
	Long:  "Remove block from a user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handle := strings.TrimPrefix(args[0], "@")

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		err := c.UnblockUser(handle)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "unblocked", "user": handle})
		} else if !flagQuiet {
			out.Printf("✓ Unblocked @%s\n", handle)
		}
	},
}

var muteCmd = &cobra.Command{
	Use:   "mute <@user>",
	Short: "Mute a user",
	Long:  "Hide user's content without unfollowing",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handle := strings.TrimPrefix(args[0], "@")

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		err := c.MuteUser(handle)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "muted", "user": handle})
		} else if !flagQuiet {
			out.Printf("✓ Muted @%s\n", handle)
		}
	},
}

var unmuteCmd = &cobra.Command{
	Use:   "unmute <@user>",
	Short: "Unmute a user",
	Long:  "Remove mute from a user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handle := strings.TrimPrefix(args[0], "@")

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		err := c.UnmuteUser(handle)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "unmuted", "user": handle})
		} else if !flagQuiet {
			out.Printf("✓ Unmuted @%s\n", handle)
		}
	},
}

var followersCmd = &cobra.Command{
	Use:   "followers [@user]",
	Short: "List followers",
	Long:  "Show followers for a user (default: yourself)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handle := ""
		if len(args) > 0 {
			handle = strings.TrimPrefix(args[0], "@")
		} else {
			// Get current user
			// cfg, _ := config.Load()
			c := getClient()
			user, err := c.GetProfile()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			handle = user.Handle
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		users, cursor, err := c.GetFollowers(handle, flagLimit, flagBefore, flagAfter)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if len(users) == 0 {
			if !flagQuiet {
				out.Println("No followers")
			}
			return
		}

		if flagJSON {
			result := map[string]interface{}{
				"users":  users,
				"cursor": cursor,
			}
			out.Success(result)
		} else {
			for _, user := range users {
				renderUser(out, user)
			}
			if cursor != "" && !flagQuiet {
				out.Printf("\nNext page: --after %s\n", cursor)
			}
		}
	},
}

var followingCmd = &cobra.Command{
	Use:   "following [@user]",
	Short: "List following",
	Long:  "Show users that a user follows (default: yourself)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handle := ""
		if len(args) > 0 {
			handle = strings.TrimPrefix(args[0], "@")
		} else {
			// Get current user
			// cfg, _ := config.Load()
			c := getClient()
			user, err := c.GetProfile()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			handle = user.Handle
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		users, cursor, err := c.GetFollowing(handle, flagLimit, flagBefore, flagAfter)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if len(users) == 0 {
			if !flagQuiet {
				out.Println("Not following anyone")
			}
			return
		}

		if flagJSON {
			result := map[string]interface{}{
				"users":  users,
				"cursor": cursor,
			}
			out.Success(result)
		} else {
			for _, user := range users {
				renderUser(out, user)
			}
			if cursor != "" && !flagQuiet {
				out.Printf("\nNext page: --after %s\n", cursor)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(followCmd)
	rootCmd.AddCommand(unfollowCmd)
	rootCmd.AddCommand(blockCmd)
	rootCmd.AddCommand(unblockCmd)
	rootCmd.AddCommand(muteCmd)
	rootCmd.AddCommand(unmuteCmd)
	rootCmd.AddCommand(followersCmd)
	rootCmd.AddCommand(followingCmd)
}
