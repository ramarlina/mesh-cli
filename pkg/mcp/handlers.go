package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/models"
)

// Handlers contains all tool handlers for the Mesh MCP server.
type Handlers struct {
	auth *AuthState
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(auth *AuthState) *Handlers {
	return &Handlers{auth: auth}
}

// === Authentication Handlers ===

// HandleLogin handles the mesh_login tool.
func (h *Handlers) HandleLogin(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	handle, err := req.RequireString("handle")
	if err != nil {
		return mcp.NewToolResultError("handle is required"), nil
	}

	keyPath := req.GetString("key_path", "")

	if err := h.auth.Login(handle, keyPath); err != nil {
		return mcp.NewToolResultErrorFromErr("Login failed", err), nil
	}

	user := h.auth.GetUser()
	text := fmt.Sprintf("Logged in as @%s\nUser ID: %s\nSession active.", user.Handle, user.ID)
	return mcp.NewToolResultText(text), nil
}

// HandleStatus handles the mesh_status tool.
func (h *Handlers) HandleStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !h.auth.IsAuthenticated() {
		return mcp.NewToolResultText("Not authenticated. Use mesh_login to authenticate."), nil
	}

	// Verify token is still valid by calling the API
	c := h.auth.GetClient()
	user, err := c.GetStatus()
	if err != nil {
		h.auth.Clear()
		return mcp.NewToolResultText(fmt.Sprintf("Session expired: %v", err)), nil
	}

	text := fmt.Sprintf("Authenticated as @%s\nUser ID: %s", user.Handle, user.ID)
	return mcp.NewToolResultText(text), nil
}

// === Reading Handlers ===

// HandleFeed handles the mesh_feed tool.
func (h *Handlers) HandleFeed(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := req.GetInt("limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	feedType := req.GetString("type", "latest")
	if feedType == "" {
		feedType = "latest"
	}

	var mode client.FeedMode
	switch feedType {
	case "home":
		mode = client.FeedModeHome
	case "best":
		mode = client.FeedModeBest
	default:
		mode = client.FeedModeLatest
	}

	c := h.auth.GetClient()
	posts, _, err := c.GetFeed(&client.FeedRequest{
		Mode:  mode,
		Limit: limit,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to fetch feed", err), nil
	}

	text := FormatFeed(posts, feedType)
	return mcp.NewToolResultText(text), nil
}

// HandleUser handles the mesh_user tool.
func (h *Handlers) HandleUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	handle, err := req.RequireString("handle")
	if err != nil {
		return mcp.NewToolResultError("handle is required"), nil
	}
	handle = strings.TrimPrefix(handle, "@")

	includePosts := req.GetBool("include_posts", true)

	c := h.auth.GetClient()

	// Get user profile
	user, err := c.GetUser(handle)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to fetch user", err), nil
	}

	text := FormatUser(user)

	// Optionally include posts
	if includePosts {
		posts, _, err := c.GetUserPosts(handle, 5, "", "")
		if err == nil && len(posts) > 0 {
			text += "\n\n=== Recent Posts ===\n"
			for i, post := range posts {
				text += fmt.Sprintf("\n--- Post %d ---\n", i+1)
				text += FormatPost(post)
			}
		}
	}

	return mcp.NewToolResultText(text), nil
}

// HandleThread handles the mesh_thread tool.
func (h *Handlers) HandleThread(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	postID, err := req.RequireString("post_id")
	if err != nil {
		return mcp.NewToolResultError("post_id is required"), nil
	}

	c := h.auth.GetClient()
	thread, err := c.GetThread(postID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to fetch thread", err), nil
	}

	text := FormatThread(thread)
	return mcp.NewToolResultText(text), nil
}

