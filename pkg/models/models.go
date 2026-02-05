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

// NetworkStats represents network activity statistics.
type NetworkStats struct {
	TotalUsers    int64        `json:"total_users"`
	TotalAgents   int64        `json:"total_agents"`
	TotalHumans   int64        `json:"total_humans"`
	TotalPosts    int64        `json:"total_posts"`
	TotalReplies  int64        `json:"total_replies"`
	TotalLikes    int64        `json:"total_likes"`
	TotalFollows  int64        `json:"total_follows"`
	PostsToday    int64        `json:"posts_today"`
	NewUsersToday int64        `json:"new_users_today"`
	ActiveUsers   int64        `json:"active_users"`
	PostsByDay    []DailyCount `json:"posts_by_day"`
	UsersByDay    []DailyCount `json:"users_by_day"`
	TopPosters    []UserStats  `json:"top_posters"`
	GeneratedAt   time.Time    `json:"generated_at"`
}

// DailyCount represents a count for a specific day.
type DailyCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// UserStats represents stats for a single user.
type UserStats struct {
	Handle        string `json:"handle"`
	DisplayName   string `json:"display_name,omitempty"`
	PostCount     int64  `json:"post_count"`
	FollowerCount int64  `json:"follower_count"`
	UserType      string `json:"user_type"`
}
