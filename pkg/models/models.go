// Package models defines shared types used across CLI and server.
package models

import "time"

// User represents a user account.
type User struct {
	ID        string    `json:"id"`
	Handle    string    `json:"handle"`
	Name      string    `json:"name,omitempty"`
	Bio       string    `json:"bio,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Post represents a post on the platform.
type Post struct {
	ID          string     `json:"id"`
	AuthorID    string     `json:"author_id"`
	Author      *User      `json:"author,omitempty"`
	Content     string     `json:"content"`
	ContentType string     `json:"content_type,omitempty"`
	Visibility  Visibility `json:"visibility"`
	ReplyTo     *string    `json:"reply_to,omitempty"`
	QuoteOf     *string    `json:"quote_of,omitempty"`
	ReplyCount  int        `json:"reply_count"`
	LikeCount   int        `json:"like_count"`
	ShareCount  int        `json:"share_count"`
	IsLiked     bool       `json:"is_liked"`
	IsShared    bool       `json:"is_shared"`
	IsBookmarked bool      `json:"is_bookmarked"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Visibility defines post visibility levels.
type Visibility string

const (
	VisibilityPublic    Visibility = "public"
	VisibilityUnlisted  Visibility = "unlisted"
	VisibilityFollowers Visibility = "followers"
	VisibilityPrivate   Visibility = "private"
)
