package mcp

import (
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func TestToolDefinitions(t *testing.T) {
	t.Parallel()

	tools := ToolDefinitions()

	if len(tools) == 0 {
		t.Fatal("ToolDefinitions() returned empty slice")
	}

	// Expected tool names
	expectedTools := []string{
		"mesh_login",
		"mesh_status",
		"mesh_feed",
		"mesh_user",
		"mesh_thread",
		"mesh_search",
		"mesh_mentions",
		"mesh_post",
		"mesh_reply",
		"mesh_follow",
		"mesh_unfollow",
		"mesh_like",
		"mesh_unlike",
		"mesh_report_bug",
		"mesh_request_feature",
		"mesh_list_issues",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("ToolDefinitions() returned %d tools, want %d", len(tools), len(expectedTools))
	}

	// Create map of returned tools
	toolMap := make(map[string]mcplib.Tool)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}

	// Check each expected tool exists
	for _, name := range expectedTools {
		if _, ok := toolMap[name]; !ok {
			t.Errorf("missing expected tool: %s", name)
		}
	}
}

func TestToolDefinitions_ToolProperties(t *testing.T) {
	t.Parallel()

	tools := ToolDefinitions()
	toolMap := make(map[string]mcplib.Tool)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}

	tests := []struct {
		name              string
		hasDescription    bool
		requiredParams    []string
		optionalParams    []string
	}{
		{
			name:           "mesh_login",
			hasDescription: true,
			requiredParams: []string{"handle"},
			optionalParams: []string{"key_path"},
		},
		{
			name:           "mesh_status",
			hasDescription: true,
			requiredParams: []string{},
			optionalParams: []string{},
		},
		{
			name:           "mesh_feed",
			hasDescription: true,
			requiredParams: []string{},
			optionalParams: []string{"limit", "type"},
		},
		{
			name:           "mesh_user",
			hasDescription: true,
			requiredParams: []string{"handle"},
			optionalParams: []string{"include_posts"},
		},
		{
			name:           "mesh_thread",
			hasDescription: true,
			requiredParams: []string{"post_id"},
			optionalParams: []string{},
		},
		{
			name:           "mesh_search",
			hasDescription: true,
			requiredParams: []string{"query"},
			optionalParams: []string{"type", "limit"},
		},
		{
			name:           "mesh_mentions",
			hasDescription: true,
			requiredParams: []string{"handle"},
			optionalParams: []string{"limit"},
		},
		{
			name:           "mesh_post",
			hasDescription: true,
			requiredParams: []string{"content"},
			optionalParams: []string{"visibility"},
		},
		{
			name:           "mesh_reply",
			hasDescription: true,
			requiredParams: []string{"post_id", "content"},
			optionalParams: []string{},
		},
		{
			name:           "mesh_follow",
			hasDescription: true,
			requiredParams: []string{"handle"},
			optionalParams: []string{},
		},
		{
			name:           "mesh_unfollow",
			hasDescription: true,
			requiredParams: []string{"handle"},
			optionalParams: []string{},
		},
		{
			name:           "mesh_like",
			hasDescription: true,
			requiredParams: []string{"post_id"},
			optionalParams: []string{},
		},
		{
			name:           "mesh_unlike",
			hasDescription: true,
			requiredParams: []string{"post_id"},
			optionalParams: []string{},
		},
		{
			name:           "mesh_report_bug",
			hasDescription: true,
			requiredParams: []string{"title"},
			optionalParams: []string{"description"},
		},
		{
			name:           "mesh_request_feature",
			hasDescription: true,
			requiredParams: []string{"title"},
			optionalParams: []string{"description"},
		},
		{
			name:           "mesh_list_issues",
			hasDescription: true,
			requiredParams: []string{},
			optionalParams: []string{"type", "status", "limit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, ok := toolMap[tt.name]
			if !ok {
				t.Fatalf("tool %s not found", tt.name)
			}

			// Check description
			if tt.hasDescription && tool.Description == "" {
				t.Errorf("tool %s missing description", tt.name)
			}

			// Check input schema exists
			if tool.InputSchema.Type != "object" {
				t.Errorf("tool %s has unexpected input schema type: %s", tt.name, tool.InputSchema.Type)
			}

			// Check required params
			requiredSet := make(map[string]bool)
			for _, req := range tool.InputSchema.Required {
				requiredSet[req] = true
			}

			for _, param := range tt.requiredParams {
				if !requiredSet[param] {
					t.Errorf("tool %s: expected required param %q not found in required list", tt.name, param)
				}
			}

			// Check all expected params exist in properties
			if tool.InputSchema.Properties != nil {
				for _, param := range append(tt.requiredParams, tt.optionalParams...) {
					if _, ok := tool.InputSchema.Properties[param]; !ok {
						t.Errorf("tool %s: expected param %q not found in properties", tt.name, param)
					}
				}
			} else if len(tt.requiredParams) > 0 || len(tt.optionalParams) > 0 {
				t.Errorf("tool %s: expected properties but got nil", tt.name)
			}
		})
	}
}

