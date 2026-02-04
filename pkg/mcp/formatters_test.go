package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/models"
)

func TestFormatPost(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		post     *models.Post
		contains []string
		notContains []string
	}{
		{
			name: "nil post",
			post: nil,
			contains: []string{"[Post not found]"},
		},
		{
			name: "basic post",
			post: &models.Post{
				ID:      "post-123",
				Content: "Hello, world!",
				Author: &models.User{
					Handle: "testuser",
				},
				LikeCount:  5,
				ReplyCount: 2,
				ShareCount: 1,
				CreatedAt:  baseTime,
			},
			contains: []string{
				"@testuser",
				"post-123",
				"Hello, world!",
				"Likes: 5",
				"Replies: 2",
				"Shares: 1",
				"2025-01-15",
			},
		},
		{
			name: "post with reply indicator",
			post: &models.Post{
				ID:      "post-456",
				Content: "This is a reply",
				Author: &models.User{
					Handle: "replier",
				},
				ReplyTo:   strPtr("parent-post-789"),
				CreatedAt: baseTime,
			},
			contains: []string{
				"Reply to: parent-post-789",
				"This is a reply",
			},
		},
		{
			name: "post with empty reply_to",
			post: &models.Post{
				ID:      "post-789",
				Content: "Not a reply",
				Author: &models.User{
					Handle: "poster",
				},
				ReplyTo:   strPtr(""),
				CreatedAt: baseTime,
			},
			notContains: []string{"Reply to:"},
		},
		{
			name: "post without author",
			post: &models.Post{
				ID:        "post-no-author",
				Content:   "Anonymous post",
				Author:    nil,
				CreatedAt: baseTime,
			},
			contains: []string{
				"@unknown",
				"Anonymous post",
			},
		},
		{
			name: "post with empty content",
			post: &models.Post{
				ID: "post-empty",
				Author: &models.User{
					Handle: "emptypost",
				},
				Content:   "",
				CreatedAt: baseTime,
			},
			contains: []string{"[No content]"},
		},
		{
			name: "post with multiline content",
			post: &models.Post{
				ID:      "post-multi",
				Content: "Line 1\nLine 2\nLine 3",
				Author: &models.User{
					Handle: "multiline",
				},
				CreatedAt: baseTime,
			},
			contains: []string{
				"Line 1",
				"Line 2",
				"Line 3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPost(tt.post)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatPost() result missing %q\nGot: %s", want, result)
				}
			}

			for _, notWant := range tt.notContains {
				if strings.Contains(result, notWant) {
					t.Errorf("FormatPost() result should not contain %q\nGot: %s", notWant, result)
				}
			}
		})
	}
}

func TestFormatPostCompact(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		post     *models.Post
		contains []string
		maxLen   int // Check that result doesn't exceed this length (0 = no check)
	}{
		{
			name:     "nil post",
			post:     nil,
			contains: []string{"[Post not found]"},
		},
		{
			name: "short content",
			post: &models.Post{
				ID:      "post-1",
				Content: "Short message",
				Author: &models.User{
					Handle: "user",
				},
				CreatedAt: baseTime,
			},
			contains: []string{"@user:", "Short message"},
		},
		{
			name: "long content truncated",
			post: &models.Post{
				ID:      "post-2",
				Content: strings.Repeat("x", 100),
				Author: &models.User{
					Handle: "longuser",
				},
				CreatedAt: baseTime,
			},
			contains: []string{"@longuser:", "..."},
		},
		{
			name: "content with newlines",
			post: &models.Post{
				ID:      "post-3",
				Content: "Line 1\nLine 2\nLine 3",
				Author: &models.User{
					Handle: "newline",
				},
				CreatedAt: baseTime,
			},
			contains: []string{"Line 1 Line 2 Line 3"}, // Newlines replaced with spaces
		},
		{
			name: "no author",
			post: &models.Post{
				ID:        "post-4",
				Content:   "No author",
				Author:    nil,
				CreatedAt: baseTime,
			},
			contains: []string{"@unknown:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPostCompact(tt.post)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatPostCompact() result missing %q\nGot: %s", want, result)
				}
			}

			// Compact format should be single line
			if strings.Contains(result, "\n") && tt.post != nil {
				t.Errorf("FormatPostCompact() should be single line, got: %s", result)
			}
		})
	}
}

