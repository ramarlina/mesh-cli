package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/context"
	"github.com/spf13/cobra"
)

var (
	postVisibility string
	postTags       []string
	postAttach     []string
	postEditor     bool
)

var postCmd = &cobra.Command{
	Use:   "post [text|-]",
	Short: "Create a new post",
	Long:  "Publish a new message. Use '-' to read from stdin or --editor to open $EDITOR",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var content string
		var err error

		if postEditor {
			content, err = getEditorInput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		} else if len(args) == 0 || args[0] == "-" {
			content, err = getStdinInput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: failed to read stdin: %v\n", err)
				os.Exit(1)
			}
		} else {
			content = args[0]
		}

		content = strings.TrimSpace(content)
		if content == "" {
			fmt.Fprintf(os.Stderr, "error: post content cannot be empty\n")
			os.Exit(1)
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		req := &client.CreatePostRequest{
			Content:    content,
			Visibility: postVisibility,
			Tags:       postTags,
			AssetIDs:   postAttach,
		}

		post, err := c.CreatePost(req)
		if err != nil {
			// Check if it's a challenge error
			if apiErr, ok := err.(*client.APIError); ok {
				if apiErr.Err.Code == "challenge_required" {
					// Handle challenge interactively
					if handleChallengeInteractive(c, out, apiErr.Err) {
						// Retry the post
						post, err = c.CreatePost(req)
						if err != nil {
							out.Error(err)
							os.Exit(1)
						}
					} else {
						os.Exit(1)
					}
				} else {
					out.Error(err)
					os.Exit(1)
				}
			} else {
				out.Error(err)
				os.Exit(1)
			}
		}

		context.Set(post.ID, "post")

		if flagJSON {
			out.Success(post)
		} else if !flagQuiet {
			out.Printf("✓ Posted: %s\n", post.ID)
		}
	},
}

var replyCmd = &cobra.Command{
	Use:   "reply <p_id|this> <text>",
	Short: "Reply to a post",
	Long:  "Create a threaded reply to an existing post",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]
		content := strings.Join(args[1:], " ")

		id, _, err := context.ResolveTarget(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		req := &client.CreatePostRequest{
			Content:    content,
			ReplyTo:    id,
			Visibility: postVisibility,
			Tags:       postTags,
			AssetIDs:   postAttach,
		}

		post, err := c.CreatePost(req)
		if err != nil {
			// Check if it's a challenge error
			if apiErr, ok := err.(*client.APIError); ok {
				if apiErr.Err.Code == "challenge_required" {
					// Handle challenge interactively
					if handleChallengeInteractive(c, out, apiErr.Err) {
						// Retry the reply
						post, err = c.CreatePost(req)
						if err != nil {
							out.Error(err)
							os.Exit(1)
						}
					} else {
						os.Exit(1)
					}
				} else {
					out.Error(err)
					os.Exit(1)
				}
			} else {
				out.Error(err)
				os.Exit(1)
			}
		}

		context.Set(post.ID, "post")

		if flagJSON {
			out.Success(post)
		} else if !flagQuiet {
			out.Printf("✓ Replied: %s\n", post.ID)
		}
	},
}

var quoteCmd = &cobra.Command{
	Use:   "quote <p_id|this> <text>",
	Short: "Quote a post",
	Long:  "Create a new post that references another post",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]
		content := strings.Join(args[1:], " ")

		id, _, err := context.ResolveTarget(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		req := &client.CreatePostRequest{
			Content:    content,
			QuoteOf:    id,
			Visibility: postVisibility,
			Tags:       postTags,
			AssetIDs:   postAttach,
		}

		post, err := c.CreatePost(req)
		if err != nil {
			// Check if it's a challenge error
			if apiErr, ok := err.(*client.APIError); ok {
				if apiErr.Err.Code == "challenge_required" {
					// Handle challenge interactively
					if handleChallengeInteractive(c, out, apiErr.Err) {
						// Retry the quote
						post, err = c.CreatePost(req)
						if err != nil {
							out.Error(err)
							os.Exit(1)
						}
					} else {
						os.Exit(1)
					}
				} else {
					out.Error(err)
					os.Exit(1)
				}
			} else {
				out.Error(err)
				os.Exit(1)
			}
		}

		context.Set(post.ID, "post")

		if flagJSON {
			out.Success(post)
		} else if !flagQuiet {
			out.Printf("✓ Quoted: %s\n", post.ID)
		}
	},
}

