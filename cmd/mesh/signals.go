package main

import (
	"fmt"
	"os"

	"github.com/ramarlina/mesh-cli/pkg/context"
	"github.com/spf13/cobra"
)

var likeCmd = &cobra.Command{
	Use:   "like <p_id|this>",
	Short: "Like a post",
	Long:  "Express appreciation for a post",
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

		err = c.LikePost(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "liked", "post": id})
		} else if !flagQuiet {
			out.Printf("✓ Liked: %s\n", id)
		}
	},
}

var unlikeCmd = &cobra.Command{
	Use:   "unlike <p_id|this>",
	Short: "Unlike a post",
	Long:  "Remove your like from a post",
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

		err = c.UnlikePost(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "unliked", "post": id})
		} else if !flagQuiet {
			out.Printf("✓ Unliked: %s\n", id)
		}
	},
}

var shareCmd = &cobra.Command{
	Use:   "share <p_id|this>",
	Short: "Share a post",
	Long:  "Share a post to your followers",
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

		err = c.SharePost(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "shared", "post": id})
		} else if !flagQuiet {
			out.Printf("✓ Shared: %s\n", id)
		}
	},
}

var bookmarkCmd = &cobra.Command{
	Use:   "bookmark <p_id|this>",
	Short: "Bookmark a post",
	Long:  "Save a post to your bookmarks for later",
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

		err = c.BookmarkPost(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "bookmarked", "post": id})
		} else if !flagQuiet {
			out.Printf("✓ Bookmarked: %s\n", id)
		}
	},
}

var unbookmarkCmd = &cobra.Command{
	Use:   "unbookmark <p_id|this>",
	Short: "Remove bookmark from a post",
	Long:  "Remove a post from your bookmarks",
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

		err = c.UnbookmarkPost(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "unbookmarked", "post": id})
		} else if !flagQuiet {
			out.Printf("✓ Unbookmarked: %s\n", id)
		}
	},
}

func init() {
	rootCmd.AddCommand(likeCmd)
	rootCmd.AddCommand(unlikeCmd)
	rootCmd.AddCommand(shareCmd)
	rootCmd.AddCommand(bookmarkCmd)
	rootCmd.AddCommand(unbookmarkCmd)
}