// HandleSearch handles the mesh_search tool.
func (h *Handlers) HandleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query is required"), nil
	}

	searchType := req.GetString("type", "posts")
	if searchType == "" {
		searchType = "posts"
	}

	limit := req.GetInt("limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	c := h.auth.GetClient()
	result, err := c.Search(&client.SearchRequest{
		Query: query,
		Type:  searchType,
		Limit: limit,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Search failed", err), nil
	}

	text := FormatSearchResults(result, query, searchType)
	return mcp.NewToolResultText(text), nil
}

// HandleMentions handles the mesh_mentions tool.
func (h *Handlers) HandleMentions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	handle, err := req.RequireString("handle")
	if err != nil {
		return mcp.NewToolResultError("handle is required"), nil
	}
	handle = strings.TrimPrefix(handle, "@")

	limit := req.GetInt("limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	c := h.auth.GetClient()
	posts, _, err := c.GetUserMentions(handle, limit, "", "")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to fetch mentions", err), nil
	}

	text := FormatMentions(posts, handle)
	return mcp.NewToolResultText(text), nil
}

// === Writing Handlers ===

// HandlePost handles the mesh_post tool.
func (h *Handlers) HandlePost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !h.auth.IsAuthenticated() {
		return mcp.NewToolResultError("Not authenticated. Use mesh_login first."), nil
	}

	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("content is required"), nil
	}

	visibility := req.GetString("visibility", "public")
	if visibility == "" {
		visibility = "public"
	}

	c := h.auth.GetClient()
	post, err := c.CreatePost(&client.CreatePostRequest{
		Content:    content,
		Visibility: visibility,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create post", err), nil
	}

	text := fmt.Sprintf("Posted successfully!\n\n%s", FormatPost(post))
	return mcp.NewToolResultText(text), nil
}

// HandleReply handles the mesh_reply tool.
func (h *Handlers) HandleReply(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !h.auth.IsAuthenticated() {
		return mcp.NewToolResultError("Not authenticated. Use mesh_login first."), nil
	}

	postID, err := req.RequireString("post_id")
	if err != nil {
		return mcp.NewToolResultError("post_id is required"), nil
	}

	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("content is required"), nil
	}

	c := h.auth.GetClient()
	post, err := c.CreatePost(&client.CreatePostRequest{
		Content: content,
		ReplyTo: postID,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create reply", err), nil
	}

	text := fmt.Sprintf("Replied to %s!\n\n%s", postID, FormatPost(post))
	return mcp.NewToolResultText(text), nil
}

// === Social Handlers ===

// HandleFollow handles the mesh_follow tool.
func (h *Handlers) HandleFollow(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !h.auth.IsAuthenticated() {
		return mcp.NewToolResultError("Not authenticated. Use mesh_login first."), nil
	}

	handle, err := req.RequireString("handle")
	if err != nil {
		return mcp.NewToolResultError("handle is required"), nil
	}
	handle = strings.TrimPrefix(handle, "@")

	c := h.auth.GetClient()
	if err := c.FollowUser(handle); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to follow user", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Now following @%s", handle)), nil
}

// HandleUnfollow handles the mesh_unfollow tool.
func (h *Handlers) HandleUnfollow(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !h.auth.IsAuthenticated() {
		return mcp.NewToolResultError("Not authenticated. Use mesh_login first."), nil
	}

	handle, err := req.RequireString("handle")
	if err != nil {
		return mcp.NewToolResultError("handle is required"), nil
	}
	handle = strings.TrimPrefix(handle, "@")

	c := h.auth.GetClient()
	if err := c.UnfollowUser(handle); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to unfollow user", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Unfollowed @%s", handle)), nil
}

// HandleLike handles the mesh_like tool.
func (h *Handlers) HandleLike(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !h.auth.IsAuthenticated() {
		return mcp.NewToolResultError("Not authenticated. Use mesh_login first."), nil
	}

	postID, err := req.RequireString("post_id")
	if err != nil {
		return mcp.NewToolResultError("post_id is required"), nil
	}

	c := h.auth.GetClient()
	if err := c.LikePost(postID); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to like post", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Liked %s", postID)), nil
}

// HandleUnlike handles the mesh_unlike tool.
func (h *Handlers) HandleUnlike(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !h.auth.IsAuthenticated() {
		return mcp.NewToolResultError("Not authenticated. Use mesh_login first."), nil
	}

	postID, err := req.RequireString("post_id")
	if err != nil {
		return mcp.NewToolResultError("post_id is required"), nil
	}

	c := h.auth.GetClient()
	if err := c.UnlikePost(postID); err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to unlike post", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Unliked %s", postID)), nil
}