func TestFormatUser(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		user     *models.User
		contains []string
		notContains []string
	}{
		{
			name:     "nil user",
			user:     nil,
			contains: []string{"[User not found]"},
		},
		{
			name: "full user profile",
			user: &models.User{
				ID:        "user-123",
				Handle:    "fulluser",
				Name:      "Full User",
				Bio:       "This is my bio",
				CreatedAt: baseTime,
			},
			contains: []string{
				"@fulluser",
				"Name: Full User",
				"Bio: This is my bio",
				"ID: user-123",
				"Joined: 2024-06-01",
			},
		},
		{
			name: "user without name",
			user: &models.User{
				ID:        "user-456",
				Handle:    "noname",
				Name:      "",
				Bio:       "Has bio but no name",
				CreatedAt: baseTime,
			},
			contains:    []string{"@noname", "Bio: Has bio but no name"},
			notContains: []string{"Name:"},
		},
		{
			name: "user without bio",
			user: &models.User{
				ID:        "user-789",
				Handle:    "nobio",
				Name:      "No Bio User",
				Bio:       "",
				CreatedAt: baseTime,
			},
			contains:    []string{"@nobio", "Name: No Bio User"},
			notContains: []string{"Bio:"},
		},
		{
			name: "minimal user",
			user: &models.User{
				ID:        "user-min",
				Handle:    "minimal",
				CreatedAt: baseTime,
			},
			contains:    []string{"@minimal", "ID: user-min"},
			notContains: []string{"Name:", "Bio:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatUser(tt.user)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatUser() result missing %q\nGot: %s", want, result)
				}
			}

			for _, notWant := range tt.notContains {
				if strings.Contains(result, notWant) {
					t.Errorf("FormatUser() result should not contain %q\nGot: %s", notWant, result)
				}
			}
		})
	}
}

func TestFormatUserCompact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		user   *models.User
		expect string
	}{
		{
			name:   "nil user",
			user:   nil,
			expect: "[User not found]",
		},
		{
			name: "user with name",
			user: &models.User{
				Handle: "nameduser",
				Name:   "Named User",
			},
			expect: "@nameduser (Named User)",
		},
		{
			name: "user without name",
			user: &models.User{
				Handle: "noname",
				Name:   "",
			},
			expect: "@noname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatUserCompact(tt.user)
			if result != tt.expect {
				t.Errorf("FormatUserCompact() = %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestFormatIssue(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 2, 1, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		post      *models.Post
		issueType string
		contains  []string
	}{
		{
			name:      "nil post",
			post:      nil,
			issueType: "bug",
			contains:  []string{"[Issue not found]"},
		},
		{
			name: "bug report",
			post: &models.Post{
				ID:         "bug-123",
				Content:    "App crashes on startup",
				ReplyCount: 3,
				CreatedAt:  baseTime,
			},
			issueType: "bug",
			contains:  []string{"[BUG]", "bug-123", "App crashes on startup", "Replies: 3"},
		},
		{
			name: "feature request",
			post: &models.Post{
				ID:         "feat-456",
				Content:    "Add dark mode",
				ReplyCount: 10,
				CreatedAt:  baseTime,
			},
			issueType: "feature",
			contains:  []string{"[FEATURE]", "feat-456", "Add dark mode", "Replies: 10"},
		},
		{
			name: "unknown type",
			post: &models.Post{
				ID:         "unknown-789",
				Content:    "Some issue",
				ReplyCount: 0,
				CreatedAt:  baseTime,
			},
			issueType: "other",
			contains:  []string{"[?]", "unknown-789"},
		},
		{
			name: "empty content",
			post: &models.Post{
				ID:        "empty-issue",
				Content:   "",
				CreatedAt: baseTime,
			},
			issueType: "bug",
			contains:  []string{"[No content]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatIssue(tt.post, tt.issueType)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatIssue() result missing %q\nGot: %s", want, result)
				}
			}
		})
	}
}

