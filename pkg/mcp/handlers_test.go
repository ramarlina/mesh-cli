package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/ramarlina/mesh-cli/pkg/models"
)

// mockRequest creates a CallToolRequest with the given arguments.
func mockRequest(name string, args map[string]any) mcplib.CallToolRequest {
	return mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

// mockServer creates a test HTTP server that returns specified responses.
type mockServer struct {
	*httptest.Server
	responses map[string]mockResponse
}

type mockResponse struct {
	statusCode int
	body       any
}

func newMockServer() *mockServer {
	ms := &mockServer{
		responses: make(map[string]mockResponse),
	}

	ms.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path

		resp, ok := ms.responses[key]
		if !ok {
			// Try with query string
			key = r.Method + " " + r.URL.String()
			resp, ok = ms.responses[key]
		}

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.statusCode)
		json.NewEncoder(w).Encode(resp.body)
	}))

	return ms
}

func (ms *mockServer) setResponse(method, path string, statusCode int, body any) {
	ms.responses[method+" "+path] = mockResponse{
		statusCode: statusCode,
		body:       body,
	}
}

func TestNewHandlers(t *testing.T) {
	t.Parallel()

	auth := &AuthState{}
	handlers := NewHandlers(auth)

	if handlers == nil {
		t.Fatal("NewHandlers returned nil")
	}

	if handlers.auth != auth {
		t.Error("handlers.auth not set correctly")
	}
}

func TestHandleStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("not authenticated", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_status", nil)
		result, err := handlers.HandleStatus(ctx, req)

		if err != nil {
			t.Fatalf("HandleStatus() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Not authenticated") {
			t.Errorf("expected 'Not authenticated', got %q", text)
		}
	})

	t.Run("authenticated valid session", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("GET", "/v1/auth/status", 200, models.User{
			ID:     "user-123",
			Handle: "testuser",
		})

		auth := NewAuthState(ms.URL)
		auth.SetAuth("valid-token", &models.User{ID: "user-123", Handle: "testuser"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_status", nil)
		result, err := handlers.HandleStatus(ctx, req)

		if err != nil {
			t.Fatalf("HandleStatus() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "@testuser") {
			t.Errorf("expected '@testuser' in result, got %q", text)
		}
		if !strings.Contains(text, "user-123") {
			t.Errorf("expected 'user-123' in result, got %q", text)
		}
	})

	t.Run("authenticated expired session", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("GET", "/v1/auth/status", 401, map[string]string{
			"error": "unauthorized",
		})

		auth := NewAuthState(ms.URL)
		auth.SetAuth("expired-token", &models.User{ID: "user-123", Handle: "testuser"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_status", nil)
		result, err := handlers.HandleStatus(ctx, req)

		if err != nil {
			t.Fatalf("HandleStatus() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "expired") {
			t.Errorf("expected 'expired' in result, got %q", text)
		}

		// Auth state should be cleared
		if auth.IsAuthenticated() {
			t.Error("expected auth to be cleared after expired session")
		}
	})
}

func TestHandleFeed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ms := newMockServer()
	defer ms.Close()

	posts := []models.Post{
		{
			ID:      "post-1",
			Content: "First post",
			Author:  &models.User{Handle: "user1"},
			CreatedAt: baseTime,
		},
		{
			ID:      "post-2",
			Content: "Second post",
			Author:  &models.User{Handle: "user2"},
			CreatedAt: baseTime,
		},
	}

	// Convert to response format
	postsResp := make([]*models.Post, len(posts))
	for i := range posts {
		postsResp[i] = &posts[i]
	}

	ms.setResponse("GET", "/v1/feed?type=latest&limit=20", 200, map[string]any{
		"posts": postsResp,
	})
	ms.setResponse("GET", "/v1/feed?type=home&limit=10", 200, map[string]any{
		"posts": postsResp,
	})
	ms.setResponse("GET", "/v1/feed?type=best&limit=20", 200, map[string]any{
		"posts": postsResp,
	})

	auth := NewAuthState(ms.URL)
	handlers := NewHandlers(auth)

	tests := []struct {
		name     string
		args     map[string]any
		contains []string
	}{
		{
			name:     "default feed",
			args:     nil,
			contains: []string{"Feed (latest, 2 posts)", "First post", "Second post"},
		},
		{
			name:     "home feed with limit",
			args:     map[string]any{"type": "home", "limit": 10},
			contains: []string{"First post", "Second post"},
		},
		{
			name:     "best feed",
			args:     map[string]any{"type": "best"},
			contains: []string{"First post"},
		},
		{
			name:     "negative limit uses default",
			args:     map[string]any{"limit": -5},
			contains: []string{"First post"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mockRequest("mesh_feed", tt.args)
			result, err := handlers.HandleFeed(ctx, req)

			if err != nil {
				t.Fatalf("HandleFeed() error = %v", err)
			}

			text := getResultText(t, result)
			for _, want := range tt.contains {
				if !strings.Contains(text, want) {
					t.Errorf("result missing %q\nGot: %s", want, text)
				}
			}
		})
	}
}

func TestHandleUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("missing handle", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_user", nil)
		result, err := handlers.HandleUser(ctx, req)

		if err != nil {
			t.Fatalf("HandleUser() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for missing handle")
		}
	})

	t.Run("successful user fetch", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("GET", "/v1/users/testuser", 200, models.User{
			ID:        "user-123",
			Handle:    "testuser",
			Name:      "Test User",
			Bio:       "Test bio",
			CreatedAt: baseTime,
		})

		ms.setResponse("GET", "/v1/users/testuser/posts?limit=5", 200, map[string]any{
			"posts": []*models.Post{
				{
					ID:        "post-1",
					Content:   "User's post",
					Author:    &models.User{Handle: "testuser"},
					CreatedAt: baseTime,
				},
			},
		})

		auth := NewAuthState(ms.URL)
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_user", map[string]any{"handle": "testuser"})
		result, err := handlers.HandleUser(ctx, req)

		if err != nil {
			t.Fatalf("HandleUser() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "@testuser") {
			t.Errorf("expected '@testuser', got %q", text)
		}
		if !strings.Contains(text, "Test User") {
			t.Errorf("expected 'Test User', got %q", text)
		}
		if !strings.Contains(text, "User's post") {
			t.Errorf("expected user's post in output, got %q", text)
		}
	})

	t.Run("handle with @ prefix", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("GET", "/v1/users/cleanhandle", 200, models.User{
			ID:        "user-456",
			Handle:    "cleanhandle",
			CreatedAt: baseTime,
		})

		ms.setResponse("GET", "/v1/users/cleanhandle/posts?limit=5", 200, map[string]any{
			"posts": []*models.Post{},
		})

		auth := NewAuthState(ms.URL)
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_user", map[string]any{"handle": "@cleanhandle"})
		result, err := handlers.HandleUser(ctx, req)

		if err != nil {
			t.Fatalf("HandleUser() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "@cleanhandle") {
			t.Errorf("expected '@cleanhandle', got %q", text)
		}
	})

	t.Run("include_posts false", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("GET", "/v1/users/nopostuser", 200, models.User{
			ID:        "user-789",
			Handle:    "nopostuser",
			CreatedAt: baseTime,
		})

		auth := NewAuthState(ms.URL)
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_user", map[string]any{
			"handle":        "nopostuser",
			"include_posts": false,
		})
		result, err := handlers.HandleUser(ctx, req)

		if err != nil {
			t.Fatalf("HandleUser() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "@nopostuser") {
			t.Errorf("expected '@nopostuser', got %q", text)
		}
		// Should not contain posts section
		if strings.Contains(text, "Recent Posts") {
			t.Errorf("should not include posts section when include_posts=false, got %q", text)
		}
	})
}

