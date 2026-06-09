// Package config loads hepler's runtime configuration from the environment.
package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// readSecret resolves a secret from one of three sources, in order of
// precedence:
//
//   - <name>_CMD — a shell command whose stdout is the secret. This keeps the
//     secret off disk entirely, e.g. "op read op://vault/hepler/key".
//   - <name>_FILE — a path to a file containing the secret. This mirrors the
//     Docker/systemd secrets convention and keeps the secret out of the
//     process environment.
//   - <name> — the secret inline in the environment variable.
//
// Output from the command and file is trimmed of surrounding whitespace.
func readSecret(name string) (string, error) {
	if cmd := os.Getenv(name + "_CMD"); cmd != "" {
		out, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			return "", fmt.Errorf("running %s_CMD: %w", name, err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	if path := os.Getenv(name + "_FILE"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading %s_FILE: %w", name, err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	return os.Getenv(name), nil
}

// Config holds the connection details for the OpenAI-compatible endpoint.
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

// Load reads configuration from the HEPLER_* environment variables. The API key
// is optional, since many local servers (Ollama, LM Studio) do not require one.
func Load() (*Config, error) {
	apiKey, err := readSecret("HEPLER_OPENAI_API_KEY")
	if err != nil {
		return nil, err
	}

	c := &Config{
		BaseURL: os.Getenv("HEPLER_OPENAI_API_BASE"),
		APIKey:  apiKey,
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