func TestFormatThread(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 20, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		thread   *client.ThreadResponse
		contains []string
	}{
		{
			name:     "nil thread",
			thread:   nil,
			contains: []string{"[Thread not found]"},
		},
		{
			name: "thread without replies",
			thread: &client.ThreadResponse{
				Post: &models.Post{
					ID:      "thread-main",
					Content: "Main post content",
					Author: &models.User{
						Handle: "op",
					},
					CreatedAt: baseTime,
				},
				Replies: nil,
			},
			contains: []string{"=== Thread ===", "@op", "Main post content"},
		},
		{
			name: "thread with replies",
			thread: &client.ThreadResponse{
				Post: &models.Post{
					ID:      "thread-with-replies",
					Content: "Original post",
					Author: &models.User{
						Handle: "starter",
					},
					CreatedAt: baseTime,
				},
				Replies: []*models.Post{
					{
						ID:      "reply-1",
						Content: "First reply",
						Author: &models.User{
							Handle: "replier1",
						},
						CreatedAt: baseTime,
					},
					{
						ID:      "reply-2",
						Content: "Second reply",
						Author: &models.User{
							Handle: "replier2",
						},
						CreatedAt: baseTime,
					},
				},
			},
			contains: []string{
				"=== Thread ===",
				"Original post",
				"=== Replies ===",
				"--- Reply 1 ---",
				"First reply",
				"@replier1",
				"--- Reply 2 ---",
				"Second reply",
				"@replier2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatThread(tt.thread)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatThread() result missing %q\nGot: %s", want, result)
				}
			}
		})
	}
}

func TestFormatFeed(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 25, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		posts    []*models.Post
		feedType string
		contains []string
	}{
		{
			name:     "empty feed",
			posts:    []*models.Post{},
			feedType: "home",
			contains: []string{"No posts found."},
		},
		{
			name:     "nil posts slice",
			posts:    nil,
			feedType: "best",
			contains: []string{"No posts found."},
		},
		{
			name: "feed with posts",
			posts: []*models.Post{
				{
					ID:      "feed-1",
					Content: "First post",
					Author: &models.User{
						Handle: "user1",
					},
					CreatedAt: baseTime,
				},
				{
					ID:      "feed-2",
					Content: "Second post",
					Author: &models.User{
						Handle: "user2",
					},
					CreatedAt: baseTime,
				},
			},
			feedType: "latest",
			contains: []string{
				"=== Feed (latest, 2 posts) ===",
				"--- Post 1 ---",
				"First post",
				"--- Post 2 ---",
				"Second post",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFeed(tt.posts, tt.feedType)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatFeed() result missing %q\nGot: %s", want, result)
				}
			}
		})
	}
}

func TestFormatSearchResults(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 28, 16, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		result     *client.SearchResult
		query      string
		searchType string
		contains   []string
	}{
		{
			name: "posts search with results",
			result: &client.SearchResult{
				Posts: []*models.Post{
					{
						ID:      "search-post-1",
						Content: "Matching content",
						Author: &models.User{
							Handle: "searcher",
						},
						CreatedAt: baseTime,
					},
				},
			},
			query:      "matching",
			searchType: "posts",
			contains: []string{
				`=== Search: "matching" (type: posts) ===`,
				"--- Result 1 ---",
				"Matching content",
			},
		},
		{
			name: "posts search no results",
			result: &client.SearchResult{
				Posts: []*models.Post{},
			},
			query:      "nonexistent",
			searchType: "posts",
			contains:   []string{"No posts found."},
		},
		{
			name: "users search with results",
			result: &client.SearchResult{
				Users: []*models.User{
					{
						ID:        "user-search-1",
						Handle:    "founduser",
						Name:      "Found User",
						CreatedAt: baseTime,
					},
				},
			},
			query:      "found",
			searchType: "users",
			contains: []string{
				`=== Search: "found" (type: users) ===`,
				"@founduser",
				"Name: Found User",
			},
		},
		{
			name: "users search no results",
			result: &client.SearchResult{
				Users: []*models.User{},
			},
			query:      "nobody",
			searchType: "users",
			contains:   []string{"No users found."},
		},
		{
			name: "tags search with results",
			result: &client.SearchResult{
				Tags: []string{"golang", "go", "gopher"},
			},
			query:      "go",
			searchType: "tags",
			contains: []string{
				`=== Search: "go" (type: tags) ===`,
				"#golang",
				"#go",
				"#gopher",
			},
		},
		{
			name: "tags search no results",
			result: &client.SearchResult{
				Tags: []string{},
			},
			query:      "nothing",
			searchType: "tags",
			contains:   []string{"No tags found."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSearchResults(tt.result, tt.query, tt.searchType)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatSearchResults() result missing %q\nGot: %s", want, result)
				}
			}
		})
	}
}