func TestHandleThread(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("missing post_id", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_thread", nil)
		result, err := handlers.HandleThread(ctx, req)

		if err != nil {
			t.Fatalf("HandleThread() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for missing post_id")
		}
	})

	t.Run("successful thread fetch", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("GET", "/v1/posts/post-123/thread", 200, map[string]any{
			"post": models.Post{
				ID:        "post-123",
				Content:   "Main thread post",
				Author:    &models.User{Handle: "op"},
				CreatedAt: baseTime,
			},
			"replies": []models.Post{
				{
					ID:        "reply-1",
					Content:   "First reply",
					Author:    &models.User{Handle: "replier"},
					CreatedAt: baseTime,
				},
			},
		})

		auth := NewAuthState(ms.URL)
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_thread", map[string]any{"post_id": "post-123"})
		result, err := handlers.HandleThread(ctx, req)

		if err != nil {
			t.Fatalf("HandleThread() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Main thread post") {
			t.Errorf("expected main post content, got %q", text)
		}
		if !strings.Contains(text, "First reply") {
			t.Errorf("expected reply content, got %q", text)
		}
	})
}

func TestHandleSearch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("missing query", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_search", nil)
		result, err := handlers.HandleSearch(ctx, req)

		if err != nil {
			t.Fatalf("HandleSearch() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for missing query")
		}
	})

	t.Run("search posts", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("GET", "/v1/search?q=golang&type=posts&limit=20", 200, map[string]any{
			"posts": []models.Post{
				{
					ID:        "post-go-1",
					Content:   "Golang is great",
					Author:    &models.User{Handle: "gopher"},
					CreatedAt: baseTime,
				},
			},
		})

		auth := NewAuthState(ms.URL)
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_search", map[string]any{"query": "golang"})
		result, err := handlers.HandleSearch(ctx, req)

		if err != nil {
			t.Fatalf("HandleSearch() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Golang is great") {
			t.Errorf("expected search result, got %q", text)
		}
	})

	t.Run("search users", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("GET", "/v1/search?q=john&type=users&limit=20", 200, map[string]any{
			"users": []models.User{
				{
					ID:        "user-john",
					Handle:    "john",
					Name:      "John Doe",
					CreatedAt: baseTime,
				},
			},
		})

		auth := NewAuthState(ms.URL)
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_search", map[string]any{
			"query": "john",
			"type":  "users",
		})
		result, err := handlers.HandleSearch(ctx, req)

		if err != nil {
			t.Fatalf("HandleSearch() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "@john") {
			t.Errorf("expected '@john', got %q", text)
		}
	})
}

func TestHandleMentions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("missing handle", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_mentions", nil)
		result, err := handlers.HandleMentions(ctx, req)

		if err != nil {
			t.Fatalf("HandleMentions() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for missing handle")
		}
	})

	t.Run("successful mentions fetch", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("GET", "/v1/users/mentioned/mentions?limit=20", 200, map[string]any{
			"posts": []models.Post{
				{
					ID:        "mention-1",
					Content:   "Hey @mentioned check this out",
					Author:    &models.User{Handle: "mentioner"},
					CreatedAt: baseTime,
				},
			},
		})

		auth := NewAuthState(ms.URL)
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_mentions", map[string]any{"handle": "mentioned"})
		result, err := handlers.HandleMentions(ctx, req)

		if err != nil {
			t.Fatalf("HandleMentions() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Hey @mentioned") {
			t.Errorf("expected mention content, got %q", text)
		}
	})
}

func TestHandlePost(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("not authenticated", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_post", map[string]any{"content": "Test post"})
		result, err := handlers.HandlePost(ctx, req)

		if err != nil {
			t.Fatalf("HandlePost() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for unauthenticated post")
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Not authenticated") {
			t.Errorf("expected 'Not authenticated', got %q", text)
		}
	})

	t.Run("missing content", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "poster"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_post", nil)
		result, err := handlers.HandlePost(ctx, req)

		if err != nil {
			t.Fatalf("HandlePost() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for missing content")
		}
	})

	t.Run("successful post", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("POST", "/v1/posts", 201, models.Post{
			ID:        "post-new",
			Content:   "My new post",
			Author:    &models.User{Handle: "poster"},
			CreatedAt: baseTime,
		})

		auth := NewAuthState(ms.URL)
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "poster"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_post", map[string]any{"content": "My new post"})
		result, err := handlers.HandlePost(ctx, req)

		if err != nil {
			t.Fatalf("HandlePost() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Posted successfully") {
			t.Errorf("expected success message, got %q", text)
		}
		if !strings.Contains(text, "My new post") {
			t.Errorf("expected post content, got %q", text)
		}
	})
}