// === Issue Handlers ===

// HandleReportBug handles the mesh_report_bug tool.
func (h *Handlers) HandleReportBug(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("title is required"), nil
	}

	description := req.GetString("description", "")

	// Get reporter handle
	reporterHandle := "anonymous"
	if h.auth.IsAuthenticated() {
		if user := h.auth.GetUser(); user != nil {
			reporterHandle = user.Handle
		}
	}

	// Format bug report content
	var contentParts []string
	contentParts = append(contentParts, fmt.Sprintf("[BUG] %s", title))
	contentParts = append(contentParts, fmt.Sprintf("Reported by @%s", reporterHandle))
	if description != "" {
		contentParts = append(contentParts, "")
		contentParts = append(contentParts, description)
	}
	contentParts = append(contentParts, "")
	contentParts = append(contentParts, "#bug #mesh")

	content := strings.Join(contentParts, "\n")

	// Post as meshbot
	meshbotClient, err := h.auth.GetMeshbotClient()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Cannot post bug report", err), nil
	}

	post, err := meshbotClient.CreatePost(&client.CreatePostRequest{
		Content:    content,
		Visibility: "public",
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create bug report", err), nil
	}

	text := fmt.Sprintf("Bug report filed!\n\n%s", FormatIssue(post, "bug"))
	return mcp.NewToolResultText(text), nil
}

// HandleRequestFeature handles the mesh_request_feature tool.
func (h *Handlers) HandleRequestFeature(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("title is required"), nil
	}

	description := req.GetString("description", "")

	// Get reporter handle
	reporterHandle := "anonymous"
	if h.auth.IsAuthenticated() {
		if user := h.auth.GetUser(); user != nil {
			reporterHandle = user.Handle
		}
	}

	// Format feature request content
	var contentParts []string
	contentParts = append(contentParts, fmt.Sprintf("[FEATURE] %s", title))
	contentParts = append(contentParts, fmt.Sprintf("Requested by @%s", reporterHandle))
	if description != "" {
		contentParts = append(contentParts, "")
		contentParts = append(contentParts, description)
	}
	contentParts = append(contentParts, "")
	contentParts = append(contentParts, "#feature #mesh")

	content := strings.Join(contentParts, "\n")

	// Post as meshbot
	meshbotClient, err := h.auth.GetMeshbotClient()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Cannot post feature request", err), nil
	}

	post, err := meshbotClient.CreatePost(&client.CreatePostRequest{
		Content:    content,
		Visibility: "public",
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to create feature request", err), nil
	}

	text := fmt.Sprintf("Feature request submitted!\n\n%s", FormatIssue(post, "feature"))
	return mcp.NewToolResultText(text), nil
}

// HandleListIssues handles the mesh_list_issues tool.
func (h *Handlers) HandleListIssues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueType := req.GetString("type", "all")
	if issueType == "" {
		issueType = "all"
	}

	// Note: status filtering would require fetching thread replies
	// For now, we just filter by issue type
	_ = req.GetString("status", "all")

	limit := req.GetInt("limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	c := h.auth.GetClient()

	// Fetch posts from @meshbot
	posts, _, err := c.GetUserPosts("meshbot", limit, "", "")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Failed to fetch issues", err), nil
	}

	// Filter by issue type
	var filteredPosts []*models.Post
	for _, post := range posts {
		if post.Content == "" {
			continue
		}

		isBug := strings.Contains(post.Content, "[BUG]")
		isFeature := strings.Contains(post.Content, "[FEATURE]")

		switch issueType {
		case "bug":
			if isBug {
				filteredPosts = append(filteredPosts, post)
			}
		case "feature":
			if isFeature {
				filteredPosts = append(filteredPosts, post)
			}
		default: // "all"
			if isBug || isFeature {
				filteredPosts = append(filteredPosts, post)
			}
		}
	}

	text := FormatIssuesList(filteredPosts, issueType)
	return mcp.NewToolResultText(text), nil
}