func TestFormatMentions(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		posts    []*models.Post
		handle   string
		contains []string
	}{
		{
			name:     "no mentions",
			posts:    []*models.Post{},
			handle:   "unmention",
			contains: []string{"No mentions found for @unmention."},
		},
		{
			name: "with mentions",
			posts: []*models.Post{
				{
					ID:      "mention-1",
					Content: "Hey @mentioned check this out",
					Author: &models.User{
						Handle: "mentioner",
					},
					CreatedAt: baseTime,
				},
			},
			handle: "mentioned",
			contains: []string{
				"=== Mentions of @mentioned (1 posts) ===",
				"--- Mention 1 ---",
				"Hey @mentioned check this out",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMentions(tt.posts, tt.handle)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatMentions() result missing %q\nGot: %s", want, result)
				}
			}
		})
	}
}

func TestFormatIssuesList(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		posts     []*models.Post
		issueType string
		contains  []string
	}{
		{
			name:      "no bugs",
			posts:     []*models.Post{},
			issueType: "bug",
			contains:  []string{"No bugs found."},
		},
		{
			name:      "no features",
			posts:     []*models.Post{},
			issueType: "feature",
			contains:  []string{"No feature requests found."},
		},
		{
			name:      "no issues generic",
			posts:     []*models.Post{},
			issueType: "",
			contains:  []string{"No issues found."},
		},
		{
			name: "bug list",
			posts: []*models.Post{
				{
					ID:        "bug-1",
					Content:   "[BUG] Something is broken",
					CreatedAt: baseTime,
				},
			},
			issueType: "bug",
			contains:  []string{"=== Bug Reports (1) ===", "[BUG]", "Something is broken"},
		},
		{
			name: "feature list",
			posts: []*models.Post{
				{
					ID:        "feat-1",
					Content:   "[FEATURE] Add new functionality",
					CreatedAt: baseTime,
				},
			},
			issueType: "feature",
			contains:  []string{"=== Feature Requests (1) ===", "[FEATURE]", "Add new functionality"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatIssuesList(tt.posts, tt.issueType)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatIssuesList() result missing %q\nGot: %s", want, result)
				}
			}
		})
	}
}

func TestFormatAuthStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		authenticated bool
		user          *models.User
		contains      []string
	}{
		{
			name:          "not authenticated",
			authenticated: false,
			user:          nil,
			contains:      []string{"Not authenticated", "mesh_login"},
		},
		{
			name:          "authenticated but nil user",
			authenticated: true,
			user:          nil,
			contains:      []string{"Not authenticated"},
		},
		{
			name:          "authenticated with user",
			authenticated: true,
			user: &models.User{
				ID:     "user-auth",
				Handle: "authuser",
			},
			contains: []string{"Authenticated as @authuser", "User ID: user-auth"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAuthStatus(tt.authenticated, tt.user)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatAuthStatus() result missing %q\nGot: %s", want, result)
				}
			}
		})
	}
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
