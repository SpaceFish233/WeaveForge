package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const maxRetries = 3

// truncateBody limits response body logged in errors to avoid leaking sensitive data.
func truncateBody(body []byte, maxLen int) string {
	if len(body) <= maxLen {
		return string(body)
	}
	return string(body[:maxLen]) + "..."
}

// doWithRetry executes an HTTP request with exponential backoff for retryable errors (429, 5xx).
// The request body is cached before the loop so it can be re-sent on retry.
func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	// Read the request body into a buffer so we can re-create it on retry.
	var reqBodyBytes []byte
	if req.Body != nil {
		var err error
		reqBodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("read request body: %w", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			baseSec := 1 << (attempt - 1) // 1, 2 seconds
			backoff := time.Duration(float64(baseSec) * (0.5 + rand.Float64()*0.5) * float64(time.Second))
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Re-create request body for each attempt
		if reqBodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp, body, nil
		}

		// Only retry on 429 (rate limit) and 5xx (server error)
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("status %d: %s", resp.StatusCode, truncateBody(body, 500))
			continue
		}

		// Non-retryable error (4xx except 429)
		return nil, nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncateBody(body, 500))
	}
	return nil, nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// ChatCompletion calls an OpenAI-compatible chat API. Also supports Anthropic
// format when using the anthropic provider (model starts with "claude-").
func (c *Client) ChatCompletion(ctx context.Context, messages []Message, model string, opts ...ChatOption) (string, error) {
	if model == "" {
		model = "deepseek-chat"
	}

	// Detect Anthropic format (Claude models)
	if strings.HasPrefix(model, "claude-") || strings.HasPrefix(model, "anthropic.") {
		return c.anthropicChat(ctx, messages, model, opts...)
	}

	req := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}
	if len(opts) > 0 {
		req["temperature"] = opts[0].Temperature
		if opts[0].MaxTokens > 0 {
			req["max_tokens"] = opts[0].MaxTokens
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("llm: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(c.BaseURL, "/")+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("llm: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.authHeader())

	_, respBody, err := c.doWithRetry(ctx, httpReq)
	if err != nil {
		return "", fmt.Errorf("llm: request failed: %w", err)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("llm: parse: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("llm: no choices")
	}
	return result.Choices[0].Message.Content, nil
}

// anthropicChat implements Anthropic's Messages API format.
func (c *Client) anthropicChat(ctx context.Context, messages []Message, model string, opts ...ChatOption) (string, error) {
	type anthropicMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	systemContent := ""
	var msgs []anthropicMsg
	for _, m := range messages {
		if m.Role == "system" {
			systemContent = m.Content
			continue
		}
		msgs = append(msgs, anthropicMsg{Role: m.Role, Content: m.Content})
	}
	if len(msgs) == 0 {
		msgs = append(msgs, anthropicMsg{Role: "user", Content: "Say OK"})
	}

	req := map[string]interface{}{
		"model":      model,
		"messages":   msgs,
		"max_tokens": 4096,
	}
	if systemContent != "" {
		req["system"] = systemContent
	}
	if len(opts) > 0 {
		req["temperature"] = opts[0].Temperature
		if opts[0].MaxTokens > 0 {
			req["max_tokens"] = opts[0].MaxTokens
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("llm: anthropic marshal: %w", err)
	}

	baseURL := strings.TrimRight(c.BaseURL, "/")
	if !strings.HasSuffix(baseURL, "/v1") && strings.Contains(baseURL, "anthropic") {
		baseURL += "/v1"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/messages", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("llm: anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	_, respBody, err := c.doWithRetry(ctx, httpReq)
	if err != nil {
		return "", fmt.Errorf("llm: anthropic failed: %w", err)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("llm: anthropic parse: %w", err)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("llm: anthropic empty content")
	}
	return result.Content[0].Text, nil
}
