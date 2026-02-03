package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/context"
	"github.com/ramarlina/mesh-cli/pkg/models"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	feedMode string
)

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "View your main timeline",
	Long:  "Display posts from your home feed, with options for different algorithms",
	Run: func(cmd *cobra.Command, args []string) {
		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		mode := client.FeedMode(feedMode)
		if mode == "" {
			mode = client.FeedModeHome
		}

		req := &client.FeedRequest{
			Mode:   mode,
			Limit:  flagLimit,
			Before: flagBefore,
			After:  flagAfter,
			Since:  flagSince,
			Until:  flagUntil,
		}

		posts, cursor, err := c.GetFeed(req)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if len(posts) == 0 {
			if !flagQuiet {
				out.Println("No posts found")
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

var catchupCmd = &cobra.Command{
	Use:   "catchup",
	Short: "High-signal posts since last login",
	Long:  "View important posts you may have missed since your last login",
	Run: func(cmd *cobra.Command, args []string) {
		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		since := flagSince
		if since == "" {
			since = "24h" // Default to last 24 hours
		}

		posts, err := c.GetCatchup(since, flagLimit)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if len(posts) == 0 {
			if !flagQuiet {
				out.Println("No new posts")
			}
			return
		}

		// Update context to the first post
		if len(posts) > 0 {
			context.Set(posts[0].ID, "post")
		}

		if flagJSON {
			out.Success(map[string]interface{}{"posts": posts})
		} else {
			for i, post := range posts {
				renderPost(out, post)
				if i < len(posts)-1 {
					out.Println()
				}
			}
		}
	},
}

var readCmd = &cobra.Command{
	Use:   "read <@user|p_id|this>",
	Short: "Read posts or a specific post",
	Long:  "View posts from a user or read a specific post by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		// Check if it's a user handle
		if strings.HasPrefix(target, "@") {
			handle := strings.TrimPrefix(target, "@")
			posts, cursor, err := c.GetUserPosts(handle, flagLimit, flagBefore, flagAfter)
			if err != nil {
				out.Error(err)
				os.Exit(1)
			}

			if len(posts) == 0 {
				if !flagQuiet {
					out.Printf("No posts from @%s\n", handle)
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
		} else {
			// It's a post ID (or "this")
			id, _, err := context.ResolveTarget(target)
			if err != nil {
				out.Error(err)
				os.Exit(1)
			}

			post, err := c.GetPost(id)
			if err != nil {
				out.Error(err)
				os.Exit(1)
			}

			context.Set(post.ID, "post")

			if flagJSON {
				out.Success(post)
			} else {
				renderPost(out, post)
			}
		}
	},
}

var threadCmd = &cobra.Command{
	Use:   "thread <p_id|this>",
	Short: "View full thread context",
	Long:  "Display the complete conversation thread for a post",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		id, _, err := context.ResolveTarget(target)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		posts, err := c.GetThread(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if len(posts) == 0 {
			if !flagQuiet {
				out.Println("No thread found")
			}
			return
		}

		// Update context to the target post
		context.Set(id, "post")

		if flagJSON {
			out.Success(map[string]interface{}{"posts": posts})
		} else {
			for i, post := range posts {
				renderPost(out, post)
				if i < len(posts)-1 {
					out.Println()
				}
			}
		}
	},
}

var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "Search posts, users, or tags",
	Long:  "Search for content across the platform (public content only)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		typ, _ := cmd.Flags().GetString("type")

		req := &client.SearchRequest{
			Query:  query,
			Type:   typ,
			Limit:  flagLimit,
			Before: flagBefore,
			After:  flagAfter,
		}

		result, err := c.Search(req)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(result)
		} else {
			// Render based on type
			if typ == "" || typ == "posts" {
				if len(result.Posts) > 0 {
					if !flagQuiet {
						out.Println("Posts:")
					}
					for i, post := range result.Posts {
						renderPost(out, post)
						if i < len(result.Posts)-1 {
							out.Println()
						}
					}
					// Update context to first post
					context.Set(result.Posts[0].ID, "post")
				}
			}

			if typ == "" || typ == "users" {
				if len(result.Users) > 0 {
					if typ == "" && len(result.Posts) > 0 {
						out.Println()
					}
					if !flagQuiet {
						out.Println("Users:")
					}
					for _, user := range result.Users {
						renderUser(out, user)
					}
				}
			}

			if typ == "" || typ == "tags" {
				if len(result.Tags) > 0 {
					if typ == "" && (len(result.Posts) > 0 || len(result.Users) > 0) {
						out.Println()
					}
					if !flagQuiet {
						out.Println("Tags:")
					}
					for _, tag := range result.Tags {
						out.Printf("  %s\n", tag)
					}
				}
			}

			if len(result.Posts) == 0 && len(result.Users) == 0 && len(result.Tags) == 0 {
				if !flagQuiet {
					out.Println("No results found")
				}
			}

			if result.Cursor != "" && !flagQuiet {
				out.Printf("\nNext page: --after %s\n", result.Cursor)
			}
		}
	},
}

func renderPost(out *output.Printer, post *models.Post) {
	if out.IsJSON() {
		data, _ := json.Marshal(post)
		out.Print(string(data))
		return
	}

	if out.IsRaw() {
		out.Printf("%s\n", post.Content)
		return
	}

	// Human-readable format
	author := "unknown"
	if post.Author != nil {
		if post.Author.Name != "" {
			author = fmt.Sprintf("%s (@%s)", post.Author.Name, post.Author.Handle)
		} else {
			author = fmt.Sprintf("@%s", post.Author.Handle)
		}
	}

	out.Printf("%s • %s • %s\n", post.ID, author, post.CreatedAt.Format("2006-01-02 15:04"))

	if post.ReplyTo != nil {
		out.Printf("  ↳ replying to %s\n", *post.ReplyTo)
	}
	if post.QuoteOf != nil {
		out.Printf("  ↺ quoting %s\n", *post.QuoteOf)
	}

	out.Println(post.Content)

	if post.Visibility != models.VisibilityPublic {
		out.Printf("  [%s]\n", post.Visibility)
	}
}

func renderUser(out *output.Printer, user *models.User) {
	if out.IsRaw() {
		out.Printf("@%s\n", user.Handle)
		return
	}

	name := user.Handle
	if user.Name != "" {
		name = fmt.Sprintf("%s (@%s)", user.Name, user.Handle)
	} else {
		name = fmt.Sprintf("@%s", user.Handle)
	}

	out.Printf("  %s\n", name)
	if user.Bio != "" {
		out.Printf("    %s\n", user.Bio)
	}
}

func init() {
	rootCmd.AddCommand(feedCmd)
	rootCmd.AddCommand(catchupCmd)
	rootCmd.AddCommand(readCmd)
	rootCmd.AddCommand(threadCmd)
	rootCmd.AddCommand(findCmd)

	feedCmd.Flags().StringVar(&feedMode, "mode", "home", "Feed mode (home|best|latest)")
	findCmd.Flags().String("type", "", "Search type (posts|users|tags)")
}
