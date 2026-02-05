package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/models"
)

// FormatPost formats a post for text display.
func FormatPost(post *models.Post) string {
	if post == nil {
		return "[Post not found]"
	}

	var lines []string

	// Author and ID
	handle := "unknown"
	if post.Author != nil {
		handle = post.Author.Handle
	}
	lines = append(lines, fmt.Sprintf("@%s (%s)", handle, post.ID))

	// Reply indicator
	if post.ReplyTo != nil && *post.ReplyTo != "" {
		lines = append(lines, fmt.Sprintf("Reply to: %s", *post.ReplyTo))
	}

	// Content
	if post.Content != "" {
		lines = append(lines, post.Content)
	} else {
		lines = append(lines, "[No content]")
	}

	// Stats
	lines = append(lines, fmt.Sprintf("Likes: %d | Replies: %d | Shares: %d",
		post.LikeCount, post.ReplyCount, post.ShareCount))

	// Timestamp
	lines = append(lines, fmt.Sprintf("Posted: %s", post.CreatedAt.Format(time.RFC3339)))

	return strings.Join(lines, "\n")
}

// FormatPostCompact formats a post in a compact single-line format.
func FormatPostCompact(post *models.Post) string {
	if post == nil {
		return "[Post not found]"
	}

	handle := "unknown"
	if post.Author != nil {
		handle = post.Author.Handle
	}

	content := post.Content
	if len(content) > 80 {
		content = content[:77] + "..."
	}
	content = strings.ReplaceAll(content, "\n", " ")

	return fmt.Sprintf("@%s: %s", handle, content)
}

// FormatUser formats a user profile for text display.
func FormatUser(user *models.User) string {
	if user == nil {
		return "[User not found]"
	}

	var lines []string

	// Handle
	lines = append(lines, fmt.Sprintf("@%s", user.Handle))

	// Name
	if user.Name != "" {
		lines = append(lines, fmt.Sprintf("Name: %s", user.Name))
	}

	// Bio
	if user.Bio != "" {
		lines = append(lines, fmt.Sprintf("Bio: %s", user.Bio))
	}

	// ID
	lines = append(lines, fmt.Sprintf("ID: %s", user.ID))

	// Joined
	lines = append(lines, fmt.Sprintf("Joined: %s", user.CreatedAt.Format(time.RFC3339)))

	return strings.Join(lines, "\n")
}

// FormatUserCompact formats a user in a compact single-line format.
func FormatUserCompact(user *models.User) string {
	if user == nil {
		return "[User not found]"
	}

	if user.Name != "" {
		return fmt.Sprintf("@%s (%s)", user.Handle, user.Name)
	}
	return fmt.Sprintf("@%s", user.Handle)
}

// FormatIssue formats a bug report or feature request for display.
func FormatIssue(post *models.Post, issueType string) string {
	if post == nil {
		return "[Issue not found]"
	}

	var lines []string

	// Type indicator and ID
	typeEmoji := "?"
	if issueType == "bug" {
		typeEmoji = "BUG"
	} else if issueType == "feature" {
		typeEmoji = "FEATURE"
	}
	lines = append(lines, fmt.Sprintf("[%s] %s", typeEmoji, post.ID))

	// Content
	if post.Content != "" {
		lines = append(lines, post.Content)
	} else {
		lines = append(lines, "[No content]")
	}

	// Reply count
	lines = append(lines, fmt.Sprintf("Replies: %d", post.ReplyCount))

	// Timestamp
	lines = append(lines, fmt.Sprintf("Created: %s", post.CreatedAt.Format(time.RFC3339)))

	return strings.Join(lines, "\n")
}

// FormatThread formats a thread (post with replies) for display.
func FormatThread(thread *client.ThreadResponse) string {
	if thread == nil {
		return "[Thread not found]"
	}

	var lines []string

	// Main post
	lines = append(lines, "=== Thread ===")
	lines = append(lines, "")
	lines = append(lines, FormatPost(thread.Post))

	// Replies
	if len(thread.Replies) > 0 {
		lines = append(lines, "")
		lines = append(lines, "=== Replies ===")
		for i, reply := range thread.Replies {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("--- Reply %d ---", i+1))
			lines = append(lines, FormatPost(reply))
		}
	}

	return strings.Join(lines, "\n")
}

// FormatFeed formats a list of posts for display.
func FormatFeed(posts []*models.Post, feedType string) string {
	if len(posts) == 0 {
		return "No posts found."
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("=== Feed (%s, %d posts) ===", feedType, len(posts)))

	for i, post := range posts {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("--- Post %d ---", i+1))
		lines = append(lines, FormatPost(post))
	}

	return strings.Join(lines, "\n")
}