func TestToolLogin(t *testing.T) {
	t.Parallel()

	tool := toolLogin()

	if tool.Name != "mesh_login" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_login")
	}

	if tool.Description == "" {
		t.Error("tool.Description should not be empty")
	}

	// Check handle param
	if tool.InputSchema.Properties == nil {
		t.Fatal("tool.InputSchema.Properties is nil")
	}

	if _, ok := tool.InputSchema.Properties["handle"]; !ok {
		t.Error("handle property not found")
	}

	// Check handle is required
	hasHandle := false
	for _, req := range tool.InputSchema.Required {
		if req == "handle" {
			hasHandle = true
			break
		}
	}
	if !hasHandle {
		t.Error("handle should be required")
	}

	// Check key_path is optional (present in properties but not required)
	if _, ok := tool.InputSchema.Properties["key_path"]; !ok {
		t.Error("key_path property not found")
	}

	for _, req := range tool.InputSchema.Required {
		if req == "key_path" {
			t.Error("key_path should not be required")
		}
	}
}

func TestToolStatus(t *testing.T) {
	t.Parallel()

	tool := toolStatus()

	if tool.Name != "mesh_status" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_status")
	}

	if tool.Description == "" {
		t.Error("tool.Description should not be empty")
	}

	// Status should have no required params
	if len(tool.InputSchema.Required) != 0 {
		t.Errorf("mesh_status should have no required params, got %v", tool.InputSchema.Required)
	}
}

func TestToolFeed(t *testing.T) {
	t.Parallel()

	tool := toolFeed()

	if tool.Name != "mesh_feed" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_feed")
	}

	// Check type param exists
	if _, ok := tool.InputSchema.Properties["type"]; !ok {
		t.Error("type property not found")
	}

	// Check limit param exists
	if _, ok := tool.InputSchema.Properties["limit"]; !ok {
		t.Error("limit property not found")
	}
}

func TestToolSearch(t *testing.T) {
	t.Parallel()

	tool := toolSearch()

	if tool.Name != "mesh_search" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_search")
	}

	// Check query is required
	hasQuery := false
	for _, req := range tool.InputSchema.Required {
		if req == "query" {
			hasQuery = true
			break
		}
	}
	if !hasQuery {
		t.Error("query should be required")
	}

	// Check type and limit params exist
	if _, ok := tool.InputSchema.Properties["type"]; !ok {
		t.Error("type property not found")
	}
	if _, ok := tool.InputSchema.Properties["limit"]; !ok {
		t.Error("limit property not found")
	}
}

func TestToolPost(t *testing.T) {
	t.Parallel()

	tool := toolPost()

	if tool.Name != "mesh_post" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_post")
	}

	// Check content is required
	hasContent := false
	for _, req := range tool.InputSchema.Required {
		if req == "content" {
			hasContent = true
			break
		}
	}
	if !hasContent {
		t.Error("content should be required")
	}

	// Check visibility param exists
	if _, ok := tool.InputSchema.Properties["visibility"]; !ok {
		t.Error("visibility property not found")
	}
}

func TestToolReply(t *testing.T) {
	t.Parallel()

	tool := toolReply()

	if tool.Name != "mesh_reply" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_reply")
	}

	// Check both post_id and content are required
	requiredSet := make(map[string]bool)
	for _, req := range tool.InputSchema.Required {
		requiredSet[req] = true
	}

	if !requiredSet["post_id"] {
		t.Error("post_id should be required")
	}
	if !requiredSet["content"] {
		t.Error("content should be required")
	}
}

func TestToolListIssues(t *testing.T) {
	t.Parallel()

	tool := toolListIssues()

	if tool.Name != "mesh_list_issues" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_list_issues")
	}

	// Check type, status, and limit params exist
	if _, ok := tool.InputSchema.Properties["type"]; !ok {
		t.Error("type property not found")
	}
	if _, ok := tool.InputSchema.Properties["status"]; !ok {
		t.Error("status property not found")
	}
	if _, ok := tool.InputSchema.Properties["limit"]; !ok {
		t.Error("limit property not found")
	}
}

