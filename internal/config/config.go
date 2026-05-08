package config

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	LLM      LLMConfig      `json:"llm"`
	VectorDB VectorDBConfig `json:"vector_db"`
	Embedding EmbeddingConfig `json:"embedding"`
}

type LLMConfig struct {
	Provider   string `json:"provider"`    // openai / anthropic
	APIKey     string `json:"api_key"`     // base64 at rest
	BaseURL    string `json:"base_url"`
	ChatModel  string `json:"chat_model"`
}

type EmbeddingConfig struct {
	Engine    string `json:"engine"`     // hash / llamacpp
	ServerPath string `json:"server_path"` // path to llama-server executable
	ModelPath  string `json:"model_path"`  // path to .gguf model file
	Port       int    `json:"port"`        // local port (default 18635)
}

type VectorDBConfig struct{}

func defaults() *Config {
	return &Config{
		LLM: LLMConfig{
			Provider:  "openai",
			BaseURL:   "https://api.deepseek.com/v1",
			ChatModel: "deepseek-chat",
		},
		Embedding: EmbeddingConfig{
			Engine: "hash",
			Port:   18635,
		},
		VectorDB: VectorDBConfig{},
	}
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".weaveforge")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func EncodeKey(key string) string {
	return base64.StdEncoding.EncodeToString([]byte(key))
}

func DecodeKey(encoded string) string {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return encoded
	}
	return string(data)
}

func Load() (*Config, error) {
	cfg := defaults()
	path, err := configPath()
	if err != nil {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Save(cfg *Config) error {
	if !isBase64(cfg.LLM.APIKey) && cfg.LLM.APIKey != "" {
		cfg.LLM.APIKey = EncodeKey(cfg.LLM.APIKey)
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func isBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil && len(s) > 20
}

func DecryptConfig(cfg *Config) {
	cfg.LLM.APIKey = DecodeKey(cfg.LLM.APIKey)
}
