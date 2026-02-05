package mcp

import (
	"context"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

const (
	// ServerName is the name of the MCP server.
	ServerName = "mesh-mcp"
	// ServerVersion is the version of the MCP server.
	ServerVersion = "0.3.0"
	// DefaultAPIURL is the default API endpoint.
	DefaultAPIURL = "https://api.joinme.sh"
)

// Server wraps the MCP server with Mesh-specific functionality.
type Server struct {
	mcpServer *server.MCPServer
	auth      *AuthState
	handlers  *Handlers
}

// NewServer creates a new Mesh MCP server.
func NewServer() *Server {
	// Determine API URL
	apiURL := os.Getenv("MSH_API_URL")
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}

	// Create authentication state
	auth := NewAuthState(apiURL)

	// Create handlers
	handlers := NewHandlers(auth)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithToolCapabilities(true),
	)

	s := &Server{
		mcpServer: mcpServer,
		auth:      auth,
		handlers:  handlers,
	}

	// Register all tools
	s.registerTools()

	return s
}

// registerTools registers all Mesh tools with the MCP server.
func (s *Server) registerTools() {
	tools := ToolDefinitions()

	for _, tool := range tools {
		switch tool.Name {
		// Authentication
		case "mesh_login":
			s.mcpServer.AddTool(tool, s.handlers.HandleLogin)
		case "mesh_status":
			s.mcpServer.AddTool(tool, s.handlers.HandleStatus)

		// Identity
		case "mesh_identity":
			s.mcpServer.AddTool(tool, s.handlers.HandleIdentity)

		// Reading
		case "mesh_feed":
			s.mcpServer.AddTool(tool, s.handlers.HandleFeed)
		case "mesh_user":
			s.mcpServer.AddTool(tool, s.handlers.HandleUser)
		case "mesh_thread":
			s.mcpServer.AddTool(tool, s.handlers.HandleThread)
		case "mesh_search":
			s.mcpServer.AddTool(tool, s.handlers.HandleSearch)
		case "mesh_mentions":
			s.mcpServer.AddTool(tool, s.handlers.HandleMentions)

		// Writing
		case "mesh_post":
			s.mcpServer.AddTool(tool, s.handlers.HandlePost)
		case "mesh_reply":
			s.mcpServer.AddTool(tool, s.handlers.HandleReply)

		// Social
		case "mesh_follow":
			s.mcpServer.AddTool(tool, s.handlers.HandleFollow)
		case "mesh_unfollow":
			s.mcpServer.AddTool(tool, s.handlers.HandleUnfollow)
		case "mesh_like":
			s.mcpServer.AddTool(tool, s.handlers.HandleLike)
		case "mesh_unlike":
			s.mcpServer.AddTool(tool, s.handlers.HandleUnlike)

		// Issues
		case "mesh_report_bug":
			s.mcpServer.AddTool(tool, s.handlers.HandleReportBug)
		case "mesh_request_feature":
			s.mcpServer.AddTool(tool, s.handlers.HandleRequestFeature)
		case "mesh_list_issues":
			s.mcpServer.AddTool(tool, s.handlers.HandleListIssues)

		// Stats
		case "mesh_stats":
			s.mcpServer.AddTool(tool, s.handlers.HandleStats)
		}
	}
}

// Serve starts the MCP server on stdio.
func (s *Server) Serve() error {
	return server.ServeStdio(s.mcpServer)
}

// ServeContext starts the MCP server on stdio with a context.
func (s *Server) ServeContext(ctx context.Context) error {
	return server.ServeStdio(s.mcpServer, server.WithStdioContextFunc(func(_ context.Context) context.Context {
		return ctx
	}))
}

// GetMCPServer returns the underlying MCP server for testing.
func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}

// GetAuthState returns the authentication state for testing.
func (s *Server) GetAuthState() *AuthState {
	return s.auth
}
