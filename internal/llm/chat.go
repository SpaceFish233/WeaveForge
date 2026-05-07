package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

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

	httpClient := &http.Client{Timeout: 120 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("llm: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llm: read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: status %d: %s", resp.StatusCode, string(respBody))
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
	// Build Anthropic-style messages from OpenAI-format messages
	type anthropicMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	// Extract system message if present
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
		"model":     model,
		"messages":  msgs,
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

	httpClient := &http.Client{Timeout: 120 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("llm: anthropic failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llm: anthropic read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: anthropic status %d: %s", resp.StatusCode, string(respBody))
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
