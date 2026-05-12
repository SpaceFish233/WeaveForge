package vectordb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LlamaCppEmbedder runs a local llama.cpp server process and provides
// embeddings via its OpenAI-compatible HTTP API.
//
// Requires:
//   - llama-server executable installed (https://github.com/ggml-org/llama.cpp)
//   - A GGUF embedding model (e.g., bge-small-zh-q5_k_m.gguf)
type LlamaCppEmbedder struct {
	serverPath string // path to llama-server executable
	modelPath  string // path to .gguf model file
	port       int    // local port (default 18635)
	baseURL    string

	cmd    *exec.Cmd
	client *http.Client

	mu        sync.Mutex
	dimension int // auto-detected on first Embed() call
}

// NewLlamaCppEmbedder connects to or starts a local llama.cpp server.
// If serverPath is empty, tries to detect llama-server in PATH.
// If modelPath is empty, only connects to an already running server.
// The output dimension is auto-detected from the model on the first Embed() call.
func NewLlamaCppEmbedder(serverPath, modelPath string, port int) *LlamaCppEmbedder {
	if port <= 0 {
		port = 18635
	}

	return &LlamaCppEmbedder{
		serverPath: serverPath,
		modelPath:  modelPath,
		port:       port,
		baseURL:    fmt.Sprintf("http://127.0.0.1:%d", port),
		client:     &http.Client{Timeout: 60 * time.Second},
	}
}

// Start launches the llama-server process if configured and not already running.
func (e *LlamaCppEmbedder) Start() error {
	// Check if server is already running
	if e.isRunning() {
		return nil
	}

	if e.modelPath == "" {
		return fmt.Errorf("no GGUF model path configured")
	}

	// Find server executable
	serverPath := e.serverPath
	if serverPath == "" {
		var err error
		serverPath, err = exec.LookPath("llama-server")
		if err != nil {
			// Try common names
			for _, name := range []string{"llama-server.exe", "llama-server"} {
				if p, err := exec.LookPath(name); err == nil {
					serverPath = p
					break
				}
			}
			if serverPath == "" {
				return fmt.Errorf("llama-server not found in PATH")
			}
		}
	}

	// Verify model exists
	if _, err := os.Stat(e.modelPath); err != nil {
		return fmt.Errorf("GGUF model not found: %s", e.modelPath)
	}

	// Build command
	args := []string{
		"-m", e.modelPath,
		"--host", "127.0.0.1",
		"--port", fmt.Sprintf("%d", e.port),
		"--embeddings",
		"-c", "2048", // context size
	}
	cmd := exec.Command(serverPath, args...)
	hideConsoleWindow(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start llama-server: %w", err)
	}
	e.cmd = cmd

	// Wait for server to be ready (up to 120s, large models need time)
	for i := 0; i < 240; i++ {
		if e.isRunning() {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("llama-server did not start within 120s")
}

func (e *LlamaCppEmbedder) isRunning() bool {
	resp, err := e.client.Get(e.baseURL + "/health")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			return true
		}
	}
	// Some llama-server versions don't have /health, try /v1/models
	resp, err = e.client.Get(e.baseURL + "/v1/models")
	if err == nil {
		defer resp.Body.Close()
		return resp.StatusCode == 200
	}
	return false
}

// Embed calls the local llama.cpp server to get an embedding vector.
func (e *LlamaCppEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if !e.isRunning() {
		return nil, fmt.Errorf("llama-server not running")
	}

	body := map[string]interface{}{
		"input": text,
		"model": filepath.Base(e.modelPath),
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.baseURL+"/v1/embeddings", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	f64 := result.Data[0].Embedding
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}

	// Auto-detect dimension from first response
	e.mu.Lock()
	if e.dimension <= 0 || len(f32) > 0 {
		e.dimension = len(f32)
	}
	e.mu.Unlock()

	return f32, nil
}

// Close stops the llama-server process.
func (e *LlamaCppEmbedder) Close() error {
	if e.cmd != nil && e.cmd.Process != nil {
		if err := e.cmd.Process.Kill(); err != nil {
			return err
		}
		// Wait for process to exit to avoid zombie processes
		_ = e.cmd.Wait()
	}
	return nil
}

// Ensure interface compliance
var _ Embedder = (*LlamaCppEmbedder)(nil)