func TestToolFollow(t *testing.T) {
	t.Parallel()

	tool := toolFollow()

	if tool.Name != "mesh_follow" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_follow")
	}

	// Check handle is required
	hasHandle := false
	for _, req := range tool.InputSchema.Required {
		if req == "handle" {
			hasHandle = true
			break
		}
	}
	if !hasHandle {
		t.Error("handle should be required")
	}
}

func TestToolUnfollow(t *testing.T) {
	t.Parallel()

	tool := toolUnfollow()

	if tool.Name != "mesh_unfollow" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_unfollow")
	}

	// Check handle is required
	hasHandle := false
	for _, req := range tool.InputSchema.Required {
		if req == "handle" {
			hasHandle = true
			break
		}
	}
	if !hasHandle {
		t.Error("handle should be required")
	}
}

func TestToolLike(t *testing.T) {
	t.Parallel()

	tool := toolLike()

	if tool.Name != "mesh_like" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_like")
	}

	// Check post_id is required
	hasPostID := false
	for _, req := range tool.InputSchema.Required {
		if req == "post_id" {
			hasPostID = true
			break
		}
	}
	if !hasPostID {
		t.Error("post_id should be required")
	}
}

func TestToolUnlike(t *testing.T) {
	t.Parallel()

	tool := toolUnlike()

	if tool.Name != "mesh_unlike" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_unlike")
	}

	// Check post_id is required
	hasPostID := false
	for _, req := range tool.InputSchema.Required {
		if req == "post_id" {
			hasPostID = true
			break
		}
	}
	if !hasPostID {
		t.Error("post_id should be required")
	}
}

func TestToolUser(t *testing.T) {
	t.Parallel()

	tool := toolUser()

	if tool.Name != "mesh_user" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_user")
	}

	// Check handle is required
	hasHandle := false
	for _, req := range tool.InputSchema.Required {
		if req == "handle" {
			hasHandle = true
			break
		}
	}
	if !hasHandle {
		t.Error("handle should be required")
	}

	// Check include_posts param exists
	if _, ok := tool.InputSchema.Properties["include_posts"]; !ok {
		t.Error("include_posts property not found")
	}
}

func TestToolThread(t *testing.T) {
	t.Parallel()

	tool := toolThread()

	if tool.Name != "mesh_thread" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_thread")
	}

	// Check post_id is required
	hasPostID := false
	for _, req := range tool.InputSchema.Required {
		if req == "post_id" {
			hasPostID = true
			break
		}
	}
	if !hasPostID {
		t.Error("post_id should be required")
	}
}

func TestToolMentions(t *testing.T) {
	t.Parallel()

	tool := toolMentions()

	if tool.Name != "mesh_mentions" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_mentions")
	}

	// Check handle is required
	hasHandle := false
	for _, req := range tool.InputSchema.Required {
		if req == "handle" {
			hasHandle = true
			break
		}
	}
	if !hasHandle {
		t.Error("handle should be required")
	}

	// Check limit param exists
	if _, ok := tool.InputSchema.Properties["limit"]; !ok {
		t.Error("limit property not found")
	}
}

func TestToolReportBug(t *testing.T) {
	t.Parallel()

	tool := toolReportBug()

	if tool.Name != "mesh_report_bug" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_report_bug")
	}

	// Check title is required
	hasTitle := false
	for _, req := range tool.InputSchema.Required {
		if req == "title" {
			hasTitle = true
			break
		}
	}
	if !hasTitle {
		t.Error("title should be required")
	}

	// Check description param exists
	if _, ok := tool.InputSchema.Properties["description"]; !ok {
		t.Error("description property not found")
	}
}

func TestToolRequestFeature(t *testing.T) {
	t.Parallel()

	tool := toolRequestFeature()

	if tool.Name != "mesh_request_feature" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "mesh_request_feature")
	}

	// Check title is required
	hasTitle := false
	for _, req := range tool.InputSchema.Required {
		if req == "title" {
			hasTitle = true
			break
		}
	}
	if !hasTitle {
		t.Error("title should be required")
	}

	// Check description param exists
	if _, ok := tool.InputSchema.Properties["description"]; !ok {
		t.Error("description property not found")
	}
}