func TestHandleReply(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("not authenticated", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_reply", map[string]any{
			"post_id": "post-123",
			"content": "Reply",
		})
		result, err := handlers.HandleReply(ctx, req)

		if err != nil {
			t.Fatalf("HandleReply() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for unauthenticated reply")
		}
	})

	t.Run("missing post_id", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "replier"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_reply", map[string]any{"content": "Reply"})
		result, err := handlers.HandleReply(ctx, req)

		if err != nil {
			t.Fatalf("HandleReply() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for missing post_id")
		}
	})

	t.Run("missing content", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "replier"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_reply", map[string]any{"post_id": "post-123"})
		result, err := handlers.HandleReply(ctx, req)

		if err != nil {
			t.Fatalf("HandleReply() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for missing content")
		}
	})

	t.Run("successful reply", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("POST", "/v1/posts", 201, models.Post{
			ID:        "reply-new",
			Content:   "My reply",
			Author:    &models.User{Handle: "replier"},
			ReplyTo:   strPtr("post-123"),
			CreatedAt: baseTime,
		})

		auth := NewAuthState(ms.URL)
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "replier"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_reply", map[string]any{
			"post_id": "post-123",
			"content": "My reply",
		})
		result, err := handlers.HandleReply(ctx, req)

		if err != nil {
			t.Fatalf("HandleReply() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Replied to post-123") {
			t.Errorf("expected success message, got %q", text)
		}
	})
}

func TestHandleFollow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("not authenticated", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_follow", map[string]any{"handle": "someuser"})
		result, err := handlers.HandleFollow(ctx, req)

		if err != nil {
			t.Fatalf("HandleFollow() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for unauthenticated follow")
		}
	})

	t.Run("missing handle", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "follower"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_follow", nil)
		result, err := handlers.HandleFollow(ctx, req)

		if err != nil {
			t.Fatalf("HandleFollow() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for missing handle")
		}
	})

	t.Run("successful follow", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("POST", "/v1/users/target/follow", 200, map[string]string{})

		auth := NewAuthState(ms.URL)
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "follower"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_follow", map[string]any{"handle": "target"})
		result, err := handlers.HandleFollow(ctx, req)

		if err != nil {
			t.Fatalf("HandleFollow() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "following @target") {
			t.Errorf("expected success message, got %q", text)
		}
	})

	t.Run("follow with @ prefix", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("POST", "/v1/users/prefixed/follow", 200, map[string]string{})

		auth := NewAuthState(ms.URL)
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "follower"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_follow", map[string]any{"handle": "@prefixed"})
		result, err := handlers.HandleFollow(ctx, req)

		if err != nil {
			t.Fatalf("HandleFollow() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "@prefixed") {
			t.Errorf("expected '@prefixed', got %q", text)
		}
	})
}

func TestHandleUnfollow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("not authenticated", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_unfollow", map[string]any{"handle": "someuser"})
		result, err := handlers.HandleUnfollow(ctx, req)

		if err != nil {
			t.Fatalf("HandleUnfollow() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for unauthenticated unfollow")
		}
	})

	t.Run("successful unfollow", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("DELETE", "/v1/users/target/follow", 200, map[string]string{})

		auth := NewAuthState(ms.URL)
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "unfollower"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_unfollow", map[string]any{"handle": "target"})
		result, err := handlers.HandleUnfollow(ctx, req)

		if err != nil {
			t.Fatalf("HandleUnfollow() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Unfollowed @target") {
			t.Errorf("expected success message, got %q", text)
		}
	})
}

