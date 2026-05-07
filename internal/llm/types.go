package llm

import "fmt"

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

// Client holds connection config for LLM API calls.
type Client struct {
	APIKey  string
	BaseURL string
}

// NewClient creates a new LLM client.
// baseURL defaults to "https://api.deepseek.com/v1".
func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	return &Client{APIKey: apiKey, BaseURL: baseURL}
}

func (c *Client) authHeader() string {
	return fmt.Sprintf("Bearer %s", c.APIKey)
}
