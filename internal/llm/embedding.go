package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GetEmbedding calls an OpenAI-compatible embeddings API and returns the vector.
// model defaults to "text-embedding-3-small" if empty.
func (c *Client) GetEmbedding(ctx context.Context, text string, model string) ([]float32, error) {
	if model == "" {
		model = "text-embedding-3-small"
	}

	body := map[string]interface{}{
		"model": model,
		"input": text,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(c.BaseURL, "/")+"/embeddings", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("llm: create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm: embed request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("llm: read embed response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm: embed API status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("llm: parse embed response: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("llm: empty embed data")
	}

	// Convert []float64 to []float32
	f64 := result.Data[0].Embedding
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32, nil
}
