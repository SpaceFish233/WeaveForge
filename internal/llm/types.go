package llm

import (
	"fmt"
	"net/http"
	"time"
)

// Message represents a chat message in OpenAI-compatible API format.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatOption configures the completion request.
type ChatOption struct {
	Temperature float64
	MaxTokens   int
}

type Client struct {
	APIKey  string
	BaseURL string
	http    *http.Client
}

// NewClient creates a new LLM client with shared HTTP client for connection reuse.
// baseURL defaults to "https://api.deepseek.com/v1".
func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	return &Client{APIKey: apiKey, BaseURL: baseURL, http: &http.Client{Timeout: 120 * time.Second}}
}

// HasConfig returns true if the client has the minimum required configuration
// (API key and base URL are both set).
func (c *Client) HasConfig() bool {
	return c.APIKey != "" && c.BaseURL != ""
}

func (c *Client) authHeader() string {
	return fmt.Sprintf("Bearer %s", c.APIKey)
}