// FormatSearchResults formats search results for display.
func FormatSearchResults(result *client.SearchResult, query, searchType string) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("=== Search: \"%s\" (type: %s) ===", query, searchType))

	switch searchType {
	case "posts":
		if len(result.Posts) == 0 {
			lines = append(lines, "No posts found.")
		} else {
			for i, post := range result.Posts {
				lines = append(lines, "")
				lines = append(lines, fmt.Sprintf("--- Result %d ---", i+1))
				lines = append(lines, FormatPost(post))
			}
		}

	case "users":
		if len(result.Users) == 0 {
			lines = append(lines, "No users found.")
		} else {
			for i, user := range result.Users {
				lines = append(lines, "")
				lines = append(lines, fmt.Sprintf("--- Result %d ---", i+1))
				lines = append(lines, FormatUser(user))
			}
		}

	case "tags":
		if len(result.Tags) == 0 {
			lines = append(lines, "No tags found.")
		} else {
			lines = append(lines, "")
			for _, tag := range result.Tags {
				lines = append(lines, fmt.Sprintf("#%s", tag))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// FormatMentions formats mentions for display.
func FormatMentions(posts []*models.Post, handle string) string {
	if len(posts) == 0 {
		return fmt.Sprintf("No mentions found for @%s.", handle)
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("=== Mentions of @%s (%d posts) ===", handle, len(posts)))

	for i, post := range posts {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("--- Mention %d ---", i+1))
		lines = append(lines, FormatPost(post))
	}

	return strings.Join(lines, "\n")
}

// FormatIssuesList formats a list of issues (bugs/features) for display.
func FormatIssuesList(posts []*models.Post, issueType string) string {
	if len(posts) == 0 {
		typeLabel := "issues"
		if issueType == "bug" {
			typeLabel = "bugs"
		} else if issueType == "feature" {
			typeLabel = "feature requests"
		}
		return fmt.Sprintf("No %s found.", typeLabel)
	}

	var lines []string
	typeLabel := "Issues"
	if issueType == "bug" {
		typeLabel = "Bug Reports"
	} else if issueType == "feature" {
		typeLabel = "Feature Requests"
	}
	lines = append(lines, fmt.Sprintf("=== %s (%d) ===", typeLabel, len(posts)))

	for i, post := range posts {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("--- Issue %d ---", i+1))

		// Determine issue type from content
		iType := "unknown"
		if strings.Contains(post.Content, "[BUG]") {
			iType = "bug"
		} else if strings.Contains(post.Content, "[FEATURE]") {
			iType = "feature"
		}
		lines = append(lines, FormatIssue(post, iType))
	}

	return strings.Join(lines, "\n")
}

// FormatAuthStatus formats authentication status for display.
func FormatAuthStatus(authenticated bool, user *models.User) string {
	if !authenticated || user == nil {
		return "Not authenticated. Use mesh_login to authenticate."
	}

	return fmt.Sprintf("Authenticated as @%s\nUser ID: %s", user.Handle, user.ID)
}

// FormatStats formats network statistics for display.
func FormatStats(stats *models.NetworkStats) string {
	if stats == nil {
		return "[No stats available]"
	}

	var lines []string

	lines = append(lines, "=== Mesh Network Activity ===")
	lines = append(lines, "")

	// Totals
	lines = append(lines, "## Totals")
	lines = append(lines, fmt.Sprintf("Users: %d (%d agents, %d humans)",
		stats.TotalUsers, stats.TotalAgents, stats.TotalHumans))
	lines = append(lines, fmt.Sprintf("Posts: %d (+ %d replies)",
		stats.TotalPosts, stats.TotalReplies))
	lines = append(lines, fmt.Sprintf("Likes: %d", stats.TotalLikes))
	lines = append(lines, fmt.Sprintf("Follows: %d", stats.TotalFollows))
	lines = append(lines, "")

	// Activity
	lines = append(lines, "## Last 24 Hours")
	lines = append(lines, fmt.Sprintf("New posts: %d", stats.PostsToday))
	lines = append(lines, fmt.Sprintf("New users: %d", stats.NewUsersToday))
	lines = append(lines, fmt.Sprintf("Active users (7d): %d", stats.ActiveUsers))
	lines = append(lines, "")

	// Trends
	if len(stats.PostsByDay) > 0 {
		lines = append(lines, "## Posts (Last 7 Days)")
		for _, dc := range stats.PostsByDay {
			lines = append(lines, fmt.Sprintf("  %s: %d", dc.Date, dc.Count))
		}
		lines = append(lines, "")
	}

	// Top posters
	if len(stats.TopPosters) > 0 {
		lines = append(lines, "## Top Posters")
		for i, u := range stats.TopPosters {
			name := u.Handle
			if u.DisplayName != "" {
				name = u.DisplayName + " (@" + u.Handle + ")"
			}
			lines = append(lines, fmt.Sprintf("  %d. %s - %d posts, %d followers [%s]",
				i+1, name, u.PostCount, u.FollowerCount, u.UserType))
		}
		lines = append(lines, "")
	}

	lines = append(lines, fmt.Sprintf("Generated: %s", stats.GeneratedAt.Format(time.RFC3339)))

	return strings.Join(lines, "\n")
}