func TestHandleLike(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("not authenticated", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_like", map[string]any{"post_id": "post-123"})
		result, err := handlers.HandleLike(ctx, req)

		if err != nil {
			t.Fatalf("HandleLike() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for unauthenticated like")
		}
	})

	t.Run("missing post_id", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "liker"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_like", nil)
		result, err := handlers.HandleLike(ctx, req)

		if err != nil {
			t.Fatalf("HandleLike() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for missing post_id")
		}
	})

	t.Run("successful like", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("POST", "/v1/posts/post-123/like", 200, map[string]string{})

		auth := NewAuthState(ms.URL)
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "liker"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_like", map[string]any{"post_id": "post-123"})
		result, err := handlers.HandleLike(ctx, req)

		if err != nil {
			t.Fatalf("HandleLike() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Liked post-123") {
			t.Errorf("expected success message, got %q", text)
		}
	})
}

func TestHandleUnlike(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("not authenticated", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_unlike", map[string]any{"post_id": "post-123"})
		result, err := handlers.HandleUnlike(ctx, req)

		if err != nil {
			t.Fatalf("HandleUnlike() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for unauthenticated unlike")
		}
	})

	t.Run("successful unlike", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("DELETE", "/v1/posts/post-123/like", 200, map[string]string{})

		auth := NewAuthState(ms.URL)
		auth.SetAuth("token", &models.User{ID: "user-1", Handle: "unliker"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_unlike", map[string]any{"post_id": "post-123"})
		result, err := handlers.HandleUnlike(ctx, req)

		if err != nil {
			t.Fatalf("HandleUnlike() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Unliked post-123") {
			t.Errorf("expected success message, got %q", text)
		}
	})
}

func TestHandleReportBug(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("missing meshbot token", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_report_bug", map[string]any{"title": "Bug title"})
		result, err := handlers.HandleReportBug(ctx, req)

		if err != nil {
			t.Fatalf("HandleReportBug() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result when meshbot token not configured")
		}
	})

	t.Run("missing title", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		auth.meshbotToken = "meshbot-token"
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_report_bug", nil)
		result, err := handlers.HandleReportBug(ctx, req)

		if err != nil {
			t.Fatalf("HandleReportBug() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result for missing title")
		}
	})

	t.Run("successful bug report anonymous", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("POST", "/v1/posts", 201, models.Post{
			ID:        "bug-post",
			Content:   "[BUG] App crashes",
			Author:    &models.User{Handle: "meshbot"},
			CreatedAt: baseTime,
		})

		auth := NewAuthState(ms.URL)
		auth.meshbotToken = "meshbot-token"
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_report_bug", map[string]any{
			"title":       "App crashes",
			"description": "When clicking button X",
		})
		result, err := handlers.HandleReportBug(ctx, req)

		if err != nil {
			t.Fatalf("HandleReportBug() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Bug report filed") {
			t.Errorf("expected success message, got %q", text)
		}
	})

	t.Run("successful bug report authenticated", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("POST", "/v1/posts", 201, models.Post{
			ID:        "bug-post-2",
			Content:   "[BUG] Something broken\nReported by @reporter",
			Author:    &models.User{Handle: "meshbot"},
			CreatedAt: baseTime,
		})

		auth := NewAuthState(ms.URL)
		auth.meshbotToken = "meshbot-token"
		auth.SetAuth("user-token", &models.User{ID: "user-1", Handle: "reporter"})
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_report_bug", map[string]any{
			"title": "Something broken",
		})
		result, err := handlers.HandleReportBug(ctx, req)

		if err != nil {
			t.Fatalf("HandleReportBug() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Bug report filed") {
			t.Errorf("expected success message, got %q", text)
		}
	})
}

