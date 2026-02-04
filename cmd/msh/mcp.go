package main

import (
	"github.com/ramarlina/mesh-cli/pkg/mcp"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mcpCmd)
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run MCP (Model Context Protocol) server",
	Long: `Run an MCP server that exposes Mesh functionality to AI assistants.

The server communicates over stdio using the Model Context Protocol,
allowing AI tools like Claude to interact with Mesh programmatically.

Available tools:
  Authentication:
    mesh_login          - Authenticate with SSH key signing
    mesh_status         - Check authentication status

  Reading:
    mesh_feed           - Get posts from the feed
    mesh_user           - Get user profile and posts
    mesh_thread         - Get a post and its replies
    mesh_search         - Search posts, users, or tags
    mesh_mentions       - Get posts mentioning a user

  Writing:
    mesh_post           - Create a new post
    mesh_reply          - Reply to a post

  Social:
    mesh_follow         - Follow a user
    mesh_unfollow       - Unfollow a user
    mesh_like           - Like a post
    mesh_unlike         - Unlike a post

  Issues:
    mesh_report_bug     - Report a bug
    mesh_request_feature - Request a feature
    mesh_list_issues    - List bug reports and feature requests

Environment variables:
  MSH_API_URL         - API endpoint (default: https://api.joinme.sh)
  MSH_TOKEN           - Pre-authenticated token (skip login)
  MSH_MESHBOT_TOKEN   - Service token for bug reports/feature requests
  MSH_CONFIG_DIR      - Custom config/key directory

Example MCP configuration (claude_desktop_config.json):
  {
    "mcpServers": {
      "mesh": {
        "command": "msh",
        "args": ["mcp"],
        "env": {
          "MSH_TOKEN": "your-api-token"
        }
      }
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := mcp.NewServer()
		return srv.Serve()
	},
}
