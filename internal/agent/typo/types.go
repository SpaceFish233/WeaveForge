package typo

import (
	"context"

	"weaveforge/internal/llm"
)

type chatClient interface {
	ChatCompletion(ctx context.Context, messages []llm.Message, model string, opts ...llm.ChatOption) (string, error)
	HasConfig() bool
}

// TypoSuggestion represents a detected typo with its location and suggested fix.
type TypoSuggestion struct {
	Sentence    string `json:"sentence"`
	StartIndex  int    `json:"start_index"`
	EndIndex    int    `json:"end_index"`
	ErrorWord   string `json:"error_word"`
	Suggestion  string `json:"suggestion"`
}
