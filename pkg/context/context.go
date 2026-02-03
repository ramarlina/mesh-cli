// Package context manages the 'this' keyword resolution for CLI commands.
package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// ContextTTL is the time-to-live for context entries (1 hour)
	ContextTTL = time.Hour
)

var (
	mu          sync.RWMutex
	globalCtx   *Context
	contextPath string
)

// Context represents the current CLI context.
type Context struct {
	LastID    string    `json:"last_id"`
	LastType  string    `json:"last_type"` // "post", "asset", "user", etc.
	UpdatedAt time.Time `json:"updated_at"`
}

// Load reads the context from disk.
func Load() (*Context, error) {
	mu.Lock()
	defer mu.Unlock()

	if globalCtx != nil {
		// Check if context has expired
		if time.Since(globalCtx.UpdatedAt) > ContextTTL {
			globalCtx = nil
		} else {
			return globalCtx, nil
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	mshDir := filepath.Join(homeDir, ".msh")
	if err := os.MkdirAll(mshDir, 0700); err != nil {
		return nil, fmt.Errorf("create .msh directory: %w", err)
	}

	contextPath = filepath.Join(mshDir, "context.json")

	// Check if context file exists
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no context available")
	}

	// Load existing context
	data, err := os.ReadFile(contextPath)
	if err != nil {
		return nil, fmt.Errorf("read context file: %w", err)
	}

	var ctx Context
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("parse context: %w", err)
	}

	// Check if context has expired
	if time.Since(ctx.UpdatedAt) > ContextTTL {
		return nil, fmt.Errorf("context expired")
	}

	globalCtx = &ctx
	return globalCtx, nil
}

// Save persists the context to disk.
func Save(ctx *Context) error {
	mu.Lock()
	defer mu.Unlock()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	mshDir := filepath.Join(homeDir, ".msh")
	if err := os.MkdirAll(mshDir, 0700); err != nil {
		return fmt.Errorf("create .msh directory: %w", err)
	}

	contextPath = filepath.Join(mshDir, "context.json")

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}

	if err := os.WriteFile(contextPath, data, 0600); err != nil {
		return fmt.Errorf("write context file: %w", err)
	}

	globalCtx = ctx
	return nil
}

// Clear removes the context from disk and memory.
func Clear() error {
	mu.Lock()
	defer mu.Unlock()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	contextPath = filepath.Join(homeDir, ".msh", "context.json")

	// Remove file if it exists
	if _, err := os.Stat(contextPath); err == nil {
		if err := os.Remove(contextPath); err != nil {
			return fmt.Errorf("remove context file: %w", err)
		}
	}

	globalCtx = nil
	return nil
}

// Set sets the current context to an object.
func Set(id, typ string) error {
	ctx := &Context{
		LastID:    id,
		LastType:  typ,
		UpdatedAt: time.Now(),
	}
	return Save(ctx)
}

// Get returns the current context ID and type.
func Get() (string, string, error) {
	ctx, err := Load()
	if err != nil {
		return "", "", err
	}
	return ctx.LastID, ctx.LastType, nil
}

// GetID returns just the current context ID.
func GetID() (string, error) {
	id, _, err := Get()
	return id, err
}

// GetType returns just the current context type.
func GetType() (string, error) {
	_, typ, err := Get()
	return typ, err
}

// ResolveTarget resolves a target string (could be "this", an ID, or a handle).
// Returns the resolved ID and whether it was resolved from context.
func ResolveTarget(target string) (string, bool, error) {
	if target == "this" {
		id, err := GetID()
		if err != nil {
			return "", false, fmt.Errorf("no context available: use an explicit ID")
		}
		return id, true, nil
	}
	return target, false, nil
}
