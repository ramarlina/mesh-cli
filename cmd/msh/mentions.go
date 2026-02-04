package main

import (
	"fmt"
	"os"

	"github.com/ramarlina/mesh-cli/pkg/context"
	"github.com/ramarlina/mesh-cli/pkg/session"
	"github.com/spf13/cobra"
)

var mentionsCmd = &cobra.Command{
	Use:   "mentions [@handle]",
	Short: "View posts mentioning you or another user",
	Long:  "Display posts that mention you or a specified user",
	Run: func(cmd *cobra.Command, args []string) {
		c := getClient()
		out := getOutputPrinter()

		// Determine whose mentions to view
		var handle string
		if len(args) > 0 && args[0] != "" {
			handle = args[0]
			// Strip @ prefix if present
			if len(handle) > 0 && handle[0] == '@' {
				handle = handle[1:]
			}
		} else {
			// Default to current user
			user := session.GetUser()
			if user == nil {
				out.Error(fmt.Errorf("not logged in - specify @handle or run 'msh auth'"))
				os.Exit(1)
			}
			handle = user.Handle
		}

		posts, cursor, err := c.GetUserMentions(handle, flagLimit, flagBefore, flagAfter)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if len(posts) == 0 {
			if !flagQuiet {
				out.Printf("No posts mentioning @%s\n", handle)
			}
			return
		}

		// Update context to the first post
		if len(posts) > 0 {
			context.Set(posts[0].ID, "post")
		}

		if flagJSON {
			result := map[string]interface{}{
				"posts":  posts,
				"cursor": cursor,
			}
			out.Success(result)
		} else {
			for i, post := range posts {
				renderPost(out, post)
				if i < len(posts)-1 {
					out.Println()
				}
			}
			if cursor != "" && !flagQuiet {
				out.Printf("\nNext page: --after %s\n", cursor)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(mentionsCmd)
}
