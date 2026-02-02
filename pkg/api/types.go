// Package api defines request/response types for the HTTP API.
package api

// Response wraps all API responses.
type Response[T any] struct {
	OK     bool   `json:"ok"`
	Result T      `json:"result,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Error  *Error `json:"error,omitempty"`
}

// Error represents an API error.
type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Error codes.
const (
	ErrNotFound          = "not_found"
	ErrUnauthorized      = "unauthorized"
	ErrForbidden         = "forbidden"
	ErrBadRequest        = "bad_request"
	ErrConflict          = "conflict"
	ErrChallengeRequired = "challenge_required"
	ErrInternal          = "internal"
)