var editCmd = &cobra.Command{
	Use:   "edit <p_id|this> [--editor | --set <text>]",
	Short: "Edit your own post",
	Long:  "Update the content of an existing post you created",
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

		var content string

		setText, _ := cmd.Flags().GetString("set")
		if setText != "" {
			content = setText
		} else if postEditor {
			// Load current post content
			post, err := c.GetPost(id)
			if err != nil {
				out.Error(err)
				os.Exit(1)
			}

			content, err = getEditorInputWithContent(post.Content)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "error: must provide --set or --editor\n")
			os.Exit(1)
		}

		content = strings.TrimSpace(content)
		if content == "" {
			fmt.Fprintf(os.Stderr, "error: post content cannot be empty\n")
			os.Exit(1)
		}

		req := &client.UpdatePostRequest{
			Content: content,
		}

		post, err := c.UpdatePost(id, req)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		context.Set(post.ID, "post")

		if flagJSON {
			out.Success(post)
		} else if !flagQuiet {
			out.Printf("✓ Updated: %s\n", post.ID)
		}
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <p_id|this>",
	Short: "Delete your own post",
	Long:  "Permanently delete a post you created",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		id, _, err := context.ResolveTarget(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// Confirm deletion unless --yes is set
		if !flagYes {
			fmt.Printf("Delete post %s? [y/N]: ", id)
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

		err = c.DeletePost(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "deleted", "id": id})
		} else if !flagQuiet {
			out.Printf("✓ Deleted: %s\n", id)
		}
	},
}

func getStdinInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	var content strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				content.WriteString(line)
				break
			}
			return "", err
		}
		content.WriteString(line)
	}

	return content.String(), nil
}

func getEditorInput() (string, error) {
	return getEditorInputWithContent("")
}

func getEditorInputWithContent(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	tmpFile, err := os.CreateTemp("", "msh-post-*.md")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if initial != "" {
		if _, err := tmpFile.WriteString(initial); err != nil {
			return "", fmt.Errorf("write initial content: %w", err)
		}
	}
	tmpFile.Close()

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("read edited content: %w", err)
	}

	return string(content), nil
}

func init() {
	rootCmd.AddCommand(postCmd)
	rootCmd.AddCommand(replyCmd)
	rootCmd.AddCommand(quoteCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(deleteCmd)

	postCmd.Flags().StringVar(&postVisibility, "visibility", "", "Post visibility (public|unlisted|followers|private)")
	postCmd.Flags().StringSliceVar(&postTags, "tag", []string{}, "Add tag (can be repeated)")
	postCmd.Flags().StringSliceVar(&postAttach, "attach", []string{}, "Attach asset (path or as_id)")
	postCmd.Flags().BoolVar(&postEditor, "editor", false, "Open $EDITOR to compose")

	replyCmd.Flags().StringVar(&postVisibility, "visibility", "", "Post visibility")
	replyCmd.Flags().StringSliceVar(&postTags, "tag", []string{}, "Add tag")
	replyCmd.Flags().StringSliceVar(&postAttach, "attach", []string{}, "Attach asset")

	quoteCmd.Flags().StringVar(&postVisibility, "visibility", "", "Post visibility")
	quoteCmd.Flags().StringSliceVar(&postTags, "tag", []string{}, "Add tag")
	quoteCmd.Flags().StringSliceVar(&postAttach, "attach", []string{}, "Attach asset")

	editCmd.Flags().String("set", "", "New content")
	editCmd.Flags().BoolVar(&postEditor, "editor", false, "Open $EDITOR to edit")
}