func TestHandleRequestFeature(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("missing meshbot token", func(t *testing.T) {
		auth := NewAuthState("http://localhost")
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_request_feature", map[string]any{"title": "Feature title"})
		result, err := handlers.HandleRequestFeature(ctx, req)

		if err != nil {
			t.Fatalf("HandleRequestFeature() error = %v", err)
		}

		if !isErrorResult(result) {
			t.Error("expected error result when meshbot token not configured")
		}
	})

	t.Run("successful feature request", func(t *testing.T) {
		ms := newMockServer()
		defer ms.Close()

		ms.setResponse("POST", "/v1/posts", 201, models.Post{
			ID:        "feature-post",
			Content:   "[FEATURE] Dark mode",
			Author:    &models.User{Handle: "meshbot"},
			CreatedAt: baseTime,
		})

		auth := NewAuthState(ms.URL)
		auth.meshbotToken = "meshbot-token"
		handlers := NewHandlers(auth)

		req := mockRequest("mesh_request_feature", map[string]any{
			"title":       "Dark mode",
			"description": "Would be nice to have dark mode",
		})
		result, err := handlers.HandleRequestFeature(ctx, req)

		if err != nil {
			t.Fatalf("HandleRequestFeature() error = %v", err)
		}

		text := getResultText(t, result)
		if !strings.Contains(text, "Feature request submitted") {
			t.Errorf("expected success message, got %q", text)
		}
	})
}

func TestHandleListIssues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	ms := newMockServer()
	defer ms.Close()

	posts := []models.Post{
		{
			ID:        "bug-1",
			Content:   "[BUG] Something broken",
			Author:    &models.User{Handle: "meshbot"},
			CreatedAt: baseTime,
		},
		{
			ID:        "feature-1",
			Content:   "[FEATURE] Add dark mode",
			Author:    &models.User{Handle: "meshbot"},
			CreatedAt: baseTime,
		},
		{
			ID:        "regular-post",
			Content:   "Regular post",
			Author:    &models.User{Handle: "meshbot"},
			CreatedAt: baseTime,
		},
	}

	postsResp := make([]*models.Post, len(posts))
	for i := range posts {
		postsResp[i] = &posts[i]
	}

	ms.setResponse("GET", "/v1/users/meshbot/posts?limit=20", 200, map[string]any{
		"posts": postsResp,
	})

	auth := NewAuthState(ms.URL)
	handlers := NewHandlers(auth)

	tests := []struct {
		name        string
		args        map[string]any
		contains    []string
		notContains []string
	}{
		{
			name:     "all issues",
			args:     nil,
			contains: []string{"[BUG]", "[FEATURE]"},
			notContains: []string{"Regular post"},
		},
		{
			name:        "bugs only",
			args:        map[string]any{"type": "bug"},
			contains:    []string{"[BUG]"},
			notContains: []string{"[FEATURE]", "Regular post"},
		},
		{
			name:        "features only",
			args:        map[string]any{"type": "feature"},
			contains:    []string{"[FEATURE]"},
			notContains: []string{"[BUG]", "Regular post"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mockRequest("mesh_list_issues", tt.args)
			result, err := handlers.HandleListIssues(ctx, req)

			if err != nil {
				t.Fatalf("HandleListIssues() error = %v", err)
			}

			text := getResultText(t, result)
			for _, want := range tt.contains {
				if !strings.Contains(text, want) {
					t.Errorf("expected %q in result, got %q", want, text)
				}
			}
			for _, notWant := range tt.notContains {
				if strings.Contains(text, notWant) {
					t.Errorf("did not expect %q in result, got %q", notWant, text)
				}
			}
		})
	}
}

// Helper functions

func getResultText(t *testing.T, result *mcplib.CallToolResult) string {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}

	for _, content := range result.Content {
		if text, ok := content.(mcplib.TextContent); ok {
			return text.Text
		}
	}

	return fmt.Sprintf("unexpected content type: %T", result.Content)
}

func isErrorResult(result *mcplib.CallToolResult) bool {
	if result == nil {
		return false
	}
	return result.IsError
}
