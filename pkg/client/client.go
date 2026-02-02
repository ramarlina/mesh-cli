// Package client provides an API client for the msh server.
package client

import (
	"net/http"
	"time"
)

// Client is an HTTP client for the msh API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// Option configures the client.
type Option func(*Client)

// New creates a new API client.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithToken sets the authentication token.
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}
