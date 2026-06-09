// Package config loads hepler's runtime configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds the connection details for the OpenAI-compatible endpoint.
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

// Load reads configuration from the HEPLER_* environment variables. The API key
// is optional, since many local servers (Ollama, LM Studio) do not require one.
func Load() (*Config, error) {
	c := &Config{
		BaseURL: os.Getenv("HEPLER_OPENAI_API_BASE"),
		APIKey:  os.Getenv("HEPLER_OPENAI_API_KEY"),
		Model:   os.Getenv("HEPLER_MODEL_NAME"),
	}

	var missing []string
	if c.BaseURL == "" {
		missing = append(missing, "HEPLER_OPENAI_API_BASE")
	}
	if c.Model == "" {
		missing = append(missing, "HEPLER_MODEL_NAME")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment: %s", strings.Join(missing, ", "))
	}
	return c, nil
}
